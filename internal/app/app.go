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
	"media_library_manager/internal/providerkeys"
	"media_library_manager/internal/providers/musicbrainz"
	"media_library_manager/internal/repository"
	"media_library_manager/internal/service/adminusers"
	authsvc "media_library_manager/internal/service/auth"
	barcodesvc "media_library_manager/internal/service/barcode"
	libsvc "media_library_manager/internal/service/library"
	mediasvc "media_library_manager/internal/service/media"
	"media_library_manager/internal/service/reliability"
	searchsvc "media_library_manager/internal/service/search"
	"media_library_manager/internal/static"
	"media_library_manager/internal/views"
)

type App struct {
	Config    config.Config
	Router    http.Handler
	Mongo     *mongo.Client
	Database  *mongo.Database
	Renderer  *views.Renderer
	UserAdmin *adminusers.Service
	Auth      *authsvc.Service
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
	mediaRepo := repository.NewMediaRecordsRepository(instance.Database)
	libEntriesRepo := repository.NewLibraryEntriesRepository(instance.Database)
	if err := mediaRepo.EnsureIndexes(ctx); err != nil {
		return nil, err
	}
	if err := libEntriesRepo.EnsureIndexes(ctx); err != nil {
		return nil, err
	}
	scanLogsRepo := repository.NewScanLogsRepository(instance.Database)
	if err := scanLogsRepo.EnsureIndexes(ctx); err != nil {
		return nil, err
	}
	providerCacheRepo := repository.NewProviderCacheRepository(instance.Database)
	if err := providerCacheRepo.EnsureIndexes(ctx); err != nil {
		return nil, err
	}
	reliabilitySvc := reliability.NewService(cfg)
	cache := reliability.NewCache(providerCacheRepo, reliabilitySvc)
	mbThrottle := musicbrainz.NewThrottle(reliabilitySvc.MusicBrainzThrottle)
	instance.Auth = authsvc.NewService(usersRepo, sessionsRepo, cfg.SessionTTL())
	instance.UserAdmin = adminusers.NewService(usersRepo, instance.Auth)
	librarySvc := libsvc.NewService(mediaRepo, libEntriesRepo)
	mediaSvc := mediasvc.NewService(cfg, mediaRepo, libEntriesRepo, cache, reliabilitySvc, mbThrottle)
	searchSvc := searchsvc.NewService(cfg, cache, reliabilitySvc, mbThrottle)
	barcodeSvc := barcodesvc.NewService(cfg, mediaRepo, libEntriesRepo, scanLogsRepo, cache, reliabilitySvc, mbThrottle)
	instance.Router = instance.newRouter(staticFS, librarySvc, mediaSvc, searchSvc, barcodeSvc, mbThrottle)

	return instance, nil
}

func (a *App) Close(ctx context.Context) error {
	if a.Mongo == nil {
		return nil
	}
	return a.Mongo.Disconnect(ctx)
}

func (a *App) newRouter(staticFS fs.FS, library *libsvc.Service, media *mediasvc.Service, search *searchsvc.Service, barcode *barcodesvc.Service, mbThrottle *musicbrainz.Throttle) http.Handler {
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
	libraryHandler := handlers.NewLibraryHandler(a.Renderer, library)
	searchHandler := handlers.NewSearchHandler(a.Renderer, search, library)
	mediaHandler := handlers.NewMediaHandler(a.Renderer, media)
	barcodeHandler := handlers.NewBarcodeHandler(barcode)
	keyStore := providerkeys.NewStore(a.Config.ProviderKeysDir)
	settingsHandler := handlers.NewSettingsHandler(a.Config, a.Renderer, keyStore, &providerkeys.Checker{MusicBrainzUA: a.Config.MusicBrainzUA, MusicBrainzThrottle: mbThrottle})
	backupHandler := handlers.NewBackupHandler(a.Config, a.Database)
	usersHandler := handlers.NewUsersHandler(a.Renderer, a.UserAdmin)
	userKeysHandler := handlers.NewUserKeysHandler(a.Renderer, adminusers.NewKeyService(a.UserAdmin, keyStore))

	r.Get("/{locale}/login", authHandler.LoginPage)
	r.Post("/{locale}/login", authHandler.LoginSubmit)
	r.Get("/{locale}/register", authHandler.RegisterPage)
	r.Post("/{locale}/register", authHandler.RegisterSubmit)
	r.Post("/{locale}/logout", authHandler.LogoutSubmit)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Get("/{locale}/", libraryHandler.LibraryPage)
		r.Get("/{locale}/catalog", libraryHandler.Catalog)
		r.Get("/{locale}/wishlist", libraryHandler.Wishlist)
		r.With(middleware.WithProviderKeys(keyStore)).Get("/{locale}/search", searchHandler.Page)
		r.With(middleware.WithProviderKeys(keyStore)).Get("/{locale}/search/group", searchHandler.Group)
		r.With(middleware.WithProviderKeys(keyStore)).Get("/{locale}/search/details", mediaHandler.Preview)
		r.Get("/{locale}/scan", handlers.LegacyScanPage)
		r.Get("/{locale}/settings", settingsHandler.Page)
		r.With(middleware.RequireAdmin).Get("/{locale}/admin/users", usersHandler.Page)
		r.With(middleware.RequireAdmin).Get("/{locale}/admin/users/{userId}/keys", userKeysHandler.Page)
		r.With(middleware.RequireAdmin).Get("/{locale}/admin/users/{userId}/keys/clear", userKeysHandler.Confirm)
		r.With(middleware.RequireAdmin).Post("/{locale}/admin/users/{userId}/keys/clear", userKeysHandler.Clear)
		r.With(middleware.RequireAdmin).Post("/{locale}/admin/users", usersHandler.Create)
		r.With(middleware.RequireAdmin).Post("/{locale}/admin/users/{userId}/role", usersHandler.ChangeRole)
		r.With(middleware.RequireWritable).Post("/{locale}/settings/providers", settingsHandler.ProviderKeysSubmit)
		r.Post("/{locale}/settings/providers/test", settingsHandler.ProviderKeyTest)
		r.Post("/settings/backup/export", backupHandler.ExportSubmit)
		r.Post("/settings/backup/import", backupHandler.ImportSubmit)
		r.With(middleware.RequireWritable, middleware.WithProviderKeys(keyStore)).Post("/media/import", mediaHandler.ImportSubmit)
		r.With(middleware.RequireWritable, middleware.WithProviderKeys(keyStore)).Post("/media/refresh/{mediaRecordId}", mediaHandler.RefreshSubmit)
		r.With(middleware.RequireWritable).Get("/{locale}/library/new", libraryHandler.NewForm)
		r.With(middleware.RequireWritable).Get("/{locale}/library/{entryId}/edit", libraryHandler.EditForm)
		r.Get("/{locale}/library/{entryId}", libraryHandler.Detail)
		r.With(middleware.RequireWritable).Post("/library", libraryHandler.CreateSubmit)
		r.With(middleware.RequireWritable).Post("/library/attach", libraryHandler.AttachSubmit)
		r.With(middleware.RequireWritable).Post("/library/{entryId}/update", libraryHandler.UpdateSubmit)
		r.With(middleware.RequireWritable).Post("/library/{entryId}/move-to-collection", libraryHandler.MoveToCollectionSubmit)
		r.With(middleware.RequireWritable).Post("/library/{entryId}/delete", libraryHandler.DeleteSubmit)
		r.With(middleware.WithProviderKeys(keyStore)).Post("/barcode/lookup", barcodeHandler.Lookup)
	})

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
