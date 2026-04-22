package search

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"media_library_manager/internal/config"
	domainlib "media_library_manager/internal/domain/library"
	domainsearch "media_library_manager/internal/domain/search"
	"media_library_manager/internal/providers/discogs"
	"media_library_manager/internal/providers/musicbrainz"
	"media_library_manager/internal/providers/openlibrary"
	"media_library_manager/internal/providers/rawg"
	"media_library_manager/internal/providers/tmdb"
)

type Service struct {
	cfg     config.Config
	tmdb    *tmdb.Client
	mb      *musicbrainz.Client
	ol      *openlibrary.Client
	rawg    *rawg.Client
	discogs *discogs.Client
}

func NewService(cfg config.Config) *Service {
	httpTMDB := tmdb.DefaultHTTPClient()
	httpMB := musicbrainz.DefaultHTTPClient()
	httpOL := openlibrary.DefaultHTTPClient()
	httpRAWG := rawg.DefaultHTTPClient()

	return &Service{
		cfg: cfg,
		tmdb: &tmdb.Client{
			APIKey: cfg.TMDBAPIKey,
			HTTP:   httpTMDB,
		},
		mb: &musicbrainz.Client{
			UserAgent: cfg.MusicBrainzUA,
			HTTP:      httpMB,
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
	Hits     []domainsearch.Hit
	Warnings []string // translation keys
}

func (s *Service) Search(ctx context.Context, mediaType, query string) (Outcome, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return Outcome{}, nil
	}
	if !domainlib.IsMediaType(mediaType) {
		return Outcome{}, fmt.Errorf("invalid media type")
	}

	var warnings []string
	var hits []domainsearch.Hit

	switch mediaType {
	case "movie":
		if strings.TrimSpace(s.cfg.TMDBAPIKey) == "" {
			warnings = append(warnings, "search.warnings.tmdbKeyMissing")
			return Outcome{Warnings: warnings}, nil
		}
		rows, err := s.tmdb.SearchMovies(ctx, query)
		if err != nil {
			return Outcome{}, err
		}
		for _, r := range rows {
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
		if strings.TrimSpace(s.cfg.TMDBAPIKey) == "" {
			warnings = append(warnings, "search.warnings.tmdbKeyMissing")
			return Outcome{Warnings: warnings}, nil
		}
		rows, err := s.tmdb.SearchTV(ctx, query)
		if err != nil {
			return Outcome{}, err
		}
		for _, r := range rows {
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
		rows, err := s.mb.SearchReleases(ctx, query)
		if err != nil {
			return Outcome{}, err
		}
		for _, r := range rows {
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
		rows, err := s.ol.SearchWorks(ctx, query)
		if err != nil {
			return Outcome{}, err
		}
		for _, w := range rows {
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
		if strings.TrimSpace(s.cfg.RAWGAPIKey) == "" {
			warnings = append(warnings, "search.warnings.rawgKeyMissing")
			return Outcome{Warnings: warnings}, nil
		}
		rows, err := s.rawg.SearchGames(ctx, query)
		if err != nil {
			return Outcome{}, err
		}
		for _, g := range rows {
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

	return Outcome{Hits: hits, Warnings: warnings}, nil
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
