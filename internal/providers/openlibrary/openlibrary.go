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
	u := "https://openlibrary.org/search.json?limit=15&q=" + url.QueryEscape(query)
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

func CoverURL(coverID int) *string {
	if coverID <= 0 {
		return nil
	}
	u := fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-M.jpg", coverID)
	return &u
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
