package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	domainauth "media_library_manager/internal/domain/auth"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrUsernameTaken = errors.New("username already exists")

type UsersRepository struct {
	coll *mongo.Collection
}

func NewUsersRepository(db *mongo.Database) *UsersRepository {
	return &UsersRepository{coll: db.Collection("users")}
}

func (r *UsersRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create users indexes: %w", err)
	}
	return nil
}

func (r *UsersRepository) Create(ctx context.Context, user *domainauth.User) error {
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.coll.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrUsernameTaken
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UsersRepository) FindByUsername(ctx context.Context, username string) (*domainauth.User, error) {
	var user domainauth.User
	err := r.coll.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by username: %w", err)
	}
	return &user, nil
}

func (r *UsersRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*domainauth.User, error) {
	var user domainauth.User
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return &user, nil
}
