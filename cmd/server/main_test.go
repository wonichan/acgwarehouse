package main

import (
	"testing"

	"github.com/wonichan/acgwarehouse-backend/internal/ai"
	"github.com/wonichan/acgwarehouse-backend/internal/worker"
)

func TestRegisterAIWorkerInjectsGovernanceService(t *testing.T) {
	manager := worker.NewManager(nil)
	client := &stubAIProvider{}
	obsRepo := &stubObservationRepository{}
	governance := &stubGovernanceService{}

	called := false
	original := registerAITagHandler
	registerAITagHandler = func(manager *worker.Manager, client ai.AIProvider, obsRepo workerTagObservationRepository, governance workerTagGovernanceService) {
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

type stubGovernanceService struct{}
