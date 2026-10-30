package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func testMongoDB(t *testing.T) *mongo.Database {
	t.Helper()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Skipf("mongo not available: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		t.Skipf("mongo not reachable: %v", err)
	}

	dbName := "mlm_backup_test_" + primitive.NewObjectID().Hex()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_ = client.Database(dbName).Drop(cleanupCtx)
		_ = client.Disconnect(cleanupCtx)
	})
	return client.Database(dbName)
}

func seedCollections(t *testing.T, db *mongo.Database) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	userID := primitive.NewObjectID()
	mediaID := primitive.NewObjectID()

	_, err := db.Collection("users").InsertOne(ctx, bson.M{
		"_id":          userID,
		"username":     "backupuser",
		"passwordHash": "hash",
		"createdAt":    now,
		"updatedAt":    now,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Collection("sessions").InsertOne(ctx, bson.M{
		"_id":        primitive.NewObjectID(),
		"userId":     userID,
		"tokenHash":  "tok",
		"expiresAt":  now.Add(time.Hour),
		"createdAt":  now,
		"lastUsedAt": now,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Collection("media_records").InsertOne(ctx, bson.M{
		"_id":       mediaID,
		"source":    "manual",
		"mediaType": "book",
		"title":     "Test Book",
		"createdAt": now,
		"updatedAt": now,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Collection("library_entries").InsertOne(ctx, bson.M{
		"_id":           primitive.NewObjectID(),
		"userId":        userID,
		"mediaRecordId": mediaID,
		"bucket":        "catalog",
		"mediaType":     "book",
		"createdAt":     now,
		"updatedAt":     now,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Collection("provider_cache").InsertOne(ctx, bson.M{
		"_id":       primitive.NewObjectID(),
		"provider":  "tmdb",
		"cacheKey":  "k1",
		"payload":   bson.M{"ok": true},
		"expiresAt": now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Collection("scan_logs").InsertOne(ctx, bson.M{
		"_id":       primitive.NewObjectID(),
		"userId":    userID,
		"barcode":   "123",
		"createdAt": now,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	src := testMongoDB(t)
	seedCollections(t, src)

	ctx := context.Background()
	data, err := ExportDatabase(ctx, src)
	if err != nil {
		t.Fatal(err)
	}

	dst := testMongoDB(t)
	_, err = ImportDatabase(ctx, dst, bytes.NewReader(data), ImportOptions{RequireDatabaseMatch: false})
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range CollectionNames {
		srcCount, err := src.Collection(name).CountDocuments(ctx, bson.M{})
		if err != nil {
			t.Fatal(err)
		}
		dstCount, err := dst.Collection(name).CountDocuments(ctx, bson.M{})
		if err != nil {
			t.Fatal(err)
		}
		if srcCount != dstCount {
			t.Fatalf("collection %s: src=%d dst=%d", name, srcCount, dstCount)
		}
	}

	var user bson.M
	if err := dst.Collection("users").FindOne(ctx, bson.M{"username": "backupuser"}).Decode(&user); err != nil {
		t.Fatal(err)
	}
}

func TestImportRejectsWrongVersion(t *testing.T) {
	db := testMongoDB(t)
	ctx := context.Background()

	emptyCollections := make(map[string][]string, len(CollectionNames))
	for _, name := range CollectionNames {
		emptyCollections[name] = []string{}
	}

	payload := File{
		Manifest: Manifest{
			Format:     FormatName,
			Version:    99,
			ExportedAt: time.Now().UTC(),
			Database:   db.Name(),
		},
		Collections: emptyCollections,
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if err := json.NewEncoder(gz).Encode(payload); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	_, err := ImportDatabase(ctx, db, bytes.NewReader(buf.Bytes()), ImportOptions{})
	if err == nil {
		t.Fatal("expected version error")
	}
}

func TestImportRequiresDatabaseMatch(t *testing.T) {
	src := testMongoDB(t)
	seedCollections(t, src)
	ctx := context.Background()

	data, err := ExportDatabase(ctx, src)
	if err != nil {
		t.Fatal(err)
	}

	dst := testMongoDB(t)
	_, err = ImportDatabase(ctx, dst, bytes.NewReader(data), ImportOptions{RequireDatabaseMatch: true})
	if err == nil {
		t.Fatal("expected database name mismatch error")
	}
}
