package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"media_library_manager/internal/config"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/providerkeys"
	"media_library_manager/internal/views"
)

type keyStoreFake struct {
	changes providerkeys.Keys
	id      primitive.ObjectID
	fail    bool
}

func (f *keyStoreFake) Get(primitive.ObjectID) (providerkeys.Keys, error) {
	return providerkeys.Keys{}, nil
}
func (f *keyStoreFake) Update(id primitive.ObjectID, changes providerkeys.Keys) error {
	f.id = id
	f.changes = changes
	if f.fail {
		return errors.New("storage failure")
	}
	return nil
}

func TestProviderSettingsSaveAndAuthentication(t *testing.T) {
	fake := &keyStoreFake{}
	handler := NewSettingsHandler(config.Config{}, nil, fake, nil)
	r := httptest.NewRequest(http.MethodPost, "/ja/settings/providers", strings.NewReader("tmdb=new-key&rawg=&remove_discogs=on"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	id := primitive.NewObjectID()
	middleware.Locale("en")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handler.saveProviderKeys(w, r, id) })).ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/ja/settings" {
		t.Fatalf("unexpected save response: %d", w.Code)
	}
	if fake.id != id || fake.changes["tmdb"] != "new-key" {
		t.Error("did not save for the selected account")
	}
	if _, ok := fake.changes["rawg"]; ok {
		t.Error("blank input must preserve existing key")
	}
	if value, ok := fake.changes["discogs"]; !ok || value != "" {
		t.Error("removal must clear only the personal override")
	}
	if strings.Contains(w.Body.String(), "new-key") {
		t.Error("response leaked submitted key")
	}
	w = httptest.NewRecorder()
	handler.ProviderKeysSubmit(w, httptest.NewRequest(http.MethodPost, "/en/settings/providers", nil))
	if w.Code != http.StatusUnauthorized {
		t.Error("saving keys must require authentication")
	}
}

func TestProviderSettingsRejectCrossOriginAndInvalidChanges(t *testing.T) {
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		origin, body string
		status       int
	}{
		{"https://elsewhere.invalid", "tmdb=secret", http.StatusForbidden},
		{"", "tmdb=secret&remove_tmdb=on", http.StatusBadRequest},
		{"", "tmdb=invalid+key", http.StatusBadRequest},
	} {
		fake := &keyStoreFake{}
		handler := NewSettingsHandler(config.Config{}, renderer, fake, nil)
		r := httptest.NewRequest(http.MethodPost, "/en/settings/providers", strings.NewReader(test.body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", test.origin)
		w := httptest.NewRecorder()
		handler.saveProviderKeys(w, r, primitive.NewObjectID())
		if w.Code != test.status {
			t.Errorf("status=%d, want %d", w.Code, test.status)
		}
		if fake.changes != nil {
			t.Error("rejected submission wrote credentials")
		}
		if strings.Contains(w.Body.String(), "secret") || strings.Contains(w.Body.String(), "invalid key") {
			t.Error("error response echoed input")
		}
	}
}
