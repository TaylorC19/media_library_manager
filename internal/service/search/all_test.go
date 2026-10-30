package search

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"media_library_manager/internal/config"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/service/reliability"
)

type allTransport func(*http.Request) (*http.Response, error)

func (f allTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSearchAllRunsConcurrentlyAndRetainsOtherTypesOnFailure(t *testing.T) {
	cfg := config.Config{TMDBAPIKey: "test", RAWGAPIKey: "test"}
	service := NewService(cfg, nil, reliability.NewService(cfg), nil)
	arrived := make(chan string, 5)
	release := make(chan struct{})
	client := &http.Client{Transport: allTransport(func(r *http.Request) (*http.Response, error) {
		arrived <- r.URL.Path
		select {
		case <-release:
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
		payload := `{"results":[{"id":1,"title":"Movie or TV"}]}`
		switch r.URL.Host {
		case "musicbrainz.org":
			payload = `{"releases":[{"id":"album1","title":"Music"}]}`
		case "openlibrary.org":
			payload = `{"docs":[{"key":"/works/OL1W","title":"Book"}]}`
		case "api.rawg.io":
			payload = `invalid JSON`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(payload))}, nil
	})}
	service.tmdb.HTTP = client
	service.mb.HTTP = client
	service.ol.HTTP = client
	service.rawg.HTTP = client
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	type result struct {
		groups []Group
		err    error
	}
	done := make(chan result, 1)
	go func() { groups, err := service.SearchAll(ctx, "cyberpunk"); done <- result{groups, err} }()
	// No provider is allowed to finish until all five lookups have started.
	for i := 0; i < 5; i++ {
		select {
		case <-arrived:
		case <-ctx.Done():
			t.Fatal("lookups did not start concurrently")
		}
	}
	close(release)
	out := <-done
	if out.err != nil {
		t.Fatal(out.err)
	}
	if len(out.groups) != 5 {
		t.Fatalf("got %d groups", len(out.groups))
	}
	for i, group := range out.groups {
		if group.MediaType != domainlib.MediaTypes[i] {
			t.Error("unstable group ordering")
		}
		if group.MediaType == "game" {
			if len(group.Warnings) != 1 || group.Warnings[0] != "search.warnings.providerUnavailable" {
				t.Error("failed provider needs a scoped warning")
			}
		} else if len(group.Hits) != 1 {
			t.Errorf("lost results for %s", group.MediaType)
		}
	}
}

func TestSearchAllEmptyAndCancelledRequests(t *testing.T) {
	service := &Service{}
	groups, err := service.SearchAll(context.Background(), "  ")
	if err != nil || len(groups) != 0 {
		t.Fatal("blank queries must not call providers")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = service.SearchAll(ctx, "query")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want cancellation", err)
	}
}
