package backup

import "time"

const (
	FormatName    = "media_library_manager.backup"
	FormatVersion = 1
)

// Manifest describes a backup file header.
type Manifest struct {
	Format     string    `json:"format"`
	Version    int       `json:"version"`
	ExportedAt time.Time `json:"exportedAt"`
	Database   string    `json:"database"`
}

// File is the on-disk JSON structure before gzip compression.
type File struct {
	Manifest    Manifest              `json:"manifest"`
	Collections map[string][]string   `json:"collections"`
}
