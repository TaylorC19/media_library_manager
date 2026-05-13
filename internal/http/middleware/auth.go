package middleware

import (
	"context"
	"net/http"

	domainauth "media_library_manager/internal/domain/auth"
	authsvc "media_library_manager/internal/service/auth"
)

type authContextKey string

const (
	authUserKey    authContextKey = "auth.user"
	authSessionKey authContextKey = "auth.session"
)

func WithAuthSession(service *authsvc.Service, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			if err == nil && cookie.Value != "" {
				user, session, resolveErr := service.ResolveSession(r.Context(), cookie.Value)
				if resolveErr == nil && user != nil && session != nil {
					ctx := context.WithValue(r.Context(), authUserKey, user)
					ctx = context.WithValue(ctx, authSessionKey, session)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if CurrentUser(r.Context()) == nil {
			locale := LocaleFromContext(r.Context())
			http.Redirect(w, r, "/"+locale+"/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func CurrentUser(ctx context.Context) *domainauth.User {
	user, _ := ctx.Value(authUserKey).(*domainauth.User)
	return user
}

func CurrentSession(ctx context.Context) *domainauth.Session {
	session, _ := ctx.Value(authSessionKey).(*domainauth.Session)
	return session
}
