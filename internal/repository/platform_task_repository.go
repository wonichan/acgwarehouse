package repository

import (
	"context"
	"database/sql"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type PlatformTaskListFilter struct {
	BatchID  *int64
	ImageID  *int64
	TaskType string
	Status   string
	Limit    int
	Offset   int
}

type PlatformTaskDedupeLookup struct {
	TaskType        string
	ImageVersionKey string
	DedupeKey       string
	Statuses        []string
}

type PlatformTaskRepository interface {
	Create(ctx context.Context, task *domain.PlatformTask) error
	FindByID(ctx context.Context, taskID int64) (*domain.PlatformTask, error)
	List(ctx context.Context, filter PlatformTaskListFilter) ([]domain.PlatformTask, error)
	FindByDedupeKey(ctx context.Context, dedupeKey string) (*domain.PlatformTask, error)
	ListByImageAndTypes(ctx context.Context, imageID int64, taskTypes []string) ([]domain.PlatformTask, error)
	ListActiveByDedupe(ctx context.Context, lookup PlatformTaskDedupeLookup) ([]domain.PlatformTask, error)
	Update(ctx context.Context, task *domain.PlatformTask) error
	SetLatestAsyncJob(ctx context.Context, taskID int64, asyncJobID *int64) error
}

type sqlitePlatformTaskRepository struct {
	db *sql.DB
}

func NewPlatformTaskRepository(db *sql.DB) PlatformTaskRepository {
	return &sqlitePlatformTaskRepository{db: db}
}

func (r *sqlitePlatformTaskRepository) Create(ctx context.Context, task *domain.PlatformTask) error {
	return ErrTaskPlatformRepositoryNotImplemented
}

func (r *sqlitePlatformTaskRepository) FindByID(ctx context.Context, taskID int64) (*domain.PlatformTask, error) {
	return nil, ErrTaskPlatformRepositoryNotImplemented
}

func (r *sqlitePlatformTaskRepository) List(ctx context.Context, filter PlatformTaskListFilter) ([]domain.PlatformTask, error) {
	return nil, ErrTaskPlatformRepositoryNotImplemented
}

func (r *sqlitePlatformTaskRepository) FindByDedupeKey(ctx context.Context, dedupeKey string) (*domain.PlatformTask, error) {
	return nil, ErrTaskPlatformRepositoryNotImplemented
}

func (r *sqlitePlatformTaskRepository) ListByImageAndTypes(ctx context.Context, imageID int64, taskTypes []string) ([]domain.PlatformTask, error) {
	return nil, ErrTaskPlatformRepositoryNotImplemented
}

func (r *sqlitePlatformTaskRepository) ListActiveByDedupe(ctx context.Context, lookup PlatformTaskDedupeLookup) ([]domain.PlatformTask, error) {
	return nil, ErrTaskPlatformRepositoryNotImplemented
}

func (r *sqlitePlatformTaskRepository) Update(ctx context.Context, task *domain.PlatformTask) error {
	return ErrTaskPlatformRepositoryNotImplemented
}

func (r *sqlitePlatformTaskRepository) SetLatestAsyncJob(ctx context.Context, taskID int64, asyncJobID *int64) error {
	return ErrTaskPlatformRepositoryNotImplemented
}
