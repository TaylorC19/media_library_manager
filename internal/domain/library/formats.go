package library

import "strings"

var formatsByMediaType = map[string][]string{
	"movie": {"blu_ray", "dvd", "vhs", "digital", "other"},
	"tv":    {"blu_ray", "dvd", "vhs", "digital", "other"},
	"album": {"cd", "vinyl", "cassette", "digital", "other"},
	"book":  {"hardcover", "paperback", "digital", "other"},
	"game":  {"switch", "ps5", "xbox", "blu_ray", "dvd", "cd", "digital", "other"},
}

func FormatsForMediaType(mediaType string) []string {
	if values, ok := formatsByMediaType[mediaType]; ok {
		return append([]string(nil), values...)
	}
	return append([]string(nil), Formats...)
}
func IsFormatForMediaType(mediaType, format string) bool {
	if format == "" {
		return true
	}
	for _, allowed := range formatsByMediaType[mediaType] {
		if allowed == format {
			return true
		}
	}
	return false
}
func FormatMediaTypes(format string) string {
	types := []string{}
	for _, kind := range MediaTypes {
		if IsFormatForMediaType(kind, format) {
			types = append(types, kind)
		}
	}
	return strings.Join(types, " ")
}

type CopyDetails struct{ Format, Notes string }
