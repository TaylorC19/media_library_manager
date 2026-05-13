package discogs

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"media_library_manager/internal/providers/httpx"
)

// Client calls the Discogs HTTP API when Token is set.
type Client struct {
	Token string
	HTTP  *http.Client
}

// SearchHit is a normalized row from database search (release by barcode).
type SearchHit struct {
	ID         int
	Title      string
	Year       *int
	Barcode    []string
	CoverImage *string
}

type searchResponse struct {
	Results []searchResultItem `json:"results"`
}

type searchResultItem struct {
	ID         int      `json:"id"`
	Title      string   `json:"title"`
	Year       string   `json:"year"`
	Barcode    []string `json:"barcode"`
	CoverImage string   `json:"cover_image"`
}

// SearchByBarcode queries Discogs release search by barcode.
func (c *Client) SearchByBarcode(ctx context.Context, barcode string, limit int) ([]SearchHit, error) {
	barcode = strings.TrimSpace(barcode)
	if barcode == "" || c == nil || strings.TrimSpace(c.Token) == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}
	var payload searchResponse
	if err := httpx.GetJSON(ctx, c.HTTP, "discogs", "barcode", "https://api.discogs.com/database/search", httpx.GetJSONOptions{
		Query: map[string]any{
			"type":     "release",
			"barcode":  barcode,
			"per_page": limit,
		},
		Headers: map[string]string{
			"Authorization": "Discogs token=" + strings.TrimSpace(c.Token),
			"User-Agent":    "MediaLibraryManager/1.0",
		},
		Timeout: 20 * time.Second,
	}, &payload); err != nil {
		return nil, err
	}

	var out []SearchHit
	for _, r := range payload.Results {
		title := strings.TrimSpace(r.Title)
		if r.ID == 0 || title == "" {
			continue
		}
		var y *int
		if ys := strings.TrimSpace(r.Year); ys != "" {
			if yi, err := strconv.Atoi(ys); err == nil {
				y = &yi
			}
		}
		var cover *string
		if strings.TrimSpace(r.CoverImage) != "" {
			c := r.CoverImage
			cover = &c
		}
		out = append(out, SearchHit{
			ID:         r.ID,
			Title:      title,
			Year:       y,
			Barcode:    append([]string{}, r.Barcode...),
			CoverImage: cover,
		})
	}
	return out, nil
}

// EnrichAlbumSubtitle is a placeholder hook for search subtitle enrichment.
func (c *Client) EnrichAlbumSubtitle(_ context.Context, artist, title, currentSubtitle string) string {
	_ = artist
	_ = title
	if c == nil || strings.TrimSpace(c.Token) == "" {
		return currentSubtitle
	}
	return currentSubtitle
}

// DefaultHTTPClient returns a shared timeout for Discogs requests.
func DefaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}
