package providerkeys

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type checkTransport func(*http.Request) (*http.Response, error)

func (f checkTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCheckerUsesUncachedAuthenticatedReadOnlyRequests(t *testing.T) {
	for _, provider := range Providers {
		calls := 0
		checker := &Checker{HTTP: &http.Client{Transport: checkTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != http.MethodGet {
				t.Error("check must be read-only")
			}
			key := r.URL.Query().Get("api_key")
			if provider == "rawg" {
				key = r.URL.Query().Get("key")
			}
			if provider == "discogs" {
				key = strings.TrimPrefix(r.Header.Get("Authorization"), "Discogs token=")
			}
			if key != "candidate" {
				t.Error("did not use candidate key")
			}
			payload := `{"id":11}`
			if provider == "rawg" {
				payload = `{"results":[]}`
			}
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(payload))}, nil
		})}}
		for i := 0; i < 2; i++ {
			if got := checker.Check(context.Background(), provider, "candidate"); got != "success" {
				t.Errorf("%s: %s", provider, got)
			}
		}
		if calls != 2 {
			t.Error("credential checks must bypass caches")
		}
	}
}

func TestCheckerDistinguishesRejectedKeysFromProviderFailures(t *testing.T) {
	for _, test := range []struct {
		status     int
		body, want string
	}{
		{401, `{}`, "rejected"}, {403, `{}`, "forbidden"}, {429, `{}`, "rate_limited"},
		{503, `{}`, "unavailable"}, {200, `broken`, "invalid_response"}, {200, `{}`, "invalid_response"},
	} {
		checker := &Checker{HTTP: &http.Client{Transport: checkTransport(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: test.status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(test.body))}, nil
		})}}
		if got := checker.Check(context.Background(), "tmdb", "candidate"); got != test.want {
			t.Errorf("status %d: got %q, want %q", test.status, got, test.want)
		}
	}
	if got := (&Checker{}).Check(context.Background(), "tmdb", ""); got != "missing" {
		t.Error("missing keys should not call providers")
	}
}

func TestPublicProviderConnectionChecksNeedNoCredentials(t *testing.T) {
	for _, provider := range []string{"musicbrainz", "openlibrary"} {
		for _, valid := range []bool{true, false} {
			checker := &Checker{MusicBrainzUA: "MediaLibraryManager/test", HTTP: &http.Client{Transport: checkTransport(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet || r.URL.Query().Get("limit") != "1" {
					t.Error("connection check must be a small read-only request")
				}
				if r.URL.Query().Get("api_key") != "" || r.URL.Query().Get("key") != "" || r.Header.Get("Authorization") != "" {
					t.Error("public checks must not send credentials")
				}
				payload := `{}`
				if provider == "musicbrainz" {
					if r.URL.Host != "musicbrainz.org" || r.Header.Get("User-Agent") != "MediaLibraryManager/test" {
						t.Error("MusicBrainz needs its configured client identity")
					}
					if valid {
						payload = `{"releases":[]}`
					}
				} else {
					if r.URL.Host != "openlibrary.org" {
						t.Error("incorrect provider endpoint")
					}
					if valid {
						payload = `{"docs":[]}`
					}
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(payload))}, nil
			})}}
			want := "success"
			if !valid {
				want = "invalid_response"
			}
			if got := checker.Check(context.Background(), provider, ""); got != want {
				t.Errorf("%s: got %s, want %s", provider, got, want)
			}
		}
	}
}
