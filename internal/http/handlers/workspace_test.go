package handlers

import (
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
