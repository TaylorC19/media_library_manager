package library

import (
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	domainlib "media_library_manager/internal/domain/library"
)

func TestDiscoverySortsHaveStableTiesAndUnknownYearsLast(t *testing.T) {
	year := func(n int32) *int32 { return &n }
	rows := []EntryWithMedia{}
	for i, record := range []*domainlib.MediaRecord{{Title: "Zulu", Year: year(2000)}, {Title: "alpha", Year: year(2020)}, {Title: "Beta"}, {Title: "Alpha", Year: year(2020)}, nil} {
		id := primitive.ObjectID{}
		id[11] = byte(i + 1)
		rows = append(rows, EntryWithMedia{Entry: &domainlib.LibraryEntry{ID: id, CreatedAt: time.Unix(int64(i), 0)}, Media: record})
	}
	tests := []struct {
		order string
		ids   []byte
	}{{"recent", []byte{5, 4, 3, 2, 1}}, {"title_asc", []byte{4, 2, 3, 1, 5}}, {"title_desc", []byte{1, 3, 4, 2, 5}}, {"year_desc", []byte{4, 2, 1, 3, 5}}, {"year_asc", []byte{1, 4, 2, 3, 5}}}
	for _, tc := range tests {
		copied := append([]EntryWithMedia(nil), rows...)
		sortLibraryRows(copied, tc.order)
		got := []byte{}
		for _, row := range copied {
			got = append(got, row.Entry.ID[11])
		}
		if !reflect.DeepEqual(got, tc.ids) {
			t.Errorf("%s: got %v want %v", tc.order, got, tc.ids)
		}
		// Starting order must not change pagination boundaries when sort keys tie.
		for i, j := 0, len(copied)-1; i < j; i, j = i+1, j-1 {
			copied[i], copied[j] = copied[j], copied[i]
		}
		sortLibraryRows(copied, tc.order)
		for i, row := range copied {
			if row.Entry.ID[11] != tc.ids[i] {
				t.Error("sort order changed on repeated visit")
			}
		}
	}
}

func TestLibrarySearchIncludesCreatorsAndTagsWithoutChangingOtherMatches(t *testing.T) {
	for _, key := range []string{"authors", "directors", "creators", "artists", "developers"} {
		for _, value := range []any{[]string{"Test Creator"}, bson.A{"Test Creator", int32(1)}, "Test Creator"} {
			rows := []EntryWithMedia{{Entry: &domainlib.LibraryEntry{Tags: []string{"Sci-Fi", "To Read"}}, Media: &domainlib.MediaRecord{Title: "Example", Details: bson.M{key: value}}}}
			for _, query := range []string{"CREATOR", "sci-fi", "to read", "example"} {
				if len(filterRows(rows, query)) != 1 {
					t.Errorf("missing %s match for %q", key, query)
				}
			}
			if len(filterTag(rows, " SCI-FI ")) != 1 || len(filterTag(rows, "sci")) != 0 {
				t.Error("tag filter must be a whole-tag match ignoring case")
			}
			if len(filterRows(filterTag(rows, "different"), "Creator")) != 0 {
				t.Error("tag and text criteria must combine")
			}
		}
	}
	options := libraryTagOptions([]string{"to read", "Sci-Fi", "sci-fi", " to read ", ""})
	if len(options) != 2 || options[0].Value != "sci-fi" || options[1].Value != "to read" {
		t.Errorf("tag options not normalized: %+v", options)
	}
}
