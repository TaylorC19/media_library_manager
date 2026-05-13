package reliability

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type cacheRepository interface {
	FindActive(ctx context.Context, provider, cacheKey string) ([]byte, error)
	Set(ctx context.Context, provider, cacheKey string, payload []byte, expiresAt time.Time) error
}

type Cache struct {
	repo        cacheRepository
	reliability *Service
	inFlight    sync.Map
}

type flight struct {
	done  chan struct{}
	value any
	err   error
}

func NewCache(repo cacheRepository, reliability *Service) *Cache {
	return &Cache{
		repo:        repo,
		reliability: reliability,
	}
}

func Wrap[T any](ctx context.Context, c *Cache, operation Operation, provider, cacheKey string, loader func(context.Context) (T, error)) (T, error) {
	var zero T
	if c == nil || c.repo == nil || c.reliability == nil {
		return loader(ctx)
	}

	cached, err := c.repo.FindActive(ctx, provider, cacheKey)
	if err != nil {
		return zero, err
	}
	if len(cached) > 0 {
		var value T
		if err := json.Unmarshal(cached, &value); err != nil {
			return zero, fmt.Errorf("decode provider cache: %w", err)
		}
		return value, nil
	}

	flightKey := provider + ":" + cacheKey
	if existing, ok := c.inFlight.Load(flightKey); ok {
		return awaitFlight[T](ctx, existing.(*flight))
	}

	f := &flight{done: make(chan struct{})}
	actual, loaded := c.inFlight.LoadOrStore(flightKey, f)
	if loaded {
		return awaitFlight[T](ctx, actual.(*flight))
	}

	defer c.inFlight.Delete(flightKey)

	value, err := loader(ctx)
	if err == nil {
		payload, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			err = fmt.Errorf("encode provider cache: %w", marshalErr)
		} else {
			policy := c.reliability.CachePolicy(operation)
			ttl := policy.TTL
			if c.reliability.IsNegativeCacheEligible(operation, value) {
				ttl = policy.NegativeTTL
			}
			if setErr := c.repo.Set(ctx, provider, cacheKey, payload, time.Now().UTC().Add(ttl)); setErr != nil {
				err = setErr
			}
		}
	}

	f.value = value
	f.err = err
	close(f.done)

	if err != nil {
		return zero, err
	}
	return value, nil
}

func awaitFlight[T any](ctx context.Context, f *flight) (T, error) {
	var zero T
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case <-f.done:
		if f.err != nil {
			return zero, f.err
		}
		value, ok := f.value.(T)
		if !ok {
			return zero, fmt.Errorf("provider cache flight returned unexpected type %T", f.value)
		}
		return value, nil
	}
}
