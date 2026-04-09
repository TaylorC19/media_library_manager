package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env                 string
	Port                int
	MongoURI            string
	MongoDatabase       string
	DefaultLocale       string
	SessionCookieName   string
	SessionTTLHours     int
	SessionCookieSecure bool
}

func Load() (Config, error) {
	cfg := Config{
		Env:                 getEnv("APP_ENV", "development"),
		Port:                getEnvInt("PORT", 8080),
		MongoURI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		MongoDatabase:       getEnv("MONGODB_DATABASE", "media_library"),
		DefaultLocale:       getEnv("DEFAULT_LOCALE", "en"),
		SessionCookieName:   getEnv("SESSION_COOKIE_NAME", "mlm_session"),
		SessionTTLHours:     getEnvInt("SESSION_TTL_HOURS", 24*14),
		SessionCookieSecure: getEnvBool("SESSION_COOKIE_SECURE", false),
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
