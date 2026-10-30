package rawg

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"media_library_manager/internal/providerkeys"
	"media_library_manager/internal/providers/httpx"
	"media_library_manager/internal/providers/providererrors"
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
	Count   int       `json:"count"`
	Results []gameDoc `json:"results"`
}

type gameDoc struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Released        string `json:"released"`
	BackgroundImage string `json:"background_image"`
}

func (c *Client) SearchGames(ctx context.Context, query string) ([]GameHit, error) {
	page, err := c.SearchGamesPage(ctx, query, 1, 50)
	return page.Items, err
}
func (c *Client) SearchGamesPage(ctx context.Context, query string, page, limit int) (httpx.Page[GameHit], error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return httpx.Page[GameHit]{}, nil
	}
	key := providerkeys.Resolve(ctx, "rawg", c.APIKey)
	if key == "" {
		return httpx.Page[GameHit]{}, nil
	}
	var payload searchResponse
	if err := httpx.GetJSON(ctx, c.HTTP, "rawg", "search", "https://api.rawg.io/api/games", httpx.GetJSONOptions{
		Query: map[string]any{
			"key":       key,
			"search":    query,
			"page_size": limit,
			"page":      page,
		},
		Timeout: 15 * time.Second,
	}, &payload); err != nil {
		return httpx.Page[GameHit]{}, err
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
	total := payload.Count
	if total == 0 {
		total = len(payload.Results)
	}
	return httpx.Page[GameHit]{Items: out, Total: total}, nil
}

func (c *Client) GetGameDetails(ctx context.Context, externalID string) (*GameDetails, error) {
	key := providerkeys.Resolve(ctx, "rawg", c.APIKey)
	if key == "" {
		return nil, nil
	}
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return nil, nil
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
	err := httpx.GetJSON(ctx, c.HTTP, "rawg", "detail", "https://api.rawg.io/api/games/"+url.PathEscape(externalID), httpx.GetJSONOptions{
		Query: map[string]any{
			"key": key,
		},
		Timeout: 15 * time.Second,
	}, &payload)
	if err != nil {
		var perr *providererrors.Error
		if errors.As(err, &perr) && perr.Code == providererrors.CodeNotFound {
			return nil, nil
		}
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
