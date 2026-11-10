package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"media_library_manager/internal/config"
	domainauth "media_library_manager/internal/domain/auth"
	bdom "media_library_manager/internal/domain/barcode"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/repository"
	barcodesvc "media_library_manager/internal/service/barcode"
	libsvc "media_library_manager/internal/service/library"
	mediasvc "media_library_manager/internal/service/media"
	"media_library_manager/internal/service/reliability"
	"media_library_manager/internal/views"
)

func TestManualPreviewAndAttachmentRequireOwnedEntry(t *testing.T) {
	db := regressionDatabase(t)
	ctx := context.Background()
	records := repository.NewMediaRecordsRepository(db)
	entries := repository.NewLibraryEntriesRepository(db)
	library := libsvc.NewService(records, entries)
	owner, other := primitive.NewObjectID(), primitive.NewObjectID()
	id, _, err := library.CreateManual(ctx, owner, libsvc.ManualCreateForm{Title: "Private fixture title", MediaType: "movie", Bucket: "catalog", Summary: "Private fixture description", Creators: "Private fixture creator", Notes: "Private fixture notes"})
	if err != nil {
		t.Fatal(err)
	}
	detail, err := library.GetEntryDetail(ctx, owner, id)
	if err != nil {
		t.Fatal(err)
	}
	render, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	mh := NewMediaHandler(render, mediasvc.NewService(config.Config{}, records, entries, nil, nil, nil))
	lh := NewLibraryHandler(render, library)
	preview := func(user primitive.ObjectID, role string, record primitive.ObjectID, fragment string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/en/search/details?media_record_id="+record.Hex()+"&fragment="+fragment, nil)
		req = req.WithContext(middleware.WithUser(req.Context(), &domainauth.User{ID: user, Role: role}))
		w := httptest.NewRecorder()
		mh.Preview(w, req)
		return w
	}
	for _, fragment := range []string{"0", "1"} {
		for _, role := range []string{"general", "admin", "demo"} {
			w := preview(other, role, detail.Media.ID, fragment)
			if w.Code != 404 || strings.Contains(w.Body.String(), "Private fixture") {
				t.Fatalf("foreign manual preview (%s, fragment %s): status %d", role, fragment, w.Code)
			}
		}
		w := preview(owner, "general", detail.Media.ID, fragment)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "Private fixture description") || strings.Contains(w.Body.String(), "Private fixture notes") {
			t.Fatal("owner preview must retain metadata without exposing copy notes")
		}
		if w := preview(other, "general", primitive.NewObjectID(), fragment); w.Code != 404 {
			t.Fatal("missing preview must return the same not-found status")
		}
	}
	for _, dialog := range []bool{false, true} {
		for _, role := range []string{"general", "admin", "demo"} {
			fields := url.Values{"locale": {"en"}, "media_record_id": {detail.Media.ID.Hex()}, "bucket": {"wishlist"}, "format": {"dvd"}}
			req := httptest.NewRequest("POST", "/library/attach", strings.NewReader(fields.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if dialog {
				req.Header.Set("X-Search-Dialog", "1")
			}
			req = req.WithContext(middleware.WithUser(req.Context(), &domainauth.User{ID: other, Role: role}))
			w := httptest.NewRecorder()
			lh.AttachSubmit(w, req)
			want := 404
			if role == "demo" {
				want = 403
			}
			if w.Code != want || strings.Contains(w.Body.String(), "Private fixture") {
				t.Fatalf("foreign manual attach (%s, dialog %t): status %d", role, dialog, w.Code)
			}
		}
	}
	if _, _, _, err := library.AttachFromMediaRecord(ctx, other, primitive.NewObjectID(), "catalog", ""); !errors.Is(err, libsvc.ErrNotFound) {
		t.Fatal("missing and inaccessible attachments must both be not found")
	}
	foreign, err := entries.FindByUserAndMediaRecordIDs(ctx, other, []primitive.ObjectID{detail.Media.ID})
	if err != nil || len(foreign) != 0 {
		t.Fatal("rejected attachment created a copy")
	}
	if _, _, _, err := library.AttachFromMediaRecord(ctx, owner, detail.Media.ID, "wishlist", "", domainlib.CopyDetails{Format: "dvd", Notes: "Additional owned copy"}); err != nil {
		t.Fatalf("owner cannot save another copy: %v", err)
	}
	// Orphaned manual records are private, even when their ID is known.
	orphan := &domainlib.MediaRecord{Source: domainlib.SourceManual, MediaType: "movie", Title: "Orphan fixture"}
	if err := records.Insert(ctx, orphan); err != nil {
		t.Fatal(err)
	}
	if w := preview(owner, "admin", orphan.ID, "1"); w.Code != 404 {
		t.Fatal("orphan manual metadata must not be public")
	}
	// Provider metadata remains shareable without disclosing the owner's copy data.
	shared := &domainlib.MediaRecord{Source: domainlib.SourceProvider, MediaType: "movie", Title: "Shared provider fixture"}
	if err := records.Insert(ctx, shared); err != nil {
		t.Fatal(err)
	}
	if w := preview(other, "general", shared.ID, "1"); w.Code != 200 {
		t.Fatal("shared provider preview was blocked")
	}
	if _, _, _, err := library.AttachFromMediaRecord(ctx, other, shared.ID, "catalog", ""); err != nil {
		t.Fatal("shared provider attachment was blocked")
	}
}

func TestBarcodeResultsExcludeOtherUsersManualMetadata(t *testing.T) {
	db := regressionDatabase(t)
	ctx := context.Background()
	records := repository.NewMediaRecordsRepository(db)
	entries := repository.NewLibraryEntriesRepository(db)
	owner, other := primitive.NewObjectID(), primitive.NewObjectID()
	code := "1234567890123"
	library := libsvc.NewService(records, entries)
	private := map[primitive.ObjectID]primitive.ObjectID{}
	for _, user := range []primitive.ObjectID{owner, other} {
		id, _, err := library.CreateManual(ctx, user, libsvc.ManualCreateForm{Title: "Private barcode fixture", MediaType: "movie", Bucket: "catalog"})
		if err != nil {
			t.Fatal(err)
		}
		detail, err := library.GetEntryDetail(ctx, user, id)
		if err != nil {
			t.Fatal(err)
		}
		// Legacy manual records can have metadata barcode candidates even when the
		// saved copy has no barcode. Owners must still be able to find those records.
		_, err = db.Collection("media_records").UpdateByID(ctx, detail.Media.ID, bson.M{"$set": bson.M{"barcodeCandidates": []string{code}}})
		if err != nil {
			t.Fatal(err)
		}
		private[user] = detail.Media.ID
	}
	shared := &domainlib.MediaRecord{Source: domainlib.SourceProvider, Title: "Shared barcode fixture", MediaType: "movie", BarcodeCandidates: []string{code}}
	if err := records.Insert(ctx, shared); err != nil {
		t.Fatal(err)
	}
	svc := barcodesvc.NewService(config.Config{}, records, entries, nil, nil, nil, nil)
	for _, user := range []primitive.ObjectID{owner, other} {
		out, err := svc.Lookup(ctx, user, bdom.Request{Barcode: code, PreferredMediaType: "movie", ReadOnly: true})
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Candidates) != 2 {
			t.Fatalf("want own manual and shared provider results, got %d", len(out.Candidates))
		}
		for _, candidate := range out.Candidates {
			if candidate.MediaRecordID != private[user].Hex() && candidate.MediaRecordID != shared.ID.Hex() {
				t.Fatal("another user's manual metadata escaped through barcode lookup")
			}
		}
	}
}

type isolationTransport func(*http.Request) (*http.Response, error)

func (f isolationTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestProviderImportPreservesManualMetadata(t *testing.T) {
	for _, scenario := range []string{"title_year_creator", "barcode", "legacy_provider_ref", "existing_provider_match"} {
		t.Run(scenario, func(t *testing.T) {
			db := regressionDatabase(t)
			ctx := context.Background()
			records := repository.NewMediaRecordsRepository(db)
			entries := repository.NewLibraryEntriesRepository(db)
			library := libsvc.NewService(records, entries)
			owner, other := primitive.NewObjectID(), primitive.NewObjectID()
			year := 2020
			kind, creator := "movie", "Fixture Director"
			provider, external, tmdbKind := "tmdb", "42", "movie"
			if scenario == "barcode" {
				kind = "album"
				creator = "Manual-only artist"
				provider = "musicbrainz"
				external = "fixture-release"
				tmdbKind = ""
			}
			id, _, err := library.CreateManual(ctx, owner, libsvc.ManualCreateForm{Title: "Matching fixture", MediaType: kind, Bucket: "catalog", Year: &year, Creators: creator, Summary: "Keep manual description", ImageURL: "https://example.invalid/manual.jpg", Notes: "Keep copy notes", Tags: "fixture"})
			if err != nil {
				t.Fatal(err)
			}
			before, err := library.GetEntryDetail(ctx, owner, id)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "barcode" || scenario == "legacy_provider_ref" {
				set := bson.M{"barcodeCandidates": []string{"1234567890123"}}
				if scenario == "legacy_provider_ref" {
					set["providerRefs"] = bson.M{"tmdb": bson.M{"id": "42", "kind": "movie"}}
				}
				if _, err := db.Collection("media_records").UpdateByID(ctx, before.Media.ID, bson.M{"$set": set}); err != nil {
					t.Fatal(err)
				}
				before, err = library.GetEntryDetail(ctx, owner, id)
				if err != nil {
					t.Fatal(err)
				}
			}
			var sharedID primitive.ObjectID
			if scenario == "existing_provider_match" {
				shared := &domainlib.MediaRecord{Source: domainlib.SourceProvider, MediaType: "movie", Title: "Matching fixture", Year: before.Media.Year, Details: bson.M{"directors": []string{"Fixture Director"}}, ProviderRefs: bson.M{"tmdb": bson.M{"id": "84", "kind": "movie"}}}
				if err := records.Insert(ctx, shared); err != nil {
					t.Fatal(err)
				}
				sharedID = shared.ID
			}
			oldTransport := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = oldTransport })
			calls := 0
			http.DefaultTransport = isolationTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				body := `{"id":42,"title":"Matching fixture","release_date":"2020-01-01","overview":"Provider description","credits":{"crew":[{"job":"Director","name":"Fixture Director"}]}}`
				if scenario == "barcode" {
					body = `{"id":"fixture-release","title":"Different provider title","date":"2020","barcode":"1234567890123","artist-credit":[{"artist":{"name":"Provider Artist"}}]}`
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			cfg := config.Config{TMDBAPIKey: "fixture-key"}
			svc := mediasvc.NewService(cfg, records, entries, nil, reliability.NewService(cfg), nil)
			// Neither a different user nor the manual item's owner may implicitly
			// convert manual metadata by importing a matching provider result.
			var importedID primitive.ObjectID
			for _, user := range []primitive.ObjectID{other, owner} {
				out, _, err := svc.Import(ctx, user, mediasvc.ImportForm{Provider: provider, ExternalID: external, MediaType: kind, TMDBKind: tmdbKind, Bucket: "catalog"})
				if err != nil {
					t.Fatal(err)
				}
				if out.MediaRecord.ID == before.Media.ID || out.MediaRecord.Source != domainlib.SourceProvider {
					t.Fatal("provider import reused a private manual record")
				}
				if importedID.IsZero() {
					importedID = out.MediaRecord.ID
				} else if out.MediaRecord.ID != importedID {
					t.Fatal("provider metadata should remain shared across users")
				}
				if !sharedID.IsZero() && out.MediaRecord.ID != sharedID {
					t.Fatal("valid provider deduplication stopped working")
				}
				after, err := library.GetEntryDetail(ctx, owner, id)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(before, after) {
					t.Fatal("provider import changed manual metadata or original copy details")
				}
			}
			refs, err := records.FindByProviderImportRefs(ctx, []repository.ProviderImportRef{{Provider: provider, ExternalID: external, MediaType: kind, TMDBKind: tmdbKind}})
			if err != nil || len(refs) != 1 || refs[0].ID != importedID {
				t.Fatal("search identity lookup must include only the shared provider record")
			}
			if calls != 1 {
				t.Fatalf("expected one mocked provider lookup, got %d", calls)
			}
			// Defense in depth: a stale/incorrect caller must not turn a manual record
			// into provider metadata through the repository update method.
			update := *before.Media
			update.Source = domainlib.SourceProvider
			update.Title = "Rejected replacement"
			if err := records.UpdateProviderData(ctx, &update); !errors.Is(err, mongo.ErrNoDocuments) {
				t.Fatal("repository allowed a provider update of a manual record")
			}
			after, err := library.GetEntryDetail(ctx, owner, id)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("rejected repository update changed manual metadata")
			}
		})
	}
}
