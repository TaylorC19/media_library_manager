package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	domainauth "media_library_manager/internal/domain/auth"
	"media_library_manager/internal/http/middleware"
)

func TestAdminKeyHandlersRejectOtherRolesBeforeReadingKeys(t *testing.T) {
	handler := NewUserKeysHandler(nil, nil)
	for _, role := range []string{"", "general", "demo", "invalid"} {
		for _, action := range []http.HandlerFunc{handler.Page, handler.Confirm, handler.Clear} {
			r := httptest.NewRequest("POST", "/en/admin/users/id/keys/clear", nil)
			r = r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{Role: role}))
			w := httptest.NewRecorder()
			action(w, r)
			if w.Code != 403 {
				t.Error("only admins may inspect or clear saved-key status")
			}
		}
	}
	w := httptest.NewRecorder()
	handler.Clear(w, httptest.NewRequest("POST", "/en/admin/users/id/keys/clear", nil))
	if w.Code != 401 {
		t.Error("clearing requires authentication")
	}
}
