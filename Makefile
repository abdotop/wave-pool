.PHONY: up down migrate-up migrate-down sqlc-generate help

up:
	docker-compose up -d

down:
	docker-compose down

migrate-up:
	cd api && goose -dir ./db/migrations up

migrate-down:
	cd api && goose -dir ./db/migrations down

sqlc-generate:
	cd api && sqlc generate

help:
	@echo "Available commands:"
	@echo "  up             - Start the Docker containers in detached mode"
	@echo "  down           - Stop and remove the Docker containers"
	@echo "  migrate-up     - Apply all pending database migrations"
	@echo "  migrate-down   - Roll back the last database migration"
	@echo "  sqlc-generate  - Generate Go code from SQL queries"
