package handlers

import (
	"context"
	"fmt"
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
	searchsvc "media_library_manager/internal/service/search"
	"media_library_manager/internal/views"
)

func TestSearchRemotePageRenderingAndStableControls(t *testing.T) {
	database := regressionDatabase(t)
	ctx := context.Background()
	media := repository.NewMediaRecordsRepository(database)
	entries := repository.NewLibraryEntriesRepository(database)
	owner := primitive.NewObjectID()
	rec := &domainlib.MediaRecord{MediaType: "book", ProviderRefs: bson.M{"openLibrary": bson.M{"workKey": "OL61W"}}}
	if err := media.Insert(ctx, rec); err != nil {
		t.Fatal(err)
	}
	if err := entries.Insert(ctx, &domainlib.LibraryEntry{UserID: owner, MediaRecordID: rec.ID, Bucket: "catalog", MediaType: "book"}); err != nil {
		t.Fatal(err)
	}
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	stub := groupSearchStub{paged: func(_ context.Context, kind, query string, page, size int) (searchsvc.Outcome, error) {
		if kind != "book" || query != "A&B" || size != 10 {
			t.Error("handler lost search criteria")
		}
		out := searchsvc.Outcome{Page: page, PageSize: size, Total: 125}
		for i := (page-1)*size + 1; i <= page*size && i <= 125; i++ {
			out.Hits = append(out.Hits, domainsearch.Hit{Title: fmt.Sprintf("Remote result %d", i), MediaType: kind, Provider: "open_library", ExternalID: fmt.Sprintf("OL%dW", i)})
		}
		return out, nil
	}}
	handler := NewSearchHandler(renderer, stub, libsvc.NewService(media, entries))
	for _, page := range []int{1, 7, 13} {
		for _, view := range []string{"grid", "list"} {
			r := httptest.NewRequest("GET", fmt.Sprintf("/en/search?media_type=book&q=A%%26B&page=%d&page_size=10&view=%s", page, view), nil)
			r.Header.Set("HX-Request", "true")
			r = r.WithContext(middleware.WithUser(ctx, &domainauth.User{ID: owner}))
			w := httptest.NewRecorder()
			handler.Page(w, r)
			html := w.Body.String()
			if w.Code != 200 {
				t.Fatalf("remote page: %d %s", w.Code, html)
			}
			if !strings.Contains(html, fmt.Sprintf("Remote result %d", (page-1)*10+1)) {
				t.Error("handler sliced a remote page a second time")
			}
			if strings.Count(html, "pagination__previous") != 1 || strings.Count(html, "pagination__next") != 1 {
				t.Error("both pagination positions must remain present")
			}
			if !strings.Contains(html, fmt.Sprintf(`aria-current="page" disabled>%d</button>`, page)) {
				t.Error("current page must be a disabled button")
			}
			if strings.Contains(html, `class="button-link pagination__previous" type="button" disabled`) != (page == 1) {
				t.Error("previous state incorrect")
			}
			if strings.Contains(html, `class="button-link pagination__next" type="button" disabled`) != (page == 13) {
				t.Error("next state incorrect")
			}
			if page == 7 && !strings.Contains(html, "In Collection") {
				t.Error("remote pages must keep personal badges")
			}
			if !strings.Contains(html, "q=A%26B") || !strings.Contains(html, "view="+view) || strings.Contains(html, "hx-get=") {
				t.Error("search pagination must preserve context and full navigation")
			}
		}
	}
}
