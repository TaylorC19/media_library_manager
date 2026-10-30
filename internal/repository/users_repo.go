package repository

import (
	"context"
	"errors"
	"fmt"
	"regexp"
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

	result, err := r.coll.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrUsernameTaken
		}
		return fmt.Errorf("create user: %w", err)
	}
	if id, ok := result.InsertedID.(primitive.ObjectID); ok {
		user.ID = id
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

// SetRole updates an already-authorized account. The admin service checks the
// actor and self-demotion rules; the trusted server command is the recovery path.
func (r *UsersRepository) SetRole(ctx context.Context, username, role string) error {
	if !domainauth.IsRole(role) {
		return fmt.Errorf("invalid role")
	}
	result, err := r.coll.UpdateOne(ctx, bson.M{"username": username}, bson.M{"$set": bson.M{"role": role, "updatedAt": time.Now().UTC()}})
	if err != nil {
		return fmt.Errorf("set user role: %w", err)
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// ListManaged excludes password hashes and all other fields not needed by the
// admin table, even though authorization is checked before this query.
func (r *UsersRepository) ListManaged(ctx context.Context, query, role string, page, pageSize int) ([]domainauth.User, int64, error) {
	if role != "" && !domainauth.IsRole(role) {
		return nil, 0, fmt.Errorf("invalid role filter")
	}
	filter := bson.M{}
	conditions := bson.A{}
	if query != "" {
		pattern := bson.M{"$regex": regexp.QuoteMeta(query), "$options": "i"}
		conditions = append(conditions, bson.M{"$or": bson.A{bson.M{"username": pattern}, bson.M{"displayName": pattern}}})
	}
	if role == domainauth.RoleGeneral {
		// Null/missing and empty legacy roles have general access too.
		conditions = append(conditions, bson.M{"$or": bson.A{bson.M{"role": domainauth.RoleGeneral}, bson.M{"role": ""}, bson.M{"role": nil}}})
	} else if role != "" {
		conditions = append(conditions, bson.M{"role": role})
	}
	if len(conditions) > 0 {
		filter["$and"] = conditions
	}
	count, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if page < 1 {
		page = 1
	}
	last := (count-1)/int64(pageSize) + 1
	if last < 1 {
		last = 1
	}
	if int64(page) > last {
		page = int(last)
	}
	cur, err := r.coll.Find(ctx, filter, options.Find().SetProjection(bson.M{"username": 1, "displayName": 1, "role": 1, "createdAt": 1}).SetSort(bson.D{{Key: "username", Value: 1}, {Key: "_id", Value: 1}}).SetSkip(int64(page-1)*int64(pageSize)).SetLimit(int64(pageSize)))
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	var users []domainauth.User
	if err := cur.All(ctx, &users); err != nil {
		return nil, 0, err
	}
	return users, count, nil
}
