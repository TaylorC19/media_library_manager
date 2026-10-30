package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	domainauth "media_library_manager/internal/domain/auth"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/repository"
	libsvc "media_library_manager/internal/service/library"
	"media_library_manager/internal/views"
)

// Exercise real BSON round trips: stored lists and optional numeric fields have
// different Go types from freshly normalized provider data.
func TestSavedLibraryDetails(t *testing.T) {
	database := regressionDatabase(t)
	ctx := context.Background()
	media := repository.NewMediaRecordsRepository(database)
	entries := repository.NewLibraryEntriesRepository(database)
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewLibraryHandler(renderer, libsvc.NewService(media, entries))
	owner := primitive.NewObjectID()
	image, summary, release := "https://example.com/cover.jpg", "Saved description <script>alert(1)</script>\nSecond paragraph", "2020-01-02"
	runtime := int32(123)
	cases := []struct {
		kind    string
		details bson.M
		want    []string
	}{
		{"movie", bson.M{"directors": []string{"Test Director"}, "cast": []string{"Test Actor"}, "genres": []string{"Drama"}, "runtimeMinutes": &runtime}, []string{"Test Director", "Test Actor", "Drama", "123"}},
		{"tv", bson.M{"creators": []string{"Test Creator"}, "seasons": int32(2), "episodes": int32(16)}, []string{"Test Creator", ">2<", ">16<"}},
		{"album", bson.M{"artists": []string{"Test Artist"}, "label": "Test Label", "catalogNumber": "ABC-42", "releaseCountry": "JP", "trackCount": int32(12)}, []string{"Test Artist", "Test Label", "ABC-42", "JP", ">12<"}},
		{"book", bson.M{"authors": []string{"First Author", "Second Author"}}, []string{"First Author, Second Author"}},
		{"game", bson.M{"developers": []string{"Test Developer"}, "publishers": []string{"Test Publisher"}, "platforms": []string{"PC", "PS5"}, "genres": []string{"Adventure"}}, []string{"Test Developer", "Test Publisher", "PC, PS5", "Adventure"}},
		{"book", bson.M{}, []string{"No cover", "No description available."}},
	}
	request := func(id, locale string, user *domainauth.User) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/"+locale+"/library/"+id, nil)
		route := chi.NewRouteContext()
		route.URLParams.Add("entryId", id)
		r = r.WithContext(context.WithValue(middleware.WithUser(r.Context(), user), chi.RouteCtxKey, route))
		w := httptest.NewRecorder()
		// Use the actual locale middleware so labels and language navigation render together.
		middleware.Locale("en")(http.HandlerFunc(handler.Detail)).ServeHTTP(w, r)
		return w
	}
	for i, tc := range cases {
		record := &domainlib.MediaRecord{Title: "Saved item", MediaType: tc.kind, Source: domainlib.SourceProvider, ImageURL: &image, Summary: &summary, ReleaseDate: &release, Details: tc.details}
		if i == len(cases)-1 {
			record.ImageURL = nil
			record.Summary = nil
			record.Source = domainlib.SourceManual
		}
		if err := media.Insert(ctx, record); err != nil {
			t.Fatal(err)
		}
		for _, bucket := range []string{"catalog", "wishlist"} {
			note := "Private copy note"
			entry := &domainlib.LibraryEntry{UserID: owner, MediaRecordID: record.ID, Bucket: bucket, MediaType: tc.kind, Notes: &note}
			if err := entries.Insert(ctx, entry); err != nil {
				t.Fatal(err)
			}
			for _, locale := range []string{"en", "ja"} {
				for _, role := range []string{"general", "demo"} {
					w := request(entry.ID.Hex(), locale, &domainauth.User{ID: owner, Role: role})
					html := w.Body.String()
					if w.Code != 200 {
						t.Fatalf("%s/%s/%s: %d %s", tc.kind, bucket, locale, w.Code, html)
					}
					wants := append([]string{"Saved item", note}, tc.want...)
					if i != len(cases)-1 {
						wants = append(wants, image, "Saved description", release)
					}
					for _, want := range wants {
						if i == len(cases)-1 && locale == "ja" && (want == "No cover" || want == "No description available.") {
							continue
						}
						if !strings.Contains(html, want) {
							t.Errorf("%s/%s/%s missing %q", tc.kind, bucket, locale, want)
						}
					}
					if strings.Contains(html, "<script>alert(1)</script>") || strings.Contains(html, "ZgotmplZ") {
						t.Error("unsafe metadata rendering")
					}
					if strings.Contains(html, `action="/library/`) != (role == "general") {
						t.Error("write controls must respect account role")
					}
				}
			}
			if w := request(entry.ID.Hex(), "en", &domainauth.User{ID: primitive.NewObjectID(), Role: "general"}); w.Code != 404 || strings.Contains(w.Body.String(), note) {
				t.Error("saved detail must remain private to its owner")
			}
		}
	}
}
