BIN_DIR := bin
SERVER_BIN := $(BIN_DIR)/server

.PHONY: build run test migrate-up migrate-down docker-build docker-compose-up docker-compose-down docker-compose-logs compose-smoke package-windows-portable

build:
	go build -o $(SERVER_BIN) ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

migrate-up:
	mkdir -p ./data
	sqlite3 ./data/acgwarehouse.db < ./migrations/001_initial_schema.up.sql
	sqlite3 ./data/acgwarehouse.db < ./migrations/002_add_thumbnail_fields.up.sql

migrate-down:
	mkdir -p ./data
	sqlite3 ./data/acgwarehouse.db < ./migrations/002_add_thumbnail_fields.down.sql
	sqlite3 ./data/acgwarehouse.db < ./migrations/001_initial_schema.down.sql

# Docker commands
docker-build:
	docker build -t acgwarehouse:latest .

docker-compose-up:
	docker compose up -d

docker-compose-down:
	docker compose down

docker-compose-logs:
	docker compose logs -f

# Smoke test: verify compose config is valid
compose-smoke:
	docker compose config
	@echo "Compose configuration is valid"

# Quick deploy: setup config and start
deploy-setup:
	@if [ ! -f ./deploy/config/config.yaml ]; then \
		mkdir -p ./deploy/config; \
		cp ./deploy/config/config.example.yaml ./deploy/config/config.yaml; \
		echo "Created deploy/config/config.yaml from example"; \
		echo "Please edit deploy/config/config.yaml with your settings"; \
	else \
		echo "deploy/config/config.yaml already exists"; \
	fi
	mkdir -p ./data ./library

package-windows-portable:
	powershell -ExecutionPolicy Bypass -File deploy/windows/package.ps1
