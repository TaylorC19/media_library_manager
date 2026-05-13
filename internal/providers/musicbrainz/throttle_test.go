package musicbrainz

import (
	"context"
	"errors"
	"testing"
	"time"

	"media_library_manager/internal/providers/providererrors"
	"media_library_manager/internal/service/reliability"
)

func TestThrottleRespectsMinimumInterval(t *testing.T) {
	throttle := NewThrottle(func() reliability.MusicBrainzThrottleSettings {
		return reliability.MusicBrainzThrottleSettings{
			MinInterval:       30 * time.Millisecond,
			RateLimitCooldown: 40 * time.Millisecond,
			RetryableCooldown: 20 * time.Millisecond,
		}
	})

	start := time.Now()
	if _, err := Run(context.Background(), throttle, func() (string, error) { return "one", nil }); err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if _, err := Run(context.Background(), throttle, func() (string, error) { return "two", nil }); err != nil {
		t.Fatalf("second call error: %v", err)
	}

	if elapsed := time.Since(start); elapsed < 25*time.Millisecond {
		t.Fatalf("elapsed = %v, want at least min interval", elapsed)
	}
}

func TestThrottleAddsRateLimitCooldown(t *testing.T) {
	throttle := NewThrottle(func() reliability.MusicBrainzThrottleSettings {
		return reliability.MusicBrainzThrottleSettings{
			MinInterval:       20 * time.Millisecond,
			RateLimitCooldown: 50 * time.Millisecond,
			RetryableCooldown: 10 * time.Millisecond,
		}
	})

	rateLimited := providererrors.New(providererrors.Options{
		Provider:  "musicbrainz",
		Operation: "search",
		Code:      providererrors.CodeRateLimited,
		Message:   "rate limited",
		Retryable: true,
	})

	if _, err := Run(context.Background(), throttle, func() (string, error) {
		return "", rateLimited
	}); !errors.Is(err, rateLimited) {
		t.Fatalf("first error = %v, want %v", err, rateLimited)
	}

	start := time.Now()
	if _, err := Run(context.Background(), throttle, func() (string, error) { return "ok", nil }); err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 65*time.Millisecond {
		t.Fatalf("elapsed = %v, want min interval plus cooldown", elapsed)
	}
}

func TestThrottleAddsRetryableCooldown(t *testing.T) {
	throttle := NewThrottle(func() reliability.MusicBrainzThrottleSettings {
		return reliability.MusicBrainzThrottleSettings{
			MinInterval:       15 * time.Millisecond,
			RateLimitCooldown: 50 * time.Millisecond,
			RetryableCooldown: 35 * time.Millisecond,
		}
	})

	retryable := providererrors.New(providererrors.Options{
		Provider:  "musicbrainz",
		Operation: "detail",
		Code:      providererrors.CodeUpstream,
		Message:   "upstream",
		Retryable: true,
	})

	if _, err := Run(context.Background(), throttle, func() (string, error) {
		return "", retryable
	}); !errors.Is(err, retryable) {
		t.Fatalf("first error = %v, want %v", err, retryable)
	}

	start := time.Now()
	if _, err := Run(context.Background(), throttle, func() (string, error) { return "ok", nil }); err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 45*time.Millisecond {
		t.Fatalf("elapsed = %v, want min interval plus retryable cooldown", elapsed)
	}
}
