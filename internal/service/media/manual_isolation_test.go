package media

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"media_library_manager/internal/config"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/service/reliability"
)

func TestLocalPreviewAccessPolicy(t *testing.T) {
	owner, other, id := primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()
	for _, tc := range []struct {
		name, source string
		user         primitive.ObjectID
		allowed      bool
	}{
		{"owner manual", domainlib.SourceManual, owner, true},
		{"foreign manual", domainlib.SourceManual, other, false},
		{"missing actor", domainlib.SourceManual, primitive.NilObjectID, false},
		{"shared provider", domainlib.SourceProvider, other, true},
		{"unknown source", "unexpected", owner, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &Service{media: &regressionMediaRepo{record: &domainlib.MediaRecord{ID: id, Source: tc.source}}, entries: &regressionEntries{owner: owner}}
			record, err := svc.PreviewLocal(context.Background(), tc.user, id)
			if tc.allowed {
				if err != nil || record == nil || record.ID != id {
					t.Fatal("authorized metadata was blocked")
				}
			} else if record != nil || !errors.Is(err, ErrNotFound) {
				t.Fatal("inaccessible metadata was not hidden as not found")
			}
		})
	}
}

type manualCandidateRepo struct {
	regressionMediaRepo
	inserted *domainlib.MediaRecord
}

func (r *manualCandidateRepo) FindLooseTitleYear(context.Context, string, string, *int32) ([]domainlib.MediaRecord, error) {
	return []domainlib.MediaRecord{*r.record}, nil
}
func (r *manualCandidateRepo) FindByAnyBarcodeCandidates(context.Context, string, []string) ([]domainlib.MediaRecord, error) {
	return []domainlib.MediaRecord{*r.record}, nil
}
func (r *manualCandidateRepo) Insert(_ context.Context, record *domainlib.MediaRecord) error {
	record.ID = primitive.NewObjectID()
	r.inserted = record
	return nil
}

func TestImportRejectsManualCandidatesEvenIfRepositoryReturnsThem(t *testing.T) {
	// Repository queries are also scoped, but the service must not rely on a
	// caller or future repository implementation preserving those filters.
	user := primitive.NewObjectID()
	year := int32(2020)
	manual := &domainlib.MediaRecord{ID: primitive.NewObjectID(), Source: domainlib.SourceManual, MediaType: "movie", Title: "Fixture movie", Year: &year, Details: bson.M{"directors": []string{"Fixture Director"}}}
	before := *manual
	records := &manualCandidateRepo{regressionMediaRepo: regressionMediaRepo{record: manual}}
	cfg := config.Config{TMDBAPIKey: "fixture"}
	svc := NewService(cfg, nil, nil, nil, reliability.NewService(cfg), nil)
	svc.media = records
	svc.entries = &regressionEntries{owner: user}
	calls := 0
	svc.tmdb.HTTP = &http.Client{Transport: previewTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"id":42,"title":"Fixture movie","release_date":"2020-01-01","credits":{"crew":[{"job":"Director","name":"Fixture Director"}]}}`))}, nil
	})}
	out, _, err := svc.Import(context.Background(), user, ImportForm{Provider: "tmdb", ExternalID: "42", TMDBKind: "movie", MediaType: "movie", Bucket: "catalog"})
	if err != nil {
		t.Fatal(err)
	}
	if out.MediaRecord.ID == manual.ID || out.MediaRecord.Source != domainlib.SourceProvider || records.inserted == nil || records.updated != nil || calls != 1 || !reflect.DeepEqual(before, *manual) {
		t.Fatal("manual metadata was reused or changed during provider import")
	}
}

func TestBarcodeDedupeConsidersOnlyProviderRecords(t *testing.T) {
	manual := domainlib.MediaRecord{ID: primitive.NewObjectID(), Source: domainlib.SourceManual}
	provider := domainlib.MediaRecord{ID: primitive.NewObjectID(), Source: domainlib.SourceProvider}
	if singleUniqueRecord([]domainlib.MediaRecord{manual}) != nil {
		t.Fatal("manual barcode candidate must not be reused")
	}
	selected := singleUniqueRecord([]domainlib.MediaRecord{manual, provider, provider})
	if selected == nil || selected.ID != provider.ID {
		t.Fatal("private candidates must not interfere with valid provider deduplication")
	}
	other := domainlib.MediaRecord{ID: primitive.NewObjectID(), Source: domainlib.SourceProvider}
	if singleUniqueRecord([]domainlib.MediaRecord{manual, provider, other}) != nil {
		t.Fatal("ambiguous provider matches must remain separate")
	}
}
