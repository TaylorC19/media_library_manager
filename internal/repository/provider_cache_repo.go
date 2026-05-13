package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProviderCacheRepository struct {
	coll *mongo.Collection
}

type providerCacheDocument struct {
	Provider  string           `bson:"provider"`
	CacheKey  string           `bson:"cacheKey"`
	Payload   primitive.Binary `bson:"payload"`
	ExpiresAt time.Time        `bson:"expiresAt"`
	CreatedAt time.Time        `bson:"createdAt"`
}

func NewProviderCacheRepository(db *mongo.Database) *ProviderCacheRepository {
	return &ProviderCacheRepository{coll: db.Collection("provider_cache")}
}

func (r *ProviderCacheRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "provider", Value: 1}, {Key: "cacheKey", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
	})
	if err != nil {
		return fmt.Errorf("create provider_cache indexes: %w", err)
	}
	return nil
}

func (r *ProviderCacheRepository) FindActive(ctx context.Context, provider, cacheKey string) ([]byte, error) {
	var doc providerCacheDocument
	err := r.coll.FindOne(ctx, bson.M{
		"provider":  provider,
		"cacheKey":  cacheKey,
		"expiresAt": bson.M{"$gt": time.Now().UTC()},
	}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find provider cache: %w", err)
	}
	return append([]byte(nil), doc.Payload.Data...), nil
}

func (r *ProviderCacheRepository) Set(ctx context.Context, provider, cacheKey string, payload []byte, expiresAt time.Time) error {
	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"provider":  provider,
			"cacheKey":  cacheKey,
			"payload":   primitive.Binary{Subtype: 0x00, Data: append([]byte(nil), payload...)},
			"expiresAt": expiresAt.UTC(),
		},
		"$setOnInsert": bson.M{
			"createdAt": now,
		},
	}
	_, err := r.coll.UpdateOne(ctx, bson.M{
		"provider": provider,
		"cacheKey": cacheKey,
	}, update, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("set provider cache: %w", err)
	}
	return nil
}
