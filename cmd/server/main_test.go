package main

import (
	"context"
	"testing"

	"github.com/wonichan/acgwarehouse-backend/internal/ai"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/worker"
)

func TestRegisterAIWorkerInjectsGovernanceService(t *testing.T) {
	manager := worker.NewManager(nil)
	client := &stubAIProvider{}
	obsRepo := &stubObservationRepository{}
	governance := &stubGovernanceService{}

	called := false
	original := registerAITagHandler
	registerAITagHandler = func(manager *worker.Manager, client ai.AIProvider, obsRepo repository.TagObservationRepository, governance worker.TagGovernanceMerger) {
		called = true
		if governance != nil {
			return
		}
		panic("expected governance service to be injected")
	}
	t.Cleanup(func() {
		registerAITagHandler = original
	})

	registerAIWorker(manager, client, obsRepo, governance)

	if !called {
		t.Fatal("expected AI worker registration to be invoked")
	}
}

type stubAIProvider struct{}

func (s *stubAIProvider) Name() string { return "stub" }

func (s *stubAIProvider) GenerateTags(ctx interface{}, imageURL, prompt string) (*ai.TagResult, error) {
	return &ai.TagResult{}, nil
}

type stubObservationRepository struct{}

func (s *stubObservationRepository) Save(ctx context.Context, obs *domain.TagObservation) error {
	return nil
}

func (s *stubObservationRepository) FindByImageID(ctx context.Context, imageID int64) ([]*domain.TagObservation, error) {
	return nil, nil
}

func (s *stubObservationRepository) FindByProvider(ctx context.Context, provider string, limit int) ([]*domain.TagObservation, error) {
	return nil, nil
}

func (s *stubObservationRepository) FindByID(ctx context.Context, id int64) (*domain.TagObservation, error) {
	return nil, nil
}

type stubGovernanceService struct{}

func (s *stubGovernanceService) MergeTags(ctx context.Context, imageID int64, tags []string, observationID int64, confidence float64) error {
	return nil
}
