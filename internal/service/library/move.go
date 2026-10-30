package library

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	domainlib "media_library_manager/internal/domain/library"
)

var ErrCollectionConflict = errors.New("collection already contains this media and format")

func (s *Service) MoveToCollection(ctx context.Context, userID, entryID primitive.ObjectID) error {
	err := s.entries.MoveWishlistToCollection(ctx, userID, entryID)
	if mongo.IsDuplicateKeyError(err) {
		return ErrCollectionConflict
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		// Repeated submissions for an already moved, owned entry are harmless.
		entry, lookupErr := s.entries.FindByIDAndUser(ctx, entryID, userID)
		if lookupErr != nil {
			return lookupErr
		}
		if entry != nil && entry.Bucket == domainlib.BucketCatalog {
			return nil
		}
		return ErrNotFound
	}
	return err
}
