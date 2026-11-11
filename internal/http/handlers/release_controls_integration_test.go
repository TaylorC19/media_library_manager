package handlers

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	domainauth "media_library_manager/internal/domain/auth"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/providerkeys"
	"media_library_manager/internal/repository"
	"media_library_manager/internal/service/adminusers"
)

func TestDurableProviderKeysEncryptedAndOwnerBound(t *testing.T) {
	db := regressionDatabase(t)
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	a, err := providerkeys.NewMongoStore(db, key)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := providerkeys.NewMongoStore(db, key)
	owner, other := primitive.NewObjectID(), primitive.NewObjectID()
	var wg sync.WaitGroup
	for p, v := range (providerkeys.Keys{"tmdb": "synthetic-tmdb", "rawg": "synthetic-rawg"}) {
		wg.Add(1)
		go func(p, v string) {
			defer wg.Done()
			if err := a.Update(owner, providerkeys.Keys{p: v}); err != nil {
				t.Error(err)
			}
		}(p, v)
	}
	wg.Wait()
	keys, err := b.Get(owner)
	if err != nil || keys["tmdb"] != "synthetic-tmdb" || keys["rawg"] != "synthetic-rawg" {
		t.Fatal("durable concurrent updates lost")
	}
	foreign, err := b.Get(other)
	if err != nil || len(foreign) != 0 {
		t.Fatal("foreign credentials returned")
	}
	var raw bson.M
	if err := db.Collection("personal_provider_keys").FindOne(context.Background(), bson.M{"_id": owner}).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	encoded, _ := bson.MarshalExtJSON(raw, false, false)
	if strings.Contains(string(encoded), "synthetic-") {
		t.Fatal("plaintext stored")
	}
	if err := b.Clear(owner, "tmdb"); err != nil {
		t.Fatal(err)
	}
	status, _ := a.Status(owner)
	if status["tmdb"] || !status["rawg"] {
		t.Fatal("clear affected other provider")
	}
	// Moving ciphertext between owner documents must fail authentication.
	raw["_id"] = other
	if _, err := db.Collection("personal_provider_keys").InsertOne(context.Background(), raw); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Get(other); err == nil {
		t.Fatal("ciphertext not owner bound")
	}
}
func TestDistributedBudgetsAreAtomic(t *testing.T) {
	db := regressionDatabase(t)
	a, b := middleware.NewBudget(db, false), middleware.NewBudget(db, false)
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			budget := a
			if i%2 == 0 {
				budget = b
			}
			ok, err := budget.Allow(context.Background(), "synthetic-account", 7, time.Hour)
			if err != nil {
				t.Error(err)
			}
			if ok {
				allowed.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if allowed.Load() != 7 {
		t.Fatalf("shared budget allowed %d requests", allowed.Load())
	}
}
func TestIndependentAdminTransactionsPreserveAnAdministrator(t *testing.T) {
	db := regressionDatabase(t)
	ctx := context.Background()
	var hello struct {
		SetName string `bson:"setName"`
	}
	_ = db.RunCommand(ctx, bson.M{"hello": 1}).Decode(&hello)
	if hello.SetName == "" {
		if os.Getenv("REQUIRE_MONGO_TESTS") == "1" {
			t.Fatal("replica set required")
		}
		t.Skip("requires a disposable MongoDB replica set")
	}
	a, b := repository.NewUsersRepository(db), repository.NewUsersRepository(db)
	a.RequireRoleTransactions()
	b.RequireRoleTransactions()
	first, second := &domainauth.User{Username: "fixture-admin-a", Role: "admin"}, &domainauth.User{Username: "fixture-admin-b", Role: "admin"}
	if err := a.Create(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := a.Create(ctx, second); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Collection("admin_guard").InsertOne(ctx, bson.M{"_id": "roles", "revision": 0}); err != nil {
		t.Fatal(err)
	}
	services := []*adminusers.Service{adminusers.NewService(a, nil), adminusers.NewService(b, nil)}
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() { <-start; results <- services[0].ChangeRole(ctx, first.ID, second.ID, "general") }()
	go func() { <-start; results <- services[1].ChangeRole(ctx, second.ID, first.ID, "general") }()
	close(start)
	one, two := <-results, <-results
	if !((one == nil && errors.Is(two, adminusers.ErrForbidden)) || (two == nil && errors.Is(one, adminusers.ErrForbidden))) {
		t.Fatalf("expected one authorized transition: %v / %v", one, two)
	}
	count, err := db.Collection("users").CountDocuments(ctx, bson.M{"role": "admin"})
	if err != nil || count != 1 {
		t.Fatal("administrator invariant failed")
	}
}

func TestDistributedProviderConcurrencyAndRelease(t *testing.T) {
	db := regressionDatabase(t)
	entered, release := make(chan struct{}, 8), make(chan struct{})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { entered <- struct{}{}; <-release; w.WriteHeader(204) })
	a, b := middleware.NewBudget(db, false).Wrap(next), middleware.NewBudget(db, false).Wrap(next)
	user := &domainauth.User{ID: primitive.NewObjectID(), Role: "general"}
	request := func() *http.Request {
		r := httptest.NewRequest("GET", "/en/search/group?q=fixture", nil)
		return r.WithContext(middleware.WithUser(r.Context(), user))
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			h := a
			if i%2 == 0 {
				h = b
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, request())
			if w.Code != 204 {
				t.Errorf("allowed request: %d", w.Code)
			}
		}(i)
	}
	defer close(release)
	for i := 0; i < 8; i++ {
		select {
		case <-entered:
		case <-time.After(10 * time.Second):
			t.Fatal("provider requests did not acquire leases")
		}
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, request())
	if w.Code != 429 {
		t.Fatalf("excess concurrent request accepted: %d", w.Code)
	}
	// Expiry recovers a crashed instance's slot; old token cleanup cannot remove a replacement.
	_, err := db.Collection("request_leases").UpdateOne(context.Background(), bson.M{"_id": user.ID.Hex() + ":0"}, bson.M{"$set": bson.M{"expiresAt": time.Now().Add(-time.Minute)}})
	if err != nil {
		t.Fatal(err)
	}
	quick := middleware.NewBudget(db, false).Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	w = httptest.NewRecorder()
	quick.ServeHTTP(w, request())
	if w.Code != 204 {
		t.Fatal("expired lease not recovered")
	}
	// Release active requests before database fixture cleanup.
	for i := 0; i < 8; i++ {
		release <- struct{}{}
	}
	wg.Wait()
	remaining, err := db.Collection("request_leases").CountDocuments(context.Background(), bson.M{})
	if err != nil || remaining != 0 {
		t.Fatal("leases not released")
	}
}

func TestSessionIndexMigrationPreservesExistingSessions(t *testing.T) {
	db := regressionDatabase(t)
	ctx := context.Background()
	if _, err := db.Collection("sessions").Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "expiresAt", Value: 1}}}); err != nil {
		t.Fatal(err)
	}
	id := primitive.NewObjectID()
	if _, err := db.Collection("sessions").InsertOne(ctx, bson.M{"_id": id, "tokenHash": "synthetic-hash", "expiresAt": time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := repository.NewSessionsRepository(db).EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	indexes, err := db.Collection("sessions").Indexes().List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer indexes.Close(ctx)
	found := false
	for indexes.Next(ctx) {
		var index struct {
			Name               string
			ExpireAfterSeconds *int64 `bson:"expireAfterSeconds"`
		}
		if err := indexes.Decode(&index); err != nil {
			t.Fatal(err)
		}
		if index.Name == "expiresAt_1" {
			found = index.ExpireAfterSeconds != nil && *index.ExpireAfterSeconds == 0
		}
	}
	if !found {
		t.Fatal("existing expiry index was not converted to TTL")
	}
	count, err := db.Collection("sessions").CountDocuments(ctx, bson.M{"_id": id})
	if err != nil || count != 1 {
		t.Fatal("migration changed an active session")
	}
}
