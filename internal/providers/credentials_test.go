package providers_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"media_library_manager/internal/providerkeys"
	"media_library_manager/internal/providers/discogs"
	"media_library_manager/internal/providers/rawg"
	"media_library_manager/internal/providers/tmdb"
)

type keyTransport func(*http.Request) (*http.Response, error)

func (f keyTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestProviderClientsUsePersonalKeysWithoutChangingDefaults(t *testing.T) {
	for _, provider := range []string{"tmdb", "rawg", "discogs"} {
		t.Run(provider, func(t *testing.T) {
			expected := "personal"
			calls := 0
			client := &http.Client{Transport: keyTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				actual := r.URL.Query().Get("api_key")
				if provider == "rawg" {
					actual = r.URL.Query().Get("key")
				}
				if provider == "discogs" {
					actual = strings.TrimPrefix(r.Header.Get("Authorization"), "Discogs token=")
				}
				if actual != expected {
					t.Error("incorrect credential selected")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"results":[]}`))}, nil
			})}
			ctx := providerkeys.WithKeys(context.Background(), providerkeys.Keys{provider: "personal"})
			var lookup func(context.Context) error
			switch provider {
			case "tmdb":
				c := &tmdb.Client{APIKey: "server", HTTP: client}
				lookup = func(ctx context.Context) error { _, err := c.SearchTV(ctx, "example"); return err }
			case "rawg":
				c := &rawg.Client{APIKey: "server", HTTP: client}
				lookup = func(ctx context.Context) error { _, err := c.SearchGames(ctx, "example"); return err }
			case "discogs":
				c := &discogs.Client{Token: "server", HTTP: client}
				lookup = func(ctx context.Context) error { _, err := c.SearchByBarcode(ctx, "12345678", 5); return err }
			}
			if err := lookup(ctx); err != nil {
				t.Fatal(err)
			}
			expected = "server"
			if err := lookup(context.Background()); err != nil {
				t.Fatal(err)
			}
			if calls != 2 {
				t.Error("provider was not called")
			}
		})
	}
}
