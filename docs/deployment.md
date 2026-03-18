# ACGWarehouse Deployment Guide

**Phase 6 - Single-Machine Docker Compose Deployment**

This guide covers deploying ACGWarehouse using Docker Compose with SQLite as the embedded database.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Directory Structure](#directory-structure)
3. [Quick Start](#quick-start)
4. [Configuration](#configuration)
5. [Health Checks](#health-checks)
6. [Admin Dashboard](#admin-dashboard)
7. [Backup & Restore](#backup--restore)
8. [Upgrades](#upgrades)
9. [Troubleshooting](#troubleshooting)

---

## Prerequisites

- Docker Engine 20.10+
- Docker Compose v2.0+
- ~2GB available disk space (for database and image library)

## Directory Structure

```
acgwarehouse/
├── data/                    # SQLite database (auto-created)
│   └── acgwarehouse.db
├── library/                # Image library directory (auto-created)
│   └── [your images]
├── deploy/
│   └── config/
│       ├── config.example.yaml   # Template (do not edit)
│       └── config.yaml            # Your configuration
├── docker-compose.yml
├── Dockerfile
└── web/                    # Static assets (admin dashboard)
    └── admin/
```

**Important**: Create the `data` and `library` directories before first run:

```bash
mkdir -p data library deploy/config
```

## Quick Start

### 1. Clone and Setup

```bash
git clone <repository>
cd acgwarehouse-backend
```

### 2. Configure

Copy the example configuration and edit with your settings:

```bash
cp deploy/config/config.example.yaml deploy/config/config.yaml
```

Edit `deploy/config/config.yaml` with your API key:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

ai:
  provider: "qwen"  # or "doubao"
  api_key: "your-api-key-here"  # Replace with your key

scan:
  roots:
    - "/library"  # Your image library path
```

### 3. Start Services

```bash
docker compose up -d
```

### 4. Verify Health

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{"status":"healthy","timestamp":"2026-03-18T12:00:00Z"}
```

---

## Configuration

### Environment Variables

You can override settings via environment variables in `docker-compose.yml` or `.env`:

| Variable | Default | Description |
|----------|---------|-------------|
| SERVER_HOST | 0.0.0.0 | Server listen address |
| SERVER_PORT | 8080 | Server port |
| DATABASE_TYPE | sqlite | Database type (sqlite only in Phase 6) |
| DATABASE_PATH | /data/acgwarehouse.db | SQLite database path |
| AI_API_KEY | - | AI service API key |
| TZ | Asia/Shanghai | Timezone |

### Directory Mounts

The docker-compose.yml uses host bind mounts:

| Container Path | Host Path | Description |
|---------------|-----------|-------------|
| `/app/config.yaml` | `./deploy/config/config.yaml` | Configuration (read-only) |
| `/data` | `./data` | SQLite database (persistent) |
| `/library` | `./library` | Image library (read-only) |

**Note**: Mount paths use `:ro` (read-only) for config and library to prevent accidental modification inside the container.

---

## Health Checks

### Automatic Health Check

The Docker Compose configuration includes a healthcheck:

```yaml
healthcheck:
  test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
  interval: 30s
  timeout: 3s
  retries: 3
  start_period: 10s
```

### Manual Health Check

```bash
# Check container health
docker ps

# Check health endpoint
curl http://localhost:8080/health

# Check admin summary
curl http://localhost:8080/admin/api/summary
```

---

## Admin Dashboard

Access the admin dashboard at:

```
http://localhost:8080/admin
```

### Features

- **Service Status**: Health check and environment info
- **Task Queue**: Queue statistics (total, ready, running, finished, failed)
- **Library Scale**: Image, tag, and collection counts
- **Configuration**: AI key, storage, and admin username status
- **Recent Jobs**: Table showing recent job details
- **Recent Errors**: List of failed jobs with error messages
- **Actions**: Pause/Resume queue, Retry failed jobs, Trigger scan

### API Endpoints

| Endpoint | Description |
|----------|-------------|
| `/admin/api/summary` | Service summary statistics |
| `/admin/api/jobs` | Recent job list |
| `/admin/api/actions/jobs/pause` | Pause job queue |
| `/admin/api/actions/jobs/resume` | Resume job queue |
| `/admin/api/actions/jobs/retry-failed` | Retry failed jobs |
| `/admin/api/actions/scan` | Trigger library scan |

---

## Backup & Restore

### Backup Database

```bash
# Stop container first
docker compose stop

# Copy database file
cp data/acgwarehouse.db data/acgwarehouse-backup-$(date +%Y%m%d).db

# Restart container
docker compose start
```

### Backup Library

```bash
# Using tar
tar -czvf library-backup-$(date +%Y%m%d).tar.gz library/

# Or rsync
rsync -av library/ backup/library/
```

### Restore

```bash
# Stop container
docker compose stop

# Restore database
cp data/acgwarehouse-backup-20260318.db data/acgwarehouse.db

# Restore library
tar -xzvf library-backup-20260318.tar.gz

# Start container
docker compose start
```

---

## Upgrades

### Minor Updates (Same Version)

```bash
docker compose pull
docker compose up -d
```

### Major Updates (Rebuild Required)

```bash
# Backup first (see Backup & Restore section)
docker compose down

# Pull new code
git pull origin master

# Rebuild
docker compose build --no-cache

# Restore config and start
docker compose up -d
```

---

## PostgreSQL Note

> **PostgreSQL deployment is out of scope for Phase 6.**
> 
> This release uses SQLite-only deployment for simplicity. The configuration schema supports future PostgreSQL migration, but the migration tooling is not included in this release.

---

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker compose logs app

# Common issues:
# - Port 8080 already in use: Change port in docker-compose.yml
# - Permission denied: Check data/ and library/ directory permissions
```

### Database Issues

```bash
# Check database file exists
ls -la data/

# Verify database integrity
docker compose exec app sqlite3 /data/acgwarehouse.db "PRAGMA integrity_check;"
```

### Performance Issues

```bash
# Check container resource usage
docker stats

# Increase resources in docker-compose.yml if needed
```

### Reset to Clean State

```bash
# WARNING: This deletes all data
docker compose down
rm -rf data/*
rm -rf library/*
docker compose up -d
```

---

## Next Steps

1. Place your images in the `./library` directory
2. Access admin dashboard at http://localhost:8080/admin
3. Trigger a scan from the admin dashboard or via API
4. Access Flutter app at http://localhost:8080

---

*For performance benchmark results, see [performance-report.md](performance-report.md)*