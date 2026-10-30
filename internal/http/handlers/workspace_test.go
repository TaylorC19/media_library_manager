package handlers

import (
	domainsearch "media_library_manager/internal/domain/search"
	searchsvc "media_library_manager/internal/service/search"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"media_library_manager/internal/http/middleware"
)

func TestLegacyScanRedirectPreservesContext(t *testing.T) {
	for _, locale := range []string{"en", "ja"} {
		r := httptest.NewRequest(http.MethodGet, "/"+locale+"/scan?bucket=wishlist&media_type=book&q=A%2FB", nil)
		w := httptest.NewRecorder()
		middleware.Locale("en")(http.HandlerFunc(LegacyScanPage)).ServeHTTP(w, r)
		if w.Code != http.StatusFound {
			t.Fatalf("status = %d", w.Code)
		}
		u, err := url.Parse(w.Header().Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		if u.Path != "/"+locale+"/search" {
			t.Errorf("path = %q", u.Path)
		}
		for key, want := range map[string]string{"bucket": "wishlist", "media_type": "book", "q": "A/B", "mode": "barcode"} {
			if got := u.Query().Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
	}
}

func TestSearchPaginationOnlyIncludesLookupCriteria(t *testing.T) {
	values := searchValues("A/B", "book")
	pagination := buildPagination("/ja/search", values, 1, 10, 25)
	u, err := url.Parse(pagination.NextURL)
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"bucket": "", "media_type": "book", "q": "A/B", "page": "2"} {
		if got := u.Query().Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestLibraryViewURLsKeepFiltersAndPagination(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/en/?q=A%2FB&media_type=book&format=paperback&page=2&view=list", nil)
	u, err := url.Parse(libraryViewURL(r, "/en/", "grid"))
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"q": "A/B", "media_type": "book", "format": "paperback", "page": "2", "view": "grid"} {
		if got := u.Query().Get(key); got != want {
			t.Errorf("%s=%q, want %q", key, got, want)
		}
	}
	for _, view := range []string{"", "invalid", "list", "grid"} {
		r = httptest.NewRequest(http.MethodGet, "/en/?view="+view, nil)
		want := "list"
		if view == "grid" {
			want = "grid"
		}
		if got := libraryView(r); got != want {
			t.Errorf("view=%q, want %q", got, want)
		}
	}
}

func TestGroupedSearchPreviewsAndSeeAllLinks(t *testing.T) {
	groups := []searchsvc.Group{
		{MediaType: "movie", Outcome: searchsvc.Outcome{Hits: []domainsearch.Hit{{Title: "1"}, {Title: "2"}, {Title: "3"}, {Title: "4"}}}},
		{MediaType: "tv"},
		{MediaType: "album", Outcome: searchsvc.Outcome{Warnings: []string{"search.warnings.providerUnavailable"}}},
		{MediaType: "book", Outcome: searchsvc.Outcome{Hits: []domainsearch.Hit{{Title: "Book"}}}},
	}
	previews, hasResults := buildSearchGroups("/ja/search", "A/B & C", 10, "grid", groups)
	if !hasResults || len(previews) != 3 {
		t.Fatalf("unexpected groups: %#v", previews)
	}
	if len(previews[0].Hits) != 3 || previews[0].Total != 4 {
		t.Error("preview must cap at three without losing total")
	}
	if previews[1].MediaType != "album" || len(previews[1].Warnings) != 1 {
		t.Error("scoped warnings must survive empty results")
	}
	for _, group := range previews {
		u, err := url.Parse(group.SeeAllURL)
		if err != nil {
			t.Fatal(err)
		}
		for key, want := range map[string]string{"q": "A/B & C", "media_type": group.MediaType, "page": "1", "page_size": "10", "view": "grid"} {
			if got := u.Query().Get(key); got != want {
				t.Errorf("%s=%q, want %q", key, got, want)
			}
		}
	}
	empty, hasResults := buildSearchGroups("/en/search", "x", 10, "grid", []searchsvc.Group{{MediaType: "tv"}})
	if len(empty) != 0 || hasResults {
		t.Error("empty types should not become result groups")
	}
}

func TestSearchLayoutDefaultAndPagination(t *testing.T) {
	for _, param := range []string{"", "invalid", "grid", "list"} {
		r := httptest.NewRequest(http.MethodGet, "/en/search?view="+param, nil)
		want := "grid"
		if param == "list" {
			want = "list"
		}
		if got := searchView(r); got != want {
			t.Errorf("view %q: got %q, want %q", param, got, want)
		}
	}
	for _, view := range []string{"grid", "list"} {
		pagination := buildPagination("/en/search", searchPageValues("cyberpunk", "tv", view), 1, 10, 25)
		u, err := url.Parse(pagination.NextURL)
		if err != nil {
			t.Fatal(err)
		}
		for key, want := range map[string]string{"q": "cyberpunk", "media_type": "tv", "view": view, "page": "2"} {
			if got := u.Query().Get(key); got != want {
				t.Errorf("%s=%q, want %q", key, got, want)
			}
		}
	}
}

func TestOutOfRangePagesShowTheSameLastPageAsTheFooter(t *testing.T) {
	for _, page := range []int{2, 3, 999} {
		start, end := pageBounds(15, page, 10)
		pagination := buildPagination("/en/search", nil, page, 10, 15)
		if start != 10 || end != 15 || pagination.Page != 2 || pagination.StartItem != start+1 || pagination.EndItem != end {
			t.Errorf("page %d: slice/footer disagree", page)
		}
	}
	if start, end := pageBounds(0, 999, 10); start != 0 || end != 0 {
		t.Error("empty pagination must remain empty")
	}
}

func TestDeleteReturnsToTheOriginalBucket(t *testing.T) {
	if libraryBucketPath("en", "catalog") != "/en/" {
		t.Error("collection delete should return to My Library")
	}
	if libraryBucketPath("ja", "wishlist") != "/ja/wishlist" {
		t.Error("wishlist delete should stay in Wishlist")
	}
}
