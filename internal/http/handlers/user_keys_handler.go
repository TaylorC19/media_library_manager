package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/service/adminusers"
	"media_library_manager/internal/views"
)

type adminKeyManager interface {
	Describe(context.Context, primitive.ObjectID, primitive.ObjectID) (*adminusers.KeyOverview, error)
	Clear(context.Context, primitive.ObjectID, primitive.ObjectID, string) error
}
type UserKeysHandler struct {
	render *views.Renderer
	keys   adminKeyManager
}

func NewUserKeysHandler(render *views.Renderer, keys adminKeyManager) *UserKeysHandler {
	return &UserKeysHandler{render: render, keys: keys}
}

func (h *UserKeysHandler) Page(w http.ResponseWriter, r *http.Request) {
	user := requireAdminUser(w, r)
	if user == nil {
		return
	}
	id, ok := parseObjectIDParam(w, r, chi.URLParam(r, "userId"))
	if !ok {
		return
	}
	overview, err := h.keys.Describe(r.Context(), user.ID, id)
	if err != nil {
		h.failure(w, err)
		return
	}
	data := h.data(r, overview)
	data["ContentTemplate"] = "pages/user_keys.content"
	if cookie, err := r.Cookie("flash"); err == nil {
		data["Flash"] = middleware.DecodeFlash(cookie.Value)
		http.SetCookie(w, &http.Cookie{Name: "flash", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	}
	h.page(w, "pages/user_keys.html", data, 200)
}
func (h *UserKeysHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	user := requireAdminUser(w, r)
	if user == nil {
		return
	}
	id, ok := parseObjectIDParam(w, r, chi.URLParam(r, "userId"))
	if !ok {
		return
	}
	h.confirmation(w, r, user.ID, id, r.URL.Query().Get("provider"), "", 200)
}
func (h *UserKeysHandler) Clear(w http.ResponseWriter, r *http.Request) {
	user := requireAdminUser(w, r)
	if user == nil {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if !sameSettingsOrigin(r) {
		http.Error(w, "Forbidden", 403)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", 400)
		return
	}
	id, ok := parseObjectIDParam(w, r, chi.URLParam(r, "userId"))
	if !ok {
		return
	}
	provider := r.PostForm.Get("provider")
	if r.PostForm.Get("confirm") != "clear" {
		h.confirmation(w, r, user.ID, id, provider, "users.keys.errors.confirm", 400)
		return
	}
	if err := h.keys.Clear(r.Context(), user.ID, id, provider); err != nil {
		if errors.Is(err, adminusers.ErrForbidden) || errors.Is(err, adminusers.ErrNotFound) || errors.Is(err, adminusers.ErrInvalidProvider) {
			h.failure(w, err)
			return
		}
		h.confirmation(w, r, user.ID, id, provider, "users.keys.errors.save", 500)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "flash", Value: middleware.EncodeFlash("common.flash.info", "users.keys.cleared"), Path: "/", MaxAge: 10, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	locale := middleware.LocaleFromContext(r.Context())
	http.Redirect(w, r, keyManagementURL(locale, id, userListValues(r)), http.StatusSeeOther)
}
func (h *UserKeysHandler) confirmation(w http.ResponseWriter, r *http.Request, actor, target primitive.ObjectID, provider, errorKey string, status int) {
	overview, err := h.keys.Describe(r.Context(), actor, target)
	if err != nil {
		h.failure(w, err)
		return
	}
	var selected *adminusers.SavedKey
	for _, key := range overview.Providers {
		if key.ID == provider {
			copy := key
			selected = &copy
			break
		}
	}
	if selected == nil {
		http.Error(w, "Invalid provider", 400)
		return
	}
	data := h.data(r, overview)
	data["Provider"] = selected
	data["ErrorKey"] = errorKey
	data["ContentTemplate"] = "pages/user_key_clear.content"
	h.page(w, "pages/user_key_clear.html", data, status)
}
func (h *UserKeysHandler) data(r *http.Request, overview *adminusers.KeyOverview) map[string]any {
	locale := middleware.LocaleFromContext(r.Context())
	values := userListValues(r)
	return map[string]any{"TitleKey": "users.keys.title", "Locale": locale, "Path": r.URL.Path, "LocalePath": r.URL.RequestURI(), "User": middleware.CurrentUser(r.Context()), "Overview": overview, "AdminUsersPage": true, "ListValues": values, "BackToUsers": userListURL(locale, values), "KeysURL": keyManagementURL(locale, overview.UserID, values)}
}
func (h *UserKeysHandler) page(w http.ResponseWriter, page string, data map[string]any, status int) {
	var body bytes.Buffer
	if err := h.render.Render(&body, page, data); err != nil {
		http.Error(w, "Could not render saved keys", 500)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	body.WriteTo(w)
}
func (h *UserKeysHandler) failure(w http.ResponseWriter, err error) {
	w.Header().Set("Cache-Control", "no-store")
	switch {
	case errors.Is(err, adminusers.ErrForbidden):
		http.Error(w, "administrator access required", 403)
	case errors.Is(err, adminusers.ErrNotFound):
		http.Error(w, "Account not found", 404)
	case errors.Is(err, adminusers.ErrInvalidProvider):
		http.Error(w, "Invalid provider", 400)
	default:
		http.Error(w, "Could not access saved API keys", 500)
	}
}
func userListValues(r *http.Request) url.Values {
	values := url.Values{}
	query, role, page := r.URL.Query().Get("q"), r.URL.Query().Get("role"), r.URL.Query().Get("page")
	if r.Method == http.MethodPost {
		query, role, page = r.PostForm.Get("q"), r.PostForm.Get("filter_role"), r.PostForm.Get("page")
	}
	if q := managedUserQuery(query); q != "" {
		values.Set("q", q)
	}
	if role == "admin" || role == "general" || role == "demo" {
		values.Set("role", role)
	}
	if parsePage(page) > 1 {
		values.Set("page", page)
	}
	return values
}
func userListURL(locale string, values url.Values) string {
	path := "/" + locale + "/admin/users"
	if len(values) > 0 {
		path += "?" + values.Encode()
	}
	return path
}
func keyManagementURL(locale string, id primitive.ObjectID, values url.Values) string {
	path := "/" + locale + "/admin/users/" + id.Hex() + "/keys"
	if len(values) > 0 {
		path += "?" + values.Encode()
	}
	return path
}
