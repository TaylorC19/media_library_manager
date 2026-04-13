SHELL := /bin/sh

.PHONY: dev-db-up dev-db-down dev-web

dev-db-up:
	docker compose -f docker-compose.go.yml up -d mongo

dev-db-down:
	docker compose -f docker-compose.go.yml stop mongo

dev-web:
	APP_ENV=development \
	PORT=8080 \
	DEFAULT_LOCALE=en \
	MONGODB_URI=mongodb://localhost:27017 \
	MONGODB_DATABASE=media_library \
	go run ./cmd/web
