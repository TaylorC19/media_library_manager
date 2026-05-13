.PHONY: docker-up docker-down docker-prod-config docker-prod-up docker-prod-up-mongo docker-prod-down docker-prod-logs

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-prod-config:
	docker compose --env-file .env.prod -f docker-compose.prod.yml config

docker-prod-up:
	docker compose --env-file .env.prod -f docker-compose.prod.yml up --build -d

docker-prod-up-mongo:
	docker compose --env-file .env.prod -f docker-compose.prod.yml --profile mongo up --build -d

docker-prod-down:
	docker compose --env-file .env.prod -f docker-compose.prod.yml down

docker-prod-logs:
	docker compose --env-file .env.prod -f docker-compose.prod.yml logs -f
