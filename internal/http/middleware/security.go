package middleware

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"media_library_manager/internal/config"
)

// Security protects unsafe requests, including login, before request parsing or
// session lookup. Production compares against a configured origin, never proxy
// headers supplied by the browser. Modern browsers send Origin on form posts.
func Security(cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "same-origin")
			w.Header().Set("Permissions-Policy", "camera=(self), microphone=(), geolocation=()")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' https: data:; connect-src 'self'; worker-src 'self' blob:; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'")
			if cfg.Env == "production" {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000")
				ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
				defer cancel()
				r = r.WithContext(ctx)
			}
			if !strings.HasPrefix(r.URL.Path, "/static/") {
				w.Header().Set("Cache-Control", "private, no-store")
				w.Header().Set("Vary", "Cookie")
			}
			if strings.HasSuffix(r.URL.Path, "/register") && !cfg.SignupEnabled {
				http.NotFound(w, r)
				return
			}
			if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
				expected := cfg.PublicOrigin
				if expected == "" {
					scheme := "http"
					if r.TLS != nil {
						scheme = "https"
					}
					expected = scheme + "://" + r.Host
				}
				origin := r.Header.Get("Origin")
				valid := origin == expected
				if origin == "" { // Referer fallback supports older browsers; no evidence fails closed in production.
					ref, err := url.Parse(r.Header.Get("Referer"))
					valid = err == nil && ref.Host != "" && ref.Scheme+"://"+ref.Host == expected
					if cfg.Env != "production" && r.Header.Get("Referer") == "" {
						valid = true
					}
				}
				if !valid || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
					http.Error(w, "Request origin rejected", http.StatusForbidden)
					return
				}
				limit := int64(1 << 20)
				if cfg.BackupEnabled && strings.HasPrefix(r.URL.Path, "/settings/backup/") {
					limit = cfg.BackupMaxUploadBytes + (1 << 20)
				}
				if r.ContentLength > limit {
					http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
					return
				}
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
			r = r.WithContext(context.WithValue(r.Context(), secureCookiesKey{}, cfg.SessionCookieSecure || cfg.Env == "production"))
			next.ServeHTTP(w, r)
		})
	}
}

type secureCookiesKey struct{}

// SecureCookies uses trusted deployment configuration rather than proxy headers.
func SecureCookies(r *http.Request) bool {
	secure, _ := r.Context().Value(secureCookiesKey{}).(bool)
	return secure || r.TLS != nil
}
