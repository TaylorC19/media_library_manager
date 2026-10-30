package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
	domainauth "media_library_manager/internal/domain/auth"
	"media_library_manager/internal/providerkeys"
)

func TestProviderKeyMiddlewareScopesCredentialsToSignedInAccount(t *testing.T) {
	store := providerkeys.NewStore(t.TempDir())
	alice, bob := primitive.NewObjectID(), primitive.NewObjectID()
	if err := store.Update(alice, providerkeys.Keys{"tmdb": "personal"}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []primitive.ObjectID{alice, bob} {
		r := httptest.NewRequest(http.MethodGet, "/en/search", nil)
		r = r.WithContext(context.WithValue(r.Context(), authUserKey, &domainauth.User{ID: id}))
		handler := RequireAuth(WithProviderKeys(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			want := "server"
			if id == alice {
				want = "personal"
			}
			if providerkeys.Resolve(r.Context(), "tmdb", "server") != want {
				t.Error("wrong account's credentials selected")
			}
		})))
		handler.ServeHTTP(httptest.NewRecorder(), r)
	}
}
