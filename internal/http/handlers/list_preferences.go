package handlers

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/http/middleware"
)

const listPreferenceLifetime = 365 * 24 * 60 * 60

func listPreferenceCookie(r *http.Request, scope string) string {
	return "mlm_view_v1_" + middleware.CurrentUser(r.Context()).ID.Hex() + "_" + scope
}

// resolveListPreferences restores only absent parameters. Empty URL values are
// explicit clearing choices. These account-scoped cookies contain browser-only
// preferences, never shared provider results, roles, or credentials.
func resolveListPreferences(w http.ResponseWriter, r *http.Request, scope string) bool {
	keys := []string{"view", "media_type", "q", "page_size"}
	defaultView := "grid"
	if scope != "search" {
		keys = append(keys, "format")
		defaultView = "list"
	}
	defaults := url.Values{"view": {defaultView}, "media_type": {""}, "q": {""}, "page_size": {"10"}}
	if scope != "search" {
		defaults.Set("format", "")
	}
	stored := url.Values{}
	cookieName := listPreferenceCookie(r, scope)
	if cookie, err := r.Cookie(cookieName); err == nil && len(cookie.Value) <= 2048 {
		if data, err := base64.RawURLEncoding.DecodeString(cookie.Value); err == nil {
			if parsed, err := url.ParseQuery(string(data)); err == nil && validListPreferences(parsed, scope) {
				stored = parsed
			}
		}
	}
	query := r.URL.Query()
	restored := false
	for _, key := range keys {
		if query.Has(key) {
			continue
		}
		if stored.Has(key) {
			query.Set(key, stored.Get(key))
			restored = true
		} else {
			query.Set(key, defaults.Get(key))
		}
	}
	// Normalize page size and view consistently with the existing page controls.
	if query.Get("view") == "" {
		query.Set("view", defaultView)
	}
	query.Set("page_size", strconv.Itoa(parsePageSize(query.Get("page_size"))))
	effective := url.Values{}
	for _, key := range keys {
		effective.Set(key, strings.TrimSpace(query.Get(key)))
	}
	if validListPreferences(effective, scope) {
		encoded := base64.RawURLEncoding.EncodeToString([]byte(effective.Encode()))
		if previous, err := r.Cookie(cookieName); err != nil || previous.Value != encoded {
			http.SetCookie(w, &http.Cookie{Name: cookieName, Value: encoded, Path: "/", MaxAge: listPreferenceLifetime, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode})
		}
	}
	r.URL.RawQuery = query.Encode()
	w.Header().Set("Cache-Control", "private, no-store")
	if restored {
		// Keep history, copied links, and locale switches tied to the filters actually
		// used. HTMX must continue returning the list fragment rather than redirecting.
		if isHTMX(r) {
			w.Header().Set("HX-Push-Url", r.URL.RequestURI())
		} else {
			http.Redirect(w, r, r.URL.RequestURI(), http.StatusFound)
			return false
		}
	}
	return true
}

func validListPreferences(values url.Values, scope string) bool {
	if view := values.Get("view"); view != "list" && view != "grid" {
		return false
	}
	if kind := values.Get("media_type"); kind != "" && !domainlib.IsMediaType(kind) {
		return false
	}
	if format := values.Get("format"); format != "" && (scope == "search" || !domainlib.IsFormat(format)) {
		return false
	}
	if len(values.Get("q")) > 256 {
		return false
	}
	size, err := strconv.Atoi(values.Get("page_size"))
	return err == nil && size >= 1 && size <= maxPageSize
}
