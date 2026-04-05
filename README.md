# ACGWarehouse

二次元图片库管理与检索系统 - ACG Image Gallery Management System

## Overview

ACGWarehouse is a Go + Flutter application for managing and searching anime-style image collections. It features AI-powered auto-tagging, duplicate detection, and a modern Flutter web interface.

## Tech Stack

- **Backend**: Go 1.23 + Gin + SQLite
- **Frontend**: Flutter Web
- **AI**: OpenAI-compatible APIs (Qwen, Doubao)
- **Deployment**: Docker Compose

## Quick Start

powershell -ExecutionPolicy Bypass -File "deploy/windows/package.ps1" -SkipTests

### Prerequisites

- Docker Engine 20.10+
- Docker Compose v2.0+
- Go 1.23+ (for development)
- Flutter 3.x (for frontend development)

### Development Mode

```bash
# Backend
go run cmd/server/main.go

# Frontend (separate terminal)
cd flutter_app
flutter run -d chrome
```

### Production Deployment

See [Deployment Guide](docs/deployment.md) for complete instructions.

```bash
# 1. Setup directories
mkdir -p data library deploy/config

# 2. Copy and edit configuration
cp deploy/config/config.example.yaml deploy/config/config.yaml
# Edit config.yaml with your API keys

# 3. Start with Docker
docker compose up -d

# 4. Verify
curl http://localhost:8080/health
```

## Key Features

- **AI Auto-Tagging**: Automatic tag generation using AI models
- **Duplicate Detection**: SHA256 + pHash for exact and similar duplicates
- **Collections**: Organize images into collections
- **Search**: Multi-tag filtering with AND semantics
- **Pagination**: Infinite scroll for large galleries
- **Admin Dashboard**: Web-based operations monitoring

## Entry Points

| Service | URL | Description |
|---------|-----|-------------|
| Backend API | http://localhost:8080 | REST API |
| Health Check | http://localhost:8080/health | Service health |
| Admin Dashboard | http://localhost:8080/admin | Operations monitoring |
| Flutter App | http://localhost:8080 | Main gallery interface |

## Documentation

- [Deployment Guide](docs/deployment.md) - Production deployment with Docker
- [Performance Report](docs/performance-report.md) - Benchmark results and analysis
- [API Documentation](docs/api.md) - REST API reference (if available)

## Project Structure

```
acgwarehouse/
├── cmd/server/          # Main application entry
├── internal/
│   ├── domain/         # Domain models
│   ├── handler/        # HTTP handlers
│   ├── repository/    # Data access layer
│   └── service/       # Business logic
├── flutter_app/        # Flutter web frontend
├── test/
│   └── perf/          # Performance benchmarks
├── deploy/
│   └── config/        # Deployment configuration
├── docs/              # Documentation
├── docker-compose.yml # Docker Compose config
└── Dockerfile         # Container definition
```

## Requirements

This system implements the following requirements:

- **DEPL-01**: Docker Compose single-machine deployment
- **DEPL-02**: SQLite-only runtime path

## Benchmark

Run performance benchmarks:

```bash
go test ./test/perf/... -run ^$ -bench . -benchmem -count=1
```

See [performance-report.md](docs/performance-report.md) for detailed results.

## License

MIT License