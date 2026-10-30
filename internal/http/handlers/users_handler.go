package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	domainauth "media_library_manager/internal/domain/auth"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/service/adminusers"
	authsvc "media_library_manager/internal/service/auth"
	"media_library_manager/internal/views"
)

type userManager interface {
	List(ctx context.Context, actor primitive.ObjectID, query, role string, page int) ([]domainauth.User, int64, error)
	ChangeRole(context.Context, primitive.ObjectID, primitive.ObjectID, string) error
	Create(context.Context, primitive.ObjectID, string, string, string, string) error
}
type UsersHandler struct {
	render *views.Renderer
	users  userManager
}

func NewUsersHandler(render *views.Renderer, users userManager) *UsersHandler {
	return &UsersHandler{render: render, users: users}
}
func requireAdminUser(w http.ResponseWriter, r *http.Request) *domainauth.User {
	user := middleware.CurrentUser(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", 401)
		return nil
	}
	if !user.IsAdmin() {
		http.Error(w, "administrator access required", 403)
		return nil
	}
	return user
}
func (h *UsersHandler) Page(w http.ResponseWriter, r *http.Request) {
	actor := requireAdminUser(w, r)
	if actor == nil {
		return
	}
	h.page(w, r, actor, "")
}
func (h *UsersHandler) page(w http.ResponseWriter, r *http.Request, actor *domainauth.User, errorKey string) {
	w.Header().Set("Cache-Control", "no-store")
	query := managedUserQuery(r.URL.Query().Get("q"))
	roleFilter := r.URL.Query().Get("role")
	page := parsePage(r.URL.Query().Get("page"))
	if r.Method == http.MethodPost {
		query = managedUserQuery(r.PostForm.Get("q"))
		roleFilter = r.PostForm.Get("filter_role")
		page = parsePage(r.PostForm.Get("page"))
	}
	users, count, err := h.users.List(r.Context(), actor.ID, query, roleFilter, page)
	if err != nil {
		if errors.Is(err, adminusers.ErrForbidden) {
			http.Error(w, "administrator access required", 403)
		} else if errors.Is(err, adminusers.ErrInvalidRole) {
			http.Error(w, "Invalid role filter", 400)
		} else {
			http.Error(w, "Could not load users", 500)
		}
		return
	}
	locale := middleware.LocaleFromContext(r.Context())
	path := "/" + locale + "/admin/users"
	data := map[string]any{"TitleKey": "users.title", "Locale": locale, "Path": r.URL.Path, "LocalePath": r.URL.RequestURI(), "User": actor, "ContentTemplate": "pages/users.content", "ManagedUsers": users, "Query": query, "RoleFilter": roleFilter, "RoleOptions": []string{domainauth.RoleGeneral, domainauth.RoleDemo, domainauth.RoleAdmin}, "ErrorKey": errorKey, "Pagination": buildPagination(path, url.Values{"q": {query}, "role": {roleFilter}}, page, 20, int(count))}
	createRole := domainauth.RoleGeneral
	if chi.URLParam(r, "userId") == "" {
		createRole = r.PostForm.Get("role")
	}
	if !domainauth.IsRole(createRole) {
		createRole = domainauth.RoleGeneral
	}
	data["CreateRole"] = createRole
	if errorKey != "" && chi.URLParam(r, "userId") == "" {
		data["CreateOpen"] = true
		data["NewUsername"] = r.PostForm.Get("username")
		data["NewDisplayName"] = r.PostForm.Get("display_name")
	}
	if cookie, err := r.Cookie("flash"); err == nil {
		data["Flash"] = middleware.DecodeFlash(cookie.Value)
		http.SetCookie(w, &http.Cookie{Name: "flash", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	}
	if err := h.render.Render(w, "pages/users.html", data); err != nil {
		http.Error(w, "Could not render users", 500)
	}
}
func (h *UsersHandler) ChangeRole(w http.ResponseWriter, r *http.Request) {
	actor := requireAdminUser(w, r)
	if actor == nil {
		return
	}
	if !h.parse(w, r) {
		return
	}
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "userId"))
	if err != nil {
		http.Error(w, "Invalid user", 400)
		return
	}
	if err := h.users.ChangeRole(r.Context(), actor.ID, id, r.PostForm.Get("role")); err != nil {
		h.failure(w, r, actor, err)
		return
	}
	h.redirect(w, r, "users.saved")
}
func (h *UsersHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor := requireAdminUser(w, r)
	if actor == nil {
		return
	}
	if !h.parse(w, r) {
		return
	}
	err := h.users.Create(r.Context(), actor.ID, r.PostForm.Get("username"), r.PostForm.Get("password"), r.PostForm.Get("display_name"), r.PostForm.Get("role"))
	if err != nil {
		h.failure(w, r, actor, err)
		return
	}
	h.redirect(w, r, "users.created")
}
func (h *UsersHandler) parse(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Cache-Control", "no-store")
	if !sameSettingsOrigin(r) {
		http.Error(w, "Forbidden", 403)
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", 400)
		return false
	}
	return true
}
func (h *UsersHandler) failure(w http.ResponseWriter, r *http.Request, actor *domainauth.User, err error) {
	key := "users.errors.save"
	switch {
	case errors.Is(err, adminusers.ErrForbidden):
		http.Error(w, "administrator access required", 403)
		return
	case errors.Is(err, adminusers.ErrSelfDemotion):
		key = "users.errors.selfDemotion"
	case errors.Is(err, adminusers.ErrNotFound):
		key = "users.errors.notFound"
	case errors.Is(err, adminusers.ErrInvalidRole), errors.Is(err, authsvc.ErrInvalidInput):
		key = "users.errors.invalid"
	case errors.Is(err, authsvc.ErrUsernameTaken):
		key = "users.errors.usernameTaken"
	}
	h.page(w, r, actor, key)
}
func (h *UsersHandler) redirect(w http.ResponseWriter, r *http.Request, message string) {
	http.SetCookie(w, &http.Cookie{Name: "flash", Value: middleware.EncodeFlash("common.flash.info", message), Path: "/", MaxAge: 10, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	values := url.Values{}
	if role := r.PostForm.Get("filter_role"); domainauth.IsRole(role) {
		values.Set("role", role)
	}
	query := managedUserQuery(r.PostForm.Get("q"))
	if query != "" {
		values.Set("q", query)
	}
	page := parsePage(r.PostForm.Get("page"))
	if page > 1 {
		values.Set("page", r.PostForm.Get("page"))
	}
	target := "/" + middleware.LocaleFromContext(r.Context()) + "/admin/users"
	if len(values) > 0 {
		target += "?" + values.Encode()
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func managedUserQuery(query string) string {
	runes := []rune(strings.TrimSpace(query))
	if len(runes) > 200 {
		runes = runes[:200]
	}
	return string(runes)
}
