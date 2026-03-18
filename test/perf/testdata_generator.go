package perf

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

// BenchmarkConfig holds configuration for benchmark test data generation
type BenchmarkConfig struct {
	ImageCount   int
	TagCount     int
	TagsPerImage int
	WithPHash    bool
	Seed         int64
}

// DefaultConfig returns a reasonable default configuration for benchmarks
func DefaultConfig() BenchmarkConfig {
	return BenchmarkConfig{
		ImageCount:   10000,
		TagCount:     100,
		TagsPerImage: 3,
		WithPHash:    false, // Skip expensive pHash for benchmark speed
		Seed:         42,
	}
}

// SeededRNG is a seeded random number generator for reproducible test data
type SeededRNG struct {
	rand *rand.Rand
}

// NewSeededRNG creates a new seeded random number generator
func NewSeededRNG(seed int64) *SeededRNG {
	return &SeededRNG{
		rand: rand.New(rand.NewSource(seed)),
	}
}

// RandomString generates a random alphanumeric string of given length
func (r *SeededRNG) RandomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[r.rand.Intn(len(chars))]
	}
	return string(result)
}

// BenchmarkDatabase wraps a SQLite database for benchmarking
type BenchmarkDatabase struct {
	db           *sql.DB
	imageRepo    repository.ImageRepository
	tagRepo      repository.TagRepository
	imageTagRepo repository.ImageTagRepository
	rng          *SeededRNG
}

// NewBenchmarkDatabase creates a new benchmark database with test data
func NewBenchmarkDatabase(cfg BenchmarkConfig) (*BenchmarkDatabase, error) {
	// Use a temporary file for the benchmark database (cross-platform)
	dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("benchmark_%d.db", time.Now().UnixNano()))

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Set up schema
	if err := repository.EnsureScanSchema(db); err != nil {
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	return &BenchmarkDatabase{
		db:           db,
		imageRepo:    repository.NewImageRepository(db),
		tagRepo:      repository.NewTagRepository(db),
		imageTagRepo: repository.NewImageTagRepository(db),
		rng:          NewSeededRNG(cfg.Seed),
	}, nil
}

// Close closes the benchmark database
func (bd *BenchmarkDatabase) Close() error {
	if bd.db != nil {
		return bd.db.Close()
	}
	return nil
}

// SeedData populates the database with benchmark test data
func (bd *BenchmarkDatabase) SeedData(cfg BenchmarkConfig) error {
	ctx := context.Background()

	// 1. Create tags first
	tags := make([]*domain.Tag, cfg.TagCount)
	for i := 0; i < cfg.TagCount; i++ {
		tag := &domain.Tag{
			PreferredLabel: fmt.Sprintf("tag_%d_%s", i, bd.rng.RandomString(6)),
			Slug:           fmt.Sprintf("tag-%d-%s", i, bd.rng.RandomString(6)),
			ReviewState:    "confirmed",
			UsageCount:     0,
		}
		if err := bd.tagRepo.Save(ctx, tag); err != nil {
			return fmt.Errorf("save tag %d: %w", i, err)
		}
		tags[i] = tag
	}

	// 2. Create images in batches for better performance
	fmt.Printf("Creating %d images...\n", cfg.ImageCount)
	batchSize := 1000

	for batch := 0; batch < (cfg.ImageCount+batchSize-1)/batchSize; batch++ {
		start := batch * batchSize
		end := start + batchSize
		if end > cfg.ImageCount {
			end = cfg.ImageCount
		}

		for i := start; i < end; i++ {
			img := &domain.Image{
				Path:       fmt.Sprintf("/library/%s/%s.png", bd.rng.RandomString(2), bd.rng.RandomString(12)),
				Filename:   fmt.Sprintf("image_%d.png", i),
				SourceRoot: "/library",
				FileSize:   int64(bd.rng.rand.Intn(10_000_000) + 100_000),
				Width:      bd.rng.rand.Intn(4000) + 1000,
				Height:     bd.rng.rand.Intn(4000) + 1000,
				Format:     "png",
				CreatedAt:  time.Now().Add(-time.Duration(bd.rng.rand.Intn(365*24)) * time.Hour),
				UpdatedAt:  time.Now(),
			}

			if cfg.WithPHash {
				img.PHash = int64(bd.rng.rand.Intn(1 << 62))
			}

			if err := bd.imageRepo.SaveImage(img); err != nil {
				return fmt.Errorf("save image %d: %w", i, err)
			}

			// Add tags to image (random subset)
			numTags := bd.rng.rand.Intn(cfg.TagsPerImage + 1)
			usedTags := make(map[int]bool)
			for j := 0; j < numTags && j < len(tags); j++ {
				tagIdx := bd.rng.rand.Intn(len(tags))
				if usedTags[tagIdx] {
					continue
				}
				usedTags[tagIdx] = true

				imageTag := &domain.ImageTag{
					ImageID:     img.ID,
					TagID:       tags[tagIdx].ID,
					ReviewState: "confirmed",
				}
				if err := bd.imageTagRepo.Save(ctx, imageTag); err != nil {
					return fmt.Errorf("save image-tag: %w", err)
				}
			}
		}

		if (batch+1)%5 == 0 {
			fmt.Printf("  Progress: %d/%d images\n", end, cfg.ImageCount)
		}
	}

	fmt.Printf("Seeding complete: %d images, %d tags\n", cfg.ImageCount, cfg.TagCount)
	return nil
}

// GetImageRepository returns the image repository
func (bd *BenchmarkDatabase) GetImageRepository() repository.ImageRepository {
	return bd.imageRepo
}

// GetDB returns the underlying database
func (bd *BenchmarkDatabase) GetDB() *sql.DB {
	return bd.db
}

// BenchmarkHelper provides helper methods for running benchmarks
type BenchmarkHelper struct {
	bd *BenchmarkDatabase
}

// NewBenchmarkHelper creates a new benchmark helper
func NewBenchmarkHelper(bd *BenchmarkDatabase) *BenchmarkHelper {
	return &BenchmarkHelper{bd: bd}
}

// CountImages returns the total number of images in the database
func (bh *BenchmarkHelper) CountImages() (int64, error) {
	return bh.bd.imageRepo.Count()
}
