package reliability

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"media_library_manager/internal/config"
)

type fakeCacheRepo struct {
	mu        sync.Mutex
	values    map[string][]byte
	expiresAt map[string]time.Time
	setCalls  int
}

func newFakeCacheRepo() *fakeCacheRepo {
	return &fakeCacheRepo{
		values:    map[string][]byte{},
		expiresAt: map[string]time.Time{},
	}
}

func (r *fakeCacheRepo) FindActive(_ context.Context, provider, cacheKey string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]byte(nil), r.values[provider+":"+cacheKey]...), nil
}

func (r *fakeCacheRepo) Set(_ context.Context, provider, cacheKey string, payload []byte, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := provider + ":" + cacheKey
	r.values[key] = append([]byte(nil), payload...)
	r.expiresAt[key] = expiresAt
	r.setCalls++
	return nil
}

func TestCacheWrapUsesExistingValue(t *testing.T) {
	repo := newFakeCacheRepo()
	cache := NewCache(repo, newReliabilityService())
	key := "tmdb:v1:search:movie:alien"
	payload, err := json.Marshal([]string{"cached"})
	if err != nil {
		t.Fatal(err)
	}
	repo.values[key] = payload

	var calls int32
	got, err := Wrap(context.Background(), cache, OperationSearch, "tmdb", "v1:search:movie:alien", func(context.Context) ([]string, error) {
		atomic.AddInt32(&calls, 1)
		return []string{"fresh"}, nil
	})
	if err != nil {
		t.Fatalf("Wrap returned error: %v", err)
	}
	if len(got) != 1 || got[0] != "cached" {
		t.Fatalf("got %v, want cached payload", got)
	}
	if calls != 0 {
		t.Fatalf("loader calls = %d, want 0", calls)
	}
}

func TestCacheWrapUsesNegativeTTLForEmptySearch(t *testing.T) {
	repo := newFakeCacheRepo()
	cache := NewCache(repo, newReliabilityService())
	start := time.Now().UTC()

	got, err := Wrap(context.Background(), cache, OperationSearch, "open_library", "v1:search:book:none", func(context.Context) ([]string, error) {
		return []string{}, nil
	})
	if err != nil {
		t.Fatalf("Wrap returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %v, want empty slice", got)
	}

	repo.mu.Lock()
	expiresAt := repo.expiresAt["open_library:v1:search:book:none"]
	repo.mu.Unlock()
	gotTTL := expiresAt.Sub(start)
	if gotTTL < 14*time.Minute || gotTTL > 16*time.Minute {
		t.Fatalf("negative TTL = %v, want about 15m", gotTTL)
	}
}

func TestCacheWrapDoesNotCacheErrors(t *testing.T) {
	repo := newFakeCacheRepo()
	cache := NewCache(repo, newReliabilityService())
	wantErr := errors.New("upstream failed")

	_, err := Wrap(context.Background(), cache, OperationDetail, "rawg", "v1:detail:game:123", func(context.Context) (*string, error) {
		return nil, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if repo.setCalls != 0 {
		t.Fatalf("set calls = %d, want 0", repo.setCalls)
	}
}

func TestCacheWrapCollapsesConcurrentMisses(t *testing.T) {
	repo := newFakeCacheRepo()
	cache := NewCache(repo, newReliabilityService())

	var calls int32
	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make([][]string, 8)
	errs := make([]error, 8)

	for i := range results {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			result, err := Wrap(context.Background(), cache, OperationSearch, "tmdb", "v1:search:movie:arrival", func(context.Context) ([]string, error) {
				if atomic.AddInt32(&calls, 1) == 1 {
					close(start)
					time.Sleep(30 * time.Millisecond)
				}
				return []string{"arrival"}, nil
			})
			results[idx] = result
			errs[idx] = err
		}(i)
	}

	<-start
	wg.Wait()

	if calls != 1 {
		t.Fatalf("loader calls = %d, want 1", calls)
	}
	for i, err := range errs {
		if err != nil {
			t.Fatalf("errs[%d] = %v, want nil", i, err)
		}
		if len(results[i]) != 1 || results[i][0] != "arrival" {
			t.Fatalf("results[%d] = %v, want arrival", i, results[i])
		}
	}
}

func newReliabilityService() *Service {
	return NewService(config.Config{
		ProviderCacheVersion:         "v1",
		ProviderCacheSearchTTL:       12 * time.Hour,
		ProviderCacheNegativeSearch:  15 * time.Minute,
		ProviderCacheDetailTTL:       7 * 24 * time.Hour,
		ProviderCacheNegativeDetail:  time.Hour,
		ProviderCacheBarcodeTTL:      7 * 24 * time.Hour,
		ProviderCacheNegativeBarcode: time.Hour,
		MusicBrainzMinInterval:       1100 * time.Millisecond,
		MusicBrainzRateLimitCooldown: 5 * time.Second,
		MusicBrainzRetryableCooldown: 2 * time.Second,
	})
}
