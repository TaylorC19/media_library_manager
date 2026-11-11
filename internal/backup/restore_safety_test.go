package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"reflect"
	"testing"
)

func archiveFixture(t *testing.T, file *File) []byte {
	t.Helper()
	var out bytes.Buffer
	gz := gzip.NewWriter(&out)
	if err := json.NewEncoder(gz).Encode(file); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
func collectionSnapshot(t *testing.T, db *mongo.Database) map[string][]string {
	t.Helper()
	result := map[string][]string{}
	for _, name := range CollectionNames {
		docs, err := exportCollection(context.Background(), db.Collection(name))
		if err != nil {
			t.Fatal(err)
		}
		result[name] = docs
	}
	return result
}
func TestRestoreValidatesAllCollectionsBeforeWrites(t *testing.T) {
	for _, kind := range []string{"invalid-document", "missing-collection"} {
		t.Run(kind, func(t *testing.T) {
			db := testMongoDB(t)
			seedCollections(t, db)
			before := collectionSnapshot(t, db)
			raw, err := ExportDatabase(context.Background(), db)
			if err != nil {
				t.Fatal(err)
			}
			file, err := decodeBackupFile(bytes.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}
			file.Collections["users"] = nil
			if kind == "missing-collection" {
				delete(file.Collections, "scan_logs")
			} else {
				file.Collections["scan_logs"] = []string{"invalid-json"}
			}
			if _, err := ImportDatabase(context.Background(), db, bytes.NewReader(archiveFixture(t, file)), ImportOptions{}); err == nil {
				t.Fatal("invalid restore accepted")
			}
			if !reflect.DeepEqual(before, collectionSnapshot(t, db)) {
				t.Fatal("validation failure changed existing data")
			}
		})
	}
}
func TestAtomicRestoreRollsBackAllCollections(t *testing.T) {
	db := testMongoDB(t)
	ctx := context.Background()
	var hello struct {
		SetName string `bson:"setName"`
	}
	_ = db.RunCommand(ctx, bson.M{"hello": 1}).Decode(&hello)
	if hello.SetName == "" {
		if os.Getenv("REQUIRE_MONGO_TESTS") == "1" {
			t.Fatal("replica set required")
		}
		t.Skip("requires test replica set")
	}
	seedCollections(t, db)
	before := collectionSnapshot(t, db)
	if _, err := db.Collection("scan_logs").Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "fixtureUnique", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)}); err != nil {
		t.Fatal(err)
	}
	raw, err := ExportDatabase(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	file, err := decodeBackupFile(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	file.Collections["users"] = nil
	file.Collections["scan_logs"] = []string{`{"_id":"fixture-a","fixtureUnique":"same"}`, `{"_id":"fixture-b","fixtureUnique":"same"}`}
	if _, err := ImportDatabase(ctx, db, bytes.NewReader(archiveFixture(t, file)), ImportOptions{Atomic: true}); err == nil {
		t.Fatal("invalid transaction accepted")
	}
	if !reflect.DeepEqual(before, collectionSnapshot(t, db)) {
		t.Fatal("transaction failure changed existing data")
	}
}
func TestRestoreRejectsTrailingDataAndExcludesPersonalKeys(t *testing.T) {
	var out bytes.Buffer
	gz := gzip.NewWriter(&out)
	_, _ = gz.Write([]byte(`{} {}`))
	_ = gz.Close()
	if _, err := decodeBackupFile(bytes.NewReader(out.Bytes())); err == nil {
		t.Fatal("trailing JSON accepted")
	}
	for _, name := range CollectionNames {
		if name == "personal_provider_keys" {
			t.Fatal("custom archive includes personal keys")
		}
	}
}
