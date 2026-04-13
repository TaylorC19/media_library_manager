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

type MediaRecordsRepository struct {
	coll *mongo.Collection
}

func NewMediaRecordsRepository(db *mongo.Database) *MediaRecordsRepository {
	return &MediaRecordsRepository{coll: db.Collection("media_records")}
}

func (r *MediaRecordsRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "mediaType", Value: 1}, {Key: "title", Value: 1}, {Key: "year", Value: 1}}},
		{Keys: bson.D{{Key: "barcodeCandidates", Value: 1}}},
		{Keys: bson.D{{Key: "source", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("create media_records indexes: %w", err)
	}
	return nil
}

func (r *MediaRecordsRepository) Insert(ctx context.Context, rec *domainlib.MediaRecord) error {
	now := time.Now().UTC()
	rec.CreatedAt = now
	rec.UpdatedAt = now
	if rec.Details == nil {
		rec.Details = bson.M{}
	}
	if rec.ProviderRefs == nil {
		rec.ProviderRefs = bson.M{}
	}

	res, err := r.coll.InsertOne(ctx, rec)
	if err != nil {
		return fmt.Errorf("insert media record: %w", err)
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		rec.ID = oid
	}
	return nil
}

func (r *MediaRecordsRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*domainlib.MediaRecord, error) {
	var rec domainlib.MediaRecord
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&rec)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find media record: %w", err)
	}
	return &rec, nil
}

func (r *MediaRecordsRepository) UpdateManualCore(ctx context.Context, id primitive.ObjectID, title string, sortTitle *string, mediaType string, year *int32) error {
	set := bson.M{
		"title":     title,
		"mediaType": mediaType,
		"updatedAt": time.Now().UTC(),
	}
	if sortTitle != nil {
		set["sortTitle"] = *sortTitle
	} else {
		set["sortTitle"] = nil
	}
	if year != nil {
		set["year"] = *year
	} else {
		set["year"] = nil
	}

	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": id, "source": domainlib.SourceManual}, bson.M{"$set": set})
	if err != nil {
		return fmt.Errorf("update media record: %w", err)
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *MediaRecordsRepository) DeleteByID(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete media record: %w", err)
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *MediaRecordsRepository) FindByIDs(ctx context.Context, ids []primitive.ObjectID) (map[primitive.ObjectID]*domainlib.MediaRecord, error) {
	if len(ids) == 0 {
		return map[primitive.ObjectID]*domainlib.MediaRecord{}, nil
	}
	cur, err := r.coll.Find(ctx, bson.M{"_id": bson.M{"$in": ids}}, options.Find().SetProjection(bson.M{
		"title": 1, "year": 1, "mediaType": 1, "source": 1, "summary": 1, "imageUrl": 1,
	}))
	if err != nil {
		return nil, fmt.Errorf("find media records by ids: %w", err)
	}
	defer cur.Close(ctx)

	out := make(map[primitive.ObjectID]*domainlib.MediaRecord)
	for cur.Next(ctx) {
		var rec domainlib.MediaRecord
		if err := cur.Decode(&rec); err != nil {
			return nil, fmt.Errorf("decode media record: %w", err)
		}
		out[rec.ID] = &rec
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
