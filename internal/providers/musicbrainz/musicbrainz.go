package musicbrainz

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"media_library_manager/internal/providers/httpx"
	"media_library_manager/internal/providers/providererrors"
)

type Client struct {
	UserAgent string
	HTTP      *http.Client
	Throttle  *Throttle
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
	return runWithThrottle(ctx, c.Throttle, func() ([]ReleaseHit, error) {
		var payload releaseSearchResponse
		if err := httpx.GetJSON(ctx, c.HTTP, "musicbrainz", "search", "https://musicbrainz.org/ws/2/release", httpx.GetJSONOptions{
			Query: map[string]any{
				"fmt":   "json",
				"limit": 50,
				"query": query,
			},
			Headers: map[string]string{
				"User-Agent": c.userAgent(),
			},
			Timeout: 20 * time.Second,
		}, &payload); err != nil {
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
	})
}

func (c *Client) GetReleaseDetails(ctx context.Context, releaseID string) (*ReleaseDetails, error) {
	releaseID = strings.TrimSpace(releaseID)
	if releaseID == "" {
		return nil, nil
	}
	return runWithThrottle(ctx, c.Throttle, func() (*ReleaseDetails, error) {
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
		err := httpx.GetJSON(ctx, c.HTTP, "musicbrainz", "detail", "https://musicbrainz.org/ws/2/release/"+url.PathEscape(releaseID), httpx.GetJSONOptions{
			Query: map[string]any{
				"fmt": "json",
				"inc": "artists+labels+recordings",
			},
			Headers: map[string]string{
				"User-Agent": c.userAgent(),
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
	})
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

func (c *Client) userAgent() string {
	ua := strings.TrimSpace(c.UserAgent)
	if ua == "" {
		return "MediaLibraryManager/1.0"
	}
	return ua
}

func runWithThrottle[T any](ctx context.Context, throttle *Throttle, fn func() (T, error)) (T, error) {
	return Run(ctx, throttle, fn)
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
