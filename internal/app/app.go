package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/mongo"

	"media_library_manager/internal/config"
	"media_library_manager/internal/db"
	"media_library_manager/internal/http/handlers"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/repository"
	authsvc "media_library_manager/internal/service/auth"
	"media_library_manager/internal/static"
	"media_library_manager/internal/views"
)

type App struct {
	Config   config.Config
	Router   http.Handler
	Mongo    *mongo.Client
	Database *mongo.Database
	Renderer *views.Renderer
	Auth     *authsvc.Service
}

func New(cfg config.Config) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.MongoConnectTimeout())
	defer cancel()

	mongoClient, err := db.ConnectMongo(ctx, cfg.MongoURI)
	if err != nil {
		return nil, err
	}

	renderer, err := views.NewRenderer(cfg.UseEmbeddedAssets())
	if err != nil {
		return nil, err
	}

	staticFS, err := static.FS(cfg.UseEmbeddedAssets())
	if err != nil {
		return nil, fmt.Errorf("load static fs: %w", err)
	}

	instance := &App{
		Config:   cfg,
		Mongo:    mongoClient,
		Database: mongoClient.Database(cfg.MongoDatabase),
		Renderer: renderer,
	}

	usersRepo := repository.NewUsersRepository(instance.Database)
	sessionsRepo := repository.NewSessionsRepository(instance.Database)
	if err := usersRepo.EnsureIndexes(ctx); err != nil {
		return nil, err
	}
	if err := sessionsRepo.EnsureIndexes(ctx); err != nil {
		return nil, err
	}
	instance.Auth = authsvc.NewService(usersRepo, sessionsRepo, cfg.SessionTTL())
	instance.Router = instance.newRouter(staticFS)

	return instance, nil
}

func (a *App) Close(ctx context.Context) error {
	if a.Mongo == nil {
		return nil
	}
	return a.Mongo.Disconnect(ctx)
}

func (a *App) newRouter(staticFS fs.FS) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recovery)
	r.Use(middleware.Logging)
	r.Use(middleware.Locale(a.Config.DefaultLocale))
	r.Use(middleware.WithAuthSession(a.Auth, a.Config.SessionCookieName))

	r.Get("/health", a.health)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/"+a.Config.DefaultLocale+"/", http.StatusFound)
	})

	authHandler := handlers.NewAuthHandler(a.Config, a.Renderer, a.Auth)
	protectedHandler := handlers.NewProtectedHandler(a.Renderer)

	r.Get("/{locale}/login", authHandler.LoginPage)
	r.Post("/{locale}/login", authHandler.LoginSubmit)
	r.Get("/{locale}/register", authHandler.RegisterPage)
	r.Post("/{locale}/register", authHandler.RegisterSubmit)
	r.Post("/{locale}/logout", authHandler.LogoutSubmit)

	r.With(middleware.RequireAuth).Get("/{locale}/", protectedHandler.Dashboard)

	return r
}

func (a *App) health(w http.ResponseWriter, r *http.Request) {
	status := map[string]any{
		"ok":       true,
		"service":  "media-library-manager",
		"database": a.Config.MongoDatabase,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
