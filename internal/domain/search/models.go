package search

// Hit is a normalized, template-safe search result (not a raw provider payload).
type Hit struct {
	// Personal saved status is populated after provider search, never cached.
	InCollection bool
	InWishlist   bool

	// Provider is a stable internal key: tmdb, musicbrainz, open_library, rawg.
	Provider   string
	ExternalID string

	MediaType string // movie, tv, album, book, game

	Title    string
	Subtitle string
	Year     *int
	Summary  string

	ImageURL *string

	// TMDBKind is "movie" or "tv" when Provider is tmdb; otherwise empty.
	TMDBKind string
}
