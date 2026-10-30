package handlers

import domainlib "media_library_manager/internal/domain/library"

// Preserve an existing legacy choice when editing/filtering an old copy, without
// offering unrelated formats for new copies.
func setFormFormats(data map[string]any, mediaType, legacy string) {
	formats := domainlib.FormatsForMediaType(mediaType)
	data["LegacyFormat"] = ""
	data["LegacyMediaType"] = ""
	if legacy != "" && domainlib.IsFormat(legacy) && domainlib.IsMediaType(mediaType) && !domainlib.IsFormatForMediaType(mediaType, legacy) {
		formats = append(formats, legacy)
		data["LegacyFormat"] = legacy
		data["LegacyMediaType"] = mediaType
	}
	data["Formats"] = formats
}
