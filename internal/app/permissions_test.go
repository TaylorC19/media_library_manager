package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"media_library_manager/internal/config"
	domainauth "media_library_manager/internal/domain/auth"
	"media_library_manager/internal/http/middleware"
)

func TestDemoCannotReachAnyWriteRouteEvenWithForgedRoleInput(t *testing.T) {
	instance := &App{Config: config.Config{DefaultLocale: "en", SessionCookieName: "session", ProviderKeysDir: t.TempDir()}}
	router := instance.newRouter(fstest.MapFS{}, nil, nil, nil, nil, nil)
	for _, test := range []struct{ method, path string }{
		{"POST", "/library"}, {"POST", "/library/attach"}, {"POST", "/library/012345678901234567890123/update"},
		{"POST", "/library/012345678901234567890123/delete"}, {"POST", "/media/import"},
		{"POST", "/media/refresh/012345678901234567890123"}, {"POST", "/en/settings/providers"},
		{"GET", "/en/library/new"}, {"GET", "/en/library/012345678901234567890123/edit"},
	} {
		r := httptest.NewRequest(test.method, test.path, strings.NewReader("role=admin"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r = r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{Role: domainauth.RoleDemo}))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Errorf("%s %s: got %d, want 403", test.method, test.path, w.Code)
		}
	}
}

func TestActualBackupRoutesRejectAllNonAdmins(t *testing.T) {
	instance := &App{Config: config.Config{BackupEnabled: true, DefaultLocale: "en", SessionCookieName: "session", ProviderKeysDir: t.TempDir()}}
	router := instance.newRouter(fstest.MapFS{}, nil, nil, nil, nil, nil)
	for _, role := range []string{"", "general", "demo", "invalid"} {
		for _, path := range []string{"/settings/backup/export", "/settings/backup/import"} {
			r := httptest.NewRequest("POST", path, nil)
			r = r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{Role: role}))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != http.StatusForbidden {
				t.Errorf("role %q %s: got %d, want 403", role, path, w.Code)
			}
		}
	}
}
