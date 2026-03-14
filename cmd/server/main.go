package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/yourusername/acgwarehouse-backend/internal/config"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if cfg.Database.Type == "sqlite" {
		db, err := sql.Open("sqlite3", cfg.Database.Path)
		if err != nil {
			log.Fatalf("failed to open sqlite database: %v", err)
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			log.Fatalf("failed to connect sqlite database: %v", err)
		}
	}

	fmt.Println("ACGWarehouse starting...")
}
