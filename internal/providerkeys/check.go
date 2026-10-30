package providerkeys

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"media_library_manager/internal/providers/httpx"
	"media_library_manager/internal/providers/musicbrainz"
	"media_library_manager/internal/providers/providererrors"
)

// Checker makes uncached, read-only provider checks. Public services are checked
// without credentials; cached metadata cannot mask a failed connection or bad key.
// It never writes settings.
type Checker struct {
	HTTP                *http.Client
	MusicBrainzUA       string
	MusicBrainzThrottle *musicbrainz.Throttle
}

func (c *Checker) Check(ctx context.Context, provider, key string) string {
	if key == "" && provider != "musicbrainz" && provider != "openlibrary" {
		return "missing"
	}
	options := httpx.GetJSONOptions{Timeout: 10 * time.Second}
	var err error
	valid := false
	switch provider {
	case "musicbrainz":
		options.Query = map[string]any{"query": "release:Nevermind", "fmt": "json", "limit": 1}
		ua := c.MusicBrainzUA
		if ua == "" {
			ua = "MediaLibraryManager/1.0"
		}
		options.Headers = map[string]string{"User-Agent": ua}
		var response struct {
			Releases *[]json.RawMessage `json:"releases"`
		}
		_, err = musicbrainz.Run(ctx, c.MusicBrainzThrottle, func() (bool, error) {
			requestErr := httpx.GetJSON(ctx, c.HTTP, provider, "connection_test", "https://musicbrainz.org/ws/2/release", options, &response)
			return false, requestErr
		})
		valid = response.Releases != nil
	case "openlibrary":
		options.Query = map[string]any{"q": "Dune", "limit": 1}
		var response struct {
			Docs *[]json.RawMessage `json:"docs"`
		}
		err = httpx.GetJSON(ctx, c.HTTP, provider, "connection_test", "https://openlibrary.org/search.json", options, &response)
		valid = response.Docs != nil
	case "tmdb":
		options.Query = map[string]any{"api_key": key}
		var response struct {
			ID int `json:"id"`
		}
		err = httpx.GetJSON(ctx, c.HTTP, provider, "credential_test", "https://api.themoviedb.org/3/movie/11", options, &response)
		valid = response.ID > 0
	case "rawg":
		options.Query = map[string]any{"key": key, "page_size": 1}
		var response struct {
			Results *[]json.RawMessage `json:"results"`
		}
		err = httpx.GetJSON(ctx, c.HTTP, provider, "credential_test", "https://api.rawg.io/api/games", options, &response)
		valid = response.Results != nil
	case "discogs":
		options.Headers = map[string]string{"Authorization": "Discogs token=" + key, "User-Agent": "MediaLibraryManager/1.0"}
		var response struct {
			ID int `json:"id"`
		}
		err = httpx.GetJSON(ctx, c.HTTP, provider, "credential_test", "https://api.discogs.com/oauth/identity", options, &response)
		valid = response.ID > 0
	default:
		return "invalid"
	}
	if err == nil {
		if valid {
			return "success"
		}
		return "invalid_response"
	}
	var providerErr *providererrors.Error
	if errors.As(err, &providerErr) {
		switch providerErr.Code {
		case providererrors.CodeUnauthorized:
			return "rejected"
		case providererrors.CodeForbidden:
			return "forbidden"
		case providererrors.CodeRateLimited:
			return "rate_limited"
		case providererrors.CodeInvalidResponse:
			return "invalid_response"
		}
	}
	return "unavailable"
}
