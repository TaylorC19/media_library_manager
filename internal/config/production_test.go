package config

import (
	"encoding/base64"
	"testing"
)

func TestProductionFailsClosed(t *testing.T) {
	t.Setenv("SESSION_COOKIE_NAME", "__Host-mlm_session")
	t.Setenv("APP_ENV", "production")
	t.Setenv("VERCEL", "")
	t.Setenv("PUBLIC_ORIGIN", "https://library.example")
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017")
	t.Setenv("MONGODB_DATABASE", "release_fixture")
	t.Setenv("PROVIDER_KEYS_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("BACKUP_ENABLED", "false")
	t.Setenv("SESSION_COOKIE_SECURE", "true")
	t.Setenv("SIGNUP_ENABLED", "false")
	t.Setenv("ENSURE_INDEXES", "false")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SignupEnabled || cfg.BackupEnabled || cfg.EnsureIndexes || !cfg.SessionCookieSecure {
		t.Fatal("unsafe production defaults")
	}
	for _, tc := range []struct{ key, value string }{{"PUBLIC_ORIGIN", "http://library.example"}, {"PUBLIC_ORIGIN", "https://library.example/path"}, {"SESSION_COOKIE_NAME", "mlm_session"}, {"SESSION_COOKIE_SECURE", "false"}, {"PROVIDER_KEYS_ENCRYPTION_KEY", "invalid"}, {"BACKUP_ENABLED", "true"}, {"MONGODB_URI", ""}, {"MONGODB_DATABASE", ""}} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			t.Setenv("MONGODB_URL", "")
			if _, err := Load(); err == nil {
				t.Fatal("invalid production configuration accepted")
			}
		})
	}
}
