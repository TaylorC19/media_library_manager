package views

import (
	"bytes"
	"go.mongodb.org/mongo-driver/bson"
	mediasvc "media_library_manager/internal/service/media"
	"strings"
	"testing"

	domainauth "media_library_manager/internal/domain/auth"
	domainlib "media_library_manager/internal/domain/library"
	domainsearch "media_library_manager/internal/domain/search"
)

// Render the actual shells, partials, and locales together: compilation alone
// does not catch missing template data or errors in nested template calls.
func TestLibraryAndSearchWorkspaces(t *testing.T) {
	renderer, err := NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, locale := range []string{"en", "ja"} {
		for _, bucket := range []string{"catalog", "wishlist"} {
			t.Run(locale+"/"+bucket, func(t *testing.T) {
				data := map[string]any{
					"Locale": locale, "Path": "/" + locale + "/", "User": &domainauth.User{Username: "test"},
					"MediaTypes": domainlib.MediaTypes, "Bucket": bucket,
					"Values":       map[string]string{"q": "A&B", "media_type": "book"},
					"CatalogCount": 3, "WishlistCount": 2, "ListTotal": 0,
					"SearchError": "", "SearchWarnings": []string{}, "Query": "A&B",
					"Hits":            []domainsearch.Hit{{Title: "A&B", Provider: "open_library", ExternalID: "OL1W", MediaType: "book"}},
					"Pagination":      map[string]any{"TotalItems": 0, "PageSize": 10},
					"ContentTemplate": "pages/catalog.content", "ListPath": "/" + locale + "/",
				}
				if bucket == "wishlist" {
					data["ContentTemplate"] = "pages/wishlist.content"
				}
				var library bytes.Buffer
				if err := renderer.Render(&library, "pages/library_list.html", data); err != nil {
					t.Fatal(err)
				}
				for _, want := range []string{"/search?media_type=", `class="section-tabs"`, `aria-current="page"`} {
					if !strings.Contains(library.String(), want) {
						t.Errorf("library missing %q", want)
					}
				}
				if strings.Contains(library.String(), "/scan") || strings.Contains(library.String(), "/library/new") {
					t.Error("library must lead to Search for adding")
				}

				data["Path"] = "/" + locale + "/search"
				data["ContentTemplate"] = "pages/search.content"
				data["ScanOpen"] = true
				data["Pagination"] = map[string]any{"TotalItems": 25, "PageSize": 10, "StartItem": 1, "EndItem": 10, "NextURL": "/" + locale + "/search?media_type=" + "&page=2"}
				delete(data, "ListPath")
				var search bytes.Buffer
				if err := renderer.Render(&search, "pages/search.html", data); err != nil {
					t.Fatal(err)
				}
				for _, want := range []string{`id="scan-root"`, `id="scan-start-cam"`, `id="search-detail-dialog"`, `data-search-detail`, `/library/new?media_type=`, `id="barcode-search" class="barcode-search" open`} {
					if !strings.Contains(search.String(), want) {
						t.Errorf("search missing %q", want)
					}
				}
				if strings.Contains(search.String(), `id="search-bucket"`) || strings.Contains(search.String(), `action="/media/import"`) {
					t.Error("search must defer save actions to the details dialog")
				}
				if strings.Contains(search.String(), `hx-get=`) {
					t.Error("search pagination must preserve the destination through full navigation")
				}
				if strings.Contains(search.String(), "ZgotmplZ") {
					t.Error("invalid URL in search template")
				}
			})
		}
	}
}

func TestLocaleSwitchPreservesSearchContext(t *testing.T) {
	for _, path := range []string{"/en/search?bucket=wishlist&media_type=book&q=A%2FB", "/en/?q=A%2FB"} {
		got := swapLocalePath(path, "ja")
		want := strings.Replace(path, "/en/", "/ja/", 1)
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestLibraryCoversAndGrid(t *testing.T) {
	renderer, err := NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	image := "https://example.com/cover.jpg"
	for _, view := range []string{"list", "grid"} {
		data := map[string]any{"Locale": "en", "View": view, "List": []map[string]any{
			{"Entry": &domainlib.LibraryEntry{MediaType: "book"}, "Media": &domainlib.MediaRecord{Title: "With cover", ImageURL: &image}},
			{"Entry": &domainlib.LibraryEntry{MediaType: "book"}, "Media": &domainlib.MediaRecord{Title: "Without cover"}},
		}}
		var out bytes.Buffer
		if err := renderer.Render(&out, "partials/library_entry_list", data); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), image) || !strings.Contains(out.String(), "No cover") {
			t.Error("missing cover image or placeholder")
		}
		if strings.Contains(out.String(), "entry-list--grid") != (view == "grid") {
			t.Error("wrong layout")
		}
	}
}

func TestPreviewHasDetailsAndBothExplicitSaveActions(t *testing.T) {
	renderer, err := NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	description := "Full description <script>alert(1)</script>"
	for _, locale := range []string{"en", "ja"} {
		for _, localID := range []string{"", "012345678901234567890123"} {
			data := map[string]any{
				"Locale": locale, "Record": &domainlib.MediaRecord{Title: "A&B", MediaType: "book", Summary: &description, Details: bson.M{}},
				"Import":  mediasvc.ImportForm{Provider: "open_library", ExternalID: "OL1W", MediaType: "book"},
				"LocalID": localID, "Buckets": domainlib.Buckets,
				"Fields": []map[string]string{{"Key": "authors", "Value": "Test Author"}},
			}
			var out bytes.Buffer
			if err := renderer.Render(&out, "pages/search_detail.content", data); err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{`id="search-detail-title"`, `name="bucket" value="catalog"`, `name="bucket" value="wishlist"`, "Test Author", "Full description"} {
				if !strings.Contains(out.String(), want) {
					t.Errorf("missing %q", want)
				}
			}
			if strings.Contains(out.String(), "<script>") {
				t.Error("provider description must be escaped")
			}
			action := "/media/import"
			if localID != "" {
				action = "/library/attach"
			}
			if strings.Count(out.String(), `action="`+action+`"`) != 2 {
				t.Error("both destinations need an explicit save form")
			}
		}
	}
}

func TestAllMediaSearchRendering(t *testing.T) {
	renderer, err := NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, locale := range []string{"en", "ja"} {
		data := map[string]any{
			"Locale": locale, "Values": map[string]string{"media_type": "", "q": "test"},
			"ContentTemplate": "pages/search.content", "User": &domainauth.User{Username: "test"},
			"Path": "/" + locale + "/search", "MediaTypes": domainlib.MediaTypes,
			"Query": "test", "SearchError": "", "AllMedia": true, "HasResults": true,
			"SearchGroups": []map[string]any{
				{"MediaType": "game", "Hits": []domainsearch.Hit{{Title: "Game title", MediaType: "game", Provider: "rawg", ExternalID: "1"}}, "Total": 1, "SeeAllURL": "/" + locale + "/search?q=test&media_type=game"},
			},
		}
		var out bytes.Buffer
		if err := renderer.Render(&out, "pages/search.html", data); err != nil {
			t.Fatal(err)
		}
		html := out.String()
		if strings.Index(html, `id="media_type"`) > strings.Index(html, `id="q"`) {
			t.Error("type selector must precede title")
		}
		for _, want := range []string{`value="" selected`, `id="results-game"`, `data-search-detail`, `media_type=game`} {
			if !strings.Contains(html, want) {
				t.Errorf("missing %q", want)
			}
		}
		if strings.Contains(html, `class="pagination"`) {
			t.Error("all-media previews should link to per-type pagination")
		}
		data["HasResults"] = false
		data["SearchGroups"] = nil
		out.Reset()
		if err := renderer.Render(&out, "partials/search_results", data); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), `class="empty-state"`) {
			t.Error("all-media search needs an empty state")
		}
	}
}

func TestSearchListAndGridLayouts(t *testing.T) {
	renderer, err := NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, view := range []string{"list", "grid"} {
		for _, grouped := range []bool{true, false} {
			hits := []domainsearch.Hit{{Title: "Cover title", MediaType: "game", Provider: "rawg", ExternalID: "1"}}
			data := map[string]any{
				"Locale": "en", "Query": "query", "SearchError": "", "View": view,
				"ViewURLs": map[string]string{"list": "/en/search?q=query&view=list", "grid": "/en/search?q=query&view=grid"},
				"AllMedia": grouped, "HasResults": true, "Hits": hits,
				"SearchGroups": []map[string]any{{"MediaType": "game", "Hits": hits, "Total": 1, "SeeAllURL": "/en/search?q=query&media_type=game&view=" + view}},
				"Pagination":   map[string]any{"TotalItems": 1, "PageSize": 10, "StartItem": 1, "EndItem": 1},
			}
			var out bytes.Buffer
			if err := renderer.Render(&out, "partials/search_results", data); err != nil {
				t.Fatal(err)
			}
			html := out.String()
			if strings.Contains(html, "search-results--grid") != (view == "grid") {
				t.Errorf("wrong %s layout, grouped=%v", view, grouped)
			}
			if strings.Count(html, `aria-current="true" disabled`) != 1 {
				t.Error("active layout must be disabled")
			}
			if !strings.Contains(html, `data-search-detail`) {
				t.Error("layout must preserve preview links")
			}
		}
	}
}

func TestSettingsProviderKeysNeverPopulateSecrets(t *testing.T) {
	renderer, err := NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, locale := range []string{"en", "ja"} {
		var out bytes.Buffer
		data := map[string]any{
			"Locale": locale, "Path": "/" + locale + "/settings", "User": &domainauth.User{Username: "test"},
			"ContentTemplate": "pages/settings.content", "CurrentLanguageKey": "common.languageSwitcher." + locale,
			"ProviderStatuses": []map[string]any{{"ID": "tmdb", "Name": "TMDB", "StatusKey": "settings.providers.status.personal", "Personal": true, "KeyURL": "https://www.themoviedb.org/settings/api"}},
			// Deliberately supply secret data outside statuses: the template must ignore it.
			"Values": map[string]string{"tmdb": "test-secret-do-not-render"},
		}
		if err := renderer.Render(&out, "pages/settings.html", data); err != nil {
			t.Fatal(err)
		}
		html := out.String()
		for _, want := range []string{`/settings/providers`, `type="password"`, `name="remove_tmdb"`, `autocomplete="new-password"`, `type="button" data-test-provider="tmdb"`, `role="status"`, `/settings/providers/test`, `href="https://www.themoviedb.org/settings/api"`, `target="_blank" rel="noopener noreferrer"`, `data-test-provider="musicbrainz"`, `data-test-provider="openlibrary"`} {
			if !strings.Contains(html, want) {
				t.Errorf("missing %q", want)
			}
		}
		if strings.Contains(html, "test-secret-do-not-render") {
			t.Error("settings leaked a key")
		}
	}
}
