include .env
export

.PHONY: up down migrate-up migrate-down sqlc-generate help

GOOSE = github.com/pressly/goose/v3/cmd/goose@latest
SQLC = github.com/sqlc-dev/sqlc/cmd/sqlc@latest

up:
	sudo docker compose --env-file .env up -d

down:
	sudo docker compose down -v

migrate-up:
	cd api && go run $(GOOSE) postgres "postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable" -dir ./db/migrations up

migrate-down:
	cd api && go run $(GOOSE) postgres "postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable" -dir ./db/migrations down

sqlc-generate:
	cd api && go run $(SQLC) generate

run:
	cd api && go run main.go

help:
	@echo "Available commands:"
	@echo "  up             - Start the Docker containers in detached mode"
	@echo "  down           - Stop and remove the Docker containers"
	@echo "  migrate-up     - Apply all pending database migrations"
	@echo "  migrate-down   - Roll back the last database migration"
	@echo "  sqlc-generate  - Generate Go code from SQL queries"
	@echo "  run            - Run the Go application"
