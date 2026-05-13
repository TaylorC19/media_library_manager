package musicbrainz

import (
	"context"
	"errors"
	"sync"
	"time"

	"media_library_manager/internal/providers/providererrors"
	"media_library_manager/internal/service/reliability"
)

type Throttle struct {
	mu          sync.Mutex
	settings    func() reliability.MusicBrainzThrottleSettings
	nextAllowed time.Time
}

func NewThrottle(settings func() reliability.MusicBrainzThrottleSettings) *Throttle {
	return &Throttle{settings: settings}
}

func Run[T any](ctx context.Context, t *Throttle, fn func() (T, error)) (T, error) {
	var zero T
	if t == nil {
		return fn()
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	settings := reliability.MusicBrainzThrottleSettings{}
	if t.settings != nil {
		settings = t.settings()
	}

	if wait := time.Until(t.nextAllowed); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-timer.C:
		}
	}

	value, err := fn()
	t.nextAllowed = time.Now().UTC().Add(settings.MinInterval)
	if err != nil {
		t.nextAllowed = time.Now().UTC().Add(settings.MinInterval + cooldownForError(err, settings))
	}
	return value, err
}

func cooldownForError(err error, settings reliability.MusicBrainzThrottleSettings) time.Duration {
	var perr *providererrors.Error
	if !errors.As(err, &perr) {
		return 0
	}

	switch perr.Code {
	case providererrors.CodeRateLimited:
		return settings.RateLimitCooldown
	case providererrors.CodeTimeout, providererrors.CodeNetwork, providererrors.CodeUpstream:
		return settings.RetryableCooldown
	default:
		return 0
	}
}
