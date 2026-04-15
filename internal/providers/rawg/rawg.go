package rawg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	APIKey string
	HTTP   *http.Client
}

type GameHit struct {
	ID              int
	Name            string
	Released        string
	BackgroundImage string
}

type searchResponse struct {
	Results []gameDoc `json:"results"`
}

type gameDoc struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Released        string `json:"released"`
	BackgroundImage string `json:"background_image"`
}

func (c *Client) SearchGames(ctx context.Context, query string) ([]GameHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, nil
	}
	u := fmt.Sprintf(
		"https://api.rawg.io/api/games?key=%s&search=%s&page_size=15",
		url.QueryEscape(c.APIKey),
		url.QueryEscape(query),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	res, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("rawg: unexpected status %d", res.StatusCode)
	}

	var payload searchResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, err
	}

	var out []GameHit
	for _, g := range payload.Results {
		name := strings.TrimSpace(g.Name)
		if g.ID == 0 || name == "" {
			continue
		}
		img := strings.TrimSpace(g.BackgroundImage)
		out = append(out, GameHit{
			ID:              g.ID,
			Name:            name,
			Released:        strings.TrimSpace(g.Released),
			BackgroundImage: img,
		})
	}
	return out, nil
}

func BackgroundURL(raw string) *string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return &raw
}

func DefaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}
