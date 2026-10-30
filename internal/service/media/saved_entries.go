package media

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
	domainlib "media_library_manager/internal/domain/library"
)

// SavedEntry exposes navigation and copy labels, never private notes or other
// accounts' entries. Metadata can be shared, but saved state is always personal.
type SavedEntry struct {
	ID     string
	Bucket string
	Format *string
}

func (s *Service) SavedEntries(ctx context.Context, userID primitive.ObjectID, record *domainlib.MediaRecord, form ImportForm) (map[string][]SavedEntry, error) {
	saved := map[string][]SavedEntry{}
	id := record.ID
	if id.IsZero() {
		existing, err := s.media.FindByProviderImportRef(ctx, form.Provider, form.ExternalID, form.TMDBKind)
		if err != nil {
			return nil, err
		}
		if existing == nil || existing.MediaType != record.MediaType {
			return saved, nil
		}
		id = existing.ID
	}
	entries, err := s.entries.FindByUserAndMediaRecordIDs(ctx, userID, []primitive.ObjectID{id})
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.Bucket != domainlib.BucketCatalog && entry.Bucket != domainlib.BucketWishlist {
			continue
		}
		saved[entry.Bucket] = append(saved[entry.Bucket], SavedEntry{ID: entry.ID.Hex(), Bucket: entry.Bucket, Format: entry.Format})
	}
	return saved, nil
}
