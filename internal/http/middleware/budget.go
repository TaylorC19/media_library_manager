package middleware

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Budget struct {
	db     *mongo.Database
	vercel bool
}

func NewBudget(db *mongo.Database, vercel bool) *Budget { return &Budget{db: db, vercel: vercel} }
func (b *Budget) Allow(ctx context.Context, key string, limit int, period time.Duration) (bool, error) {
	now := time.Now().UTC()
	bucket := now.UnixNano() / period.Nanoseconds()
	id := fmt.Sprintf("%x:%d", sha256.Sum256([]byte(key)), bucket)
	var result struct {
		Count int `bson:"count"`
	}
	err := b.db.Collection("request_budgets").FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$inc": bson.M{"count": 1}, "$setOnInsert": bson.M{"expiresAt": now.Add(2 * period)}}, options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&result)
	if mongo.IsDuplicateKeyError(err) {
		return b.Allow(ctx, key, limit, period)
	}
	return result.Count <= limit, err
}
func (b *Budget) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/static/") || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		// Vercel overwrites this header at its trusted ingress. Never trust ordinary
		// X-Forwarded-For or accept this header on standalone deployments.
		if b.vercel {
			ip = r.Header.Get("X-Vercel-Forwarded-For")
			if ip == "" || strings.Contains(ip, ",") {
				http.Error(w, "Missing client identity", 503)
				return
			}
		}
		type rule struct {
			key    string
			limit  int
			period time.Duration
		}
		rules := []rule{{"ip:" + ip, 600, 10 * time.Minute}}
		path := r.URL.Path
		if r.Method == "POST" && (strings.HasSuffix(path, "/login") || strings.HasSuffix(path, "/register")) {
			rules = append(rules, rule{"auth:" + ip, 20, 15 * time.Minute})
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Invalid form", 400)
				return
			}
			name := strings.ToLower(strings.TrimSpace(r.Form.Get("username")))
			if len(name) > 80 {
				http.Error(w, "Invalid credentials", 400)
				return
			}
			rules = append(rules, rule{"account-login:" + name, 30, 15 * time.Minute})
		}
		user := CurrentUser(r.Context())
		provider := strings.Contains(path, "/search") || strings.HasPrefix(path, "/media/") || path == "/barcode/lookup" || strings.HasSuffix(path, "/providers/test")
		if user != nil {
			id := user.ID.Hex()
			rules = append(rules, rule{"user:" + id, 600, 10 * time.Minute})
			if provider {
				rules = append(rules, rule{"provider:" + id, 60, time.Minute}, rule{"provider-day:" + id, 1000, 24 * time.Hour})
			}
			if r.Method == "POST" {
				rules = append(rules, rule{"writes:" + id, 500, 24 * time.Hour})
			}
		}
		for _, rule := range rules {
			ok, err := b.Allow(r.Context(), rule.key, rule.limit, rule.period)
			if err != nil {
				http.Error(w, "Request budget unavailable", 503)
				return
			}
			if !ok {
				w.Header().Set("Retry-After", strconv.Itoa(int(rule.period.Seconds())))
				http.Error(w, "Too many requests. Try again later.", 429)
				return
			}
		}
		if provider && user != nil {
			token := primitive.NewObjectID()
			var lease string
			now := time.Now().UTC()
			for slot := 0; slot < 8; slot++ {
				id := fmt.Sprintf("%s:%d", user.ID.Hex(), slot)
				_, err := b.db.Collection("request_leases").UpdateOne(r.Context(), bson.M{"_id": id, "expiresAt": bson.M{"$lt": now}}, bson.M{"$set": bson.M{"token": token, "expiresAt": now.Add(2 * time.Minute)}}, options.Update().SetUpsert(true))
				if err == nil {
					lease = id
					break
				}
				if !mongo.IsDuplicateKeyError(err) {
					http.Error(w, "Request budget unavailable", 503)
					return
				}
			}
			if lease == "" {
				w.Header().Set("Retry-After", "5")
				http.Error(w, "Too many concurrent requests", 429)
				return
			}
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_, _ = b.db.Collection("request_leases").DeleteOne(ctx, bson.M{"_id": lease, "token": token})
			}()
			ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
			defer cancel()
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}
