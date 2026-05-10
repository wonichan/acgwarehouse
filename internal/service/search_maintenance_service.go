package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

type SearchMaintenanceService struct {
	rebuild func(context.Context) error
}

func NewSearchMaintenanceService(db *sql.DB) *SearchMaintenanceService {
	return &SearchMaintenanceService{
		rebuild: func(ctx context.Context) error {
			done := make(chan error, 1)
			go func() {
				done <- repository.RebuildFTSIndex(db)
			}()

			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-done:
				return err
			}
		},
	}
}

func NewD1SearchMaintenanceService(client *d1client.Client) *SearchMaintenanceService {
	return &SearchMaintenanceService{
		rebuild: func(ctx context.Context) error {
			return repository.RebuildD1FTSIndex(ctx, client)
		},
	}
}

func NewDisabledSearchMaintenanceService(message string) *SearchMaintenanceService {
	return &SearchMaintenanceService{
		rebuild: func(context.Context) error {
			return errors.New(message)
		},
	}
}

func (s *SearchMaintenanceService) RebuildFTSIndex(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return s.rebuild(ctx)
}
