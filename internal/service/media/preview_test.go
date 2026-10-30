package media

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"media_library_manager/internal/config"
	"media_library_manager/internal/service/reliability"
)

type previewTransport func(*http.Request) (*http.Response, error)

func (f previewTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPreviewFetchesFullMetadataWithoutLibraryRepositories(t *testing.T) {
	cfg := config.Config{TMDBAPIKey: "test"}
	// Nil repositories ensure opening a preview cannot insert a record or entry.
	service := NewService(cfg, nil, nil, nil, reliability.NewService(cfg), nil)
	calls := 0
	service.tmdb.HTTP = &http.Client{Transport: previewTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != "/3/movie/42" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"id":42,"title":"Full movie","overview":"Complete description","runtime":120,"genres":[{"name":"Drama"}],"credits":{"crew":[{"job":"Director","name":"Test Director"}]}}`))}, nil
	})}
	record, err := service.Preview(context.Background(), ImportForm{Provider: "tmdb", ExternalID: "42", MediaType: "movie", TMDBKind: "movie"})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || record.Title != "Full movie" || record.Summary == nil || *record.Summary != "Complete description" {
		t.Fatal("incomplete preview")
	}
	if record.ID.IsZero() == false {
		t.Error("preview should not be persisted")
	}
	_, err = service.Preview(context.Background(), ImportForm{Provider: "tmdb", ExternalID: "42", MediaType: "book", TMDBKind: "movie"})
	if !errors.Is(err, ErrValidation) || calls != 1 {
		t.Error("invalid references must not reach the provider")
	}
}
