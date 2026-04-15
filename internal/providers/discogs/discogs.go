package discogs

import (
	"context"
	"strings"
)

// Client is an optional enrichment layer for music metadata (releases, barcodes).
// Milestone 4 only wires the type; enrichment calls remain a no-op until provider
// reliability work lands.
type Client struct {
	Token string
}

// EnrichAlbumSubtitle is a placeholder hook. When Token is set, future versions may
// query Discogs to append label/catalog hints to the subtitle line.
func (c *Client) EnrichAlbumSubtitle(_ context.Context, artist, title, currentSubtitle string) string {
	_ = artist
	_ = title
	if c == nil || strings.TrimSpace(c.Token) == "" {
		return currentSubtitle
	}
	return currentSubtitle
}
