# Architecture Research

**Domain:** Image Library Management System (二次元图片库)
**Researched:** 2026-03-14
**Confidence:** HIGH

## Summary

Image library/gallery management systems follow a traditional **client-server architecture** with clear separation between:
1. **Ingestion Layer** — File scanning and import
2. **Processing Layer** — Thumbnails, AI analysis, deduplication
3. **Storage Layer** — File system for images, database for metadata
4. **API Layer** — RESTful endpoints for client access
5. **Client Layer** — Flutter frontend with offline-first capabilities

This architecture aligns with production systems like [Immich](https://immich.app/docs/developer/architecture) and [PhotoPrism](https://www.photoprism.app/kb/architecture).

---

## Standard Architecture

### System Overview

```
+-------------------------------------------------------------------------+
|                          CLIENT LAYER (Flutter)                          |
+-------------------------------------------------------------------------+
|  +--------------+  +--------------+  +--------------+  +-------------+  |
|  |   Gallery    |  |    Tags      |  | Collections  |  |   Search    |  |
|  |  (瀑布流/Grid)|  |   (筛选器)    |  |  (收藏夹管理) |  | (以图搜图)   |  |
|  +------+-------+  +------+-------+  +------+-------+  +------+------+  |
+--------+----------------+----------------+---------------+--------------+
|                           State Management                              |
|                    (Riverpod / BLoC / Provider)                         |
+-------------------------------------------------------------------------+
|                          Local Database (SQLite)                        |
|                    (Offline-first, sync with server)                    |
+--------------------------------+----------------------------------------+
                                 | HTTPS/REST
                                 v
+-------------------------------------------------------------------------+
|                           API GATEWAY (Go)                               |
+-------------------------------------------------------------------------+
|  +--------------+  +--------------+  +--------------+  +-------------+  |
|  | Image REST   |  |   Tag REST   |  |Collection    |  |  Search     |  |
|  |  Controller  |  |  Controller  |  |  Controller  |  | Controller  |  |
|  +------+-------+  +------+-------+  +------+-------+  +------+------+  |
+--------+----------------+----------------+---------------+--------------+
|                              SERVICE LAYER                               |
|         (Business Logic: Validation, Processing, Orchestration)         |
+-------------------------------------------------------------------------+
|                             REPOSITORY LAYER                             |
|              (Data Access: SQL Queries, File Operations)                |
+-------------------------------------------------------------------------+
|  +-----------------------------------------------------------------+    |
|  |                    BACKGROUND WORKER                             |    |
|  |   +----------+ +----------+ +----------+ +----------+          |    |
|  |   | Scanner  | |Thumbnail | |  AI      | | Duplicate|          |    |
|  |   | Service  | |Generator | | Service  | | Detector |          |    |
|  |   +----------+ +----------+ +----------+ +----------+          |    |
|  +-----------------------------------------------------------------+    |
+-------------------------------------------------------------------------+
                                 |
                                 v
+-------------------------------------------------------------------------+
|                           DATA LAYER                                     |
|  +-----------------+  +-----------------+  +-------------------------+ |
|  | SQLite/Postgres |  |  File System    |  |     AI API (External)   | |
|  |   (Metadata)    |  |  (Images/Thumbs)|  |  (角色识别/标签生成)      | |
|  +-----------------+  +-----------------+  +-------------------------+ |
+-------------------------------------------------------------------------+
```

---

## Component Responsibilities

### Backend (Go)

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| **Controller/Handler** | HTTP request handling, input validation, response formatting | Gin/Fiber/Echo routers with JSON handlers |
| **Service** | Business logic, orchestration between repositories | Structs with injected repository interfaces |
| **Repository** | Data access abstraction, SQL queries | GORM/sqlx with interface-based design |
| **Background Worker** | Async processing: scanning, thumbnails, AI calls | go-co-op/gocron or custom worker pool |
| **File Store** | Image file operations, thumbnail generation | Standard library + imaging/disintegration |
| **AI Client** | External API integration for recognition/tagging | HTTP client with retry logic |

### Frontend (Flutter)

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| **UI Layer** | Widgets, layouts, animations | Stateless/Stateful widgets |
| **State Management** | Business logic, data flow | Riverpod (recommended) or BLoC |
| **Repository** | API calls, local DB operations | Dio for HTTP, sqflite for local |
| **Models** | Data classes, serialization | freezed/json_serializable |
| **Local DB** | Offline storage, cache | SQLite via sqflite |

---

## Recommended Project Structure

### Go Backend Structure

```
acgwarehouse-backend/
├── cmd/
│   └── server/           # Application entrypoints
│       └── main.go       # HTTP server startup
├── internal/
│   ├── config/           # Configuration loading
│   │   └── config.go
│   ├── domain/           # Business entities (core models)
│   │   ├── image.go      # Image entity
│   │   ├── tag.go        # Tag entity
│   │   └── collection.go # Collection entity
│   ├── handler/          # HTTP handlers (controllers)
│   │   ├── image_handler.go
│   │   ├── tag_handler.go
│   │   └── collection_handler.go
│   ├── service/          # Business logic layer
│   │   ├── image_service.go
│   │   ├── scanner_service.go    # Directory scanning
│   │   ├── thumbnail_service.go  # Thumbnail generation
│   │   └── ai_service.go         # AI integration
│   ├── repository/       # Data access layer
│   │   ├── image_repository.go
│   │   ├── tag_repository.go
│   │   └── collection_repository.go
│   ├── worker/           # Background job processing
│   │   ├── scanner.go
│   │   ├── thumbnailer.go
│   │   └── ai_processor.go
│   ├── pkg/              # Shared utilities (optional)
│   │   ├── database/
│   │   ├── storage/
│   │   └── api/
│   └── middleware/       # HTTP middleware
│       ├── auth.go
│       ├── cors.go
│       └── logging.go
├── migrations/           # Database migrations
├── uploads/              # Image storage (gitignored)
└── go.mod
```

**Rationale:**
- **`internal/`:** Go convention for private code (cannot be imported externally)
- **`cmd/server/`:** Clean separation of entrypoints, supports multiple binaries later
- **`domain/`:** Core business entities, independent of storage/transport
- **`handler/`:** HTTP-specific code, thin layer delegating to services
- **`service/`:** Business logic orchestration, testable without HTTP
- **`repository/`:** Data access abstraction, enables testing with mocks
- **`worker/`:** Background processing separate from API handlers

### Flutter Frontend Structure

```
acgwarehouse-app/
├── lib/
│   ├── main.dart                 # App entrypoint
│   ├── app.dart                  # MaterialApp configuration
│   ├── config/                   # App configuration
│   │   ├── routes.dart
│   │   └── theme.dart
│   ├── core/                     # Shared utilities
│   │   ├── constants/
│   │   ├── utils/
│   │   └── widgets/
│   ├── data/                     # Data layer
│   │   ├── models/               # Entity classes
│   │   │   ├── image_model.dart
│   │   │   ├── tag_model.dart
│   │   │   └── collection_model.dart
│   │   ├── repositories/         # Data access
│   │   │   ├── image_repository.dart
│   │   │   └── local_database.dart
│   │   └── services/             # API clients
│   │       └── api_service.dart
│   ├── presentation/             # UI layer
│   │   ├── gallery/              # Gallery feature
│   │   │   ├── bloc/             # State management (optional)
│   │   │   ├── widgets/
│   │   │   │   ├── image_grid.dart      # Grid view
│   │   │   │   ├── waterfall_view.dart  # Waterfall
│   │   │   │   └── image_card.dart
│   │   │   └── pages/
│   │   │       └── gallery_page.dart
│   │   ├── tags/                 # Tag management
│   │   │   ├── widgets/
│   │   │   │   ├── tag_filter.dart      # Tag filter
│   │   │   │   └── tag_cloud.dart
│   │   │   └── pages/
│   │   │       └── tags_page.dart
│   │   ├── collections/          # Collection management
│   │   │   ├── widgets/
│   │   │   └── pages/
│   │   └── search/               # Search feature
│   │       ├── widgets/
│   │       │   └── image_search.dart    # Image search
│   │       └── pages/
│   └── providers.dart            # Riverpod providers
├── assets/
│   ├── images/
│   └── fonts/
└── pubspec.yaml
```

**Rationale:**
- **Feature-based organization:** Each feature (gallery, tags, collections) is self-contained
- **`data/`:** Centralized data layer with models, repositories, and API services
- **`presentation/`:** UI code separated from business logic
- **State management at feature level:** Either BLoC per feature or Riverpod providers

---

## Data Flow

### Image Ingestion Flow

```
[File System]
     |
     v
[Scanner Worker] -> Detect new/modified files
     |
     v
[Thumbnail Generator] -> Create preview/thumbnail
     |
     v
[AI Service] -> Character recognition, tag generation
     |
     v
[Database] -> Store metadata, tags, paths
     |
     v
[API] -> Notify clients of new images
     |
     v
[Flutter App] -> Update gallery view
```

### Gallery Browse Flow

```
[User Opens Gallery]
     |
     v
[Flutter: Check Local DB] -> Display cached images
     | (async)
     v
[API: GET /images?page=N] -> Fetch from server
     |
     v
[Go: Repository Layer] -> Query DB
     |
     v
[Flutter: Update Local DB] -> Cache results
     |
     v
[Flutter: Update UI] -> Display with animations
```

### Tag Filter Flow

```
[User Selects Tags]
     |
     v
[Flutter: Update Filter State] (Riverpod/BLoC)
     |
     v
[API: GET /images?tags=tag1,tag2] -> Query with filters
     |
     v
[Go: Service Layer] -> Build dynamic SQL
     |
     v
[Go: Repository] -> Execute query
     |
     v
[Flutter: Display Results] -> Update grid/waterfall
```

---

## API Design Patterns for Image Management

### RESTful Endpoints

| Resource | Method | Endpoint | Description |
|----------|--------|----------|-------------|
| Image | GET | /api/v1/images | List with pagination, filtering |
| Image | GET | /api/v1/images/{id} | Get specific image |
| Image | POST | /api/v1/images | Upload new image |
| Image | DELETE | /api/v1/images/{id} | Delete image |
| Tag | GET | /api/v1/tags | List all tags |
| Tag | GET | /api/v1/tags/{id}/images | Images by tag |
| Collection | GET | /api/v1/collections | List collections |
| Collection | POST | /api/v1/collections/{id}/images | Add to collection |
| Search | POST | /api/v1/search/similar | Image similarity search |
| Scan | POST | /api/v1/scan | Trigger directory scan |

### Key Patterns

**Pagination:**
```go
// Request: GET /images?page=1&limit=50
// Response:
{
  "data": [...],
  "pagination": {
    "current_page": 1,
    "total_pages": 20,
    "total_items": 1000,
    "items_per_page": 50
  }
}
```

**Filtering:**
```go
// Request: GET /images?tags=anime,sakura&character=rem&sort=date_desc
```

**Batch Operations:**
```go
// POST /images/batch
{
  "operation": "delete",
  "image_ids": [1, 2, 3, 4, 5]
}
```

---

## Suggested Build Order (Dependencies)

### Phase 1: Foundation (Weeks 1-2)
**Dependencies:** None

1. **Go Backend Skeleton**
   - Project structure setup
   - Database connection (SQLite first)
   - Basic HTTP server with Gin/Fiber
   - Health check endpoint

2. **Database Schema**
   - Images table (id, path, hash, width, height, created_at)
   - Tags table (id, name, type)
   - Image_Tags junction table
   - Collections tables

3. **Flutter App Skeleton**
   - Project structure
   - Basic navigation
   - HTTP client setup

### Phase 2: Core Image Operations (Weeks 3-4)
**Dependencies:** Phase 1

1. **Image Upload API**
   - POST /images endpoint
   - File upload handling
   - Basic validation

2. **Image Storage Service**
   - File system operations
   - Directory structure: /uploads/{year}/{month}/{hash}.ext
   - Duplicate detection via hash

3. **Flutter Gallery View**
   - Grid layout implementation
   - Image lazy loading
   - Basic pagination

### Phase 3: Image Processing (Weeks 5-6)
**Dependencies:** Phase 2

1. **Thumbnail Generation**
   - Background worker setup
   - Thumbnail generation service
   - Multiple sizes (small, medium, large)

2. **Directory Scanner**
   - File system watcher or periodic scanner
   - Auto-import from configured folders

### Phase 4: AI Integration (Weeks 7-8)
**Dependencies:** Phase 3

1. **AI Service Client**
   - External API integration
   - Character recognition
   - Tag generation

2. **Tag System**
   - Tag CRUD APIs
   - Auto-tagging on import
   - Flutter tag display

### Phase 5: Advanced Features (Weeks 9-10)
**Dependencies:** Phase 4

1. **Search & Filter**
   - Tag filtering API
   - Full-text search
   - Flutter filter UI

2. **Collections**
   - Collection CRUD
   - Add/remove images

3. **Similar Image Detection**
   - Perceptual hashing
   - Similarity search endpoint

### Phase 6: Polish & PostgreSQL (Weeks 11-12)
**Dependencies:** Phase 5

1. **PostgreSQL Support**
   - Migration from SQLite
   - Connection pooling

2. **Offline-First Sync**
   - Local SQLite in Flutter
   - Sync mechanism

3. **Performance Optimization**
   - API response caching
   - Image lazy loading optimization

---

## Component Boundaries

### Backend Internal Boundaries

```
Handler (HTTP) --------> Service --------> Repository --------> Database
     |                      |               |                |
     |                      |               |                |
     v                      v               v                v
   Transport           Business         Data Access      Persistence
   Layer               Logic            Abstraction       Layer
   (Gin/Fiber)         Layer            (GORM/sqlx)      (SQLite/PG)
```

**Rules:**
- Handlers never call repositories directly - always through services
- Services contain business logic but no HTTP or SQL specifics
- Repositories handle all database interactions
- Domain models are shared across layers

### Frontend-Backend Boundary

```
Flutter App ---HTTP/REST---> Go API
     |                           |
     |                           |
     v                           v
  Local DB                    Service Layer
 (Offline)                   (Business Logic)
```

**Communication:**
- RESTful JSON API
- Standard HTTP methods (GET, POST, PUT, DELETE)
- JWT or session-based authentication (if needed)
- File uploads via multipart/form-data

### External Service Boundary (AI)

```
Go Backend ---HTTP/REST---> AI API (External)
     |                           |
     |                           |
     v                           v
  Async Queue               Rate Limiting
  (Background)              Retry Logic
```

**Considerations:**
- AI calls are slow - always async
- Implement retry with exponential backoff
- Cache AI results to avoid duplicate calls
- Handle API limits gracefully

---

## Scaling Considerations

### Current Scale: Single User/Personal Use
**Architecture:** Monolithic, SQLite is fine
- Single binary deployment
- SQLite on local filesystem
- In-memory job queue
- Local file storage

### Future Scale: Multi-User/Family
**Architecture:** PostgreSQL, separate job worker
- PostgreSQL for concurrent access
- Redis for job queue
- Separate worker process
- Nginx reverse proxy

### Future Scale: Many Users
**Architecture:** Microservices consideration
- Split into API gateway + services
- Object storage (S3/MinIO) for images
- CDN for thumbnails
- Kubernetes deployment

**Note:** Start simple. Do not build for scale you do not have.

---

## Anti-Patterns

### Anti-Pattern 1: Storing Images in Database

**What people do:** Store image binary data in BLOB columns

**Why it is wrong:**
- Database bloat and slow queries
- Hard to serve images via CDN
- Backup/restore becomes painful

**Do this instead:**
- Store files on filesystem (or S3)
- Store only paths and metadata in DB
- Serve files directly via HTTP server or CDN

### Anti-Pattern 2: Synchronous AI Processing

**What people do:** Wait for AI API response during HTTP request

**Why it is wrong:**
- HTTP timeouts (30s+)
- Poor user experience
- Blocks API server

**Do this instead:**
- Queue AI jobs for background processing
- Return 202 Accepted immediately
- Use WebSocket or polling for completion status

### Anti-Pattern 3: Tight Coupling to AI Provider

**What people do:** Direct AI client calls scattered throughout code

**Why it is wrong:**
- Cannot swap providers easily
- Hard to mock for testing
- No central retry/caching logic

**Do this instead:**
- Create AI service interface
- Implement provider-specific clients behind interface
- Inject via dependency injection

### Anti-Pattern 4: No Local Cache in Flutter

**What people do:** Fetch from server on every screen open

**Why it is wrong:**
- Slow UI, poor UX
- Wastes bandwidth
- App unusable offline

**Do this instead:**
- Cache images locally (sqflite + cached_network_image)
- Implement offline-first architecture
- Sync changes when online

---

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| AI Recognition API | HTTP REST with retry | Use interface for swappability |
| Object Storage (S3) | SDK or REST API | Optional, for future scale |
| Reverse Image Search | HTTP API | Perceptual hashing service |

### Internal Communication

| Boundary | Communication | Notes |
|----------|---------------|-------|
| API <-> Worker | Job Queue (Redis/in-memory) | Async processing |
| Service <-> Repository | Interface methods | Testable, swappable |
| Flutter <-> Go API | HTTP REST + JSON | Standard RESTful |

---

## Technology Choices Summary

### Backend (Go)

| Component | Recommendation | Alternatives |
|-----------|----------------|--------------|
| Web Framework | Gin or Fiber | Echo, Chi |
| ORM | GORM or sqlx | Bun |
| Database | SQLite (dev), PostgreSQL (prod) | MySQL |
| Thumbnails | disintegration/imaging | nfnt/resize |
| Hashing | perceptual image hash | SHA256 |
| Job Queue | go-co-op/gocron | Asynq |
| Config | godotenv | viper |

### Frontend (Flutter)

| Component | Recommendation | Alternatives |
|-----------|----------------|--------------|
| State Management | Riverpod | BLoC, Provider |
| HTTP Client | Dio | http package |
| Local DB | sqflite | hive |
| Image Caching | cached_network_image | - |
| Gallery Layout | flutter_staggered_grid_view | masonry_grid |
| Photo View | photo_view | - |

---

## Sources

- [Immich Architecture Documentation](https://immich.app/docs/developer/architecture)
- [PhotoPrism Application Architecture](https://www.photoprism.app/kb/architecture)
- [Go Project Structure Best Practices](https://alnah.io/post/go-project-layout/)
- [Flutter App Architecture](https://docs.flutter.dev/app-architecture)
- [REST API Design Best Practices 2026](https://www.toolbrew.dev/blog/rest-api-design-2026)
- [Go Clean Architecture](https://backendbytes.com/articles/production-go-api-design)

---

## Quality Gate Checklist

- [x] Components clearly defined with boundaries
- [x] Data flow direction explicit
- [x] Build order implications noted
- [x] Go project structure best practices included
- [x] Flutter project structure best practices included
- [x] API design patterns documented

---

*Architecture research for: ACGWarehouse Image Library System*  
*Researched: 2026-03-14*
