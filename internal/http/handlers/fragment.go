package handlers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	defaultPageSize = 10
	maxPageSize     = 50
)

type PaginationLink struct {
	Page    int
	Label   string
	URL     string
	Current bool
}

type PaginationData struct {
	Page       int
	PageSize   int
	TotalItems int
	TotalPages int
	StartItem  int
	EndItem    int
	PrevURL    string
	NextURL    string
	Links      []PaginationLink
}

type InlineNotice struct {
	Level     string
	Message   string
	DetailURL string
	DetailKey string
}

func isHTMX(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("HX-Request")), "true")
}

func parsePage(raw string) int {
	return parsePositiveInt(raw, 1, 10_000)
}

func parsePageSize(raw string) int {
	return parsePositiveInt(raw, defaultPageSize, maxPageSize)
}

func parsePositiveInt(raw string, fallback, upperBound int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	if upperBound > 0 && value > upperBound {
		return upperBound
	}
	return value
}

func pageBounds(totalItems, page, pageSize int) (int, int) {
	if totalItems <= 0 {
		return 0, 0
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	start := (page - 1) * pageSize
	if start >= totalItems {
		return totalItems, totalItems
	}
	end := start + pageSize
	if end > totalItems {
		end = totalItems
	}
	return start, end
}

func buildPagination(basePath string, values url.Values, page, pageSize, totalItems int) PaginationData {
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	totalPages := 0
	if totalItems > 0 {
		totalPages = (totalItems + pageSize - 1) / pageSize
	}
	if totalPages == 0 {
		totalPages = 1
	}
	if page <= 0 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	startItem := 0
	endItem := 0
	if totalItems > 0 {
		startItem = ((page - 1) * pageSize) + 1
		endItem = startItem + pageSize - 1
		if endItem > totalItems {
			endItem = totalItems
		}
	}

	pagination := PaginationData{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
		StartItem:  startItem,
		EndItem:    endItem,
	}
	if totalItems <= pageSize {
		return pagination
	}

	windowStart := page - 2
	if windowStart < 1 {
		windowStart = 1
	}
	windowEnd := windowStart + 4
	if windowEnd > totalPages {
		windowEnd = totalPages
	}
	if windowEnd-windowStart < 4 {
		windowStart = windowEnd - 4
		if windowStart < 1 {
			windowStart = 1
		}
	}

	for current := windowStart; current <= windowEnd; current++ {
		pagination.Links = append(pagination.Links, PaginationLink{
			Page:    current,
			Label:   strconv.Itoa(current),
			URL:     pageURL(basePath, values, current, pageSize),
			Current: current == page,
		})
	}
	if page > 1 {
		pagination.PrevURL = pageURL(basePath, values, page-1, pageSize)
	}
	if page < totalPages {
		pagination.NextURL = pageURL(basePath, values, page+1, pageSize)
	}
	return pagination
}

func pageURL(basePath string, values url.Values, page, pageSize int) string {
	query := cloneValues(values)
	query.Set("page", strconv.Itoa(page))
	query.Set("page_size", strconv.Itoa(pageSize))
	encoded := query.Encode()
	if encoded == "" {
		return basePath
	}
	return basePath + "?" + encoded
}

func cloneValues(values url.Values) url.Values {
	out := make(url.Values, len(values))
	for key, items := range values {
		out[key] = append([]string(nil), items...)
	}
	return out
}
