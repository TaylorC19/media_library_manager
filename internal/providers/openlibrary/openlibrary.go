package openlibrary

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
	HTTP *http.Client
}

type WorkHit struct {
	Key            string
	Title          string
	AuthorNames    []string
	FirstYear      *int
	CoverID        *int
	FirstSentence  string
	SubjectSnippet string
}

type WorkDetails struct {
	WorkKey          string
	Title            string
	Authors          []string
	FirstPublishDate string
	CoverID          *int
	Description      string
}

type searchResponse struct {
	Docs []workDoc `json:"docs"`
}

type workDoc struct {
	Key           string   `json:"key"`
	Title         string   `json:"title"`
	AuthorName    []string `json:"author_name"`
	FirstSentence any      `json:"first_sentence"`
	Subject       []string `json:"subject"`
	CoverI        *int     `json:"cover_i"`
	FirstPublish  *int     `json:"first_publish_year"`
}

func (c *Client) SearchWorks(ctx context.Context, query string) ([]WorkHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	u := "https://openlibrary.org/search.json?limit=50&q=" + url.QueryEscape(query)
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
		return nil, fmt.Errorf("openlibrary: unexpected status %d", res.StatusCode)
	}

	var payload searchResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, err
	}

	var out []WorkHit
	for _, d := range payload.Docs {
		key := strings.TrimSpace(d.Key)
		title := strings.TrimSpace(d.Title)
		if key == "" || title == "" {
			continue
		}
		summary := firstSentenceString(d.FirstSentence)
		if summary == "" && len(d.Subject) > 0 {
			summary = strings.TrimSpace(d.Subject[0])
		}
		out = append(out, WorkHit{
			Key:            key,
			Title:          title,
			AuthorNames:    append([]string{}, d.AuthorName...),
			FirstYear:      d.FirstPublish,
			CoverID:        d.CoverI,
			FirstSentence:  summary,
			SubjectSnippet: summary,
		})
	}
	return out, nil
}

func (c *Client) GetWorkDetails(ctx context.Context, workKey string) (*WorkDetails, error) {
	workKey = strings.TrimSpace(workKey)
	workKey = strings.TrimPrefix(workKey, "/works/")
	workKey = strings.TrimPrefix(workKey, "works/")
	if workKey == "" {
		return nil, nil
	}

	workURL := "https://openlibrary.org/works/" + url.PathEscape(workKey) + ".json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, workURL, nil)
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
		return nil, fmt.Errorf("openlibrary: unexpected status %d", res.StatusCode)
	}

	var payload struct {
		Key              string `json:"key"`
		Title            string `json:"title"`
		Description      any    `json:"description"`
		Covers           []int  `json:"covers"`
		FirstPublishDate string `json:"first_publish_date"`
		Authors          []struct {
			Author struct {
				Key string `json:"key"`
			} `json:"author"`
		} `json:"authors"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, err
	}

	title := strings.TrimSpace(payload.Title)
	if title == "" {
		return nil, nil
	}

	var coverID *int
	if len(payload.Covers) > 0 && payload.Covers[0] > 0 {
		coverID = &payload.Covers[0]
	}

	authors, err := c.fetchAuthorNames(ctx, payload.Authors)
	if err != nil {
		return nil, err
	}

	return &WorkDetails{
		WorkKey:          workKey,
		Title:            title,
		Authors:          authors,
		FirstPublishDate: strings.TrimSpace(payload.FirstPublishDate),
		CoverID:          coverID,
		Description:      descriptionString(payload.Description),
	}, nil
}

func CoverURL(coverID int) *string {
	if coverID <= 0 {
		return nil
	}
	u := fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-M.jpg", coverID)
	return &u
}

func CoverURLFromID(coverID *int) *string {
	if coverID == nil {
		return nil
	}
	return CoverURL(*coverID)
}

func firstSentenceString(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case []any:
		if len(x) == 0 {
			return ""
		}
		if s, ok := x[0].(string); ok {
			return strings.TrimSpace(s)
		}
		return ""
	default:
		return ""
	}
}

func DefaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}

func JoinAuthors(names []string) string {
	var b strings.Builder
	seen := map[string]struct{}{}
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		key := strings.ToLower(n)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if b.Len() > 0 {
			b.WriteString(", ")
		}
		b.WriteString(n)
	}
	return b.String()
}

func (c *Client) fetchAuthorNames(ctx context.Context, refs []struct {
	Author struct {
		Key string `json:"key"`
	} `json:"author"`
}) ([]string, error) {
	seen := map[string]struct{}{}
	var out []string
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}

	for _, ref := range refs {
		key := strings.TrimSpace(ref.Author.Key)
		key = strings.TrimPrefix(key, "/authors/")
		key = strings.TrimPrefix(key, "authors/")
		if key == "" {
			continue
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://openlibrary.org/authors/"+url.PathEscape(key)+".json", nil)
		if err != nil {
			return nil, err
		}
		res, err := hc.Do(req)
		if err != nil {
			return nil, err
		}
		func() {
			defer res.Body.Close()
			if res.StatusCode < 200 || res.StatusCode >= 300 {
				return
			}
			var payload struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
				return
			}
			name := strings.TrimSpace(payload.Name)
			if name == "" {
				return
			}
			lower := strings.ToLower(name)
			if _, ok := seen[lower]; ok {
				return
			}
			seen[lower] = struct{}{}
			out = append(out, name)
		}()
	}

	return out, nil
}

func descriptionString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		if text, ok := v["value"].(string); ok {
			return strings.TrimSpace(text)
		}
	}
	return ""
}
