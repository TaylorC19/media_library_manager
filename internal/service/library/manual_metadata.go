package library

import (
	"net/url"
	"strings"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/bson"
	domainlib "media_library_manager/internal/domain/library"
)

func validateManualMetadata(image, summary, creators string) []string {
	errs := []string{}
	image = strings.TrimSpace(image)
	if image != "" {
		parsed, err := url.Parse(image)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil || len(image) > 2048 {
			errs = append(errs, "library.errors.invalidCoverURL")
		}
	}
	if utf8.RuneCountInString(summary) > 10000 {
		errs = append(errs, "library.errors.descriptionTooLong")
	}
	if utf8.RuneCountInString(creators) > 5000 {
		errs = append(errs, "library.errors.creatorsTooLong")
	}
	return errs
}

func manualCreatorKey(mediaType string) string {
	switch mediaType {
	case "book":
		return "authors"
	case "movie":
		return "directors"
	case "album":
		return "artists"
	case "game":
		return "developers"
	default:
		return "creators"
	}
}
func parseCreators(raw string) []string {
	values := []string{}
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		name := strings.TrimSpace(line)
		key := strings.ToLower(name)
		if name != "" && !seen[key] {
			values = append(values, name)
			seen[key] = true
		}
	}
	return values
}
func ManualCreatorText(record *domainlib.MediaRecord) string {
	if record == nil {
		return ""
	}
	return strings.Join(creatorValues(record.Details), "\n")
}
func manualDetails(existing bson.M, mediaType, creators string) bson.M {
	details := bson.M{}
	for key, value := range existing {
		details[key] = value
	}
	for _, key := range []string{"authors", "directors", "creators", "artists", "developers"} {
		delete(details, key)
	}
	if values := parseCreators(creators); len(values) > 0 {
		details[manualCreatorKey(mediaType)] = values
	}
	return details
}
