package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
	domainauth "media_library_manager/internal/domain/auth"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/repository"
	libsvc "media_library_manager/internal/service/library"
	"media_library_manager/internal/views"
)

func TestLibraryReturnsWithSavedFiltersAndLayout(t *testing.T) {
	database := regressionDatabase(t)
	ctx := context.Background()
	owner := primitive.NewObjectID()
	media := repository.NewMediaRecordsRepository(database)
	entries := repository.NewLibraryEntriesRepository(database)
	format := "paperback"
	for _, title := range []string{"Remembered Book", "Unrelated Book"} {
		rec := &domainlib.MediaRecord{Title: title, SortTitle: &title, MediaType: "book", Source: "manual"}
		if err := media.Insert(ctx, rec); err != nil {
			t.Fatal(err)
		}
		if err := entries.Insert(ctx, &domainlib.LibraryEntry{UserID: owner, MediaRecordID: rec.ID, Bucket: "catalog", MediaType: "book", Format: &format}); err != nil {
			t.Fatal(err)
		}
	}
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewLibraryHandler(renderer, libsvc.NewService(media, entries))
	request := func(path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		r := preferenceRequest(path, owner, cookies...)
		r = r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{ID: owner, Role: "general"}))
		w := httptest.NewRecorder()
		middleware.Locale("en")(http.HandlerFunc(handler.Catalog)).ServeHTTP(w, r)
		return w
	}
	first := request("/en/?q=Remembered&media_type=book&format=paperback&view=grid&page_size=20&page=4")
	if first.Code != 200 {
		t.Fatalf("first visit failed: %d %s", first.Code, first.Body.String())
	}
	cookie := preferenceCookie(t, first)
	back := request("/ja/", cookie)
	if back.Code != 302 {
		t.Fatal("return must resolve saved filters")
	}
	restored := request(back.Header().Get("Location"), cookie)
	html := restored.Body.String()
	if restored.Code != 200 || !strings.Contains(html, "entry-list--grid") || !strings.Contains(html, "Remembered Book") || strings.Contains(html, "Unrelated Book") {
		t.Fatal("restored page did not use saved layout and filters")
	}
	if !strings.Contains(html, `value="Remembered"`) || !strings.Contains(html, `value="paperback" selected`) {
		t.Error("restored filters must appear in the form")
	}
	if !strings.Contains(html, "q=&amp;media_type=&amp;format=&amp;view=grid&amp;page_size=20") {
		t.Error("clear link must explicitly clear saved fields")
	}
	clear := request("/en/?q=&media_type=&format=&view=grid&page_size=20", cookie)
	clearedCookie := preferenceCookie(t, clear)
	back = request("/en/", clearedCookie)
	restored = request(back.Header().Get("Location"), clearedCookie)
	if restored.Code != 200 || !strings.Contains(restored.Body.String(), "Unrelated Book") || !strings.Contains(restored.Body.String(), "Remembered Book") {
		t.Error("cleared saved filters must stay clear across visits")
	}
}
