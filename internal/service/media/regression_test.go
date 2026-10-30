package media

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"media_library_manager/internal/config"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/providers/tmdb"
	"media_library_manager/internal/service/reliability"
)

type regressionMediaRepo struct {
	mediaRepository
	record, updated *domainlib.MediaRecord
}

func (r *regressionMediaRepo) FindByID(context.Context, primitive.ObjectID) (*domainlib.MediaRecord, error) {
	return r.record, nil
}
func (r *regressionMediaRepo) FindByProviderImportRef(context.Context, string, string, string) (*domainlib.MediaRecord, error) {
	return r.record, nil
}
func (r *regressionMediaRepo) UpdateProviderData(_ context.Context, record *domainlib.MediaRecord) error {
	r.updated = record
	return nil
}

type regressionEntries struct {
	entryRepository
	owner              primitive.ObjectID
	existing, inserted *domainlib.LibraryEntry
	barcode            string
}

func (r *regressionEntries) FindByUserAndMediaRecordIDs(_ context.Context, userID primitive.ObjectID, _ []primitive.ObjectID) ([]domainlib.LibraryEntry, error) {
	if userID != r.owner {
		return nil, nil
	}
	return []domainlib.LibraryEntry{{UserID: r.owner}}, nil
}
func (r *regressionEntries) FindByUserMediaBucketAndFormat(context.Context, primitive.ObjectID, primitive.ObjectID, string, *string) (*domainlib.LibraryEntry, error) {
	return r.existing, nil
}
func (r *regressionEntries) Insert(_ context.Context, entry *domainlib.LibraryEntry) error {
	r.inserted = entry
	entry.ID = primitive.NewObjectID()
	return nil
}
func (r *regressionEntries) SetBarcodeIfMissing(_ context.Context, owner, entryID primitive.ObjectID, barcode string) error {
	r.barcode = barcode
	return nil
}

type regressionCache struct {
	payload []byte
	writes  int
}

func (r *regressionCache) FindActive(context.Context, string, string) ([]byte, error) {
	return r.payload, nil
}
func (r *regressionCache) Set(_ context.Context, _, _ string, payload []byte, _ time.Time) error {
	r.payload = payload
	r.writes++
	return nil
}

func TestProviderImportRetainsScannedBarcodeForNewAndExistingCopies(t *testing.T) {
	for _, existing := range []bool{false, true} {
		owner := primitive.NewObjectID()
		media := &regressionMediaRepo{record: &domainlib.MediaRecord{ID: primitive.NewObjectID(), MediaType: "book"}}
		entries := &regressionEntries{owner: owner}
		notes := "Keep my notes"
		if existing {
			entries.existing = &domainlib.LibraryEntry{ID: primitive.NewObjectID(), UserID: owner, Notes: &notes}
		}
		service := &Service{media: media, entries: entries}
		out, _, err := service.Import(context.Background(), owner, ImportForm{Provider: "open_library", ExternalID: "OL1W", MediaType: "book", Bucket: "catalog", Barcode: "9781234567897"})
		if err != nil {
			t.Fatal(err)
		}
		if out.LibraryEntry.Barcode == nil || *out.LibraryEntry.Barcode != "9781234567897" {
			t.Error("barcode lost during import")
		}
		if existing && (entries.barcode != "9781234567897" || *entries.existing.Notes != notes) {
			t.Error("existing copy must retain notes and gain missing barcode")
		}
	}
}

func TestRefreshFetchesFreshMetadataUpdatesCacheAndRequiresOwnership(t *testing.T) {
	cfg := config.Config{TMDBAPIKey: "test", ProviderCacheDetailTTL: time.Hour}
	owner := primitive.NewObjectID()
	media := &regressionMediaRepo{record: &domainlib.MediaRecord{ID: primitive.NewObjectID(), Source: domainlib.SourceProvider, MediaType: "movie", Title: "Old title", ProviderRefs: bson.M{"tmdb": bson.M{"id": "42", "kind": "movie"}}}}
	entries := &regressionEntries{owner: owner}
	stale, _ := json.Marshal(&tmdb.Detail{ID: 42, Kind: "movie", Title: "Cached old title"})
	cacheRepo := &regressionCache{payload: stale}
	rel := reliability.NewService(cfg)
	service := NewService(cfg, nil, nil, reliability.NewCache(cacheRepo, rel), rel, nil)
	service.media = media
	service.entries = entries
	calls := 0
	failed := false
	service.tmdb.HTTP = &http.Client{Transport: previewTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		status := 200
		body := `{"id":42,"title":"Fresh title","overview":"Fresh description"}`
		if failed {
			status = 503
			body = `{}`
		}
		return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	_, err := service.Refresh(context.Background(), primitive.NewObjectID(), media.record.ID)
	if err != ErrNotFound || calls != 0 {
		t.Error("another user's record must not be refreshed")
	}
	outcome, err := service.Refresh(context.Background(), owner, media.record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.WasRefreshed || calls != 1 || media.updated == nil || media.updated.Title != "Fresh title" || media.updated.ID != media.record.ID {
		t.Error("refresh did not persist freshly fetched metadata")
	}
	if cacheRepo.writes != 1 {
		t.Error("refresh must replace stale cached details")
	}
	preview, err := service.Preview(context.Background(), ImportForm{Provider: "tmdb", ExternalID: "42", MediaType: "movie", TMDBKind: "movie"})
	if err != nil || preview.Title != "Fresh title" || calls != 1 {
		t.Error("subsequent previews should use refreshed cache")
	}
	media.updated = nil
	failed = true
	_, err = service.Refresh(context.Background(), owner, media.record.ID)
	if err == nil || media.updated != nil || cacheRepo.writes != 1 {
		t.Error("failed refresh must not overwrite metadata or cache")
	}
}
