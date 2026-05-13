package middleware

import (
	"context"
	"net/http"
	"strings"
)

type localeContextKey string

const LocaleContextKey localeContextKey = "locale"

var allowedLocales = map[string]struct{}{
	"en": {},
	"ja": {},
}

func Locale(defaultLocale string) func(http.Handler) http.Handler {
	defaultLocale = normalizeLocale(defaultLocale)
	if defaultLocale == "" {
		defaultLocale = "en"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			locale := detectLocaleFromPath(r.URL.Path)
			if !isAllowedLocale(locale) {
				locale = defaultLocale
			}

			ctx := context.WithValue(r.Context(), LocaleContextKey, locale)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func LocaleFromContext(ctx context.Context) string {
	locale, _ := ctx.Value(LocaleContextKey).(string)
	if !isAllowedLocale(locale) {
		return "en"
	}
	return locale
}

func detectLocaleFromPath(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 {
		return ""
	}
	return normalizeLocale(parts[0])
}

func normalizeLocale(locale string) string {
	return strings.ToLower(strings.TrimSpace(locale))
}

func isAllowedLocale(locale string) bool {
	_, ok := allowedLocales[normalizeLocale(locale)]
	return ok
}
