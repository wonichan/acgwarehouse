package service

import (
	"context"
	"testing"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

// TestPreviewBackfillRejectsUnfiltered verifies D-02: preview rejects unfiltered requests.
func TestPreviewBackfillRejectsUnfiltered(t *testing.T) {
	svc := NewAIBackfillService(nil, nil)

	// No filters at all — must be rejected per D-02
	filter := repository.BackfillCandidateFilter{}
	_, err := svc.PreviewBackfill(context.Background(), filter)
	if err == nil {
		t.Fatal("expected error for unfiltered backfill request, got nil")
	}
}

// TestIsFilterNarrowed verifies the filter narrowing detection.
func TestIsFilterNarrowed(t *testing.T) {
	tests := []struct {
		name     string
		filter   repository.BackfillCandidateFilter
		narrowed bool
	}{
		{
			name:     "empty filter is not narrowed",
			filter:   repository.BackfillCandidateFilter{},
			narrowed: false,
		},
		{
			name:     "tag_ids filter is narrowed",
			filter:   repository.BackfillCandidateFilter{TagIDs: []int64{1, 2}},
			narrowed: true,
		},
		{
			name:     "has_tags=false is narrowed",
			filter:   repository.BackfillCandidateFilter{HasTags: boolPtr(false)},
			narrowed: true,
		},
		{
			name:     "has_tags=true is narrowed",
			filter:   repository.BackfillCandidateFilter{HasTags: boolPtr(true)},
			narrowed: true,
		},
		{
			name:     "sort-only is not narrowed",
			filter:   repository.BackfillCandidateFilter{SortBy: "id", SortDir: "desc"},
			narrowed: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsFilterNarrowed(tt.filter)
			if got != tt.narrowed {
				t.Errorf("IsFilterNarrowed(%+v) = %v, want %v", tt.filter, got, tt.narrowed)
			}
		})
	}
}

// TestPreviewBackfillClassifiesSkipReasons verifies D-05/D-09/D-12: preview returns
// hit count, enqueueable count, skipped-with-ai-tag count, and skipped-with-active-task count.
func TestPreviewBackfillClassifiesSkipReasons(t *testing.T) {
	// This test uses a mock image repository to verify classification logic.
	mockRepo := &mockBackfillImageRepo{
		hitCount:              10,
		enqueueableCount:      4,
		skippedWithAITag:      3,
		skippedWithActiveTask: 3,
	}
	svc := NewAIBackfillService(mockRepo, nil)

	filter := repository.BackfillCandidateFilter{HasTags: boolPtr(false)}
	result, err := svc.PreviewBackfill(context.Background(), filter)
	if err != nil {
		t.Fatalf("PreviewBackfill() error = %v", err)
	}
	if result.HitCount != 10 {
		t.Errorf("HitCount = %d, want 10", result.HitCount)
	}
	if result.EnqueueableCount != 4 {
		t.Errorf("EnqueueableCount = %d, want 4", result.EnqueueableCount)
	}
	if result.SkippedWithAITag != 3 {
		t.Errorf("SkippedWithAITag = %d, want 3", result.SkippedWithAITag)
	}
	if result.SkippedWithActiveTask != 3 {
		t.Errorf("SkippedWithActiveTask = %d, want 3", result.SkippedWithActiveTask)
	}
	if result.SkippedTotal != 6 {
		t.Errorf("SkippedTotal = %d, want 6", result.SkippedTotal)
	}
}

// TestExecuteBackfillReturnsNoOpForZeroEligible verifies D-13: execute returns
// explicit no-op result when preview has zero eligible images.
func TestExecuteBackfillReturnsNoOpForZeroEligible(t *testing.T) {
	mockRepo := &mockBackfillImageRepo{
		hitCount:              5,
		enqueueableCount:      0,
		skippedWithAITag:      3,
		skippedWithActiveTask: 2,
	}
	svc := NewAIBackfillService(mockRepo, nil)

	filter := repository.BackfillCandidateFilter{TagIDs: []int64{1}}
	result, err := svc.ExecuteBackfill(context.Background(), filter, "")
	if err != nil {
		t.Fatalf("ExecuteBackfill() error = %v", err)
	}
	if result.Success {
		t.Error("expected Success=false for zero eligible images, got true")
	}
	if result.CreatedTasks != 0 {
		t.Errorf("CreatedTasks = %d, want 0", result.CreatedTasks)
	}
	if result.NoOpReason == "" {
		t.Error("expected non-empty NoOpReason for zero eligible images")
	}
}

// --- mock implementations ---

type mockBackfillImageRepo struct {
	hitCount              int64
	enqueueableCount      int64
	skippedWithAITag      int64
	skippedWithActiveTask int64
	err                   error
}

func (m *mockBackfillImageRepo) SaveImage(_ *domain.Image) (bool, error)    { return false, nil }
func (m *mockBackfillImageRepo) FindByID(_ int64) (*domain.Image, error)    { return nil, nil }
func (m *mockBackfillImageRepo) FindByPath(_ string) (*domain.Image, error) { return nil, nil }
func (m *mockBackfillImageRepo) FindAll(_, _ int, _, _ string) ([]domain.Image, error) {
	return nil, nil
}
func (m *mockBackfillImageRepo) FindByTagIDs(_ context.Context, _ []int64, _, _ int, _, _ string) ([]domain.Image, error) {
	return nil, nil
}
func (m *mockBackfillImageRepo) CountByTagIDs(_ context.Context, _ []int64) (int64, error) {
	return 0, nil
}
func (m *mockBackfillImageRepo) FindUntagged(_ context.Context, _, _ int, _, _ string) ([]domain.Image, error) {
	return nil, nil
}
func (m *mockBackfillImageRepo) CountUntagged(_ context.Context) (int64, error) { return 0, nil }
func (m *mockBackfillImageRepo) FindImagesWithoutAITags(_ context.Context, _ int) ([]domain.Image, error) {
	return nil, nil
}
func (m *mockBackfillImageRepo) FindBackfillCandidates(_ context.Context, _ repository.BackfillCandidateFilter) ([]domain.Image, error) {
	return nil, m.err
}
func (m *mockBackfillImageRepo) CountBackfillCandidates(_ context.Context, _ repository.BackfillCandidateFilter) (int64, error) {
	return m.enqueueableCount, m.err
}
func (m *mockBackfillImageRepo) CountBackfillSkippedWithAITag(_ context.Context, _ repository.BackfillCandidateFilter) (int64, error) {
	return m.skippedWithAITag, m.err
}
func (m *mockBackfillImageRepo) CountBackfillSkippedWithActiveTask(_ context.Context, _ repository.BackfillCandidateFilter) (int64, error) {
	return m.skippedWithActiveTask, m.err
}
func (m *mockBackfillImageRepo) CountBackfillHitCount(_ context.Context, _ repository.BackfillCandidateFilter) (int64, error) {
	return m.hitCount, m.err
}
func (m *mockBackfillImageRepo) UpdateThumbnails(_ int64, _, _ string) error { return nil }
func (m *mockBackfillImageRepo) Count() (int64, error)                       { return 0, nil }
func (m *mockBackfillImageRepo) Delete(_ int64) error                        { return nil }

func boolPtr(v bool) *bool { return &v }
