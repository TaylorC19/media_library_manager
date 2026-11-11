package dbsetup

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"media_library_manager/internal/repository"
	"time"
)

var Collections = []string{"users", "sessions", "media_records", "library_entries", "scan_logs", "provider_cache", "request_budgets", "request_leases", "admin_guard", "personal_provider_keys"}

// Initialize is repeatable and never replaces documents or account credentials.
func Initialize(ctx context.Context, database *mongo.Database) error {
	for _, name := range Collections {
		if err := database.CreateCollection(ctx, name); err != nil {
			var ce mongo.CommandError
			if !errors.As(err, &ce) || ce.Code != 48 {
				return fmt.Errorf("create collection %s: %w", name, err)
			}
		}
	}
	for _, ensure := range []func(context.Context) error{repository.NewUsersRepository(database).EnsureIndexes, repository.NewSessionsRepository(database).EnsureIndexes, repository.NewMediaRecordsRepository(database).EnsureIndexes, repository.NewLibraryEntriesRepository(database).EnsureIndexes, repository.NewScanLogsRepository(database).EnsureIndexes, repository.NewProviderCacheRepository(database).EnsureIndexes} {
		if err := ensure(ctx); err != nil {
			return err
		}
	}
	for _, name := range []string{"request_budgets", "request_leases"} {
		_, err := database.Collection(name).Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)})
		if err != nil {
			return err
		}
	}
	_, err := database.Collection("admin_guard").UpdateByID(ctx, "roles", bson.M{"$setOnInsert": bson.M{"revision": 0}}, options.Update().SetUpsert(true))
	if err != nil {
		return err
	}
	return nil
}

// Replica initializes only a fresh bundled single-node set, then waits for primary.
func Replica(ctx context.Context, client *mongo.Client) error {
	admin := client.Database("admin")
	var status struct {
		Set string `bson:"set"`
	}
	err := admin.RunCommand(ctx, bson.D{{Key: "replSetGetStatus", Value: 1}}).Decode(&status)
	if err != nil {
		var ce mongo.CommandError
		if !errors.As(err, &ce) || ce.Code != 94 {
			return errors.New("could not inspect replica set")
		}
		config := bson.M{"_id": "mlm", "members": bson.A{bson.M{"_id": 0, "host": "mongo:27017"}}}
		if err := admin.RunCommand(ctx, bson.D{{Key: "replSetInitiate", Value: config}}).Err(); err != nil {
			return errors.New("could not initialize replica set")
		}
	} else if status.Set != "mlm" {
		return errors.New("bundled setup requires replica set mlm")
	}
	return WaitPrimary(ctx, client)
}
func WaitPrimary(ctx context.Context, client *mongo.Client) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		var hello struct {
			Primary bool   `bson:"isWritablePrimary"`
			Set     string `bson:"setName"`
			Msg     string `bson:"msg"`
		}
		if err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&hello); err == nil && hello.Primary {
			if hello.Set == "" && hello.Msg != "isdbgrid" {
				return errors.New("database setup requires a replica set or sharded cluster")
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.New("database did not become a writable replica set before timeout")
		case <-ticker.C:
		}
	}
}
