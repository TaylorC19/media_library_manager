package repository

import (
	"context"
	"fmt"
	"time"

	domainauth "media_library_manager/internal/domain/auth"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SessionsRepository struct {
	coll *mongo.Collection
}

func NewSessionsRepository(db *mongo.Database) *SessionsRepository {
	return &SessionsRepository{coll: db.Collection("sessions")}
}

func (r *SessionsRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "tokenHash", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "expiresAt", Value: 1}},
		},
	})
	if err != nil {
		return fmt.Errorf("create sessions indexes: %w", err)
	}
	return nil
}

func (r *SessionsRepository) Create(ctx context.Context, session *domainauth.Session) error {
	now := time.Now().UTC()
	session.CreatedAt = now
	session.LastUsed = now

	_, err := r.coll.InsertOne(ctx, session)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *SessionsRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domainauth.Session, error) {
	filter := bson.M{
		"tokenHash": tokenHash,
		"expiresAt": bson.M{"$gt": time.Now().UTC()},
	}

	var session domainauth.Session
	err := r.coll.FindOne(ctx, filter).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("find session by token hash: %w", err)
	}
	return &session, nil
}

func (r *SessionsRepository) Touch(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.coll.UpdateByID(ctx, id, bson.M{
		"$set": bson.M{"lastUsedAt": time.Now().UTC()},
	})
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return nil
}

func (r *SessionsRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"tokenHash": tokenHash})
	if err != nil {
		return fmt.Errorf("delete session by token hash: %w", err)
	}
	return nil
}

func (r *SessionsRepository) DeleteByID(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete session by id: %w", err)
	}
	return nil
}
