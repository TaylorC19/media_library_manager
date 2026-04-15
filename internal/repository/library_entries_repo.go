package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	domainlib "media_library_manager/internal/domain/library"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LibraryEntriesRepository struct {
	coll *mongo.Collection
}

func NewLibraryEntriesRepository(db *mongo.Database) *LibraryEntriesRepository {
	return &LibraryEntriesRepository{coll: db.Collection("library_entries")}
}

func (r *LibraryEntriesRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "bucket", Value: 1}, {Key: "mediaType", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "tags", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "mediaRecordId", Value: 1}}},
		{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "mediaRecordId", Value: 1},
				{Key: "bucket", Value: 1},
				{Key: "format", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		return fmt.Errorf("create library_entries indexes: %w", err)
	}
	return nil
}

func (r *LibraryEntriesRepository) Insert(ctx context.Context, entry *domainlib.LibraryEntry) error {
	now := time.Now().UTC()
	entry.CreatedAt = now
	entry.UpdatedAt = now

	res, err := r.coll.InsertOne(ctx, entry)
	if err != nil {
		return err
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		entry.ID = oid
	}
	return nil
}

func (r *LibraryEntriesRepository) FindByIDAndUser(ctx context.Context, id, userID primitive.ObjectID) (*domainlib.LibraryEntry, error) {
	var entry domainlib.LibraryEntry
	err := r.coll.FindOne(ctx, bson.M{"_id": id, "userId": userID}).Decode(&entry)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find library entry: %w", err)
	}
	return &entry, nil
}

func (r *LibraryEntriesRepository) FindByUserMediaBucketAndFormat(ctx context.Context, userID, mediaRecordID primitive.ObjectID, bucket string, format *string) (*domainlib.LibraryEntry, error) {
	query := bson.M{
		"userId":        userID,
		"mediaRecordId": mediaRecordID,
		"bucket":        bucket,
	}
	if format == nil || *format == "" {
		query["format"] = bson.M{"$in": []any{nil, ""}}
	} else {
		query["format"] = *format
	}

	var entry domainlib.LibraryEntry
	err := r.coll.FindOne(ctx, query).Decode(&entry)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find library entry by media record: %w", err)
	}
	return &entry, nil
}

func (r *LibraryEntriesRepository) ListByUserAndBucket(ctx context.Context, userID primitive.ObjectID, bucket string) ([]domainlib.LibraryEntry, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.coll.Find(ctx, bson.M{"userId": userID, "bucket": bucket}, opts)
	if err != nil {
		return nil, fmt.Errorf("list library entries: %w", err)
	}
	defer cur.Close(ctx)

	var entries []domainlib.LibraryEntry
	for cur.Next(ctx) {
		var e domainlib.LibraryEntry
		if err := cur.Decode(&e); err != nil {
			return nil, fmt.Errorf("decode library entry: %w", err)
		}
		entries = append(entries, e)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *LibraryEntriesRepository) CountByUserAndBucket(ctx context.Context, userID primitive.ObjectID, bucket string) (int64, error) {
	n, err := r.coll.CountDocuments(ctx, bson.M{"userId": userID, "bucket": bucket})
	if err != nil {
		return 0, fmt.Errorf("count library entries: %w", err)
	}
	return n, nil
}

func (r *LibraryEntriesRepository) CountByMediaRecordID(ctx context.Context, mediaID primitive.ObjectID) (int64, error) {
	n, err := r.coll.CountDocuments(ctx, bson.M{"mediaRecordId": mediaID})
	if err != nil {
		return 0, fmt.Errorf("count entries by media: %w", err)
	}
	return n, nil
}

func (r *LibraryEntriesRepository) Update(ctx context.Context, entry *domainlib.LibraryEntry) error {
	entry.UpdatedAt = time.Now().UTC()
	set := bson.M{
		"bucket":       entry.Bucket,
		"mediaType":    entry.MediaType,
		"format":       entry.Format,
		"barcode":      entry.Barcode,
		"purchaseDate": entry.PurchaseDate,
		"notes":        entry.Notes,
		"tags":         entry.Tags,
		"updatedAt":    entry.UpdatedAt,
	}
	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": entry.ID, "userId": entry.UserID}, bson.M{"$set": set})
	if err != nil {
		return fmt.Errorf("update library entry: %w", err)
	}
	return nil
}

func (r *LibraryEntriesRepository) DeleteByIDAndUser(ctx context.Context, id, userID primitive.ObjectID) error {
	res, err := r.coll.DeleteOne(ctx, bson.M{"_id": id, "userId": userID})
	if err != nil {
		return fmt.Errorf("delete library entry: %w", err)
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
