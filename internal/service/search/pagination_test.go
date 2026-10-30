package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"media_library_manager/internal/config"
	"media_library_manager/internal/service/reliability"
)

type pageCacheRepo struct {
	mu     sync.Mutex
	values map[string][]byte
}

func (r *pageCacheRepo) FindActive(_ context.Context, provider, key string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.values[provider+key], nil
}
func (r *pageCacheRepo) Set(_ context.Context, provider, key string, payload []byte, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[provider+key] = append([]byte(nil), payload...)
	return nil
}

func pagedSearchFixture(t *testing.T, total int) (*Service, *[]string) {
	t.Helper()
	cfg := config.Config{TMDBAPIKey: "test", RAWGAPIKey: "test", ProviderCacheSearchTTL: time.Hour}
	rel := reliability.NewService(cfg)
	service := NewService(cfg, reliability.NewCache(&pageCacheRepo{values: map[string][]byte{}}, rel), rel, nil)
	calls := []string{}
	client := &http.Client{Transport: allTransport(func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.URL.Path+"?"+r.URL.RawQuery)
		q := r.URL.Query()
		get := func(key string) int { n, _ := strconv.Atoi(q.Get(key)); return n }
		offset, limit := get("offset"), get("limit")
		kind := ""
		switch r.URL.Host {
		case "api.themoviedb.org":
			offset = (get("page") - 1) * 20
			limit = 20
			kind = "tmdb"
		case "api.rawg.io":
			limit = get("page_size")
			offset = (get("page") - 1) * limit
			kind = "rawg"
		case "openlibrary.org":
			kind = "book"
		case "musicbrainz.org":
			kind = "album"
		default:
			t.Fatalf("unexpected provider host: %s", r.URL.Host)
		}
		if limit < 1 || offset < 0 || (offset >= total && total > 0) {
			t.Errorf("invalid upstream page offset=%d limit=%d total=%d", offset, limit, total)
		}
		rows := []map[string]any{}
		for i := offset + 1; i <= offset+limit && i <= total; i++ {
			row := map[string]any{"id": i, "title": fmt.Sprintf("Result %d", i), "name": fmt.Sprintf("Result %d", i)}
			if kind == "book" {
				row["key"] = fmt.Sprintf("/works/OL%dW", i)
			}
			if kind == "album" {
				row["id"] = fmt.Sprint(i)
			}
			rows = append(rows, row)
		}
		payload := map[string]any{"count": total, "numFound": total, "total_results": total, "total_pages": (total + 19) / 20, "results": rows, "docs": rows, "releases": rows}
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(data)))}, nil
	})}
	service.tmdb.HTTP = client
	service.mb.HTTP = client
	service.ol.HTTP = client
	service.rawg.HTTP = client
	return service, &calls
}

func TestProviderPaginationBeyondInitialBatch(t *testing.T) {
	for _, kind := range []string{"movie", "tv", "book", "album", "game"} {
		t.Run(kind, func(t *testing.T) {
			service, calls := pagedSearchFixture(t, 125)
			out, err := service.SearchPage(context.Background(), kind, "example", 7, 10)
			if err != nil {
				t.Fatal(err)
			}
			if out.Total != 125 || out.Page != 7 || len(out.Hits) != 10 || out.Hits[0].Title != "Result 61" || out.Hits[9].Title != "Result 70" {
				t.Fatalf("wrong remote page: %+v", out)
			}
			before := len(*calls)
			cached, err := service.SearchPage(context.Background(), kind, "example", 7, 10)
			if err != nil || len(*calls) != before || cached.Hits[0].Title != "Result 61" {
				t.Fatal("repeated page should use its own cache")
			}
			last, err := service.SearchPage(context.Background(), kind, "example", 10000, 10)
			if err != nil || last.Page != 13 || len(last.Hits) != 5 || last.Hits[0].Title != "Result 121" {
				t.Fatalf("invalid page must fetch the real last page: %+v err=%v", last, err)
			}
			preview, err := service.Search(context.Background(), kind, "example")
			if err != nil || len(preview.Hits) != 3 || preview.Total != 125 {
				t.Fatal("preview must retain full provider total")
			}
		})
	}
}

func TestTMDBPageBoundariesAndPageSizeCacheIsolation(t *testing.T) {
	service, _ := pagedSearchFixture(t, 125)
	for _, tc := range []struct{ page, size, first, last int }{{1, 10, 1, 10}, {2, 10, 11, 20}, {2, 15, 16, 30}, {2, 50, 51, 100}, {3, 50, 101, 125}} {
		out, err := service.SearchPage(context.Background(), "movie", "example", tc.page, tc.size)
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Hits) != tc.last-tc.first+1 || out.Hits[0].Title != fmt.Sprintf("Result %d", tc.first) || out.Hits[len(out.Hits)-1].Title != fmt.Sprintf("Result %d", tc.last) {
			t.Fatalf("page %d size %d crossed native pages incorrectly: %+v", tc.page, tc.size, out)
		}
	}
}

func TestTMDBAccessiblePageLimitAndEmptyResults(t *testing.T) {
	service, calls := pagedSearchFixture(t, 12000)
	out, err := service.SearchPage(context.Background(), "movie", "example", 10000, 50)
	if err != nil || out.Total != 10000 || out.Page != 200 || len(out.Hits) != 50 || out.Hits[49].Title != "Result 10000" {
		t.Fatalf("TMDB access cap: %+v %v", out, err)
	}
	for _, call := range *calls {
		if strings.Contains(call, "page=501&") {
			t.Fatal("requested unavailable TMDB page")
		}
	}
	service, _ = pagedSearchFixture(t, 0)
	out, err = service.SearchPage(context.Background(), "book", "nothing", 5, 10)
	if err != nil || out.Total != 0 || out.Page != 1 || len(out.Hits) != 0 {
		t.Fatal("empty results must settle on page 1")
	}
}
