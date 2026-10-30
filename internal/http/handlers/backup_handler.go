package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"

	"media_library_manager/internal/backup"
	"media_library_manager/internal/config"
	"media_library_manager/internal/http/middleware"
)

type BackupHandler struct {
	cfg config.Config
	db  *mongo.Database
}

func NewBackupHandler(cfg config.Config, db *mongo.Database) *BackupHandler {
	return &BackupHandler{cfg: cfg, db: db}
}

func (h *BackupHandler) ExportSubmit(w http.ResponseWriter, r *http.Request) {
	if middleware.CurrentUser(r.Context()) == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !h.cfg.BackupEnabled {
		http.Error(w, "backup disabled", http.StatusForbidden)
		return
	}

	ctx := r.Context()
	data, err := backup.ExportDatabase(ctx, h.db)
	if err != nil {
		http.Error(w, "export failed", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("%s-%s.mlm-backup.json.gz",
		h.cfg.MongoDatabase,
		time.Now().UTC().Format("20060102-150405"),
	)
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	_, _ = w.Write(data)
}

func (h *BackupHandler) ImportSubmit(w http.ResponseWriter, r *http.Request) {
	locale := middleware.LocaleFromContext(r.Context())
	if middleware.CurrentUser(r.Context()) == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !h.cfg.BackupEnabled {
		h.flashRedirect(w, r, "/"+locale+"/settings", "common.flash.error", "settings.backup.errors.disabled")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.BackupMaxUploadBytes)
	if err := r.ParseMultipartForm(h.cfg.BackupMaxUploadBytes); err != nil {
		h.flashRedirect(w, r, "/"+locale+"/settings", "common.flash.error", "settings.backup.errors.invalidUpload")
		return
	}

	confirm := strings.TrimSpace(r.FormValue("confirm"))
	if confirm != h.cfg.MongoDatabase {
		h.flashRedirect(w, r, "/"+locale+"/settings", "common.flash.error", "settings.backup.errors.confirmMismatch")
		return
	}

	file, _, err := r.FormFile("backup")
	if err != nil {
		h.flashRedirect(w, r, "/"+locale+"/settings", "common.flash.error", "settings.backup.errors.invalidUpload")
		return
	}
	defer file.Close()

	_, err = backup.ImportDatabase(r.Context(), h.db, file, backup.ImportOptions{RequireDatabaseMatch: true})
	if err != nil {
		h.flashRedirect(w, r, "/"+locale+"/settings", "common.flash.error", "settings.backup.errors.importFailed")
		return
	}

	h.flashRedirect(w, r, "/"+locale+"/settings", "common.flash.info", "settings.backup.flash.imported")
}

func (h *BackupHandler) flashRedirect(w http.ResponseWriter, r *http.Request, target, level, msgKey string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    middleware.EncodeFlash(level, msgKey),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   10,
	})
	http.Redirect(w, r, target, http.StatusFound)
}
