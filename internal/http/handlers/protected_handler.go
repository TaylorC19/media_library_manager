package handlers

import (
	"net/http"

	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/views"
)

type ProtectedHandler struct {
	render *views.Renderer
}

func NewProtectedHandler(render *views.Renderer) *ProtectedHandler {
	return &ProtectedHandler{render: render}
}

func (h *ProtectedHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.CurrentUser(r.Context())
	if user == nil {
		http.Error(w, "missing user", http.StatusUnauthorized)
		return
	}

	data := map[string]any{
		"Title":           "Dashboard - Media Library Manager",
		"PageTitle":       "Dashboard",
		"Locale":          middleware.LocaleFromContext(r.Context()),
		"User":            user,
		"ContentTemplate": "pages/dashboard.content",
	}
	if err := h.render.Render(w, "pages/dashboard.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
