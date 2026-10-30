package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"media_library_manager/internal/http/middleware"
	libsvc "media_library_manager/internal/service/library"
)

func (h *LibraryHandler) MoveToCollectionSubmit(w http.ResponseWriter, r *http.Request) {
	user := h.requireUser(w, r)
	if user == nil {
		return
	}
	if !user.CanWrite() || !sameSettingsOrigin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	entryID, ok := parseObjectIDParam(w, r, chi.URLParam(r, "entryId"))
	if !ok {
		return
	}
	locale := r.PostForm.Get("locale")
	if locale != "en" && locale != "ja" {
		locale = middleware.LocaleFromContext(r.Context())
	}
	target := "/" + locale + "/library/" + entryID.Hex()
	err := h.library.MoveToCollection(r.Context(), user.ID, entryID)
	if errors.Is(err, libsvc.ErrCollectionConflict) {
		h.setFlashRedirect(w, r, target, "common.flash.error", "library.errors.moveConflict")
		return
	}
	if errors.Is(err, libsvc.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Could not move item", http.StatusInternalServerError)
		return
	}
	h.setFlashRedirect(w, r, target, "common.flash.info", "library.flash.movedToCollection")
}
