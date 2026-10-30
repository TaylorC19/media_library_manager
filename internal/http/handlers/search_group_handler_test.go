package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

type groupSearchStub struct {
	paged func(context.Context, string, string, int, int) (searchsvc.Outcome, error)
	one   func(context.Context, string, string) (searchsvc.Outcome, error)
	all   func(context.Context, string) ([]searchsvc.Group, error)
}

func (s groupSearchStub) Search(ctx context.Context, kind, query string) (searchsvc.Outcome, error) {
	return s.one(ctx, kind, query)
}
func (s groupSearchStub) SearchPage(ctx context.Context, kind, query string, page, size int) (searchsvc.Outcome, error) {
	if s.paged != nil {
		return s.paged(ctx, kind, query, page, size)
	}
	return s.Search(ctx, kind, query)
}
func (s groupSearchStub) SearchAll(ctx context.Context, query string) ([]searchsvc.Group, error) {
	return s.all(ctx, query)
}

func TestGroupedSearchShellDoesNotWaitForProviders(t *testing.T) {
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	stub := groupSearchStub{one: func(context.Context, string, string) (searchsvc.Outcome, error) {
		t.Fatal("shell must not call providers")
		return searchsvc.Outcome{}, nil
	}, all: func(context.Context, string) ([]searchsvc.Group, error) { calls++; return nil, nil }}
	handler := NewSearchHandler(renderer, stub, nil)
	for _, locale := range []string{"en", "ja"} {
		r := httptest.NewRequest("GET", "/"+locale+"/search?q=A%26B&view=list&page_size=20", nil)
		r = r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{ID: primitive.NewObjectID()}))
		w := httptest.NewRecorder()
		middleware.Locale("en")(http.HandlerFunc(handler.Page)).ServeHTTP(w, r)
		html := w.Body.String()
		if w.Code != 200 || strings.Count(html, `data-search-group-url=`) != 5 || calls != 0 {
			t.Fatalf("search shell blocked on providers: %d calls=%d", w.Code, calls)
		}
		for _, want := range []string{"/search/group?", "q=A%26B", "view=list", "page_size=20", "complete=1", "<noscript>", "/static/js/search-groups.js"} {
			if !strings.Contains(html, want) {
				t.Errorf("shell missing %q", want)
			}
		}
		if !strings.Contains(html, `data-groups-empty hidden`) {
			t.Error("no-results state must wait for pending groups")
		}
	}
	r := httptest.NewRequest("GET", "/en/search?q=test&complete=1", nil)
	r = r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{}))
	w := httptest.NewRecorder()
	handler.Page(w, r)
	if calls != 1 || strings.Contains(w.Body.String(), "data-search-group-url=") {
		t.Error("complete fallback must render synchronously")
	}
	for _, path := range []string{"/en/search/group?q=test", "/en/search/group?q=test&media_type=invalid", "/en/search/group?media_type=book"} {
		r := httptest.NewRequest("GET", path, nil)
		r = r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{}))
		w := httptest.NewRecorder()
		handler.Group(w, r)
		if w.Code != 400 {
			t.Errorf("invalid group got %d", w.Code)
		}
	}
	w = httptest.NewRecorder()
	handler.Group(w, httptest.NewRequest("GET", "/en/search/group?q=test&media_type=book", nil))
	if w.Code != 401 {
		t.Error("group requires authentication")
	}
}

func TestIndependentSearchGroupKeepsBadgesAndDoesNotWaitForSlowType(t *testing.T) {
	database := regressionDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	media := repository.NewMediaRecordsRepository(database)
	entries := repository.NewLibraryEntriesRepository(database)
	owner := primitive.NewObjectID()
	record := &domainlib.MediaRecord{Title: "Saved book", MediaType: "book", ProviderRefs: bson.M{"openLibrary": bson.M{"workKey": "OL1W"}}}
	if err := media.Insert(ctx, record); err != nil {
		t.Fatal(err)
	}
	if err := entries.Insert(ctx, &domainlib.LibraryEntry{UserID: owner, MediaRecordID: record.ID, Bucket: "wishlist", MediaType: "book"}); err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	stub := groupSearchStub{one: func(ctx context.Context, kind, query string) (searchsvc.Outcome, error) {
		if kind == "movie" {
			close(started)
			select {
			case <-release:
			case <-ctx.Done():
				return searchsvc.Outcome{}, ctx.Err()
			}
			return searchsvc.Outcome{}, nil
		}
		if kind == "game" {
			return searchsvc.Outcome{}, errors.New("private provider error")
		}
		hits := []domainsearch.Hit{{Title: "Saved book", Provider: "open_library", ExternalID: "OL1W", MediaType: "book"}}
		for i := 0; i < 4; i++ {
			hits = append(hits, domainsearch.Hit{Title: "Other book", Provider: "open_library", ExternalID: "other", MediaType: "book"})
		}
		return searchsvc.Outcome{Hits: hits}, nil
	}}
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewSearchHandler(renderer, stub, libsvc.NewService(media, entries))
	request := func(kind string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/en/search/group?q=test&view=grid&page_size=20&media_type="+kind, nil)
		r = r.WithContext(middleware.WithUser(ctx, &domainauth.User{ID: owner}))
		w := httptest.NewRecorder()
		handler.Group(w, r)
		return w
	}
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- request("movie") }()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("slow group did not start")
	}
	fast := request("book")
	if fast.Code != 200 {
		t.Fatalf("fast group failed: %d %s", fast.Code, fast.Body.String())
	}
	select {
	case <-done:
		t.Error("slow group completed before release")
	default:
	}
	for _, want := range []string{"In Wishlist", "search-results--grid", "Showing: 3 / 5", "media_type=book", "view=grid", "page_size=20", "data-search-detail", "data-has-results=\"true\""} {
		if !strings.Contains(fast.Body.String(), want) {
			t.Errorf("fast group missing %q", want)
		}
	}
	if fast.Header().Get("Cache-Control") != "private, no-store" {
		t.Error("group badges must not be cached")
	}
	close(release)
	select {
	case w := <-done:
		if w.Code != 200 || !strings.Contains(w.Body.String(), `data-has-results="false"`) {
			t.Error("empty group must settle without results")
		}
	case <-ctx.Done():
		t.Fatal("slow group did not finish")
	}
	failed := request("game")
	if failed.Code != 502 || strings.Contains(failed.Body.String(), "private provider error") {
		t.Error("provider failure must be generic and scoped to its group")
	}
}
