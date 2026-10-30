package tmdb

import (
	"context"
	"strings"

	"media_library_manager/internal/providers/httpx"
)

// SearchPage translates an arbitrary app offset/limit into TMDB's fixed 20-item
// pages, fetching only overlapping pages and respecting its 500-page API limit.
func (c *Client) SearchPage(ctx context.Context, kind, query string, offset, limit int) (httpx.Page[Result], error) {
	endpoint := "https://api.themoviedb.org/3/search/" + kind
	first := offset/20 + 1
	if first > 500 {
		first = 500
	}
	last := (offset+limit-1)/20 + 1
	if last > 500 {
		last = 500
	}
	result := httpx.Page[Result]{}
	for page := first; page <= last; page++ {
		batch, err := c.decodeSearchPage(ctx, endpoint, kind, strings.TrimSpace(query), page)
		if err != nil {
			return httpx.Page[Result]{}, err
		}
		result.Total = batch.Total
		for i, row := range batch.Items {
			position := (page-1)*20 + i
			if position >= offset && position < offset+limit && row.ID != 0 {
				result.Items = append(result.Items, row)
			}
		}
		if page*20 >= batch.Total {
			break
		}
	}
	return result, nil
}
