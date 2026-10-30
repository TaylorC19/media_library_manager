package middleware

import (
	"media_library_manager/internal/providerkeys"
	"net/http"
)

func WithProviderKeys(store *providerkeys.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if user := CurrentUser(r.Context()); user != nil {
				keys, err := store.Get(user.ID)
				if err != nil {
					http.Error(w, "Could not load provider settings", http.StatusInternalServerError)
					return
				}
				r = r.WithContext(providerkeys.WithKeys(r.Context(), keys))
			}
			next.ServeHTTP(w, r)
		})
	}
}
