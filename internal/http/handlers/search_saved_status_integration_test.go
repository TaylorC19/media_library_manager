package handlers

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	domainauth "media_library_manager/internal/domain/auth"
	domainlib "media_library_manager/internal/domain/library"
	domainsearch "media_library_manager/internal/domain/search"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/repository"
	libsvc "media_library_manager/internal/service/library"
	"media_library_manager/internal/views"
)

func TestSearchSavedBadgesArePersonalAndMatchProviderIdentity(t *testing.T) {
	database := regressionDatabase(t)
	ctx := context.Background()
	media := repository.NewMediaRecordsRepository(database)
	entries := repository.NewLibraryEntriesRepository(database)
	owner, other := primitive.NewObjectID(), primitive.NewObjectID()
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	service := libsvc.NewService(media, entries)
	handler := NewSearchHandler(renderer, nil, service)
	cases := []struct {
		hit                  domainsearch.Hit
		refs                 bson.M
		buckets              []string
		collection, wishlist bool
	}{
		{domainsearch.Hit{Title: "Movie", Provider: "tmdb", ExternalID: "42", MediaType: "movie", TMDBKind: "movie"}, bson.M{"tmdb": bson.M{"id": "42", "kind": "movie"}}, []string{"catalog", "catalog"}, true, false},
		{domainsearch.Hit{Title: "TV", Provider: "tmdb", ExternalID: "42", MediaType: "tv", TMDBKind: "tv"}, bson.M{"tmdb": bson.M{"id": "42", "kind": "tv"}}, []string{"wishlist"}, false, true},
		{domainsearch.Hit{Title: "Album", Provider: "musicbrainz", ExternalID: "release-1", MediaType: "album"}, bson.M{"musicbrainz": bson.M{"releaseId": "release-1"}}, []string{"catalog", "wishlist"}, true, true},
		{domainsearch.Hit{Title: "Book", Provider: "open_library", ExternalID: "OL1W", MediaType: "book"}, bson.M{"openLibrary": bson.M{"workKey": "OL1W"}}, []string{"catalog"}, true, false},
		{domainsearch.Hit{Title: "Game", Provider: "rawg", ExternalID: "100", MediaType: "game"}, bson.M{"rawg": bson.M{"id": "100"}}, []string{"wishlist"}, false, true},
		{domainsearch.Hit{Title: "Other user's book", Provider: "open_library", ExternalID: "OL2W", MediaType: "book"}, bson.M{"openLibrary": bson.M{"workKey": "OL2W"}}, nil, false, false},
	}
	hits := []domainsearch.Hit{}
	var remove *domainlib.LibraryEntry
	for _, tc := range cases {
		record := &domainlib.MediaRecord{Title: tc.hit.Title, MediaType: tc.hit.MediaType, Source: domainlib.SourceProvider, ProviderRefs: tc.refs}
		if err := media.Insert(ctx, record); err != nil {
			t.Fatal(err)
		}
		for _, bucket := range tc.buckets {
			entry := &domainlib.LibraryEntry{UserID: owner, MediaRecordID: record.ID, MediaType: record.MediaType, Bucket: bucket}
			if err := entries.Insert(ctx, entry); err != nil {
				t.Fatal(err)
			}
			if tc.hit.Title == "Game" {
				remove = entry
			}
		}
		// All records also belong to another account in both buckets. Those saves
		// must never supplement the signed-in user's status.
		for _, bucket := range []string{"catalog", "wishlist"} {
			entry := &domainlib.LibraryEntry{UserID: other, MediaRecordID: record.ID, MediaType: record.MediaType, Bucket: bucket}
			if err := entries.Insert(ctx, entry); err != nil {
				t.Fatal(err)
			}
		}
		hits = append(hits, tc.hit)
	}
	// Equal title and type with a different provider ID is not the saved movie.
	hits = append(hits, domainsearch.Hit{Title: "Movie", Provider: "tmdb", ExternalID: "999", MediaType: "movie", TMDBKind: "movie"})
	for _, grouped := range []bool{false, true} {
		for _, view := range []string{"list", "grid"} {
			for _, locale := range []string{"en", "ja"} {
				data := map[string]any{"Locale": locale, "Query": "test", "SearchError": "", "SearchWarnings": []string{}, "AllMedia": grouped, "HasResults": true, "View": view, "Hits": hits, "SearchGroups": []searchGroup{}, "Pagination": map[string]any{"TotalItems": len(hits), "PageSize": 10}}
				if grouped {
					data["Hits"] = []domainsearch.Hit{}
					data["SearchGroups"] = []searchGroup{{MediaType: "movie", Hits: hits[:2], Total: 2}, {MediaType: "book", Hits: hits[2:], Total: len(hits) - 2}}
				}
				r := httptest.NewRequest("GET", "/"+locale+"/search?q=test", nil)
				r = r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{ID: owner, Role: "demo"}))
				handler.addSavedStatus(r, data)
				enriched := data["Hits"].([]domainsearch.Hit)
				for _, group := range data["SearchGroups"].([]searchGroup) {
					enriched = append(enriched, group.Hits...)
				}
				for i, hit := range enriched {
					collection, wishlist := false, false
					if i < len(cases) {
						collection, wishlist = cases[i].collection, cases[i].wishlist
					}
					if hit.InCollection != collection || hit.InWishlist != wishlist {
						t.Errorf("%s status = %v/%v, want %v/%v", hit.Title, hit.InCollection, hit.InWishlist, collection, wishlist)
					}
				}
				var out bytes.Buffer
				if err := renderer.Render(&out, "partials/search_results", data); err != nil {
					t.Fatal(err)
				}
				if strings.Count(out.String(), `class="saved-badge saved-badge--collection"`) != 3 || strings.Count(out.String(), `class="saved-badge saved-badge--wishlist"`) != 3 {
					t.Errorf("wrong badge count, grouped=%v view=%s locale=%s", grouped, view, locale)
				}
				if strings.Contains(out.String(), "search.results.in") {
					t.Error("untranslated badge")
				}
			}
		}
	}
	for _, hit := range hits {
		if hit.InCollection || hit.InWishlist {
			t.Error("enrichment mutated provider results")
		}
	}
	// Fresh results must reflect removal, even though provider hits did not change.
	if err := entries.DeleteByIDAndUser(ctx, remove.ID, owner); err != nil {
		t.Fatal(err)
	}
	enriched, err := service.SavedSearchHits(ctx, owner, hits)
	if err != nil {
		t.Fatal(err)
	}
	if enriched[4].InWishlist {
		t.Error("removed save retained stale badge")
	}
	// A database failure should preserve results and clearly signal unknown status.
	failedCtx, cancel := context.WithCancel(ctx)
	cancel()
	r := httptest.NewRequest("GET", "/en/search", nil)
	r = r.WithContext(middleware.WithUser(failedCtx, &domainauth.User{ID: owner}))
	data := map[string]any{"Hits": hits, "SearchGroups": []searchGroup{}, "SearchWarnings": []string{}}
	handler.addSavedStatus(r, data)
	if len(data["Hits"].([]domainsearch.Hit)) != len(hits) || len(data["SearchWarnings"].([]string)) != 1 {
		t.Error("status failure must keep search results and show a warning")
	}
}
