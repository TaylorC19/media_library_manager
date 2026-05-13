package reliability

import (
	"reflect"
	"strings"
	"time"
	"unicode"

	"media_library_manager/internal/config"
)

type Operation string

const (
	OperationSearch  Operation = "search"
	OperationDetail  Operation = "detail"
	OperationBarcode Operation = "barcode"
)

type CachePolicy struct {
	Operation   Operation
	TTL         time.Duration
	NegativeTTL time.Duration
}

type MusicBrainzThrottleSettings struct {
	MinInterval       time.Duration
	RateLimitCooldown time.Duration
	RetryableCooldown time.Duration
}

type Service struct {
	cfg config.Config
}

func NewService(cfg config.Config) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) CachePolicy(operation Operation) CachePolicy {
	switch operation {
	case OperationSearch:
		return CachePolicy{
			Operation:   operation,
			TTL:         s.cfg.ProviderCacheSearchTTL,
			NegativeTTL: s.cfg.ProviderCacheNegativeSearch,
		}
	case OperationDetail:
		return CachePolicy{
			Operation:   operation,
			TTL:         s.cfg.ProviderCacheDetailTTL,
			NegativeTTL: s.cfg.ProviderCacheNegativeDetail,
		}
	case OperationBarcode:
		return CachePolicy{
			Operation:   operation,
			TTL:         s.cfg.ProviderCacheBarcodeTTL,
			NegativeTTL: s.cfg.ProviderCacheNegativeBarcode,
		}
	default:
		return CachePolicy{Operation: operation}
	}
}

func (s *Service) CacheVersionPrefix() string {
	return strings.TrimSpace(s.cfg.ProviderCacheVersion)
}

func (s *Service) SearchCacheKey(mediaType, query string) string {
	return s.withVersion("search:" + strings.TrimSpace(mediaType) + ":" + strings.ToLower(strings.TrimSpace(query)))
}

func (s *Service) DetailCacheKey(mediaType, providerID string) string {
	return s.withVersion("detail:" + strings.TrimSpace(mediaType) + ":" + strings.TrimSpace(providerID))
}

func (s *Service) BarcodeCacheKey(mediaType *string, barcode string) string {
	typed := "any"
	if mediaType != nil && strings.TrimSpace(*mediaType) != "" {
		typed = strings.TrimSpace(*mediaType)
	}
	return s.withVersion("barcode:" + typed + ":" + normalizeBarcode(barcode))
}

func (s *Service) IsNegativeCacheEligible(operation Operation, value any) bool {
	switch operation {
	case OperationSearch, OperationBarcode:
		v := reflect.ValueOf(value)
		if !v.IsValid() {
			return true
		}
		switch v.Kind() {
		case reflect.Slice, reflect.Array:
			return v.Len() == 0
		default:
			return false
		}
	case OperationDetail:
		if value == nil {
			return true
		}
		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
			return v.IsNil()
		default:
			return false
		}
	default:
		return false
	}
}

func (s *Service) MusicBrainzThrottle() MusicBrainzThrottleSettings {
	return MusicBrainzThrottleSettings{
		MinInterval:       s.cfg.MusicBrainzMinInterval,
		RateLimitCooldown: s.cfg.MusicBrainzRateLimitCooldown,
		RetryableCooldown: s.cfg.MusicBrainzRetryableCooldown,
	}
}

func (s *Service) withVersion(cacheKey string) string {
	return s.CacheVersionPrefix() + ":" + cacheKey
}

func normalizeBarcode(value string) string {
	s := strings.TrimSpace(strings.ToUpper(value))
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsDigit(r):
			b.WriteRune(r)
		case r == 'X':
			b.WriteRune(r)
		}
	}
	return b.String()
}
