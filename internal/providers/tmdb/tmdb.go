package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"media_library_manager/internal/providerkeys"
	"media_library_manager/internal/providers/httpx"
	"media_library_manager/internal/providers/providererrors"
)

const (
	imageBase = "https://image.tmdb.org/t/p/w185"
)

type Client struct {
	APIKey string
	HTTP   *http.Client
}

func (c *Client) SearchMovies(ctx context.Context, query string) ([]Result, error) {
	key := providerkeys.Resolve(ctx, "tmdb", c.APIKey)
	if key == "" {
		return nil, nil
	}
	return c.decodeSearch(ctx, "https://api.themoviedb.org/3/search/movie", "movie", strings.TrimSpace(query))
}

func (c *Client) SearchTV(ctx context.Context, query string) ([]Result, error) {
	key := providerkeys.Resolve(ctx, "tmdb", c.APIKey)
	if key == "" {
		return nil, nil
	}
	return c.decodeSearch(ctx, "https://api.themoviedb.org/3/search/tv", "tv", strings.TrimSpace(query))
}

type Result struct {
	Kind        string // movie | tv
	ID          int
	Title       string
	ReleaseDate string
	Overview    string
	PosterPath  string
}

type Detail struct {
	Kind           string
	ID             int
	Title          string
	ReleaseDate    string
	Overview       string
	PosterPath     string
	Genres         []string
	Directors      []string
	Cast           []string
	RuntimeMinutes *int
	Creators       []string
	Seasons        *int
	Episodes       *int
	VoteAverage    *float64
}

type searchResponse struct {
	Total      int               `json:"total_results"`
	TotalPages int               `json:"total_pages"`
	Results    []json.RawMessage `json:"results"`
}

func (c *Client) decodeSearch(ctx context.Context, endpoint, kind, query string) ([]Result, error) {
	page, err := c.decodeSearchPage(ctx, endpoint, kind, query, 1)
	valid := make([]Result, 0, len(page.Items))
	for _, row := range page.Items {
		if row.ID != 0 {
			valid = append(valid, row)
		}
	}
	return valid, err
}
func (c *Client) decodeSearchPage(ctx context.Context, endpoint, kind, query string, page int) (httpx.Page[Result], error) {
	key := providerkeys.Resolve(ctx, "tmdb", c.APIKey)
	var envelope searchResponse
	if err := httpx.GetJSON(ctx, c.HTTP, "tmdb", "search", endpoint, httpx.GetJSONOptions{
		Query: map[string]any{
			"api_key":  key,
			"query":    query,
			"page":     page,
			"language": "en-US",
		},
		Timeout: 20 * time.Second,
	}, &envelope); err != nil {
		return httpx.Page[Result]{}, err
	}

	var out []Result
	for _, raw := range envelope.Results {
		parsed, ok := decodeResultItem(raw, kind)
		if !ok {
			parsed = Result{}
		}
		out = append(out, parsed)
	}
	total := envelope.Total
	if total == 0 {
		total = len(envelope.Results)
	}
	if envelope.TotalPages > 0 && envelope.TotalPages*20 < total {
		total = envelope.TotalPages * 20
	}
	if total > 10000 {
		total = 10000
	}
	return httpx.Page[Result]{Items: out, Total: total}, nil
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

func (c *Client) GetDetails(ctx context.Context, externalID, kind string) (*Detail, error) {
	key := providerkeys.Resolve(ctx, "tmdb", c.APIKey)
	if key == "" {
		return nil, nil
	}
	if kind != "movie" && kind != "tv" {
		return nil, nil
	}
	id, err := strconv.Atoi(strings.TrimSpace(externalID))
	if err != nil || id <= 0 {
		return nil, nil
	}

	var payload struct {
		ID               int     `json:"id"`
		Title            string  `json:"title"`
		Name             string  `json:"name"`
		Overview         string  `json:"overview"`
		PosterPath       string  `json:"poster_path"`
		ReleaseDate      string  `json:"release_date"`
		FirstAirDate     string  `json:"first_air_date"`
		Runtime          *int    `json:"runtime"`
		NumberOfSeasons  *int    `json:"number_of_seasons"`
		NumberOfEpisodes *int    `json:"number_of_episodes"`
		VoteAverage      float64 `json:"vote_average"`
		Genres           []struct {
			Name string `json:"name"`
		} `json:"genres"`
		CreatedBy []struct {
			Name string `json:"name"`
		} `json:"created_by"`
		Credits struct {
			Cast []struct {
				Name string `json:"name"`
			} `json:"cast"`
			Crew []struct {
				Job        string `json:"job"`
				Department string `json:"department"`
				Name       string `json:"name"`
			} `json:"crew"`
		} `json:"credits"`
	}
	err = httpx.GetJSON(ctx, c.HTTP, "tmdb", "detail", "https://api.themoviedb.org/3/"+kind+"/"+strconv.Itoa(id), httpx.GetJSONOptions{
		Query: map[string]any{
			"api_key":            key,
			"language":           "en-US",
			"append_to_response": "credits",
		},
		Timeout: 20 * time.Second,
	}, &payload)
	if err != nil {
		var perr *providererrors.Error
		if errors.As(err, &perr) && perr.Code == providererrors.CodeNotFound {
			return nil, nil
		}
		return nil, err
	}

	title := strings.TrimSpace(payload.Title)
	if title == "" {
		title = strings.TrimSpace(payload.Name)
	}
	if payload.ID == 0 || title == "" {
		return nil, nil
	}

	date := strings.TrimSpace(payload.ReleaseDate)
	if kind == "tv" {
		date = strings.TrimSpace(payload.FirstAirDate)
	}

	detail := &Detail{
		Kind:           kind,
		ID:             payload.ID,
		Title:          title,
		ReleaseDate:    date,
		Overview:       strings.TrimSpace(payload.Overview),
		PosterPath:     strings.TrimSpace(payload.PosterPath),
		Genres:         uniqueNamesFromGenreList(payload.Genres),
		RuntimeMinutes: payload.Runtime,
		Seasons:        payload.NumberOfSeasons,
		Episodes:       payload.NumberOfEpisodes,
		VoteAverage:    floatPtr(payload.VoteAverage),
	}
	if kind == "movie" {
		detail.Directors = uniqueTMDBCrewNames(payload.Credits.Crew, "Director")
		detail.Cast = uniqueTMDBCastNames(payload.Credits.Cast, 5)
	}
	if kind == "tv" {
		detail.Creators = uniqueTMDBCreatedBy(payload.CreatedBy)
	}
	return detail, nil
}

func PosterURL(posterPath string) *string {
	posterPath = strings.TrimSpace(posterPath)
	if posterPath == "" {
		return nil
	}
	u := imageBase + posterPath
	return &u
}

func (d *Detail) ExternalID() string {
	if d == nil || d.ID <= 0 {
		return ""
	}
	return strconv.Itoa(d.ID)
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

func uniqueNamesFromGenreList(genres []struct {
	Name string `json:"name"`
}) []string {
	var names []string
	for _, genre := range genres {
		names = append(names, genre.Name)
	}
	return uniqueTrimmedNames(names, 0)
}

func uniqueTMDBCrewNames(crew []struct {
	Job        string `json:"job"`
	Department string `json:"department"`
	Name       string `json:"name"`
}, job string) []string {
	var names []string
	for _, person := range crew {
		if strings.TrimSpace(person.Job) != job {
			continue
		}
		names = append(names, person.Name)
	}
	return uniqueTrimmedNames(names, 0)
}

func uniqueTMDBCastNames(cast []struct {
	Name string `json:"name"`
}, limit int) []string {
	var names []string
	for _, person := range cast {
		names = append(names, person.Name)
	}
	return uniqueTrimmedNames(names, limit)
}

func uniqueTMDBCreatedBy(creators []struct {
	Name string `json:"name"`
}) []string {
	var names []string
	for _, creator := range creators {
		names = append(names, creator.Name)
	}
	return uniqueTrimmedNames(names, 0)
}

func uniqueTrimmedNames(values []string, limit int) []string {
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
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func floatPtr(v float64) *float64 {
	if v <= 0 {
		return nil
	}
	return &v
}
