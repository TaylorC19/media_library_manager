package library

import (
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	domainlib "media_library_manager/internal/domain/library"
)

type TagOption struct{ Value, Label string }

func libraryTagOptions(tags []string) []TagOption {
	sort.Slice(tags, func(i, j int) bool {
		a, b := strings.ToLower(strings.TrimSpace(tags[i])), strings.ToLower(strings.TrimSpace(tags[j]))
		if a != b {
			return a < b
		}
		return tags[i] < tags[j]
	})
	result := []TagOption{}
	seen := map[string]bool{}
	for _, tag := range tags {
		label := strings.TrimSpace(tag)
		value := strings.ToLower(label)
		if value != "" && !seen[value] {
			result = append(result, TagOption{value, label})
			seen[value] = true
		}
	}
	return result
}

func filterTag(rows []EntryWithMedia, tag string) []EntryWithMedia {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return rows
	}
	out := []EntryWithMedia{}
	for _, row := range rows {
		for _, value := range row.Entry.Tags {
			if strings.EqualFold(strings.TrimSpace(value), tag) {
				out = append(out, row)
				break
			}
		}
	}
	return out
}

func creatorValues(details bson.M) []string {
	values := []string{}
	for _, key := range []string{"authors", "directors", "creators", "artists", "developers"} {
		switch value := details[key].(type) {
		case string:
			values = append(values, value)
		case []string:
			values = append(values, value...)
		case bson.A:
			for _, item := range value {
				if text, ok := item.(string); ok {
					values = append(values, text)
				}
			}
		}
	}
	return values
}

func sortLibraryRows(rows []EntryWithMedia, order string) {
	title := func(row EntryWithMedia) string {
		if row.Media == nil {
			return ""
		}
		return strings.ToLower(strings.TrimSpace(row.Media.Title))
	}
	year := func(row EntryWithMedia) int32 {
		if row.Media == nil || row.Media.Year == nil {
			return 0
		}
		return *row.Media.Year
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if order == "year_asc" || order == "year_desc" {
			ay, by := year(a), year(b)
			if ay != by {
				if ay == 0 {
					return false
				}
				if by == 0 {
					return true
				}
				if order == "year_asc" {
					return ay < by
				}
				return ay > by
			}
		}
		if order != "" && order != domainlib.DefaultSort {
			at, bt := title(a), title(b)
			if at != bt {
				if at == "" {
					return false
				}
				if bt == "" {
					return true
				}
				if order == "title_desc" {
					return at > bt
				}
				return at < bt
			}
		}
		if !a.Entry.CreatedAt.Equal(b.Entry.CreatedAt) {
			return a.Entry.CreatedAt.After(b.Entry.CreatedAt)
		}
		return a.Entry.ID.Hex() > b.Entry.ID.Hex()
	})
}
