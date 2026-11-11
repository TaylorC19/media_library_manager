package dbsetup

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"media_library_manager/internal/repository"
	authsvc "media_library_manager/internal/service/auth"
	"os"
	"testing"
	"time"
)

func fixture(t *testing.T) *mongo.Database {
	t.Helper()
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		uri = "mongodb://127.0.0.1:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		if os.Getenv("REQUIRE_MONGO_TESTS") == "1" {
			t.Fatal("test MongoDB unavailable")
		}
		t.Skip("test MongoDB unavailable")
	}
	db := client.Database("mlm_setup_test_" + primitive.NewObjectID().Hex())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})
	return db
}
func TestInitializeIsRepeatableAndCreatesAllCollections(t *testing.T) {
	db := fixture(t)
	ctx := context.Background()
	if err := Initialize(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Collection("library_entries").InsertOne(ctx, bson.M{"_id": "fixture", "notes": "keep"}); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(ctx, db); err != nil {
		t.Fatal(err)
	}
	names, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range Collections {
		found := false
		for _, name := range names {
			if name == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing collection %s", want)
		}
	}
	count, err := db.Collection("library_entries").CountDocuments(ctx, bson.M{"_id": "fixture", "notes": "keep"})
	if err != nil || count != 1 {
		t.Fatal("repeat setup changed saved data")
	}
}
func TestBootstrapAdminPreservesExistingCredentials(t *testing.T) {
	db := fixture(t)
	ctx := context.Background()
	var hello struct {
		SetName string `bson:"setName"`
	}
	_ = db.RunCommand(ctx, bson.M{"hello": 1}).Decode(&hello)
	if hello.SetName == "" {
		if os.Getenv("REQUIRE_MONGO_TESTS") == "1" {
			t.Fatal("replica set required")
		}
		t.Skip("replica set required")
	}
	if err := Initialize(ctx, db); err != nil {
		t.Fatal(err)
	}
	created, err := BootstrapAdmin(ctx, db, "fixture-admin", "fixture-password")
	if err != nil || !created {
		t.Fatal("initial admin bootstrap failed")
	}
	before, err := repository.NewUsersRepository(db).FindByUsername(ctx, "fixture-admin")
	if err != nil {
		t.Fatal(err)
	}
	created, err = BootstrapAdmin(ctx, db, "fixture-admin", "different-password")
	if err != nil || created {
		t.Fatal("repeat bootstrap changed account")
	}
	after, _ := repository.NewUsersRepository(db).FindByUsername(ctx, "fixture-admin")
	if after.PasswordHash != before.PasswordHash {
		t.Fatal("existing password replaced")
	}
	_, _ = db.Collection("users").DeleteMany(ctx, bson.M{})
	service := authsvc.NewService(repository.NewUsersRepository(db), repository.NewSessionsRepository(db), time.Hour)
	if _, err := service.CreateAccount(ctx, "fixture-admin", "fixture-password", "", "general"); err != nil {
		t.Fatal(err)
	}
	if _, err := BootstrapAdmin(ctx, db, "fixture-admin", "different-password"); err == nil {
		t.Fatal("existing general account promoted")
	}
	user, _ := repository.NewUsersRepository(db).FindByUsername(ctx, "fixture-admin")
	if user.Role != "general" {
		t.Fatal("existing account role changed")
	}
}

func TestRuntimeProvisioningPreservesExistingPassword(t *testing.T) {
	db := fixture(t)
	ctx := context.Background()
	if err := Initialize(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := RuntimeUser(ctx, db, "fixture_runtime", "fixture-runtime-password"); err != nil {
		t.Fatal(err)
	}
	if err := RuntimeUser(ctx, db, "fixture_runtime", "different-runtime-password"); err != nil {
		t.Fatal(err)
	}
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		uri = "mongodb://127.0.0.1:27017"
	}
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(timeout, options.Client().ApplyURI(uri).SetAuth(options.Credential{AuthSource: db.Name(), Username: "fixture_runtime", Password: "fixture-runtime-password"}))
	if err != nil {
		t.Fatal("could not connect runtime fixture")
	}
	defer client.Disconnect(context.Background())
	if err := client.Ping(timeout, nil); err != nil {
		t.Fatal("repeat provisioning replaced the runtime password")
	}
}
