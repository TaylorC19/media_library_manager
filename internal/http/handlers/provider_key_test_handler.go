package handlers

import (
	"context"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/providerkeys"
)

type providerKeyChecker interface {
	Check(context.Context, string, string) string
}

func (h *SettingsHandler) ProviderKeyTest(w http.ResponseWriter, r *http.Request) {
	user := h.requireUser(w, r)
	if user == nil {
		return
	}
	h.testProviderKey(w, r, user.ID)
}

func (h *SettingsHandler) testProviderKey(w http.ResponseWriter, r *http.Request, id primitive.ObjectID) {
	w.Header().Set("Cache-Control", "no-store")
	if !sameSettingsOrigin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		h.keyTestResult(w, r, "invalid")
		return
	}
	provider := r.PostForm.Get("provider")
	publicProvider := provider == "musicbrainz" || provider == "openlibrary"
	allowed := publicProvider
	for _, name := range providerkeys.Providers {
		if provider == name {
			allowed = true
			break
		}
	}
	if !allowed {
		h.keyTestResult(w, r, "invalid")
		return
	}
	if publicProvider {
		if r.PostForm.Get("key") != "" || r.PostForm.Get("remove") != "" {
			h.keyTestResult(w, r, "invalid")
			return
		}
		result := h.checker.Check(r.Context(), provider, "")
		if result == "success" {
			result = "connection_success"
		}
		if result == "rejected" || result == "forbidden" {
			result = "connection_denied"
		}
		h.keyTestResult(w, r, result)
		return
	}
	key := strings.TrimSpace(r.PostForm.Get("key"))
	remove := r.PostForm.Get("remove") == "on"
	if !validProviderKey(key) || (key != "" && remove) {
		h.keyTestResult(w, r, "invalid")
		return
	}
	if key == "" {
		keys := providerkeys.Keys{}
		if !remove {
			var err error
			keys, err = h.keys.Get(id)
			if err != nil {
				h.keyTestResult(w, r, "unavailable")
				return
			}
		}
		defaults := map[string]string{"tmdb": h.cfg.TMDBAPIKey, "rawg": h.cfg.RAWGAPIKey, "discogs": h.cfg.DiscogsToken}
		key = providerkeys.Resolve(providerkeys.WithKeys(r.Context(), keys), provider, defaults[provider])
	}
	h.keyTestResult(w, r, h.checker.Check(r.Context(), provider, key))
}

func (h *SettingsHandler) keyTestResult(w http.ResponseWriter, r *http.Request, result string) {
	// Only fixed messages are rendered. Never include a key, provider response,
	// request URL, or account identity in the result.
	switch result {
	case "connection_success", "connection_denied", "success", "missing", "rejected", "forbidden", "rate_limited", "invalid_response", "unavailable", "invalid":
	default:
		result = "unavailable"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.renderPage(w, "partials/provider_key_test", map[string]any{
		"Locale": middleware.LocaleFromContext(r.Context()), "OK": result == "success" || result == "connection_success",
		"MessageKey": "settings.providers.testResults." + result,
	})
}
