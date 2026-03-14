module github.com/yourusername/acgwarehouse-backend

// Phase 01-01 foundation module baseline.
// Keep explicit core dependencies for upcoming API and data layers.
// SQLite path uses pure-Go driver to avoid CGO.
// PostgreSQL dependency is preloaded for dual-database evolution.

go 1.25.0

require (
	github.com/evanoberholster/imagemeta v0.3.1
	github.com/gin-gonic/gin v1.11.0
	github.com/jackc/pgx/v5 v5.7.6
	github.com/ncruces/go-sqlite3 v0.32.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/tetratelabs/wazero v1.11.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)
