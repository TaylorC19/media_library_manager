package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ListTags is scoped to the signed-in account and current library bucket.
func (r *LibraryEntriesRepository) ListTags(ctx context.Context, userID primitive.ObjectID, bucket string) ([]string, error) {
	values, err := r.coll.Distinct(ctx, "tags", bson.M{"userId": userID, "bucket": bucket})
	if err != nil {
		return nil, err
	}
	tags := []string{}
	for _, value := range values {
		if text, ok := value.(string); ok {
			tags = append(tags, text)
		}
	}
	return tags, nil
}
