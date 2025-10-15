# Wave Simulator

This repository contains the source code for the Wave Simulator project.

## Quick Start

This guide will walk you through setting up the development environment for the
Wave Simulator.

### Prerequisites

- Docker
- Docker Compose
- Go
- sqlc
- Goose

### 1. Set up environment variables

Copy the `.env.example` file to `.env` and fill in the required values.

```bash
cp .env.example .env
```

### 2. Start the services

Launch the Postgres and Redis containers using Docker Compose.

```bash
docker-compose up -d
```

### 3. Apply database migrations

Use Goose to apply the database migrations. You will need to have Go and Goose
installed on your local machine for this to work. Make sure you are in the `api`
directory.

```bash
cd api
goose -dir ./db/migrations up
```

### 4. Generate SQL code

Use sqlc to generate Go code from your SQL queries.

```bash
cd api
sqlc generate
```
