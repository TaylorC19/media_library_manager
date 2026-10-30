package handlers

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
	domainauth "media_library_manager/internal/domain/auth"
	"media_library_manager/internal/http/middleware"
)

func preferenceRequest(path string, user primitive.ObjectID, cookies ...*http.Cookie) *http.Request {
	r := httptest.NewRequest("GET", path, nil)
	for _, cookie := range cookies {
		r.AddCookie(cookie)
	}
	return r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{ID: user, Role: "demo"}))
}
func preferenceCookie(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one preference cookie, got %d", len(cookies))
	}
	return cookies[0]
}

func TestListPreferencesPersistAcrossVisitsAndExplicitClearsWin(t *testing.T) {
	owner := primitive.NewObjectID()
	r := preferenceRequest("/en/?view=grid&media_type=book&format=paperback&q=Author&page_size=20&page=9&mode=barcode&complete=1", owner)
	w := httptest.NewRecorder()
	if !resolveListPreferences(w, r, "catalog") {
		t.Fatal("explicit filters must render immediately")
	}
	cookie := preferenceCookie(t, w)
	if cookie.MaxAge != listPreferenceLifetime || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
		t.Error("preferences must survive browser restarts")
	}
	data, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	saved, _ := url.ParseQuery(string(data))
	if saved.Has("page") || saved.Has("mode") || saved.Has("complete") {
		t.Error("transient state must not be persisted")
	}
	r = preferenceRequest("/ja/", owner, cookie)
	w = httptest.NewRecorder()
	if resolveListPreferences(w, r, "catalog") || w.Code != 302 {
		t.Fatal("restored visit should canonicalize actual filters")
	}
	target, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"view": "grid", "media_type": "book", "format": "paperback", "q": "Author", "page_size": "20"} {
		if target.Query().Get(key) != want {
			t.Errorf("restored %s=%q", key, target.Query().Get(key))
		}
	}
	if target.Query().Has("page") {
		t.Error("return visits must start on the first page")
	}
	r = preferenceRequest(target.String(), owner, cookie)
	w = httptest.NewRecorder()
	if !resolveListPreferences(w, r, "catalog") {
		t.Error("canonical redirect must not loop")
	}
	if len(w.Result().Cookies()) != 0 {
		t.Error("unchanged preference should not be rewritten")
	}
	// The clear link encodes empty fields so saved filters cannot reappear.
	r = preferenceRequest("/en/?view=grid&q=&media_type=&format=&page_size=20", owner, cookie)
	w = httptest.NewRecorder()
	if !resolveListPreferences(w, r, "catalog") {
		t.Error("explicit clearing must render immediately")
	}
	cleared := preferenceCookie(t, w)
	r = preferenceRequest("/en/", owner, cleared)
	w = httptest.NewRecorder()
	resolveListPreferences(w, r, "catalog")
	for _, key := range []string{"q", "media_type", "format"} {
		if r.URL.Query().Get(key) != "" {
			t.Errorf("cleared %s resurrected", key)
		}
	}
	if r.URL.Query().Get("view") != "grid" {
		t.Error("clear filters should retain layout")
	}
}

func TestListPreferencesAreScopedAndHTMXRestoresWithoutRedirecting(t *testing.T) {
	owner := primitive.NewObjectID()
	r := preferenceRequest("/en/search?view=list&q=avatar&media_type=movie&page_size=25", owner)
	w := httptest.NewRecorder()
	resolveListPreferences(w, r, "search")
	cookie := preferenceCookie(t, w)
	for _, scope := range []string{"catalog", "wishlist"} {
		r := preferenceRequest("/en/", owner, cookie)
		w := httptest.NewRecorder()
		if !resolveListPreferences(w, r, scope) || r.URL.Query().Get("q") != "" || r.URL.Query().Get("view") != "list" {
			t.Error("search preferences leaked into library")
		}
	}
	r = preferenceRequest("/en/search", primitive.NewObjectID(), cookie)
	w = httptest.NewRecorder()
	if !resolveListPreferences(w, r, "search") || r.URL.Query().Get("q") != "" || r.URL.Query().Get("view") != "grid" {
		t.Error("preferences crossed account boundary")
	}
	r = preferenceRequest("/ja/search?q=other", owner, cookie)
	r.Header.Set("HX-Request", "true")
	w = httptest.NewRecorder()
	if !resolveListPreferences(w, r, "search") || w.Header().Get("Location") != "" {
		t.Error("HTMX must continue rendering its fragment")
	}
	if !strings.Contains(w.Header().Get("HX-Push-Url"), "q=other") || r.URL.Query().Get("view") != "list" || r.URL.Query().Get("media_type") != "movie" {
		t.Error("partial URL overrides should merge with saved defaults and canonicalize history")
	}
	// A group's movie filter must never become the overall All media preference:
	// group endpoints do not call the preference resolver.
	if cookie.Name == listPreferenceCookie(preferenceRequest("/en/", owner), "wishlist") {
		t.Error("scope cookie collision")
	}
}

func TestMalformedAndOversizedPreferencesFallBackSafely(t *testing.T) {
	owner := primitive.NewObjectID()
	for _, value := range []string{"bad!base64", strings.Repeat("x", 2050), base64.RawURLEncoding.EncodeToString([]byte("view=grid&media_type=invalid&page_size=10"))} {
		r := preferenceRequest("/en/search", owner)
		r.AddCookie(&http.Cookie{Name: listPreferenceCookie(r, "search"), Value: value})
		w := httptest.NewRecorder()
		if !resolveListPreferences(w, r, "search") || r.URL.Query().Get("view") != "grid" || r.URL.Query().Get("media_type") != "" {
			t.Error("corrupt preference should fall back to defaults")
		}
	}
	r := preferenceRequest("/en/search?view=grid&q="+strings.Repeat("a", 257), owner)
	w := httptest.NewRecorder()
	resolveListPreferences(w, r, "search")
	if len(w.Result().Cookies()) != 0 {
		t.Error("oversized query must not enter a preference cookie")
	}
	if len(r.URL.Query().Get("q")) != 257 {
		t.Error("storage limit must not truncate the current search")
	}
}
