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

type GameDetails struct {
	ID              int
	Name            string
	Released        string
	BackgroundImage string
	Description     string
	Genres          []string
	Developers      []string
	Publishers      []string
	Platforms       []string
	Rating          *float64
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
		"https://api.rawg.io/api/games?key=%s&search=%s&page_size=50",
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

func (c *Client) GetGameDetails(ctx context.Context, externalID string) (*GameDetails, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, nil
	}
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return nil, nil
	}

	u := fmt.Sprintf(
		"https://api.rawg.io/api/games/%s?key=%s",
		url.PathEscape(externalID),
		url.QueryEscape(c.APIKey),
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
	if res.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("rawg: unexpected status %d", res.StatusCode)
	}

	var payload struct {
		ID              int     `json:"id"`
		Name            string  `json:"name"`
		Released        string  `json:"released"`
		BackgroundImage string  `json:"background_image"`
		DescriptionRaw  string  `json:"description_raw"`
		Rating          float64 `json:"rating"`
		Genres          []struct {
			Name string `json:"name"`
		} `json:"genres"`
		Developers []struct {
			Name string `json:"name"`
		} `json:"developers"`
		Publishers []struct {
			Name string `json:"name"`
		} `json:"publishers"`
		Platforms []struct {
			Platform struct {
				Name string `json:"name"`
			} `json:"platform"`
		} `json:"platforms"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.ID == 0 || strings.TrimSpace(payload.Name) == "" {
		return nil, nil
	}

	return &GameDetails{
		ID:              payload.ID,
		Name:            strings.TrimSpace(payload.Name),
		Released:        strings.TrimSpace(payload.Released),
		BackgroundImage: strings.TrimSpace(payload.BackgroundImage),
		Description:     strings.TrimSpace(payload.DescriptionRaw),
		Genres:          rawgNames(payload.Genres),
		Developers:      rawgDeveloperNames(payload.Developers),
		Publishers:      rawgPublisherNames(payload.Publishers),
		Platforms:       rawgPlatformNames(payload.Platforms),
		Rating:          rawgFloatPtr(payload.Rating),
	}, nil
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

func rawgNames(values []struct {
	Name string `json:"name"`
}) []string {
	var names []string
	for _, value := range values {
		names = append(names, value.Name)
	}
	return uniqueNames(names)
}

func rawgDeveloperNames(values []struct {
	Name string `json:"name"`
}) []string {
	return rawgNames(values)
}

func rawgPublisherNames(values []struct {
	Name string `json:"name"`
}) []string {
	return rawgNames(values)
}

func rawgPlatformNames(values []struct {
	Platform struct {
		Name string `json:"name"`
	} `json:"platform"`
}) []string {
	var names []string
	for _, value := range values {
		names = append(names, value.Platform.Name)
	}
	return uniqueNames(names)
}

func uniqueNames(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func rawgFloatPtr(v float64) *float64 {
	if v <= 0 {
		return nil
	}
	return &v
}
