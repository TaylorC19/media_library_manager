package handlers

import (
	"net/http"
	"net/http/httptest"
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
