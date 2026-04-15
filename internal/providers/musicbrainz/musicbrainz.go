package musicbrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Client struct {
	UserAgent string
	HTTP      *http.Client

	mu       sync.Mutex
	lastCall time.Time
}

func (c *Client) throttle() {
	c.mu.Lock()
	defer c.mu.Unlock()
	// MusicBrainz asks for ~1 request per second for anonymous clients.
	wait := time.Until(c.lastCall.Add(1100 * time.Millisecond))
	if wait > 0 {
		time.Sleep(wait)
	}
	c.lastCall = time.Now()
}

type ReleaseHit struct {
	ID             string
	Title          string
	Date           string
	ArtistName     string
	Disambiguation string
}

type releaseSearchResponse struct {
	Releases []releaseDoc `json:"releases"`
}

type releaseDoc struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Date           string `json:"date"`
	Disambiguation string `json:"disambiguation"`
	ArtistCredit   []struct {
		Artist struct {
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"artist-credit"`
}

func (c *Client) SearchReleases(ctx context.Context, query string) ([]ReleaseHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	c.throttle()

	u := "https://musicbrainz.org/ws/2/release?fmt=json&limit=15&query=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	ua := strings.TrimSpace(c.UserAgent)
	if ua == "" {
		ua = "MediaLibraryManager/1.0"
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "application/json")

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
		return nil, fmt.Errorf("musicbrainz: unexpected status %d", res.StatusCode)
	}

	var payload releaseSearchResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, err
	}

	var out []ReleaseHit
	for _, r := range payload.Releases {
		id := strings.TrimSpace(r.ID)
		title := strings.TrimSpace(r.Title)
		if id == "" || title == "" {
			continue
		}
		artist := ""
		if len(r.ArtistCredit) > 0 {
			artist = strings.TrimSpace(r.ArtistCredit[0].Artist.Name)
		}
		out = append(out, ReleaseHit{
			ID:             id,
			Title:          title,
			Date:           strings.TrimSpace(r.Date),
			ArtistName:     artist,
			Disambiguation: strings.TrimSpace(r.Disambiguation),
		})
	}
	return out, nil
}

func YearFromPartialDate(s string) *int {
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
	return &http.Client{Timeout: 20 * time.Second}
}
