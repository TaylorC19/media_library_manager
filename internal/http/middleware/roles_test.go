package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	domainauth "media_library_manager/internal/domain/auth"
)

func TestWritableRoleGuard(t *testing.T) {
	for _, role := range []string{domainauth.RoleAdmin, domainauth.RoleGeneral, domainauth.RoleDemo, "invalid", ""} {
		called := false
		next := RequireWritable(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(204) }))
		r := httptest.NewRequest(http.MethodPost, "/library", nil)
		user := &domainauth.User{Role: role}
		r = r.WithContext(WithUser(r.Context(), user))
		w := httptest.NewRecorder()
		next.ServeHTTP(w, r)
		if called != user.CanWrite() {
			t.Errorf("wrong write access for %q", role)
		}
		if !user.CanWrite() && w.Code != 403 {
			t.Errorf("read-only role needs 403, got %d", w.Code)
		}
	}
}
