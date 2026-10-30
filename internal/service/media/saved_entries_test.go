package media

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
	domainlib "media_library_manager/internal/domain/library"
)

type savedEntriesRepo struct {
	entryRepository
	rows []domainlib.LibraryEntry
}

func (r savedEntriesRepo) FindByUserAndMediaRecordIDs(_ context.Context, user primitive.ObjectID, ids []primitive.ObjectID) ([]domainlib.LibraryEntry, error) {
	out := []domainlib.LibraryEntry{}
	for _, row := range r.rows {
		for _, id := range ids {
			if row.UserID == user && row.MediaRecordID == id {
				out = append(out, row)
			}
		}
	}
	return out, nil
}
func TestPreviewSavedEntriesArePersonalAndIncludeFormattedCopies(t *testing.T) {
	owner, other, id := primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()
	format, notes := "blu_ray", "private notes must never be rendered"
	entryID, otherID := primitive.NewObjectID(), primitive.NewObjectID()
	service := &Service{media: &regressionMediaRepo{record: &domainlib.MediaRecord{ID: id, MediaType: "movie"}}, entries: savedEntriesRepo{rows: []domainlib.LibraryEntry{
		{ID: entryID, UserID: owner, MediaRecordID: id, Bucket: "catalog", Format: &format, Notes: &notes},
		{ID: primitive.NewObjectID(), UserID: owner, MediaRecordID: id, Bucket: "wishlist"},
		{ID: otherID, UserID: other, MediaRecordID: id, Bucket: "catalog"},
	}}}
	for _, record := range []*domainlib.MediaRecord{{ID: id, MediaType: "movie"}, {MediaType: "movie"}} {
		saved, err := service.SavedEntries(context.Background(), owner, record, ImportForm{Provider: "tmdb", ExternalID: "42", TMDBKind: "movie"})
		if err != nil || len(saved["catalog"]) != 1 || len(saved["wishlist"]) != 1 || saved["catalog"][0].ID != entryID.Hex() || *saved["catalog"][0].Format != format {
			t.Fatalf("incorrect personal saved copies: %+v %v", saved, err)
		}
		data, _ := json.Marshal(saved)
		if strings.Contains(string(data), notes) || strings.Contains(string(data), otherID.Hex()) {
			t.Error("saved-item navigation leaked private data")
		}
	}
	saved, err := service.SavedEntries(context.Background(), owner, &domainlib.MediaRecord{MediaType: "tv"}, ImportForm{Provider: "tmdb", ExternalID: "42", TMDBKind: "tv"})
	if err != nil || len(saved) != 0 {
		t.Error("same-number movie/TV references must remain distinct")
	}
}
