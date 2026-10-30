package handlers

import (
	"bytes"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"media_library_manager/internal/http/middleware"
	libsvc "media_library_manager/internal/service/library"
)

// DeleteConfirmation is read-only; only the explicit confirmation POST deletes.
func (h *LibraryHandler) DeleteConfirmation(w http.ResponseWriter, r *http.Request) {
	user := h.requireUser(w, r)
	if user == nil {
		return
	}
	if !user.CanWrite() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	id, ok := parseObjectIDParam(w, r, chi.URLParam(r, "entryId"))
	if !ok {
		return
	}
	detail, err := h.library.GetEntryDetail(r.Context(), user.ID, id)
	if errors.Is(err, libsvc.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Could not load item", http.StatusInternalServerError)
		return
	}
	h.renderDeleteConfirmation(w, r, detail, middleware.LocaleFromContext(r.Context()), "", http.StatusOK)
}

func (h *LibraryHandler) renderDeleteConfirmation(w http.ResponseWriter, r *http.Request, detail *libsvc.EntryWithMedia, locale, errorKey string, status int) {
	data := h.baseAppData(r, "library.delete.heading")
	data["Locale"] = locale
	data["ContentTemplate"] = "pages/library_delete.content"
	data["Detail"] = detail
	data["DeleteError"] = errorKey
	var body bytes.Buffer
	if err := h.render.Render(&body, "pages/library_delete.html", data); err != nil {
		http.Error(w, "Could not render confirmation", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	body.WriteTo(w)
}
