package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"media_library_manager/internal/config"
)

func TestSecurityUnsafeOrigins(t *testing.T) {
	cfg := config.Config{Env: "production", PublicOrigin: "https://library.example", SignupEnabled: true}
	h := Security(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	for _, path := range []string{"/en/login", "/en/logout", "/en/register", "/library/x/update", "/media/refresh/x", "/barcode/lookup"} {
		for _, tc := range []struct {
			origin, ref, site string
			want              int
		}{
			{"https://library.example", "", "same-origin", 204}, {"https://other.example", "", "cross-site", 403}, {"http://library.example", "", "", 403}, {"null", "", "", 403}, {"", "", "", 403}, {"", "https://library.example/en/", "same-origin", 204}, {"https://library.example", "", "cross-site", 403},
		} {
			r := httptest.NewRequest("POST", path, strings.NewReader("x=1"))
			r.Host = "untrusted.example"
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("Referer", tc.ref)
			r.Header.Set("Sec-Fetch-Site", tc.site)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("%s origin %q: got %d want %d", path, tc.origin, w.Code, tc.want)
			}
		}
	}
}
func TestSecurityHeadersAndBounds(t *testing.T) {
	cfg := config.Config{Env: "production", PublicOrigin: "https://library.example"}
	called := false
	h := Security(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/en/", nil))
	for _, key := range []string{"Content-Security-Policy", "Permissions-Policy", "Strict-Transport-Security", "X-Content-Type-Options"} {
		if w.Header().Get(key) == "" {
			t.Fatal("missing " + key)
		}
	}
	if w.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("private response can be cached")
	}
	called = false
	r := httptest.NewRequest("POST", "/library", strings.NewReader(strings.Repeat("a", (1<<20)+1)))
	r.Header.Set("Origin", cfg.PublicOrigin)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 413 || called {
		t.Fatal("oversized body accepted")
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/en/register", nil))
	if w.Code != 404 {
		t.Fatal("signup enabled without opt in")
	}
}

func TestSecureCookiesUsesDeploymentConfiguration(t *testing.T) {
	for _, production := range []bool{false, true} {
		cfg := config.Config{SignupEnabled: true}
		if production {
			cfg.Env = "production"
		}
		r := httptest.NewRequest("GET", "http://library.example/en/", nil)
		r.Header.Set("X-Forwarded-Proto", "https")
		Security(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if SecureCookies(r) != production {
				t.Fatal("cookie security trusted an unverified proxy header")
			}
		})).ServeHTTP(httptest.NewRecorder(), r)
	}
}
