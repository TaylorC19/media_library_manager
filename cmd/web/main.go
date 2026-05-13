package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"media_library_manager/internal/app"
	"media_library_manager/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("initialize app: %v", err)
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr(),
		Handler:           application.Router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("web server listening on %s", cfg.HTTPAddr())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-shutdownCtx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}
	if err := application.Close(ctx); err != nil {
		log.Printf("app shutdown error: %v", err)
	}
}
