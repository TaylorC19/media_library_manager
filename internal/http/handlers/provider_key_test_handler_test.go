package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"media_library_manager/internal/config"
	"media_library_manager/internal/providerkeys"
	"media_library_manager/internal/views"
)

type testKeyStore struct {
	keyStoreFake
	reads int
}

func (s *testKeyStore) Get(primitive.ObjectID) (providerkeys.Keys, error) {
	s.reads++
	return providerkeys.Keys{"tmdb": "saved-secret"}, nil
}

type checkerFake struct {
	provider, key string
	calls         int
}

func (c *checkerFake) Check(_ context.Context, p, k string) string {
	c.calls++
	c.provider = p
	c.key = k
	return "success"
}

func TestProviderKeyTestSelectsCandidateSavedOrDefaultWithoutSaving(t *testing.T) {
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ key, remove, want string }{
		{"new-secret", "", "new-secret"}, {"", "", "saved-secret"}, {"", "on", "server-secret"},
	} {
		store := &testKeyStore{}
		checker := &checkerFake{}
		handler := NewSettingsHandler(config.Config{TMDBAPIKey: "server-secret"}, renderer, store, nil)
		handler.checker = checker
		values := url.Values{"provider": {"tmdb"}, "key": {test.key}, "remove": {test.remove}}
		r := httptest.NewRequest(http.MethodPost, "/en/settings/providers/test", strings.NewReader(values.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		handler.testProviderKey(w, r, primitive.NewObjectID())
		if w.Code != 200 || checker.key != test.want {
			t.Errorf("wrong test credential selected, status %d", w.Code)
		}
		if store.changes != nil {
			t.Error("testing must never save credentials")
		}
		for _, secret := range []string{"new-secret", "saved-secret", "server-secret"} {
			if strings.Contains(w.Body.String(), secret) {
				t.Error("test response leaked a credential")
			}
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Error("test results must not be cached")
		}
	}
}

func TestProviderKeyTestRejectsUnauthorizedRequestsAndMalformedInput(t *testing.T) {
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	store := &testKeyStore{}
	checker := &checkerFake{}
	handler := NewSettingsHandler(config.Config{}, renderer, store, nil)
	handler.checker = checker
	w := httptest.NewRecorder()
	handler.ProviderKeyTest(w, httptest.NewRequest(http.MethodPost, "/en/settings/providers/test", nil))
	if w.Code != http.StatusUnauthorized {
		t.Error("test endpoint needs authentication")
	}
	for _, body := range []string{"provider=unknown&key=secret", "provider=tmdb&key=invalid+key", "provider=tmdb&key=secret&remove=on"} {
		r := httptest.NewRequest(http.MethodPost, "/en/settings/providers/test", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w = httptest.NewRecorder()
		handler.testProviderKey(w, r, primitive.NewObjectID())
		if checker.calls != 0 || store.reads != 0 {
			t.Error("invalid input must not access saved credentials or call providers")
		}
	}
	r := httptest.NewRequest(http.MethodPost, "/en/settings/providers/test", nil)
	r.Header.Set("Origin", "https://attacker.invalid")
	w = httptest.NewRecorder()
	handler.testProviderKey(w, r, primitive.NewObjectID())
	if w.Code != http.StatusForbidden {
		t.Error("cross-origin tests must be rejected")
	}
}

func TestPublicConnectionChecksDoNotAccessOrSaveKeys(t *testing.T) {
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, provider := range []string{"musicbrainz", "openlibrary"} {
		store := &testKeyStore{}
		checker := &checkerFake{}
		handler := NewSettingsHandler(config.Config{}, renderer, store, checker)
		r := httptest.NewRequest(http.MethodPost, "/en/settings/providers/test", strings.NewReader("provider="+provider))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		handler.testProviderKey(w, r, primitive.NewObjectID())
		if checker.calls != 1 || checker.provider != provider || checker.key != "" {
			t.Error("incorrect public connection check")
		}
		if store.reads != 0 || store.changes != nil {
			t.Error("public checks must not access credential storage")
		}
		if !strings.Contains(w.Body.String(), "Connection successful") || strings.Contains(w.Body.String(), "Key accepted") {
			t.Error("public checks need connection-specific feedback")
		}
	}
}
