// dbsetup is the developer entrypoint; installers use the packaged setup tool.
package main

import (
	"context"
	"log"
	"media_library_manager/internal/config"
	"media_library_manager/internal/db"
	"media_library_manager/internal/dbsetup"
	"time"
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
	defer client.Disconnect(context.Background())
	if err := dbsetup.Initialize(ctx, client.Database(cfg.MongoDatabase)); err != nil {
		log.Fatal(err)
	}
	log.Print("database setup complete")
}
