package search

import (
	"context"
	"strings"
	"sync"

	domainlib "media_library_manager/internal/domain/library"
)

// Group keeps each media type's provider ranking and warnings independent.
type Group struct {
	MediaType string
	Outcome
}

// SearchAll runs type-specific lookups concurrently. A failed provider must not
// discard results from the others; groups remain in the app's media-type order.
func (s *Service) SearchAll(ctx context.Context, query string) ([]Group, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	groups := make([]Group, len(domainlib.MediaTypes))
	var pending sync.WaitGroup
	for i, mediaType := range domainlib.MediaTypes {
		pending.Add(1)
		go func(index int, kind string) {
			defer pending.Done()
			outcome, err := s.Search(ctx, kind, query)
			if err != nil {
				outcome = Outcome{Warnings: []string{"search.warnings.providerUnavailable"}}
			}
			groups[index] = Group{MediaType: kind, Outcome: outcome}
		}(i, mediaType)
	}
	pending.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return groups, nil
}
