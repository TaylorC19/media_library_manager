package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	domainlib "media_library_manager/internal/domain/library"
)

// ProviderImportRef identifies a provider item without relying on its title.
type ProviderImportRef struct {
	Provider, ExternalID, MediaType, TMDBKind string
}

// FindByProviderImportRefs batches metadata lookup for the visible search hits.
func (r *MediaRecordsRepository) FindByProviderImportRefs(ctx context.Context, refs []ProviderImportRef) ([]domainlib.MediaRecord, error) {
	filters := bson.A{}
	for _, ref := range refs {
		if ref.ExternalID == "" {
			continue
		}
		filter := bson.M{"mediaType": ref.MediaType}
		switch ref.Provider {
		case "tmdb":
			filter["providerRefs.tmdb.id"] = ref.ExternalID
			filter["providerRefs.tmdb.kind"] = ref.TMDBKind
		case "musicbrainz":
			filter["providerRefs.musicbrainz.releaseId"] = ref.ExternalID
		case "open_library":
			filter["providerRefs.openLibrary.workKey"] = ref.ExternalID
		case "rawg":
			filter["providerRefs.rawg.id"] = ref.ExternalID
		default:
			continue
		}
		filters = append(filters, filter)
	}
	if len(filters) == 0 {
		return nil, nil
	}
	cursor, err := r.coll.Find(ctx, bson.M{"$or": filters}, options.Find().SetProjection(bson.M{"_id": 1, "mediaType": 1, "providerRefs": 1}))
	if err != nil {
		return nil, fmt.Errorf("find media by search refs: %w", err)
	}
	defer cursor.Close(ctx)
	var records []domainlib.MediaRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, fmt.Errorf("decode search refs: %w", err)
	}
	return records, nil
}
