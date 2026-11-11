package config

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	PublicOrigin              string
	SignupEnabled             bool
	ProviderKeysEncryptionKey string
	EnsureIndexes             bool
	Env                       string
	Port                      int
	MongoURI                  string
	MongoDatabase             string
	DefaultLocale             string
	SessionCookieName         string
	SessionTTLHours           int
	SessionCookieSecure       bool

	// Provider API keys (optional; search degrades gracefully when unset).
	ProviderKeysDir              string
	TMDBAPIKey                   string
	RAWGAPIKey                   string
	DiscogsToken                 string
	MusicBrainzUA                string
	ProviderCacheVersion         string
	ProviderCacheSearchTTL       time.Duration
	ProviderCacheNegativeSearch  time.Duration
	ProviderCacheDetailTTL       time.Duration
	ProviderCacheNegativeDetail  time.Duration
	ProviderCacheBarcodeTTL      time.Duration
	ProviderCacheNegativeBarcode time.Duration
	MusicBrainzMinInterval       time.Duration
	MusicBrainzRateLimitCooldown time.Duration
	MusicBrainzRetryableCooldown time.Duration

	BackupEnabled        bool
	BackupMaxUploadBytes int64
}

func Load() (Config, error) {
	// Best-effort .env loading for local development.
	// Existing OS environment values still win.
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") && os.Getenv("VERCEL") != "1" {
		_ = godotenv.Load(".env")
	}
	production := strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") || os.Getenv("VERCEL") == "1"

	cookieName := "mlm_session"
	if production {
		cookieName = "__Host-mlm_session"
	}
	cfg := Config{
		Env:                          getEnv("APP_ENV", "development"),
		PublicOrigin:                 getEnv("PUBLIC_ORIGIN", ""),
		SignupEnabled:                getEnvBool("SIGNUP_ENABLED", !production),
		ProviderKeysEncryptionKey:    getEnv("PROVIDER_KEYS_ENCRYPTION_KEY", ""),
		EnsureIndexes:                getEnvBool("ENSURE_INDEXES", !production),
		Port:                         getEnvInt("PORT", 8080),
		MongoURI:                     getEnvWithAliases([]string{"MONGODB_URI", "MONGODB_URL"}, "mongodb://localhost:27017"),
		MongoDatabase:                getEnv("MONGODB_DATABASE", "media_library"),
		DefaultLocale:                getEnv("DEFAULT_LOCALE", "en"),
		SessionCookieName:            getEnv("SESSION_COOKIE_NAME", cookieName),
		SessionTTLHours:              getEnvInt("SESSION_TTL_HOURS", 24*14),
		SessionCookieSecure:          getEnvBool("SESSION_COOKIE_SECURE", production),
		ProviderKeysDir:              getEnv("PROVIDER_KEYS_DIR", "data/provider-keys"),
		TMDBAPIKey:                   getEnv("TMDB_API_KEY", ""),
		RAWGAPIKey:                   getEnv("RAWG_API_KEY", ""),
		DiscogsToken:                 getEnv("DISCOGS_TOKEN", ""),
		MusicBrainzUA:                getEnv("MUSICBRAINZ_USER_AGENT", "MediaLibraryManager/1.0 (https://example.invalid/contact)"),
		ProviderCacheVersion:         getEnv("PROVIDER_CACHE_VERSION", "v1"),
		ProviderCacheSearchTTL:       getEnvDurationMS("PROVIDER_CACHE_SEARCH_TTL_MS", 12*time.Hour),
		ProviderCacheNegativeSearch:  getEnvDurationMS("PROVIDER_CACHE_NEGATIVE_SEARCH_TTL_MS", 15*time.Minute),
		ProviderCacheDetailTTL:       getEnvDurationMS("PROVIDER_CACHE_DETAIL_TTL_MS", 7*24*time.Hour),
		ProviderCacheNegativeDetail:  getEnvDurationMS("PROVIDER_CACHE_NEGATIVE_DETAIL_TTL_MS", time.Hour),
		ProviderCacheBarcodeTTL:      getEnvDurationMS("PROVIDER_CACHE_BARCODE_TTL_MS", 7*24*time.Hour),
		ProviderCacheNegativeBarcode: getEnvDurationMS("PROVIDER_CACHE_NEGATIVE_BARCODE_TTL_MS", time.Hour),
		MusicBrainzMinInterval:       getEnvDurationMS("MUSICBRAINZ_MIN_INTERVAL_MS", 1100*time.Millisecond),
		MusicBrainzRateLimitCooldown: getEnvDurationMS("MUSICBRAINZ_RATE_LIMIT_COOLDOWN_MS", 5*time.Second),
		MusicBrainzRetryableCooldown: getEnvDurationMS("MUSICBRAINZ_RETRYABLE_COOLDOWN_MS", 2*time.Second),
		BackupEnabled:                getEnvBool("BACKUP_ENABLED", false),
		BackupMaxUploadBytes:         getEnvInt64("BACKUP_MAX_UPLOAD_BYTES", 100*1024*1024),
	}

	cfg.Env = strings.ToLower(strings.TrimSpace(cfg.Env))
	if cfg.Env != "development" && cfg.Env != "production" {
		return Config{}, fmt.Errorf("APP_ENV must be development or production")
	}
	if production {
		cfg.Env = "production"
		if !strings.HasPrefix(cfg.SessionCookieName, "__Host-") {
			return Config{}, fmt.Errorf("production requires a __Host- session cookie name")
		}
		if !cfg.SessionCookieSecure {
			return Config{}, fmt.Errorf("production requires secure session cookies")
		}
		origin, err := url.Parse(cfg.PublicOrigin)
		if err != nil || origin.Scheme != "https" || origin.Host == "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" || (origin.Path != "" && origin.Path != "/") {
			return Config{}, fmt.Errorf("PUBLIC_ORIGIN must be an HTTPS origin")
		}
		cfg.PublicOrigin = strings.TrimSuffix(cfg.PublicOrigin, "/")
		key, err := base64.StdEncoding.DecodeString(cfg.ProviderKeysEncryptionKey)
		if err != nil || len(key) != 32 {
			return Config{}, fmt.Errorf("PROVIDER_KEYS_ENCRYPTION_KEY must encode 32 random bytes in base64")
		}
		if cfg.BackupEnabled {
			return Config{}, fmt.Errorf("web backups are unavailable in production; use database maintenance tools")
		}
		if os.Getenv("MONGODB_URI") == "" && os.Getenv("MONGODB_URL") == "" {
			return Config{}, fmt.Errorf("production requires an explicit MongoDB URI")
		}
		if os.Getenv("MONGODB_DATABASE") == "" {
			return Config{}, fmt.Errorf("production requires an explicit database name")
		}
	}
	if strings.HasPrefix(cfg.SessionCookieName, "__Host-") && !cfg.SessionCookieSecure {
		return Config{}, fmt.Errorf("host cookies require HTTPS")
	}
	if strings.TrimSpace(cfg.ProviderKeysDir) == "" {
		return Config{}, fmt.Errorf("PROVIDER_KEYS_DIR is required")
	}

	if cfg.MongoDatabase == "" {
		return Config{}, fmt.Errorf("MONGODB_DATABASE is required")
	}

	cfg.DefaultLocale = strings.ToLower(strings.TrimSpace(cfg.DefaultLocale))
	if cfg.DefaultLocale != "en" && cfg.DefaultLocale != "ja" {
		return Config{}, fmt.Errorf("DEFAULT_LOCALE must be one of: en, ja")
	}
	if strings.TrimSpace(cfg.SessionCookieName) == "" {
		return Config{}, fmt.Errorf("SESSION_COOKIE_NAME is required")
	}
	if cfg.SessionTTLHours <= 0 {
		return Config{}, fmt.Errorf("SESSION_TTL_HOURS must be > 0")
	}
	if strings.TrimSpace(cfg.ProviderCacheVersion) == "" {
		return Config{}, fmt.Errorf("PROVIDER_CACHE_VERSION is required")
	}
	if cfg.ProviderCacheSearchTTL <= 0 {
		return Config{}, fmt.Errorf("PROVIDER_CACHE_SEARCH_TTL_MS must be > 0")
	}
	if cfg.ProviderCacheNegativeSearch <= 0 {
		return Config{}, fmt.Errorf("PROVIDER_CACHE_NEGATIVE_SEARCH_TTL_MS must be > 0")
	}
	if cfg.ProviderCacheDetailTTL <= 0 {
		return Config{}, fmt.Errorf("PROVIDER_CACHE_DETAIL_TTL_MS must be > 0")
	}
	if cfg.ProviderCacheNegativeDetail <= 0 {
		return Config{}, fmt.Errorf("PROVIDER_CACHE_NEGATIVE_DETAIL_TTL_MS must be > 0")
	}
	if cfg.ProviderCacheBarcodeTTL <= 0 {
		return Config{}, fmt.Errorf("PROVIDER_CACHE_BARCODE_TTL_MS must be > 0")
	}
	if cfg.ProviderCacheNegativeBarcode <= 0 {
		return Config{}, fmt.Errorf("PROVIDER_CACHE_NEGATIVE_BARCODE_TTL_MS must be > 0")
	}
	if cfg.MusicBrainzMinInterval < 0 {
		return Config{}, fmt.Errorf("MUSICBRAINZ_MIN_INTERVAL_MS must be >= 0")
	}
	if cfg.MusicBrainzRateLimitCooldown < 0 {
		return Config{}, fmt.Errorf("MUSICBRAINZ_RATE_LIMIT_COOLDOWN_MS must be >= 0")
	}
	if cfg.MusicBrainzRetryableCooldown < 0 {
		return Config{}, fmt.Errorf("MUSICBRAINZ_RETRYABLE_COOLDOWN_MS must be >= 0")
	}
	if cfg.BackupMaxUploadBytes <= 0 {
		return Config{}, fmt.Errorf("BACKUP_MAX_UPLOAD_BYTES must be > 0")
	}

	return cfg, nil
}

func (c Config) HTTPAddr() string {
	return fmt.Sprintf(":%d", c.Port)
}

func (c Config) UseEmbeddedAssets() bool {
	return strings.EqualFold(c.Env, "production")
}

func (c Config) MongoConnectTimeout() time.Duration {
	return 10 * time.Second
}

func (c Config) SessionTTL() time.Duration {
	return time.Duration(c.SessionTTLHours) * time.Hour
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}

func getEnvWithAliases(keys []string, fallback string) string {
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			return strings.TrimSpace(value)
		}
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvInt(key string, fallback int) int {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(getEnv(key, "")))
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func getEnvDurationMS(key string, fallback time.Duration) time.Duration {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return time.Duration(parsed) * time.Millisecond
}
