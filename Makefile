.PHONY: docker-up docker-down docker-prod-config docker-prod-up docker-prod-up-mongo docker-prod-down docker-prod-logs db-backup db-restore setup-image setup-packages

db-backup:
	./scripts/db-backup.sh

db-restore:
	@if [ -z "$(ARCHIVE)" ]; then echo "usage: make db-restore ARCHIVE=backups/mongodump-..."; exit 2; fi
	./scripts/db-restore.sh "$(ARCHIVE)"

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-prod-config:
	docker compose --env-file .env.prod -f docker-compose.prod.yml config

docker-prod-up:
	docker compose --env-file .env.prod -f docker-compose.prod.yml up --build -d

docker-prod-up-mongo:
	docker compose --env-file .env.prod -f docker-compose.prod.yml up --build -d

docker-prod-down:
	docker compose --env-file .env.prod -f docker-compose.prod.yml down

docker-prod-logs:
	docker compose --env-file .env.prod -f docker-compose.prod.yml logs -f

# Packaged installer (Docker provides the build toolchain).
setup-image:
	docker build --target setup -t media-library-manager-setup:local .

setup-packages:
	./scripts/package-setup.sh $(or $(VERSION),dev)
