// dbsetup runs with maintenance credentials before starting a new deployment.
package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"media_library_manager/internal/config"
	"media_library_manager/internal/db"
	"media_library_manager/internal/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	client, err := db.ConnectMongo(ctx, cfg.MongoURI)
	if err != nil {
		log.Fatal("database connection failed")
	}
	defer client.Disconnect(ctx)
	database := client.Database(cfg.MongoDatabase)
	for _, ensure := range []func(context.Context) error{repository.NewUsersRepository(database).EnsureIndexes, repository.NewSessionsRepository(database).EnsureIndexes, repository.NewMediaRecordsRepository(database).EnsureIndexes, repository.NewLibraryEntriesRepository(database).EnsureIndexes, repository.NewScanLogsRepository(database).EnsureIndexes, repository.NewProviderCacheRepository(database).EnsureIndexes} {
		if err := ensure(ctx); err != nil {
			log.Fatal(err)
		}
	}
	for _, name := range []string{"request_budgets", "request_leases"} {
		_, err := database.Collection(name).Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)})
		if err != nil {
			log.Fatal(err)
		}
	}
	_, err = database.Collection("admin_guard").UpdateByID(ctx, "roles", bson.M{"$setOnInsert": bson.M{"revision": 0}}, options.Update().SetUpsert(true))
	if err != nil {
		log.Fatal(err)
	}
	log.Print("database setup complete")
}
