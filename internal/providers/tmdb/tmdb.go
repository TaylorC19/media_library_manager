package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	imageBase = "https://image.tmdb.org/t/p/w185"
)

type Client struct {
	APIKey string
	HTTP   *http.Client
}

func (c *Client) SearchMovies(ctx context.Context, query string) ([]Result, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, nil
	}
	u := fmt.Sprintf(
		"https://api.themoviedb.org/3/search/movie?api_key=%s&query=%s&language=en-US",
		url.QueryEscape(c.APIKey),
		url.QueryEscape(strings.TrimSpace(query)),
	)
	return c.decodeSearch(ctx, u, "movie")
}

func (c *Client) SearchTV(ctx context.Context, query string) ([]Result, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, nil
	}
	u := fmt.Sprintf(
		"https://api.themoviedb.org/3/search/tv?api_key=%s&query=%s&language=en-US",
		url.QueryEscape(c.APIKey),
		url.QueryEscape(strings.TrimSpace(query)),
	)
	return c.decodeSearch(ctx, u, "tv")
}

type Result struct {
	Kind        string // movie | tv
	ID          int
	Title       string
	ReleaseDate string
	Overview    string
	PosterPath  string
}

type searchResponse struct {
	Results []json.RawMessage `json:"results"`
}

func (c *Client) decodeSearch(ctx context.Context, rawURL, kind string) ([]Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
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
		return nil, fmt.Errorf("tmdb: unexpected status %d", res.StatusCode)
	}

	var envelope searchResponse
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		return nil, err
	}

	var out []Result
	for _, raw := range envelope.Results {
		if len(out) >= 15 {
			break
		}
		parsed, ok := decodeResultItem(raw, kind)
		if !ok {
			continue
		}
		out = append(out, parsed)
	}
	return out, nil
}

func decodeResultItem(raw json.RawMessage, kind string) (Result, bool) {
	var common struct {
		ID           int    `json:"id"`
		Overview     string `json:"overview"`
		PosterPath   string `json:"poster_path"`
		Title        string `json:"title"`
		Name         string `json:"name"`
		ReleaseDate  string `json:"release_date"`
		FirstAirDate string `json:"first_air_date"`
	}
	if err := json.Unmarshal(raw, &common); err != nil {
		return Result{}, false
	}
	title := strings.TrimSpace(common.Title)
	if title == "" {
		title = strings.TrimSpace(common.Name)
	}
	if common.ID == 0 || title == "" {
		return Result{}, false
	}
	date := common.ReleaseDate
	if kind == "tv" {
		date = common.FirstAirDate
	}
	return Result{
		Kind:        kind,
		ID:          common.ID,
		Title:       title,
		ReleaseDate: date,
		Overview:    strings.TrimSpace(common.Overview),
		PosterPath:  strings.TrimSpace(common.PosterPath),
	}, true
}

func PosterURL(posterPath string) *string {
	posterPath = strings.TrimSpace(posterPath)
	if posterPath == "" {
		return nil
	}
	u := imageBase + posterPath
	return &u
}

func YearFromDate(s string) *int {
	s = strings.TrimSpace(s)
	if len(s) < 4 {
		return nil
	}
	y, err := strconv.Atoi(s[:4])
	if err != nil {
		return nil
	}
	return &y
}

func DefaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}
