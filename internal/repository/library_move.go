package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	domainlib "media_library_manager/internal/domain/library"
)

// MoveWishlistToCollection changes only bucket and updatedAt. Copy fields are
// never read and rewritten, so concurrent edits to them are preserved.
func (r *LibraryEntriesRepository) MoveWishlistToCollection(ctx context.Context, userID, entryID primitive.ObjectID) error {
	result, err := r.coll.UpdateOne(ctx, bson.M{"_id": entryID, "userId": userID, "bucket": domainlib.BucketWishlist}, bson.M{"$set": bson.M{"bucket": domainlib.BucketCatalog, "updatedAt": time.Now().UTC()}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
