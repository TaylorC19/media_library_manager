package barcode

// JSON provider keys align with @media-library/types `providerNames`.
const (
	ProviderTmdb         = "tmdb"
	ProviderMusicBrainz  = "musicbrainz"
	ProviderDiscogs      = "discogs"
	ProviderOpenLibrary  = "openlibrary"
	ProviderRAWG         = "rawg"
)

// FailureCode values match BarcodeLookupFailureCode in types.
const (
	CodeUnavailable     = "unavailable"
	CodeTimeout         = "timeout"
	CodeUnsupported     = "unsupported"
	CodeInvalidResponse = "invalid_response"
)

// FallbackReason values match BarcodeLookupFallbackReason in types.
const (
	ReasonNoCandidates        = "no_candidates"
	ReasonWeakBarcodeCoverage = "weak_barcode_coverage"
	ReasonProviderUnavailable = "provider_unavailable"
	ReasonManualConfirmation  = "manual_confirmation_required"
)

// Request is the decoded POST body for /barcode/lookup.
type Request struct {
	Barcode            string
	PreferredMediaType string
}

// LinkedEntry is a user-owned entry pointing at a local media record.
type LinkedEntry struct {
	EntryID string `json:"entryId"`
	Bucket  string `json:"bucket"`
}

// Candidate is a normalized local or provider result (source discriminates: "local" | "provider").
type Candidate struct {
	Source                string         `json:"source"`
	MediaRecordID         string         `json:"mediaRecordId,omitempty"`
	MediaType             string         `json:"mediaType"`
	Title                 string         `json:"title"`
	Year                  *int32         `json:"year,omitempty"`
	ImageURL              *string        `json:"imageUrl"`
	CreatorLine           *string        `json:"creatorLine"`
	HasLinkedLibraryEntry bool           `json:"hasLinkedLibraryEntry,omitempty"`
	LinkedLibraryEntries  []LinkedEntry  `json:"linkedLibraryEntries,omitempty"`
	Provider              string         `json:"provider,omitempty"`
	ProviderID            string         `json:"providerId,omitempty"`
}

// Failure records a per-provider error without aborting the whole lookup.
type Failure struct {
	Provider string `json:"provider"`
	Code     string `json:"code"`
}

// Fallback is explicit guidance when results are empty or need confirmation.
type Fallback struct {
	Reason      string  `json:"reason"`
	ManualQuery *string `json:"manualQuery"`
	MediaType   *string `json:"mediaType"`
}

// Response is the body of POST /barcode/lookup.
type Response struct {
	Barcode    string      `json:"barcode"`
	MediaType  *string     `json:"mediaType"`
	Candidates []Candidate `json:"candidates"`
	Failures   []Failure   `json:"failures"`
	Fallback   *Fallback   `json:"fallback"`
}
