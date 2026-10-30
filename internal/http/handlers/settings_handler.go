package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"media_library_manager/internal/config"
	domainauth "media_library_manager/internal/domain/auth"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/providerkeys"
	"media_library_manager/internal/views"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type providerKeyStore interface {
	Get(primitive.ObjectID) (providerkeys.Keys, error)
	Update(primitive.ObjectID, providerkeys.Keys) error
}

type SettingsHandler struct {
	checker providerKeyChecker
	keys    providerKeyStore
	cfg     config.Config
	render  *views.Renderer
}

func NewSettingsHandler(cfg config.Config, render *views.Renderer, keys providerKeyStore, checker providerKeyChecker) *SettingsHandler {
	if checker == nil {
		checker = &providerkeys.Checker{MusicBrainzUA: cfg.MusicBrainzUA}
	}
	return &SettingsHandler{cfg: cfg, render: render, keys: keys, checker: checker}
}

func (h *SettingsHandler) baseAppData(r *http.Request, titleKey string) map[string]any {
	user := middleware.CurrentUser(r.Context())
	locale := middleware.LocaleFromContext(r.Context())
	return map[string]any{
		"TitleKey":        titleKey,
		"Locale":          locale,
		"Path":            r.URL.Path,
		"User":            user,
		"MediaTypes":      domainlib.MediaTypes,
		"Buckets":         domainlib.Buckets,
		"Formats":         domainlib.Formats,
		"Values":          map[string]string{},
		"FormErrors":      []string{},
		"ContentTemplate": "",
	}
}

func (h *SettingsHandler) renderPage(w http.ResponseWriter, page string, data map[string]any) {
	if err := h.render.Render(w, page, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *SettingsHandler) consumeFlash(w http.ResponseWriter, r *http.Request, data map[string]any) {
	c, err := r.Cookie("flash")
	if err != nil {
		return
	}
	flash := middleware.DecodeFlash(c.Value)
	if flash != nil {
		data["Flash"] = flash
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *SettingsHandler) requireUser(w http.ResponseWriter, r *http.Request) *domainauth.User {
	user := middleware.CurrentUser(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil
	}
	return user
}

func (h *SettingsHandler) Page(w http.ResponseWriter, r *http.Request) {
	user := h.requireUser(w, r)
	if user == nil {
		return
	}

	data := h.settingsData(r)
	keys, err := h.keys.Get(user.ID)
	if err != nil {
		data["FormErrors"] = []string{"settings.providers.errors.load"}
	}
	data["ProviderStatuses"] = providerStatuses(keys, h.cfg)
	w.Header().Set("Cache-Control", "no-store")
	h.consumeFlash(w, r, data)
	h.renderPage(w, "pages/settings.html", data)
}

func (h *SettingsHandler) settingsData(r *http.Request) map[string]any {
	data := h.baseAppData(r, "settings.title")
	data["ContentTemplate"] = "pages/settings.content"
	data["PageScripts"] = []string{"/static/js/settings.js"}
	data["CurrentLanguageKey"] = "common.languageSwitcher." + data["Locale"].(string)
	data["DatabaseName"] = h.cfg.MongoDatabase
	data["BackupEnabled"] = h.cfg.BackupEnabled && middleware.CurrentUser(r.Context()).IsAdmin()
	return data
}

type providerStatus struct {
	ID, Name, StatusKey, KeyURL string
	Personal                    bool
}

func providerStatuses(keys providerkeys.Keys, cfg config.Config) []providerStatus {
	defaults := map[string]string{"tmdb": cfg.TMDBAPIKey, "rawg": cfg.RAWGAPIKey, "discogs": cfg.DiscogsToken}
	names := map[string]string{"tmdb": "TMDB", "rawg": "RAWG", "discogs": "Discogs"}
	keyURLs := map[string]string{
		"tmdb":    "https://www.themoviedb.org/settings/api",
		"rawg":    "https://rawg.io/apidocs",
		"discogs": "https://www.discogs.com/settings/developers",
	}
	statuses := make([]providerStatus, 0, len(providerkeys.Providers))
	for _, id := range providerkeys.Providers {
		personal := keys[id] != ""
		status := "settings.providers.status.missing"
		if defaults[id] != "" {
			status = "settings.providers.status.server"
		}
		if personal {
			status = "settings.providers.status.personal"
		}
		statuses = append(statuses, providerStatus{ID: id, Name: names[id], StatusKey: status, Personal: personal, KeyURL: keyURLs[id]})
	}
	return statuses
}

func (h *SettingsHandler) ProviderKeysSubmit(w http.ResponseWriter, r *http.Request) {
	user := h.requireUser(w, r)
	if user == nil {
		return
	}
	h.saveProviderKeys(w, r, user.ID)
}

func (h *SettingsHandler) saveProviderKeys(w http.ResponseWriter, r *http.Request, userID primitive.ObjectID) {
	if !sameSettingsOrigin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	if err := r.ParseForm(); err != nil {
		h.providerKeyError(w, r, "settings.providers.errors.invalid", http.StatusBadRequest)
		return
	}
	changes := providerkeys.Keys{}
	for _, id := range providerkeys.Providers {
		value := strings.TrimSpace(r.PostForm.Get(id))
		remove := r.PostForm.Get("remove_"+id) == "on"
		if !validProviderKey(value) || (remove && value != "") {
			h.providerKeyError(w, r, "settings.providers.errors.invalid", http.StatusBadRequest)
			return
		}
		if remove {
			changes[id] = ""
		} else if value != "" {
			changes[id] = value
		}
	}
	if len(changes) > 0 {
		if err := h.keys.Update(userID, changes); err != nil {
			h.providerKeyError(w, r, "settings.providers.errors.save", http.StatusInternalServerError)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: "flash", Value: middleware.EncodeFlash("common.flash.info", "settings.providers.saved"), Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 10})
	http.Redirect(w, r, "/"+middleware.LocaleFromContext(r.Context())+"/settings", http.StatusSeeOther)
}

func (h *SettingsHandler) providerKeyError(w http.ResponseWriter, r *http.Request, key string, status int) {
	data := h.settingsData(r)
	data["FormErrors"] = []string{key}
	// Never echo submitted values, even when validation or storage fails.
	if user := middleware.CurrentUser(r.Context()); user != nil {
		keys, _ := h.keys.Get(user.ID)
		data["ProviderStatuses"] = providerStatuses(keys, h.cfg)
	} else {
		data["ProviderStatuses"] = providerStatuses(nil, h.cfg)
	}
	w.WriteHeader(status)
	h.renderPage(w, "pages/settings.html", data)
}

func validProviderKey(value string) bool {
	if len(value) > 512 {
		return false
	}
	for _, char := range value {
		if char < 33 || char > 126 {
			return false
		}
	}
	return true
}

func sameSettingsOrigin(r *http.Request) bool {
	if origin := r.Header.Get("Origin"); origin != "" {
		parsed, err := url.Parse(origin)
		return err == nil && strings.EqualFold(parsed.Host, r.Host) && (parsed.Scheme == "http" || parsed.Scheme == "https")
	}
	return r.Header.Get("Sec-Fetch-Site") != "cross-site"
}
