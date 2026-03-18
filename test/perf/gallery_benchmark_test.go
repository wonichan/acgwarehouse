package perf

import (
	"context"
	"fmt"
	"testing"

	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

// runGalleryBenchmark runs gallery benchmarks with the given parameters
func runGalleryBenchmark(b *testing.B, imageCount, tagCount, tagsPerImage int) {
	cfg := BenchmarkConfig{
		ImageCount:   imageCount,
		TagCount:     tagCount,
		TagsPerImage: tagsPerImage,
		WithPHash:    false,
		Seed:         42,
	}

	bd, err := NewBenchmarkDatabase(cfg)
	if err != nil {
		b.Fatalf("create benchmark db: %v", err)
	}
	defer bd.Close()

	// Seed data
	if err := bd.SeedData(cfg); err != nil {
		b.Fatalf("seed data: %v", err)
	}

	repo := bd.GetImageRepository()
	ctx := context.Background()

	// Benchmark FindAll
	b.ResetTimer()
	b.Run("FindAll-50", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := repo.FindAll(50, 0)
			if err != nil {
				b.Fatalf("FindAll: %v", err)
			}
		}
	})

	// Benchmark FindAll with offset (pagination)
	b.Run("FindAll-50-offset-500", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := repo.FindAll(50, 500)
			if err != nil {
				b.Fatalf("FindAll with offset: %v", err)
			}
		}
	})

	// Benchmark Count
	b.Run("Count", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := repo.Count()
			if err != nil {
				b.Fatalf("Count: %v", err)
			}
		}
	})

	// Get some tag IDs for filtering
	tags, err := repository.NewTagRepository(bd.db).FindAll(ctx, 10, 0)
	if err != nil {
		b.Fatalf("find tags: %v", err)
	}
	if len(tags) > 0 {
		tagIDs := make([]int64, 0, 3)
		for i := 0; i < 3 && i < len(tags); i++ {
			tagIDs = append(tagIDs, tags[i].ID)
		}

		// Benchmark FindByTagIDs
		b.Run("FindByTagIDs-3tags", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := repo.FindByTagIDs(ctx, tagIDs, 50, 0)
				if err != nil {
					b.Fatalf("FindByTagIDs: %v", err)
				}
			}
		})
	}
}

// BenchmarkSmallDataset runs a benchmark on a small dataset (100 images)
//
// Usage: go test ./test/perf/... -run ^$ -bench BenchmarkSmallDataset -benchmem
func BenchmarkSmallDataset(b *testing.B) {
	runGalleryBenchmark(b, 100, 10, 2)
}

// BenchmarkMediumDataset runs a benchmark on a medium dataset (1000 images)
//
// Usage: go test ./test/perf/... -run ^$ -bench BenchmarkMediumDataset -benchmem
func BenchmarkMediumDataset(b *testing.B) {
	runGalleryBenchmark(b, 1000, 50, 3)
}

// BenchmarkLargeDataset runs a benchmark on a large dataset (10000 images)
//
// Usage: go test ./test/perf/... -run ^$ -bench BenchmarkLargeDataset -benchmem -count=1
func BenchmarkLargeDataset(b *testing.B) {
	runGalleryBenchmark(b, 10000, 100, 3)
}

// SmokeTest verifies the benchmark harness works correctly
func TestSmokeTest(t *testing.T) {
	cfg := BenchmarkConfig{
		ImageCount:   50,
		TagCount:     5,
		TagsPerImage: 2,
		WithPHash:    false,
		Seed:         42,
	}

	bd, err := NewBenchmarkDatabase(cfg)
	if err != nil {
		t.Fatalf("create benchmark db: %v", err)
	}
	defer bd.Close()

	if err := bd.SeedData(cfg); err != nil {
		t.Fatalf("seed data: %v", err)
	}

	repo := bd.GetImageRepository()

	// Test basic operations
	images, err := repo.FindAll(10, 0)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(images) == 0 {
		t.Fatal("expected images, got none")
	}

	count, err := repo.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 50 {
		t.Errorf("expected 50 images, got %d", count)
	}

	// Test tag filtering
	ctx := context.Background()
	tags, err := repository.NewTagRepository(bd.db).FindAll(ctx, 5, 0)
	if err != nil {
		t.Fatalf("FindAll tags: %v", err)
	}
	if len(tags) > 0 {
		filtered, err := repo.FindByTagIDs(ctx, []int64{tags[0].ID}, 10, 0)
		if err != nil {
			t.Fatalf("FindByTagIDs: %v", err)
		}
		fmt.Printf("Smoke test: filtered images by tag %d: %d results\n", tags[0].ID, len(filtered))
	}
}
