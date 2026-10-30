package handlers

import (
	domainauth "media_library_manager/internal/domain/auth"
	"media_library_manager/internal/http/middleware"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"media_library_manager/internal/config"
)

func TestBackupExportRequiresAuth(t *testing.T) {
	h := NewBackupHandler(config.Config{BackupEnabled: true}, nil)
	req := httptest.NewRequest(http.MethodPost, "/settings/backup/export", nil)
	rec := httptest.NewRecorder()

	h.ExportSubmit(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestBackupImportRequiresAuth(t *testing.T) {
	h := NewBackupHandler(config.Config{BackupEnabled: true}, nil)
	req := httptest.NewRequest(http.MethodPost, "/settings/backup/import", nil)
	rec := httptest.NewRecorder()

	h.ImportSubmit(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestBackupExportDisabled(t *testing.T) {
	h := NewBackupHandler(config.Config{BackupEnabled: false}, nil)
	req := httptest.NewRequest(http.MethodPost, "/settings/backup/export", nil)
	rec := httptest.NewRecorder()

	// Without auth, unauthorized wins first; test disabled path via nil user is 401.
	// Disabled is tested at import with auth in integration; here we only verify handler exists.
	h.ExportSubmit(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestBackupActionsRejectEveryNonAdminBeforeDatabaseAccess(t *testing.T) {
	for _, role := range []string{"", "general", "demo", "unknown"} {
		handler := NewBackupHandler(config.Config{BackupEnabled: true}, nil)
		for _, action := range []http.HandlerFunc{handler.ExportSubmit, handler.ImportSubmit} {
			r := httptest.NewRequest(http.MethodPost, "/settings/backup", nil)
			r = r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{Role: role}))
			w := httptest.NewRecorder()
			action(w, r)
			if w.Code != http.StatusForbidden {
				t.Errorf("role %q: got %d, want 403", role, w.Code)
			}
		}
	}
}

func TestAdminStillHonorsBackupDisableAndOriginCheck(t *testing.T) {
	handler := NewBackupHandler(config.Config{BackupEnabled: false}, nil)
	r := httptest.NewRequest(http.MethodPost, "/settings/backup/export", nil)
	r = r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{Role: domainauth.RoleAdmin}))
	w := httptest.NewRecorder()
	handler.ExportSubmit(w, r)
	if w.Code != 403 || !strings.Contains(w.Body.String(), "backup disabled") {
		t.Error("admin must honor disabled backups")
	}
	handler.cfg.BackupEnabled = true
	r.Header.Set("Origin", "https://untrusted.invalid")
	w = httptest.NewRecorder()
	handler.ExportSubmit(w, r)
	if w.Code != 403 {
		t.Error("admin backup requests must reject cross-origin submissions")
	}
}
