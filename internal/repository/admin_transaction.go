package repository

import (
	"context"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// WithRoleChange serializes authorization and role writes across instances.
// Updating the shared guard forces concurrent transactions to conflict and
// retry with a fresh actor role, preventing cross-demotion write skew.
func (r *UsersRepository) WithRoleChange(ctx context.Context, fn func(context.Context) error) error {
	if !r.requireTransactions {
		localRoleChanges.Lock()
		defer localRoleChanges.Unlock()
		return fn(ctx)
	}
	session, err := r.coll.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		_, err := r.coll.Database().Collection("admin_guard").UpdateByID(sc, "roles", bson.M{"$inc": bson.M{"revision": 1}}, options.Update().SetUpsert(true))
		if err != nil {
			return nil, err
		}
		return nil, fn(sc)
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	return err
}

var localRoleChanges sync.Mutex
