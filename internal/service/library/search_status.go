package library

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	domainlib "media_library_manager/internal/domain/library"
	domainsearch "media_library_manager/internal/domain/search"
	"media_library_manager/internal/repository"
)

// SavedSearchHits enriches a request-local copy of visible hits. User-specific
// status never enters the provider cache, and titles are never used as identity.
func (s *Service) SavedSearchHits(ctx context.Context, userID primitive.ObjectID, hits []domainsearch.Hit) ([]domainsearch.Hit, error) {
	result := append([]domainsearch.Hit(nil), hits...)
	refs := make([]repository.ProviderImportRef, 0, len(hits))
	for i, hit := range result {
		result[i].InCollection = false
		result[i].InWishlist = false
		refs = append(refs, repository.ProviderImportRef{Provider: hit.Provider, ExternalID: hit.ExternalID, MediaType: hit.MediaType, TMDBKind: hit.TMDBKind})
	}
	records, err := s.media.FindByProviderImportRefs(ctx, refs)
	if err != nil {
		return nil, err
	}
	ids := make([]primitive.ObjectID, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ID)
	}
	entries, err := s.entries.FindByUserAndMediaRecordIDs(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	type status struct{ collection, wishlist bool }
	saved := map[primitive.ObjectID]status{}
	for _, entry := range entries {
		state := saved[entry.MediaRecordID]
		state.collection = state.collection || entry.Bucket == domainlib.BucketCatalog
		state.wishlist = state.wishlist || entry.Bucket == domainlib.BucketWishlist
		saved[entry.MediaRecordID] = state
	}
	for i, hit := range result {
		for _, record := range records {
			if !matchesSearchHit(record, hit) {
				continue
			}
			state := saved[record.ID]
			result[i].InCollection = result[i].InCollection || state.collection
			result[i].InWishlist = result[i].InWishlist || state.wishlist
		}
	}
	return result, nil
}

func matchesSearchHit(record domainlib.MediaRecord, hit domainsearch.Hit) bool {
	if record.MediaType != hit.MediaType {
		return false
	}
	provider, key := hit.Provider, "id"
	switch provider {
	case "tmdb":
	case "musicbrainz":
		key = "releaseId"
	case "open_library":
		provider = "openLibrary"
		key = "workKey"
	case "rawg":
	default:
		return false
	}
	ref, ok := record.ProviderRefs[provider].(bson.M)
	if !ok || ref[key] != hit.ExternalID {
		return false
	}
	return hit.Provider != "tmdb" || ref["kind"] == hit.TMDBKind
}
