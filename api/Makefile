# Variables
BINARY := wave-pool
BIN_DIR := bin
DB_FILE := wave-pool.db
MIGRATIONS_DIR := db/migrations

# Tools (assumes installed via `go install`)
GOOSE := goose
SQLC := sqlc

.PHONY: all build run test fmt lint tidy sqlc gen db-up db-down db-status migrate-create docker-build docker-run clean

all: build

build: | $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY) .

run: build
	./$(BIN_DIR)/$(BINARY)

test:
	go test -race -v ./...

fmt:
	gofmt -w .
	go mod tidy

lint:
	golangci-lint run

sqlc:
	$(SQLC) generate

# Run migrations up on local sqlite DB
# Usage: make db-up
# To specify a custom db file: make db-up DB_FILE=my.db
# To specify a different dir: make db-up MIGRATIONS_DIR=path

db-up:
	$(GOOSE) -dir $(MIGRATIONS_DIR) sqlite3 ./$(DB_FILE) up

db-down:
	$(GOOSE) -dir $(MIGRATIONS_DIR) sqlite3 ./$(DB_FILE) down

db-status:
	$(GOOSE) -dir $(MIGRATIONS_DIR) sqlite3 ./$(DB_FILE) status

migrate-create:
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=add_something"; exit 1; fi
	$(GOOSE) -dir $(MIGRATIONS_DIR) create $(name) sql

# Docker helpers

docker-build:
	docker build -t $(BINARY):dev .

docker-run:
	docker run --rm -p 8080:8080 -v $(PWD)/$(DB_FILE):/root/$(DB_FILE) $(BINARY):dev

clean:
	rm -rf $(BIN_DIR)
	rm -f $(DB_FILE)

$(BIN_DIR):
	mkdir -p $(BIN_DIR)
