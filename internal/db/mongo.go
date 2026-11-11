package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectMongo(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetMaxPoolSize(10).SetMaxConnecting(2).SetMaxConnIdleTime(time.Minute).SetServerSelectionTimeout(5*time.Second).SetTimeout(10*time.Second))
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	return client, nil
}
