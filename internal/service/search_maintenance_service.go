package service

import (
	"context"
	"database/sql"

	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

type SearchMaintenanceService struct {
	db *sql.DB
}

func NewSearchMaintenanceService(db *sql.DB) *SearchMaintenanceService {
	return &SearchMaintenanceService{db: db}
}

func (s *SearchMaintenanceService) RebuildFTSIndex(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	done := make(chan error, 1)
	go func() {
		done <- repository.RebuildFTSIndex(s.db)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}
