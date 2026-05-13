package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// ScanLog is a user-scoped record of a barcode lookup attempt (optional collection).
type ScanLog struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	UserID            primitive.ObjectID `bson:"userId"`
	Barcode           string             `bson:"barcode"`
	MatchedMediaType  *string            `bson:"matchedMediaType"`
	MatchedProvider   *string            `bson:"matchedProvider"`
	CreatedAt         time.Time          `bson:"createdAt"`
}

// ScanLogsRepository persists to scan_logs.
type ScanLogsRepository struct {
	coll *mongo.Collection
}

func NewScanLogsRepository(db *mongo.Database) *ScanLogsRepository {
	return &ScanLogsRepository{coll: db.Collection("scan_logs")}
}

func (r *ScanLogsRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "barcode", Value: 1}, {Key: "createdAt", Value: -1}}},
	})
	if err != nil {
		return fmt.Errorf("create scan_logs indexes: %w", err)
	}
	return nil
}

// Insert records a lookup attempt. matchedProvider is only set for provider-sourced top candidates.
func (r *ScanLogsRepository) Insert(ctx context.Context, userID primitive.ObjectID, barcode string, matchedMediaType *string, matchedProvider *string) error {
	barcode = strings.TrimSpace(barcode)
	now := time.Now().UTC()
	doc := ScanLog{
		UserID:           userID,
		Barcode:          barcode,
		MatchedMediaType: matchedMediaType,
		MatchedProvider:  matchedProvider,
		CreatedAt:        now,
	}
	_, err := r.coll.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("insert scan log: %w", err)
	}
	return nil
}
