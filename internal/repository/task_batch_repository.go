package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

var ErrTaskPlatformRepositoryNotImplemented = errors.New("task platform repository not implemented")

type TaskBatchListFilter struct {
	SourceType string
	Status     string
	Limit      int
	Offset     int
}

type TaskBatchAggregate struct {
	BatchID      int64
	TaskType     string
	Status       string
	Count        int64
	LatestTaskAt *string
}

type TaskBatchRepository interface {
	Create(ctx context.Context, batch *domain.TaskBatch) error
	AddSource(ctx context.Context, source *domain.TaskBatchSource) error
	FindByID(ctx context.Context, batchID int64) (*domain.TaskBatch, error)
	List(ctx context.Context, filter TaskBatchListFilter) ([]domain.TaskBatch, error)
	ListSources(ctx context.Context, batchID int64) ([]domain.TaskBatchSource, error)
	ListAggregates(ctx context.Context, batchID int64) ([]TaskBatchAggregate, error)
	Update(ctx context.Context, batch *domain.TaskBatch) error
}

type sqliteTaskBatchRepository struct {
	db *sql.DB
}

func NewTaskBatchRepository(db *sql.DB) TaskBatchRepository {
	return &sqliteTaskBatchRepository{db: db}
}

func (r *sqliteTaskBatchRepository) Create(ctx context.Context, batch *domain.TaskBatch) error {
	return ErrTaskPlatformRepositoryNotImplemented
}

func (r *sqliteTaskBatchRepository) AddSource(ctx context.Context, source *domain.TaskBatchSource) error {
	return ErrTaskPlatformRepositoryNotImplemented
}

func (r *sqliteTaskBatchRepository) FindByID(ctx context.Context, batchID int64) (*domain.TaskBatch, error) {
	return nil, ErrTaskPlatformRepositoryNotImplemented
}

func (r *sqliteTaskBatchRepository) List(ctx context.Context, filter TaskBatchListFilter) ([]domain.TaskBatch, error) {
	return nil, ErrTaskPlatformRepositoryNotImplemented
}

func (r *sqliteTaskBatchRepository) ListSources(ctx context.Context, batchID int64) ([]domain.TaskBatchSource, error) {
	return nil, ErrTaskPlatformRepositoryNotImplemented
}

func (r *sqliteTaskBatchRepository) ListAggregates(ctx context.Context, batchID int64) ([]TaskBatchAggregate, error) {
	return nil, ErrTaskPlatformRepositoryNotImplemented
}

func (r *sqliteTaskBatchRepository) Update(ctx context.Context, batch *domain.TaskBatch) error {
	return ErrTaskPlatformRepositoryNotImplemented
}
