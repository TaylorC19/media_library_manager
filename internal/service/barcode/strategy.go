package barcode

import (
	"strings"
	"unicode"

	domainbarcode "media_library_manager/internal/domain/barcode"
	domainlib "media_library_manager/internal/domain/library"
)

// Hint drives local filter and provider stage selection.
type Hint struct {
	InferredMediaType   *string // from ISBN or preferred
	IsIsbnLike          bool
	PreferredMediaType  *string
}

// Stage is one ordered provider pass for a media type.
type Stage struct {
	MediaType string
	Providers []string
	RankTier  int
}

// NormalizeBarcode trims, uppercases, removes spaces and hyphens, keeps digits and X.
func NormalizeBarcode(value string) string {
	s := strings.TrimSpace(value)
	s = strings.ToUpper(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsDigit(r):
			b.WriteRune(r)
		case r == 'X':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// IsIsbnLike returns true for 10- or 13-digit normalized barcodes.
func IsIsbnLike(normalized string) bool {
	n := len(normalized)
	return n == 10 || n == 13
}

// HasWeakCoverage is true for media types with unreliable barcode lookup.
func HasWeakCoverage(mediaType string) bool {
	switch mediaType {
	case "movie", "tv", "game":
		return true
	default:
		return false
	}
}

// InferHint produces lookup hints from normalized barcode and optional preferred type.
func InferHint(normalized, preferred string) Hint {
	pref := strings.TrimSpace(preferred)
	var prefPtr *string
	if pref != "" && domainlib.IsMediaType(pref) {
		prefPtr = &pref
	}

	isbn := IsIsbnLike(normalized)
	var inferred *string
	if prefPtr != nil {
		inferred = prefPtr
	} else if isbn {
		t := "book"
		inferred = &t
	}

	return Hint{
		InferredMediaType:  inferred,
		IsIsbnLike:         isbn,
		PreferredMediaType: prefPtr,
	}
}

// BuildStages returns ordered provider stages for barcode lookup.
func BuildStages(h Hint) []Stage {
	if h.PreferredMediaType != nil {
		switch *h.PreferredMediaType {
		case "book":
			return []Stage{{MediaType: "book", Providers: []string{domainbarcode.ProviderOpenLibrary}, RankTier: 2}}
		case "album":
			return []Stage{
				{MediaType: "album", Providers: []string{domainbarcode.ProviderDiscogs}, RankTier: 2},
				{MediaType: "album", Providers: []string{domainbarcode.ProviderMusicBrainz}, RankTier: 2},
			}
		case "movie", "tv", "game":
			return nil
		}
	}

	if h.IsIsbnLike {
		return []Stage{
			{MediaType: "book", Providers: []string{domainbarcode.ProviderOpenLibrary}, RankTier: 2},
			{MediaType: "album", Providers: []string{domainbarcode.ProviderDiscogs}, RankTier: 3},
			{MediaType: "album", Providers: []string{domainbarcode.ProviderMusicBrainz}, RankTier: 3},
		}
	}

	return []Stage{
		{MediaType: "album", Providers: []string{domainbarcode.ProviderDiscogs}, RankTier: 2},
		{MediaType: "album", Providers: []string{domainbarcode.ProviderMusicBrainz}, RankTier: 2},
	}
}
