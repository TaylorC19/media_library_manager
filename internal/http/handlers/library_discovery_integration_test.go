package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	domainauth "media_library_manager/internal/domain/auth"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/repository"
	libsvc "media_library_manager/internal/service/library"
	"media_library_manager/internal/views"
)

func TestLibraryDiscoveryFiltersSortsAndPreferences(t *testing.T) {
	database := regressionDatabase(t)
	ctx := context.Background()
	owner, other := primitive.NewObjectID(), primitive.NewObjectID()
	media := repository.NewMediaRecordsRepository(database)
	entries := repository.NewLibraryEntriesRepository(database)
	fixtures := []struct {
		title, kind, creatorKey, creator string
		year                             int32
		tags                             []string
		bucket                           string
		user                             primitive.ObjectID
	}{
		{"Zulu", "movie", "directors", "James Cameron", 2000, []string{"Sci-Fi", "favorite"}, "catalog", owner},
		{"Alpha", "book", "authors", "Ursula Le Guin", 2020, []string{"Read", "favorite"}, "catalog", owner},
		{"Beta", "game", "developers", "CD Projekt", 2024, []string{"sci-fi", "gaming"}, "catalog", owner},
		{"Gamma", "album", "artists", "Music Maker", 1990, []string{"Read"}, "catalog", owner},
		{"Delta", "tv", "creators", "Jane Creator", 0, []string{"Watch"}, "catalog", owner},
		{"Wish", "book", "authors", "Other Author", 2010, []string{"wishlist-only"}, "wishlist", owner},
		{"Private", "book", "authors", "Private Author", 2010, []string{"other-user-secret"}, "catalog", other},
	}
	for i, fixture := range fixtures {
		record := &domainlib.MediaRecord{Title: fixture.title, MediaType: fixture.kind, Details: bson.M{fixture.creatorKey: []string{fixture.creator}}, Source: "manual"}
		if fixture.year > 0 {
			value := fixture.year
			record.Year = &value
		}
		if err := media.Insert(ctx, record); err != nil {
			t.Fatal(err)
		}
		entry := &domainlib.LibraryEntry{UserID: fixture.user, MediaRecordID: record.ID, MediaType: fixture.kind, Bucket: fixture.bucket, Tags: fixture.tags}
		if err := entries.Insert(ctx, entry); err != nil {
			t.Fatal(err)
		}
		// Give each entry a deterministic added date, independent of insertion speed.
		if _, err := database.Collection("library_entries").UpdateByID(ctx, entry.ID, bson.M{"$set": bson.M{"createdAt": time.Unix(int64(i), 0)}}); err != nil {
			t.Fatal(err)
		}
	}
	service := libsvc.NewService(media, entries)
	for _, tc := range []struct {
		filters    libsvc.ListBucketFilters
		page, size int
		want       []string
		total      int
	}{
		{libsvc.ListBucketFilters{Sort: "title_asc"}, 2, 2, []string{"Delta", "Gamma"}, 5},
		{libsvc.ListBucketFilters{Sort: "year_desc"}, 1, 10, []string{"Beta", "Alpha", "Zulu", "Gamma", "Delta"}, 5},
		{libsvc.ListBucketFilters{Sort: "year_asc"}, 1, 10, []string{"Gamma", "Zulu", "Alpha", "Beta", "Delta"}, 5},
		{libsvc.ListBucketFilters{Sort: "recent"}, 1, 2, []string{"Delta", "Gamma"}, 5},
		{libsvc.ListBucketFilters{Tag: "SCI-FI", Sort: "title_asc"}, 1, 10, []string{"Beta", "Zulu"}, 2},
		{libsvc.ListBucketFilters{Query: "CAMERON", Tag: "Sci-Fi"}, 1, 10, []string{"Zulu"}, 1},
		{libsvc.ListBucketFilters{Query: "favorite", Sort: "title_desc"}, 1, 10, []string{"Zulu", "Alpha"}, 2},
		{libsvc.ListBucketFilters{Query: "Maker"}, 1, 10, []string{"Gamma"}, 1},
	} {
		out, err := service.ListBucket(ctx, owner, "catalog", tc.filters, tc.page, tc.size)
		if err != nil {
			t.Fatal(err)
		}
		if out.TotalItems != tc.total || len(out.Items) != len(tc.want) {
			t.Fatalf("wrong filtered page for %+v: %+v", tc.filters, out)
		}
		for i, want := range tc.want {
			if out.Items[i].Media.Title != want {
				t.Errorf("got %s want %s", out.Items[i].Media.Title, want)
			}
		}
		for _, tag := range out.Tags {
			if tag.Value == "other-user-secret" || tag.Value == "wishlist-only" {
				t.Error("tag choices leaked another account or bucket")
			}
		}
	}
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewLibraryHandler(renderer, service)
	request := func(path string, cookie *http.Cookie, htmx bool) *httptest.ResponseRecorder {
		r := preferenceRequest(path, owner)
		if cookie != nil {
			r.AddCookie(cookie)
		}
		r = r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{ID: owner, Role: "general"}))
		if htmx {
			r.Header.Set("HX-Request", "true")
		}
		w := httptest.NewRecorder()
		middleware.Locale("en")(http.HandlerFunc(handler.Catalog)).ServeHTTP(w, r)
		return w
	}
	first := request("/en/?q=&media_type=&format=&tag=Sci-Fi&sort=title_asc&view=grid&page_size=1", nil, false)
	html := first.Body.String()
	if first.Code != 200 || !strings.Contains(html, `value="sci-fi" selected`) || !strings.Contains(html, `value="title_asc" selected`) || !strings.Contains(html, "tag=sci-fi") || !strings.Contains(html, "sort=title_asc") {
		t.Fatalf("discovery controls/context missing: %d %s", first.Code, html)
	}
	cookie := preferenceCookie(t, first)
	back := request("/ja/", cookie, false)
	restoredURL, _ := url.Parse(back.Header().Get("Location"))
	if back.Code != 302 || restoredURL.Query().Get("sort") != "title_asc" || restoredURL.Query().Get("tag") != "sci-fi" {
		t.Error("sort/tag preferences did not survive return visit")
	}
	restored := request(back.Header().Get("Location"), cookie, false)
	if restored.Code != 200 || !strings.Contains(restored.Body.String(), `value="sci-fi" selected`) {
		t.Error("restored discovery filters missing")
	}
	next := request("/en/?q=&media_type=&format=&tag=sci-fi&sort=title_asc&view=grid&page_size=1&page=2", cookie, true)
	if next.Code != 200 || !strings.Contains(next.Body.String(), "Zulu") || !strings.Contains(next.Body.String(), "sort=title_asc") || !strings.Contains(next.Body.String(), `id="library-clear-filters" hx-swap-oob="outerHTML"`) {
		t.Error("HTMX pagination lost sort/tag")
	}
	clear := request("/en/?q=&media_type=&format=&tag=&sort=title_asc&view=grid&page_size=1", cookie, false)
	cleared := preferenceCookie(t, clear)
	back = request("/en/", cleared, false)
	restoredURL, _ = url.Parse(back.Header().Get("Location"))
	if restoredURL.Query().Get("tag") != "" || restoredURL.Query().Get("sort") != "title_asc" {
		t.Error("clear should remove saved tag and retain sort")
	}
	if bad := request("/en/?sort=invalid", nil, false); bad.Code != 400 {
		t.Error("invalid sort should return 400")
	}
}
