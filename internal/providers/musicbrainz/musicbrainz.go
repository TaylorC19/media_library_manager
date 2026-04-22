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
	Barcode        string
	ArtistName     string
	Disambiguation string
}

type ReleaseDetails struct {
	ID            string
	Title         string
	Date          string
	Country       string
	Barcode       string
	Artists       []string
	Label         string
	CatalogNumber string
	TrackCount    *int
}

type releaseSearchResponse struct {
	Releases []releaseDoc `json:"releases"`
}

type releaseDoc struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Date           string `json:"date"`
	Barcode        string `json:"barcode"`
	Disambiguation string `json:"disambiguation"`
	ArtistCredit   []struct {
		Artist struct {
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"artist-credit"`
}

// SearchByBarcode queries MusicBrainz for releases whose barcode field matches the given code.
func (c *Client) SearchByBarcode(ctx context.Context, barcode string) ([]ReleaseHit, error) {
	barcode = strings.TrimSpace(barcode)
	if barcode == "" {
		return nil, nil
	}
	return c.SearchReleases(ctx, "barcode:"+barcode)
}

func (c *Client) SearchReleases(ctx context.Context, query string) ([]ReleaseHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	c.throttle()

	u := "https://musicbrainz.org/ws/2/release?fmt=json&limit=50&query=" + url.QueryEscape(query)
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
			Barcode:        strings.TrimSpace(r.Barcode),
			ArtistName:     artist,
			Disambiguation: strings.TrimSpace(r.Disambiguation),
		})
	}
	return out, nil
}

func (c *Client) GetReleaseDetails(ctx context.Context, releaseID string) (*ReleaseDetails, error) {
	releaseID = strings.TrimSpace(releaseID)
	if releaseID == "" {
		return nil, nil
	}
	c.throttle()

	u := "https://musicbrainz.org/ws/2/release/" + url.PathEscape(releaseID) + "?fmt=json&inc=artists+labels+recordings"
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
	if res.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("musicbrainz: unexpected status %d", res.StatusCode)
	}

	var payload struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		Date         string `json:"date"`
		Country      string `json:"country"`
		Barcode      string `json:"barcode"`
		ArtistCredit []struct {
			Name   string `json:"name"`
			Artist struct {
				Name string `json:"name"`
			} `json:"artist"`
		} `json:"artist-credit"`
		LabelInfo []struct {
			CatalogNumber string `json:"catalog-number"`
			Label         *struct {
				Name string `json:"name"`
			} `json:"label"`
		} `json:"label-info"`
		Media []struct {
			TrackCount int `json:"track-count"`
		} `json:"media"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, err
	}

	title := strings.TrimSpace(payload.Title)
	if strings.TrimSpace(payload.ID) == "" || title == "" {
		return nil, nil
	}

	var trackCount *int
	totalTracks := 0
	for _, medium := range payload.Media {
		if medium.TrackCount > 0 {
			totalTracks += medium.TrackCount
		}
	}
	if totalTracks > 0 {
		trackCount = &totalTracks
	}

	label := ""
	catalogNumber := ""
	for _, info := range payload.LabelInfo {
		if label == "" && info.Label != nil {
			label = strings.TrimSpace(info.Label.Name)
		}
		if catalogNumber == "" {
			catalogNumber = strings.TrimSpace(info.CatalogNumber)
		}
	}

	return &ReleaseDetails{
		ID:            strings.TrimSpace(payload.ID),
		Title:         title,
		Date:          strings.TrimSpace(payload.Date),
		Country:       strings.TrimSpace(payload.Country),
		Barcode:       strings.TrimSpace(payload.Barcode),
		Artists:       uniqueArtistNames(payload.ArtistCredit),
		Label:         label,
		CatalogNumber: catalogNumber,
		TrackCount:    trackCount,
	}, nil
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

func uniqueArtistNames(credits []struct {
	Name   string `json:"name"`
	Artist struct {
		Name string `json:"name"`
	} `json:"artist"`
}) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, credit := range credits {
		name := strings.TrimSpace(credit.Artist.Name)
		if name == "" {
			name = strings.TrimSpace(credit.Name)
		}
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	return out
}
