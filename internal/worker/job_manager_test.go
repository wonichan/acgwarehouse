package worker

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/yourusername/acgwarehouse-backend/internal/repository"
)

func TestManagerProcessesJobsSequentially(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", t.TempDir()+"/jobs.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	jobRepo := repository.NewJobRepository(db)
	mgr := NewManager(jobRepo)

	var (
		mu    sync.Mutex
		order []int64
	)
	mgr.RegisterHandler("image_imported", func(ctx context.Context, id int64, payload string) error {
		mu.Lock()
		order = append(order, id)
		mu.Unlock()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mgr.Start(ctx)
	defer mgr.Stop()

	id1, err := mgr.AddJob(ctx, "image_imported", `{"path":"a.png"}`)
	if err != nil {
		t.Fatalf("AddJob() first error = %v", err)
	}
	id2, err := mgr.AddJob(ctx, "image_imported", `{"path":"b.png"}`)
	if err != nil {
		t.Fatalf("AddJob() second error = %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		j1, err1 := jobRepo.FindByID(id1)
		j2, err2 := jobRepo.FindByID(id2)
		if err1 == nil && err2 == nil && j1.Status == "finished" && j2.Status == "finished" {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	j1, err := jobRepo.FindByID(id1)
	if err != nil {
		t.Fatalf("FindByID(id1) error = %v", err)
	}
	j2, err := jobRepo.FindByID(id2)
	if err != nil {
		t.Fatalf("FindByID(id2) error = %v", err)
	}

	if j1.Status != "finished" || j2.Status != "finished" {
		t.Fatalf("unexpected statuses: job1=%s job2=%s", j1.Status, j2.Status)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 || order[0] != id1 || order[1] != id2 {
		t.Fatalf("handler order = %v, want [%d %d]", order, id1, id2)
	}
}
