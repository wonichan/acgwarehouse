BIN_DIR := bin
SERVER_BIN := $(BIN_DIR)/server

.PHONY: build run test migrate-up migrate-down

build:
	go build -o $(SERVER_BIN) ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

migrate-up:
	mkdir -p ./data
	sqlite3 ./data/acgwarehouse.db < ./migrations/001_initial_schema.up.sql

migrate-down:
	mkdir -p ./data
	sqlite3 ./data/acgwarehouse.db < ./migrations/001_initial_schema.down.sql
