.PHONY: build run test fmt fmt-check vet lint check clean setup db-init db-migrate import import-batch normalize dev smoke help

# Default target
help:
	@echo "Drift FM - Development Commands"
	@echo ""
	@echo "Development:"
	@echo "  make build          Build the server binary"
	@echo "  make run            Run the server (localhost:8080)"
	@echo "  make test           Run all tests"
	@echo "  make clean          Remove build artifacts"
	@echo ""
	@echo "Code Quality:"
	@echo "  make check          Full quality gate (fmt, vet, lint, test)"
	@echo "  make fmt            Format code (gofmt + goimports)"
	@echo "  make fmt-check      Check formatting (non-modifying)"
	@echo "  make vet            Run go vet"
	@echo "  make lint           Run linter"
	@echo ""
	@echo "Setup:"
	@echo "  make setup          Create data/ and audio/ directories"
	@echo "  make db-init        Initialize SQLite database"
	@echo "  make db-migrate     Run pending migrations"
	@echo "  make dev            Run with hot reload (requires: go install github.com/air-verse/air@latest)"
	@echo "  make smoke          Build, init DB, start server, test /health and /api/moods"
	@echo ""
	@echo "Audio:"
	@echo "  make import FILE=<path> MOOD=focus       Import single track"
	@echo "  make import-batch ARGS=<dir>             Import directory (interactive)"
	@echo "  make normalize FILE=<path>           Normalize audio file"

# Build
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	@echo "Building server..."
	@mkdir -p bin
	go build -ldflags "-X main.version=$(VERSION)" -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	@go test ./...

clean:
	rm -rf bin/ coverage.out coverage.html

# ═══════════════════════════════════════════════════════════════════════════
# Code Quality
# ═══════════════════════════════════════════════════════════════════════════

fmt:
	@gofmt -w .
	@goimports -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

vet:
	@go vet ./...

lint:
	@golangci-lint run

check: fmt-check vet lint test
	@echo "All checks passed"

dev:
	@command -v air >/dev/null 2>&1 || { echo "air not found. Install: go install github.com/air-verse/air@latest"; exit 1; }
	air

smoke: build db-init
	@echo "Starting smoke test..."
	@bin/server & SERVER_PID=$$!; \
	sleep 2; \
	FAIL=0; \
	curl -sf http://localhost:8080/health > /dev/null || { echo "FAIL: /health"; FAIL=1; }; \
	curl -sf http://localhost:8080/api/moods > /dev/null || { echo "FAIL: /api/moods"; FAIL=1; }; \
	kill $$SERVER_PID 2>/dev/null; wait $$SERVER_PID 2>/dev/null; \
	if [ $$FAIL -eq 0 ]; then echo "Smoke test passed"; else echo "Smoke test FAILED"; exit 1; fi

# ═══════════════════════════════════════════════════════════════════════════
# Database
# ═══════════════════════════════════════════════════════════════════════════

setup:
	@mkdir -p data audio/tracks
	@echo "Created data/ and audio/tracks/ directories"

db-init: setup
	@echo "Initializing database..."
	sqlite3 data/inventory.db < scripts/migrations/schema.sql
	@echo "Database initialized at data/inventory.db"

db-migrate:
	@echo "Running database migrations..."
	./scripts/migrate.sh
	@echo "Migrations complete"

# ═══════════════════════════════════════════════════════════════════════════
# Audio
# ═══════════════════════════════════════════════════════════════════════════

normalize:
ifndef FILE
	$(error FILE is required. Usage: make normalize FILE=path/to/audio.mp3)
endif
	./scripts/normalize.sh $(FILE)

import:
ifndef FILE
	$(error FILE is required. Usage: make import FILE=path/to/audio.mp3 MOOD=focus)
endif
	./scripts/import-track.sh $(FILE) --mood $(or $(MOOD),focus)

import-batch:
	./scripts/import-tracks.sh $(ARGS)
