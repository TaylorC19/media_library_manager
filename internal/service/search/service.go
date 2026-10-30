package search

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"media_library_manager/internal/config"
	domainlib "media_library_manager/internal/domain/library"
	domainsearch "media_library_manager/internal/domain/search"
	"media_library_manager/internal/providerkeys"
	"media_library_manager/internal/providers/discogs"
	"media_library_manager/internal/providers/httpx"
	"media_library_manager/internal/providers/musicbrainz"
	"media_library_manager/internal/providers/openlibrary"
	"media_library_manager/internal/providers/providererrors"
	"media_library_manager/internal/providers/rawg"
	"media_library_manager/internal/providers/tmdb"
	"media_library_manager/internal/service/reliability"
)

type Service struct {
	cfg         config.Config
	cache       *reliability.Cache
	reliability *reliability.Service
	tmdb        *tmdb.Client
	mb          *musicbrainz.Client
	ol          *openlibrary.Client
	rawg        *rawg.Client
	discogs     *discogs.Client
}

func NewService(cfg config.Config, cache *reliability.Cache, reliabilitySvc *reliability.Service, mbThrottle *musicbrainz.Throttle) *Service {
	httpTMDB := tmdb.DefaultHTTPClient()
	httpMB := musicbrainz.DefaultHTTPClient()
	httpOL := openlibrary.DefaultHTTPClient()
	httpRAWG := rawg.DefaultHTTPClient()

	return &Service{
		cfg:         cfg,
		cache:       cache,
		reliability: reliabilitySvc,
		tmdb: &tmdb.Client{
			APIKey: cfg.TMDBAPIKey,
			HTTP:   httpTMDB,
		},
		mb: &musicbrainz.Client{
			UserAgent: cfg.MusicBrainzUA,
			HTTP:      httpMB,
			Throttle:  mbThrottle,
		},
		ol: &openlibrary.Client{HTTP: httpOL},
		rawg: &rawg.Client{
			APIKey: cfg.RAWGAPIKey,
			HTTP:   httpRAWG,
		},
		discogs: &discogs.Client{
			Token: cfg.DiscogsToken,
			HTTP:  discogs.DefaultHTTPClient(),
		},
	}
}

type Outcome struct {
	Total    int
	Page     int
	PageSize int
	Hits     []domainsearch.Hit
	Warnings []string // translation keys
}

// Search returns the small first-page preview used by grouped searches.
func (s *Service) Search(ctx context.Context, mediaType, query string) (Outcome, error) {
	return s.SearchPage(ctx, mediaType, query, 1, 3)
}

func (s *Service) SearchPage(ctx context.Context, mediaType, query string, page, pageSize int) (Outcome, error) {
	if page < 1 {
		page = 1
	}
	if page > 10000 {
		page = 10000
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}
	if page > 1 {
		// Cached first-page metadata lets us clamp stale or forged page links before
		// calling providers which reject offsets beyond the last page.
		first, err := s.searchPage(ctx, mediaType, query, 1, pageSize, true)
		if err != nil || len(first.Warnings) > 0 {
			return first, err
		}
		last := (first.Total + pageSize - 1) / pageSize
		if last < 1 {
			last = 1
		}
		if page > last {
			page = last
		}
	}
	return s.searchPage(ctx, mediaType, query, page, pageSize, false)
}

func (s *Service) searchPage(ctx context.Context, mediaType, query string, page, pageSize int, metadataOnly bool) (Outcome, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return Outcome{}, nil
	}
	if !domainlib.IsMediaType(mediaType) {
		return Outcome{}, fmt.Errorf("invalid media type")
	}

	var warnings []string
	var hits []domainsearch.Hit
	total := 0
	offset := (page - 1) * pageSize
	cacheKey := s.reliability.SearchCacheKey(mediaType, query) + fmt.Sprintf(":provider-page:v1:%d:%d", page, pageSize)

	switch mediaType {
	case "movie":
		if providerkeys.Resolve(ctx, "tmdb", s.cfg.TMDBAPIKey) == "" {
			warnings = append(warnings, "search.warnings.tmdbKeyMissing")
			return Outcome{Warnings: warnings}, nil
		}
		rows, err := reliability.Wrap(ctx, s.cache, reliability.OperationSearch, "tmdb", cacheKey, func(ctx context.Context) (httpx.Page[tmdb.Result], error) {
			return s.tmdb.SearchPage(ctx, "movie", query, offset, pageSize)
		})
		if err != nil {
			if searchWarning, ok := providerSearchWarning(err); ok {
				warnings = append(warnings, searchWarning)
				return Outcome{Warnings: warnings}, nil
			}
			return Outcome{}, err
		}
		total = rows.Total
		if metadataOnly {
			break
		}
		for _, r := range rows.Items {
			img := tmdb.PosterURL(r.PosterPath)
			year := tmdb.YearFromDate(r.ReleaseDate)
			hits = append(hits, domainsearch.Hit{
				Provider:   "tmdb",
				ExternalID: strconv.Itoa(r.ID),
				MediaType:  "movie",
				Title:      r.Title,
				Year:       year,
				Summary:    truncate(r.Overview, 280),
				ImageURL:   img,
				TMDBKind:   "movie",
			})
		}

	case "tv":
		if providerkeys.Resolve(ctx, "tmdb", s.cfg.TMDBAPIKey) == "" {
			warnings = append(warnings, "search.warnings.tmdbKeyMissing")
			return Outcome{Warnings: warnings}, nil
		}
		rows, err := reliability.Wrap(ctx, s.cache, reliability.OperationSearch, "tmdb", cacheKey, func(ctx context.Context) (httpx.Page[tmdb.Result], error) {
			return s.tmdb.SearchPage(ctx, "tv", query, offset, pageSize)
		})
		if err != nil {
			if searchWarning, ok := providerSearchWarning(err); ok {
				warnings = append(warnings, searchWarning)
				return Outcome{Warnings: warnings}, nil
			}
			return Outcome{}, err
		}
		total = rows.Total
		if metadataOnly {
			break
		}
		for _, r := range rows.Items {
			img := tmdb.PosterURL(r.PosterPath)
			year := tmdb.YearFromDate(r.ReleaseDate)
			hits = append(hits, domainsearch.Hit{
				Provider:   "tmdb",
				ExternalID: strconv.Itoa(r.ID),
				MediaType:  "tv",
				Title:      r.Title,
				Year:       year,
				Summary:    truncate(r.Overview, 280),
				ImageURL:   img,
				TMDBKind:   "tv",
			})
		}

	case "album":
		rows, err := reliability.Wrap(ctx, s.cache, reliability.OperationSearch, "musicbrainz", cacheKey, func(ctx context.Context) (httpx.Page[musicbrainz.ReleaseHit], error) {
			return s.mb.SearchReleasesPage(ctx, query, offset, pageSize)
		})
		if err != nil {
			if searchWarning, ok := providerSearchWarning(err); ok {
				warnings = append(warnings, searchWarning)
				return Outcome{Warnings: warnings}, nil
			}
			return Outcome{}, err
		}
		total = rows.Total
		if metadataOnly {
			break
		}
		for _, r := range rows.Items {
			sub := strings.TrimSpace(r.ArtistName)
			if r.Disambiguation != "" {
				if sub != "" {
					sub = sub + " — " + r.Disambiguation
				} else {
					sub = r.Disambiguation
				}
			}
			sub = s.discogs.EnrichAlbumSubtitle(ctx, r.ArtistName, r.Title, sub)
			hits = append(hits, domainsearch.Hit{
				Provider:   "musicbrainz",
				ExternalID: r.ID,
				MediaType:  "album",
				Title:      r.Title,
				Subtitle:   sub,
				Year:       musicbrainz.YearFromPartialDate(r.Date),
				Summary:    "",
			})
		}

	case "book":
		rows, err := reliability.Wrap(ctx, s.cache, reliability.OperationSearch, "open_library", cacheKey, func(ctx context.Context) (httpx.Page[openlibrary.WorkHit], error) {
			return s.ol.SearchWorksPage(ctx, query, offset, pageSize)
		})
		if err != nil {
			if searchWarning, ok := providerSearchWarning(err); ok {
				warnings = append(warnings, searchWarning)
				return Outcome{Warnings: warnings}, nil
			}
			return Outcome{}, err
		}
		total = rows.Total
		if metadataOnly {
			break
		}
		for _, w := range rows.Items {
			var img *string
			if w.CoverID != nil {
				img = openlibrary.CoverURL(*w.CoverID)
			}
			sub := openlibrary.JoinAuthors(w.AuthorNames)
			extID := strings.TrimPrefix(strings.TrimSpace(w.Key), "/works/")
			extID = strings.TrimPrefix(extID, "works/")
			hits = append(hits, domainsearch.Hit{
				Provider:   "open_library",
				ExternalID: extID,
				MediaType:  "book",
				Title:      w.Title,
				Subtitle:   sub,
				Year:       intPtrFromInt(w.FirstYear),
				Summary:    truncate(w.FirstSentence, 280),
				ImageURL:   img,
			})
		}

	case "game":
		if providerkeys.Resolve(ctx, "rawg", s.cfg.RAWGAPIKey) == "" {
			warnings = append(warnings, "search.warnings.rawgKeyMissing")
			return Outcome{Warnings: warnings}, nil
		}
		rows, err := reliability.Wrap(ctx, s.cache, reliability.OperationSearch, "rawg", cacheKey, func(ctx context.Context) (httpx.Page[rawg.GameHit], error) {
			return s.rawg.SearchGamesPage(ctx, query, page, pageSize)
		})
		if err != nil {
			if searchWarning, ok := providerSearchWarning(err); ok {
				warnings = append(warnings, searchWarning)
				return Outcome{Warnings: warnings}, nil
			}
			return Outcome{}, err
		}
		total = rows.Total
		if metadataOnly {
			break
		}
		for _, g := range rows.Items {
			img := rawg.BackgroundURL(g.BackgroundImage)
			hits = append(hits, domainsearch.Hit{
				Provider:   "rawg",
				ExternalID: strconv.Itoa(g.ID),
				MediaType:  "game",
				Title:      g.Name,
				Year:       yearFromReleased(g.Released),
				Summary:    "",
				ImageURL:   img,
			})
		}
	}

	return Outcome{Hits: hits, Warnings: warnings, Total: total, Page: page, PageSize: pageSize}, nil
}

func providerSearchWarning(err error) (string, bool) {
	var perr *providererrors.Error
	if !errors.As(err, &perr) {
		return "", false
	}
	return "search.warnings.providerUnavailable", true
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func intPtrFromInt(v *int) *int {
	if v == nil || *v <= 0 {
		return nil
	}
	x := *v
	return &x
}

func yearFromReleased(s string) *int {
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
