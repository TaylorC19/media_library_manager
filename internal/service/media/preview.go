package media

import (
	"context"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
	domainlib "media_library_manager/internal/domain/library"
)

// Preview fetches complete provider metadata without importing a media record or
// creating a library entry. Provider detail caching still applies.
func (s *Service) Preview(ctx context.Context, form ImportForm) (*domainlib.MediaRecord, error) {
	form.Provider = strings.TrimSpace(form.Provider)
	if form.Provider == "openlibrary" {
		form.Provider = "open_library"
	}
	form.ExternalID = strings.TrimSpace(form.ExternalID)
	form.MediaType = strings.TrimSpace(form.MediaType)
	form.TMDBKind = strings.TrimSpace(form.TMDBKind)
	form.Bucket = ""
	if len(validateImportForm(form)) > 0 {
		return nil, ErrValidation
	}
	return s.fetchNormalizedRecord(ctx, form)
}

// PreviewLocal returns shared metadata only; it never exposes another user's copy fields.
func (s *Service) PreviewLocal(ctx context.Context, id primitive.ObjectID) (*domainlib.MediaRecord, error) {
	record, err := s.media.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrNotFound
	}
	return record, nil
}
