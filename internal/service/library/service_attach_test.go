package library

import (
	"context"
	"os"
	"testing"
	"time"

	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func testMongo(t *testing.T) (*mongo.Client, *mongo.Database) {
	t.Helper()
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		uri = "mongodb://127.0.0.1:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Skipf("mongo connect: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		t.Skipf("mongo ping: %v (set MONGO_TEST_URI or run local mongod)", err)
	}
	name := "mlm_lib_test_" + primitive.NewObjectID().Hex()
	return client, client.Database(name)
}

func TestAttachFromMediaRecord_Table(t *testing.T) {
	client, db := testMongo(t)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})

	ctx := context.Background()
	mediaRepo := repository.NewMediaRecordsRepository(db)
	libRepo := repository.NewLibraryEntriesRepository(db)
	if err := mediaRepo.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	if err := libRepo.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	svc := NewService(mediaRepo, libRepo)

	userID := primitive.NewObjectID()
	otherID := primitive.NewObjectID()
	missingID := primitive.NewObjectID()

	mediaID := insertTestMedia(t, ctx, mediaRepo, "Test Album", "album")
	t.Run("media_not_found", func(t *testing.T) {
		_, _, errs, err := svc.AttachFromMediaRecord(ctx, userID, missingID, domainlib.BucketCatalog, "9780000000000")
		if err != ErrValidation {
			t.Fatalf("err = %v, want ErrValidation", err)
		}
		if len(errs) != 1 || errs[0] != "library.errors.mediaNotFound" {
			t.Fatalf("errs = %v", errs)
		}
	})
	t.Run("invalid_bucket", func(t *testing.T) {
		_, _, errs, err := svc.AttachFromMediaRecord(ctx, userID, mediaID, "invalid", "")
		if err != ErrValidation {
			t.Fatalf("err = %v, want ErrValidation", err)
		}
		if len(errs) < 1 || errs[0] != "library.errors.invalidBucket" {
			t.Fatalf("errs = %v", errs)
		}
	})
	t.Run("happy_path", func(t *testing.T) {
		id, wasEx, errs, err := svc.AttachFromMediaRecord(ctx, userID, mediaID, domainlib.BucketCatalog, "1234567890123")
		if err != nil {
			t.Fatalf("AttachFromMediaRecord: %v %v", err, errs)
		}
		if id == primitive.NilObjectID {
			t.Fatal("empty entry id")
		}
		if wasEx {
			t.Fatal("first attach should be new")
		}
		again, wasEx2, errs, err := svc.AttachFromMediaRecord(ctx, userID, mediaID, domainlib.BucketCatalog, "")
		if err != nil {
			t.Fatalf("second call: %v %v", err, errs)
		}
		if again != id {
			t.Fatalf("duplicate: got %s, want %s", again.Hex(), id.Hex())
		}
		if !wasEx2 {
			t.Fatal("second attach should be existing")
		}
	})
	t.Run("isolated_user", func(t *testing.T) {
		uid, wasEx, errs, err := svc.AttachFromMediaRecord(ctx, otherID, mediaID, domainlib.BucketCatalog, "")
		if err != nil {
			t.Fatalf("other user: %v %v", err, errs)
		}
		if uid == primitive.NilObjectID {
			t.Fatal("other user should get new entry")
		}
		if wasEx {
			t.Fatal("other user should be new")
		}
	})
}

func insertTestMedia(t *testing.T, ctx context.Context, media *repository.MediaRecordsRepository, title, mediaType string) primitive.ObjectID {
	t.Helper()
	st := "x"
	rec := &domainlib.MediaRecord{
		Source:    domainlib.SourceProvider,
		MediaType: mediaType,
		Title:     title,
		SortTitle: &st,
	}
	if err := media.Insert(ctx, rec); err != nil {
		t.Fatal(err)
	}
	return rec.ID
}
