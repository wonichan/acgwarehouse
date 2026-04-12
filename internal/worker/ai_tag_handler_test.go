package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/ai"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

func TestRegisterBatchAITagHandler_Registration(t *testing.T) {
	// Test 1: AI 标签任务处理器正确注册
	mockJobRepo := &mockJobRepoForAI{}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{
		result: &ai.TagResult{
			Tags:       []string{"girl", "outdoors", "sunny"},
			Confidence: 0.92,
			ModelName:  "qwen-vl-max",
		},
	}

	manager := NewManager(mockJobRepo)
	RegisterBatchAITagHandler(manager, mockJobRepo, mockClient, mockObsRepo, &mockTagGovernanceService{}, nil, nil, nil, "auto")

	// 验证处理器已注册
	if _, ok := manager.handlers["ai_tag_generation"]; !ok {
		t.Error("expected ai_tag_generation handler to be registered")
	}
}

func TestBatchAITagHandler_ParsesPayload(t *testing.T) {
	// Test 2: 处理器解析 payload 并调用 AI 服务
	mockJobRepo := &mockJobRepoForAI{}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{
		result: &ai.TagResult{
			Tags:       []string{"girl", "outdoors", "anime"},
			Confidence: 0.9,
			ModelName:  "qwen-vl-max",
		},
	}

	manager := NewManager(mockJobRepo)
	governance := &mockTagGovernanceService{}
	RegisterBatchAITagHandler(manager, mockJobRepo, mockClient, mockObsRepo, governance, nil, nil, nil, "auto")

	// 创建测试 payload
	payload := AITagPayload{
		ImageID: 123,
		Path:    "/path/to/image.jpg",
	}
	payloadBytes, _ := json.Marshal(payload)

	// 调用处理器
	handler := manager.handlers["ai_tag_generation"]
	err := handler(context.Background(), 1, string(payloadBytes))
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	// 验证 AI 客户端被调用
	if mockClient.lastImageURL != "/path/to/image.jpg" {
		t.Errorf("expected image URL '/path/to/image.jpg', got %s", mockClient.lastImageURL)
	}

	// 验证观测记录被保存
	if mockObsRepo.savedObservation == nil {
		t.Fatal("expected observation to be saved")
	}
	if mockObsRepo.savedObservation.ImageID != 123 {
		t.Errorf("expected image ID 123, got %d", mockObsRepo.savedObservation.ImageID)
	}
	if !governance.called {
		t.Fatal("expected governance merge to be called")
	}
	if governance.imageID != 123 {
		t.Errorf("expected governance image ID 123, got %d", governance.imageID)
	}
	if governance.observationID != 1 {
		t.Errorf("expected observation ID 1, got %d", governance.observationID)
	}
	if governance.confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", governance.confidence)
	}
}

func TestBatchAITagHandler_SavesObservation(t *testing.T) {
	// Test 3: 处理器保存观测记录
	mockJobRepo := &mockJobRepoForAI{}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{
		result: &ai.TagResult{
			Tags:        []string{"blue hair", "girl", "outdoors"},
			Confidence:  0.95,
			ModelName:   "doubao-vision-pro",
			RawResponse: `{"tags": ["blue hair", "girl", "outdoors"]}`,
		},
	}

	manager := NewManager(mockJobRepo)
	RegisterBatchAITagHandler(manager, mockJobRepo, mockClient, mockObsRepo, &mockTagGovernanceService{}, nil, nil, nil, "auto")

	payload := AITagPayload{
		ImageID: 456,
		Path:    "/test/image.png",
	}
	payloadBytes, _ := json.Marshal(payload)

	handler := manager.handlers["ai_tag_generation"]
	err := handler(context.Background(), 2, string(payloadBytes))
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	obs := mockObsRepo.savedObservation
	if obs == nil {
		t.Fatal("expected observation to be saved")
	}
	if obs.ImageID != 456 {
		t.Errorf("expected image ID 456, got %d", obs.ImageID)
	}
	if obs.Provider != "doubao" {
		t.Errorf("expected provider 'doubao', got %s", obs.Provider)
	}
	if obs.ModelName != "doubao-vision-pro" {
		t.Errorf("expected model 'doubao-vision-pro', got %s", obs.ModelName)
	}
	if obs.Confidence != 0.95 {
		t.Errorf("expected confidence 0.95, got %f", obs.Confidence)
	}
	if obs.RawText != "blue hair, girl, outdoors" {
		t.Errorf("expected raw text 'blue hair, girl, outdoors', got %s", obs.RawText)
	}
}

func TestBatchAITagHandler_PersistsPendingImageTagsForReview(t *testing.T) {
	db := mustOpenAIWorkerDB(t)
	seedAIWorkerImage(t, db, 456)

	obsRepo := repository.NewTagObservationRepository(db)
	tagRepo := repository.NewTagRepository(db)
	aliasRepo := repository.NewTagAliasRepository(db)
	imageTagRepo := repository.NewImageTagRepository(db)
	governance := service.NewTagGovernanceService(tagRepo, aliasRepo, obsRepo, imageTagRepo)
	mockClient := &mockAIClient{
		result: &ai.TagResult{
			Tags:       []string{"blue hair", "girl", "outdoors"},
			Confidence: 0.95,
			ModelName:  "doubao-vision-pro",
		},
	}

	mockJobRepo := &mockJobRepoForAI{}
	manager := NewManager(mockJobRepo)
	RegisterBatchAITagHandler(manager, mockJobRepo, mockClient, obsRepo, governance, nil, nil, nil, "auto")

	payloadBytes, _ := json.Marshal(AITagPayload{ImageID: 456, Path: "/test/image.png"})
	handler := manager.handlers["ai_tag_generation"]
	if err := handler(context.Background(), 2, string(payloadBytes)); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	observations, err := obsRepo.FindByImageID(context.Background(), 456)
	if err != nil {
		t.Fatalf("FindByImageID() observations error = %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(observations))
	}

	imageTags, err := imageTagRepo.FindByImageID(context.Background(), 456)
	if err != nil {
		t.Fatalf("FindByImageID() image tags error = %v", err)
	}
	if len(imageTags) != 3 {
		t.Fatalf("expected 3 image tags, got %d", len(imageTags))
	}
	for _, imageTag := range imageTags {
		if imageTag.ReviewState != "pending" {
			t.Fatalf("image tag review state = %q, want pending", imageTag.ReviewState)
		}
		if imageTag.Source != domain.ImageTagSourceAI {
			t.Fatalf("image tag source = %q, want %q", imageTag.Source, domain.ImageTagSourceAI)
		}
		if imageTag.SourceObservationID == nil || *imageTag.SourceObservationID != observations[0].ID {
			t.Fatalf("image tag source observation = %v, want %d", imageTag.SourceObservationID, observations[0].ID)
		}
	}
}

func TestBatchAITagHandler_RemovesRejectedAITagsBeforeSavingFreshResults(t *testing.T) {
	db := mustOpenAIWorkerDB(t)
	seedAIWorkerImage(t, db, 457)
	ctx := context.Background()

	obsRepo := repository.NewTagObservationRepository(db)
	tagRepo := repository.NewTagRepository(db)
	aliasRepo := repository.NewTagAliasRepository(db)
	imageTagRepo := repository.NewImageTagRepository(db)
	governance := service.NewTagGovernanceService(tagRepo, aliasRepo, obsRepo, imageTagRepo)

	rejected := &domain.Tag{PreferredLabel: "old-rejected", Slug: "old-rejected", ReviewState: "confirmed", UsageCount: 1}
	if err := tagRepo.Save(ctx, rejected); err != nil {
		t.Fatalf("Save(rejected seed tag) error = %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: 457, TagID: rejected.ID, Source: domain.ImageTagSourceAI, ReviewState: "rejected", Confidence: 0.3}); err != nil {
		t.Fatalf("Save(rejected ai tag) error = %v", err)
	}

	mockClient := &mockAIClient{
		result: &ai.TagResult{
			Tags:       []string{"fresh-tag"},
			Confidence: 0.93,
			ModelName:  "doubao-vision-pro",
		},
	}

	mockJobRepo := &mockJobRepoForAI{}
	manager := NewManager(mockJobRepo)
	RegisterBatchAITagHandler(manager, mockJobRepo, mockClient, obsRepo, governance, nil, nil, nil, "auto")

	payloadBytes, _ := json.Marshal(AITagPayload{ImageID: 457, Path: "/test/image.png"})
	handler := manager.handlers["ai_tag_generation"]
	if err := handler(ctx, 3, string(payloadBytes)); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	items, err := imageTagRepo.FindByImageID(ctx, 457)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	for _, item := range items {
		if item.TagID == rejected.ID {
			t.Fatalf("rejected AI tag should be removed before fresh generation merge: %+v", item)
		}
	}
	newTag, err := tagRepo.FindByLabel(ctx, "fresh-tag")
	if err != nil {
		t.Fatalf("FindByLabel(fresh-tag) error = %v", err)
	}
	if len(items) != 1 || items[0].TagID != newTag.ID {
		t.Fatalf("unexpected image tags after fresh generation: %+v", items)
	}
}

func TestAITagRegenerationHandlerReplacesPendingAITagsButPreservesConfirmedTags(t *testing.T) {
	db := mustOpenAIWorkerDB(t)
	seedAIWorkerImage(t, db, 999)
	ctx := context.Background()

	obsRepo := repository.NewTagObservationRepository(db)
	tagRepo := repository.NewTagRepository(db)
	aliasRepo := repository.NewTagAliasRepository(db)
	imageTagRepo := repository.NewImageTagRepository(db)
	governance := service.NewTagGovernanceService(tagRepo, aliasRepo, obsRepo, imageTagRepo)

	keepManual := &domain.Tag{PreferredLabel: "keep-manual", Slug: "keep-manual", ReviewState: "confirmed", UsageCount: 1}
	oldPending := &domain.Tag{PreferredLabel: "old-pending-ai", Slug: "old-pending-ai", ReviewState: "confirmed", UsageCount: 1}
	keepConfirmedAI := &domain.Tag{PreferredLabel: "keep-confirmed-ai", Slug: "keep-confirmed-ai", ReviewState: "confirmed", UsageCount: 1}
	oldRejected := &domain.Tag{PreferredLabel: "old-rejected-ai", Slug: "old-rejected-ai", ReviewState: "confirmed", UsageCount: 1}
	for _, tag := range []*domain.Tag{keepManual, oldPending, keepConfirmedAI, oldRejected} {
		if err := tagRepo.Save(ctx, tag); err != nil {
			t.Fatalf("Save(seed tag %s) error = %v", tag.PreferredLabel, err)
		}
	}
	oldObs := &domain.TagObservation{ImageID: 999, RawText: "old-pending-ai", Confidence: 0.8, EvidenceType: "ai_generated", Provider: "mock", ModelName: "seed", PromptVersion: "v1", CreatedAt: time.Now()}
	confirmedObs := &domain.TagObservation{ImageID: 999, RawText: "keep-confirmed-ai", Confidence: 0.85, EvidenceType: "ai_generated", Provider: "mock", ModelName: "seed", PromptVersion: "v1", CreatedAt: time.Now()}
	rejectedObs := &domain.TagObservation{ImageID: 999, RawText: "old-rejected-ai", Confidence: 0.2, EvidenceType: "ai_generated", Provider: "mock", ModelName: "seed", PromptVersion: "v1", CreatedAt: time.Now()}
	if err := obsRepo.Save(ctx, oldObs); err != nil {
		t.Fatalf("Save(old observation) error = %v", err)
	}
	if err := obsRepo.Save(ctx, confirmedObs); err != nil {
		t.Fatalf("Save(confirmed observation) error = %v", err)
	}
	if err := obsRepo.Save(ctx, rejectedObs); err != nil {
		t.Fatalf("Save(rejected observation) error = %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: 999, TagID: keepManual.ID, Source: domain.ImageTagSourceManual, ReviewState: "confirmed", Confidence: 1}); err != nil {
		t.Fatalf("Save(manual confirmed) error = %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: 999, TagID: oldPending.ID, Source: domain.ImageTagSourceAI, SourceObservationID: &oldObs.ID, ReviewState: "pending", Confidence: 0.8}); err != nil {
		t.Fatalf("Save(old pending ai) error = %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: 999, TagID: keepConfirmedAI.ID, Source: domain.ImageTagSourceAI, SourceObservationID: &confirmedObs.ID, ReviewState: "confirmed", Confidence: 0.9}); err != nil {
		t.Fatalf("Save(confirmed ai) error = %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: 999, TagID: oldRejected.ID, Source: domain.ImageTagSourceAI, SourceObservationID: &rejectedObs.ID, ReviewState: "rejected", Confidence: 0.2}); err != nil {
		t.Fatalf("Save(rejected ai) error = %v", err)
	}

	mockClient := &mockAIClient{
		result: &ai.TagResult{
			Tags:       []string{"new-generated", "keep-manual", "keep-confirmed-ai"},
			Confidence: 0.91,
			ModelName:  "qwen-vl-max",
		},
	}

	handler := NewAITagRegenerationJobHandler(mockClient, obsRepo, governance)
	payloadBytes, _ := json.Marshal(AITagPayload{ImageID: 999, Path: "/images/test.png"})
	if err := handler(ctx, 7, string(payloadBytes)); err != nil {
		t.Fatalf("regeneration handler failed: %v", err)
	}

	items, err := imageTagRepo.FindByImageID(ctx, 999)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	states := make(map[int64]domain.ImageTag)
	for _, item := range items {
		states[item.TagID] = *item
	}
	if _, exists := states[oldPending.ID]; exists {
		t.Fatalf("old pending AI tag should be removed during regeneration: %+v", states[oldPending.ID])
	}
	if _, exists := states[oldRejected.ID]; exists {
		t.Fatalf("old rejected AI tag should be removed during regeneration: %+v", states[oldRejected.ID])
	}
	if states[keepManual.ID].ReviewState != "confirmed" || states[keepManual.ID].Source != domain.ImageTagSourceManual {
		t.Fatalf("manual confirmed tag changed unexpectedly: %+v", states[keepManual.ID])
	}
	if states[keepConfirmedAI.ID].ReviewState != "confirmed" || states[keepConfirmedAI.ID].Source != domain.ImageTagSourceAI {
		t.Fatalf("confirmed AI tag changed unexpectedly: %+v", states[keepConfirmedAI.ID])
	}
	newTag, err := tagRepo.FindByLabel(ctx, "new-generated")
	if err != nil {
		t.Fatalf("FindByLabel(new-generated) error = %v", err)
	}
	if states[newTag.ID].ReviewState != "pending" || states[newTag.ID].Source != domain.ImageTagSourceAI {
		t.Fatalf("new regenerated tag not saved as pending AI: %+v", states[newTag.ID])
	}
}

func TestBatchAITagHandler_InvalidPayload(t *testing.T) {
	// Test: 处理无效 payload
	mockJobRepo := &mockJobRepoForAI{}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{}

	manager := NewManager(mockJobRepo)
	RegisterBatchAITagHandler(manager, mockJobRepo, mockClient, mockObsRepo, &mockTagGovernanceService{}, nil, nil, nil, "auto")

	handler := manager.handlers["ai_tag_generation"]
	err := handler(context.Background(), 1, "invalid json")
	if err == nil {
		t.Error("expected error for invalid payload, got nil")
	}
}

func TestBatchAITagHandler_AIServiceError(t *testing.T) {
	// Test: AI 服务错误处理
	mockJobRepo := &mockJobRepoForAI{}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{
		err: context.DeadlineExceeded,
	}

	manager := NewManager(mockJobRepo)
	RegisterBatchAITagHandler(manager, mockJobRepo, mockClient, mockObsRepo, &mockTagGovernanceService{}, nil, nil, nil, "auto")

	payload := AITagPayload{
		ImageID: 1,
		Path:    "/test.jpg",
	}
	payloadBytes, _ := json.Marshal(payload)

	handler := manager.handlers["ai_tag_generation"]
	err := handler(context.Background(), 1, string(payloadBytes))
	if err == nil {
		t.Error("expected error from AI service, got nil")
	}
}

func TestBatchAITagHandler_RejectsErrorLikeModelOutput(t *testing.T) {
	db := mustOpenAIWorkerDB(t)
	seedAIWorkerImage(t, db, 777)

	obsRepo := repository.NewTagObservationRepository(db)
	tagRepo := repository.NewTagRepository(db)
	aliasRepo := repository.NewTagAliasRepository(db)
	imageTagRepo := repository.NewImageTagRepository(db)
	governance := service.NewTagGovernanceService(tagRepo, aliasRepo, obsRepo, imageTagRepo)
	mockClient := &mockAIClient{
		result: &ai.TagResult{
			Tags:        []string{"无法分析图片内容，因当前输入中未提供任何图像或图片描述。请上传或描述图片内容后重试。"},
			Confidence:  0.85,
			ModelName:   "qwen-plus",
			RawResponse: `{"choices":[{"message":{"content":"无法分析图片内容，因当前输入中未提供任何图像或图片描述。请上传或描述图片内容后重试。"}}]}`,
		},
	}

	handler := NewBatchAITagJobHandler(&mockJobRepoForAI{}, mockClient, obsRepo, governance, nil, nil, nil, "auto")
	payloadBytes, _ := json.Marshal(AITagPayload{ImageID: 777, Path: "/images/test.png"})
	err := handler(context.Background(), 9, string(payloadBytes))
	if err == nil || !strings.Contains(err.Error(), "validate tags") {
		t.Fatalf("expected validate tags error, got %v", err)
	}

	observations, err := obsRepo.FindByImageID(context.Background(), 777)
	if err != nil {
		t.Fatalf("FindByImageID() observations error = %v", err)
	}
	if len(observations) != 0 {
		t.Fatalf("expected 0 observations for invalid AI output, got %d", len(observations))
	}
	imageTags, err := imageTagRepo.FindByImageID(context.Background(), 777)
	if err != nil {
		t.Fatalf("FindByImageID() image tags error = %v", err)
	}
	if len(imageTags) != 0 {
		t.Fatalf("expected 0 image tags for invalid AI output, got %d", len(imageTags))
	}
}

func TestBatchAITagHandler_SingleModeDoesNotClaimExtraJobsAndUsesGenerateTags(t *testing.T) {
	mockJobRepo := &mockJobRepoForAI{extraJobs: []domain.AsyncJob{{ID: 2, Payload: `{"image_id":2,"path":"/extra.jpg"}`}}}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{result: &ai.TagResult{Tags: []string{"girl", "anime"}, Confidence: 0.9, ModelName: "doubao"}}
	governance := &mockTagGovernanceService{}

	handler := NewBatchAITagJobHandler(mockJobRepo, mockClient, mockObsRepo, governance, nil, nil, nil, "single")
	payloadBytes, _ := json.Marshal(AITagPayload{ImageID: 1, Path: "/single.jpg"})
	if err := handler(context.Background(), 1, string(payloadBytes)); err != nil {
		t.Fatalf("handler failed: %v", err)
	}
	if mockJobRepo.claimCalls != 0 {
		t.Fatalf("expected no claim calls in single mode, got %d", mockJobRepo.claimCalls)
	}
	if mockClient.generateTagsCalls != 1 || mockClient.generateTagsBatchCalls != 0 {
		t.Fatalf("expected single GenerateTags call, got generate=%d batch=%d", mockClient.generateTagsCalls, mockClient.generateTagsBatchCalls)
	}
}

func TestBatchAITagHandler_AutoModeMultiRequestUsesGenerateTagsBatch(t *testing.T) {
	mockJobRepo := &mockJobRepoForAI{extraJobs: []domain.AsyncJob{{ID: 2, Payload: `{"image_id":2,"path":"/extra.jpg"}`}}}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{result: &ai.TagResult{Tags: []string{"girl", "anime"}, Confidence: 0.9, ModelName: "doubao"}}
	governance := &mockTagGovernanceService{}

	handler := NewBatchAITagJobHandler(mockJobRepo, mockClient, mockObsRepo, governance, nil, nil, nil, "auto")
	payloadBytes, _ := json.Marshal(AITagPayload{ImageID: 1, Path: "/single.jpg"})
	if err := handler(context.Background(), 1, string(payloadBytes)); err != nil {
		t.Fatalf("handler failed: %v", err)
	}
	if mockJobRepo.claimCalls != 1 {
		t.Fatalf("expected claim call in auto mode, got %d", mockJobRepo.claimCalls)
	}
	if mockClient.generateTagsCalls != 0 || mockClient.generateTagsBatchCalls != 1 {
		t.Fatalf("expected batch call in auto mode with extra jobs, got generate=%d batch=%d", mockClient.generateTagsCalls, mockClient.generateTagsBatchCalls)
	}
}

func TestBatchAITagHandler_MultiModeSingleRequestUsesGenerateTagsBatch(t *testing.T) {
	mockJobRepo := &mockJobRepoForAI{}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{result: &ai.TagResult{Tags: []string{"girl", "anime"}, Confidence: 0.9, ModelName: "doubao"}}
	governance := &mockTagGovernanceService{}

	handler := NewBatchAITagJobHandler(mockJobRepo, mockClient, mockObsRepo, governance, nil, nil, nil, "multi")
	payloadBytes, _ := json.Marshal(AITagPayload{ImageID: 1, Path: "/single.jpg"})
	if err := handler(context.Background(), 1, string(payloadBytes)); err != nil {
		t.Fatalf("handler failed: %v", err)
	}
	if mockClient.generateTagsCalls != 0 || mockClient.generateTagsBatchCalls != 1 {
		t.Fatalf("expected forced batch call in multi mode, got generate=%d batch=%d", mockClient.generateTagsCalls, mockClient.generateTagsBatchCalls)
	}
}

func TestBatchAITagHandler_NonDoubaoProviderIgnoresDoubaoBatchMode(t *testing.T) {
	mockJobRepo := &mockJobRepoForAI{}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{
		name:   "qwen",
		result: &ai.TagResult{Tags: []string{"girl", "anime"}, Confidence: 0.9, ModelName: "qwen-vl-max"},
	}
	governance := &mockTagGovernanceService{}

	handler := NewBatchAITagJobHandler(mockJobRepo, mockClient, mockObsRepo, governance, nil, nil, nil, "single")
	payloadBytes, _ := json.Marshal(AITagPayload{ImageID: 1, Path: "/single.jpg"})
	if err := handler(context.Background(), 1, string(payloadBytes)); err != nil {
		t.Fatalf("handler failed: %v", err)
	}
	if mockJobRepo.claimCalls != 1 {
		t.Fatalf("expected non-doubao provider to ignore doubao_batch_mode and still claim, got %d", mockJobRepo.claimCalls)
	}
}

func TestBatchAITagHandler_SkipsWhenImageAlreadyHasAITags(t *testing.T) {
	mockJobRepo := &mockJobRepoForAI{}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{
		result: &ai.TagResult{Tags: []string{"blue hair", "girl", "outdoors"}, Confidence: 0.95, ModelName: "doubao-vision-pro"},
	}
	governance := &mockTagGovernanceService{}
	checker := &mockAITagPresenceChecker{hasAITags: true}

	handler := NewBatchAITagJobHandler(mockJobRepo, mockClient, mockObsRepo, governance, nil, nil, checker, "auto")
	payloadBytes, _ := json.Marshal(AITagPayload{ImageID: 789, Path: "/test/image.png"})
	if err := handler(context.Background(), 3, string(payloadBytes)); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if governance.called {
		t.Fatal("expected governance merge to be skipped when image already has AI tags")
	}
	if mockClient.generateTagsCalls != 0 || mockClient.generateTagsBatchCalls != 0 {
		t.Fatalf("expected no AI calls when image already has AI tags, got generate=%d batch=%d", mockClient.generateTagsCalls, mockClient.generateTagsBatchCalls)
	}
}

func TestBatchAITagHandler_DoesNotOverwriteSkippedJobsWhenBatchFails(t *testing.T) {
	mockJobRepo := &mockJobRepoForAI{extraJobs: []domain.AsyncJob{{ID: 2, Payload: `{"image_id":2,"path":"/extra.jpg"}`}}}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{
		result: &ai.TagResult{Tags: []string{"girl", "anime"}, Confidence: 0.9, ModelName: "doubao"},
		err:    errors.New("upstream failed"),
	}
	governance := &mockTagGovernanceService{}
	checker := &mockAITagPresenceChecker{hasAITagsForImage: map[int64]bool{1: true}}
	platformSvc := &mockBatchPlatformService{}

	handler := NewBatchAITagJobHandler(mockJobRepo, mockClient, mockObsRepo, governance, platformSvc, nil, checker, "auto")
	payloadBytes, _ := json.Marshal(AITagPayload{ImageID: 1, Path: "/single.jpg"})
	err := handler(context.Background(), 1, string(payloadBytes))
	if err == nil {
		t.Fatal("expected batch failure")
	}
	if got := mockJobRepo.updatesByID[1].Status; got != "finished" {
		t.Fatalf("expected skipped triggering job to stay finished, got %q", got)
	}
	if mockJobRepo.updatesByID[1].Error != nil {
		t.Fatalf("expected skipped triggering job to keep nil error, got %v", *mockJobRepo.updatesByID[1].Error)
	}
	if got := mockJobRepo.updatesByID[2].Status; got != "failed" {
		t.Fatalf("expected requested sibling to fail, got %q", got)
	}
	if len(platformSvc.completed) != 1 || platformSvc.completed[0][0] != 1 {
		t.Fatalf("expected skipped job completion sync, got %+v", platformSvc.completed)
	}
	if len(platformSvc.failed) != 1 || len(platformSvc.failed[0]) != 1 || platformSvc.failed[0][0] != 2 {
		t.Fatalf("expected only request job failure sync, got %+v", platformSvc.failed)
	}
}

func TestBatchAITagHandler_MarksInvalidClaimedPayloadJobsFailed(t *testing.T) {
	mockJobRepo := &mockJobRepoForAI{extraJobs: []domain.AsyncJob{{ID: 2, Payload: `not-json`}}}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{result: &ai.TagResult{Tags: []string{"girl", "anime"}, Confidence: 0.9, ModelName: "doubao"}}
	governance := &mockTagGovernanceService{}
	platformSvc := &mockBatchPlatformService{}

	handler := NewBatchAITagJobHandler(mockJobRepo, mockClient, mockObsRepo, governance, platformSvc, nil, nil, "auto")
	payloadBytes, _ := json.Marshal(AITagPayload{ImageID: 1, Path: "/single.jpg"})
	if err := handler(context.Background(), 1, string(payloadBytes)); err != nil {
		t.Fatalf("handler failed: %v", err)
	}
	if got := mockJobRepo.updatesByID[2].Status; got != "failed" {
		t.Fatalf("expected invalid claimed payload job to fail, got %q", got)
	}
	if mockJobRepo.updatesByID[2].Error == nil || !strings.Contains(*mockJobRepo.updatesByID[2].Error, "parse payload") {
		t.Fatalf("expected parse payload error on invalid claimed job, got %+v", mockJobRepo.updatesByID[2].Error)
	}
	if len(platformSvc.failed) != 1 || len(platformSvc.failed[0]) != 1 || platformSvc.failed[0][0] != 2 {
		t.Fatalf("expected invalid claimed payload job failure sync, got %+v", platformSvc.failed)
	}
	if got := mockJobRepo.updatesByID[1].Status; got != "finished" {
		t.Fatalf("expected triggering job to finish successfully, got %q", got)
	}
}

func TestBatchAITagHandler_SyncsInvalidClaimedPayloadFailuresBeforeAIError(t *testing.T) {
	mockJobRepo := &mockJobRepoForAI{extraJobs: []domain.AsyncJob{{ID: 2, Payload: `not-json`}}}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{
		result: &ai.TagResult{Tags: []string{"girl", "anime"}, Confidence: 0.9, ModelName: "doubao"},
		err:    errors.New("upstream failed"),
	}
	governance := &mockTagGovernanceService{}
	platformSvc := &mockBatchPlatformService{}

	handler := NewBatchAITagJobHandler(mockJobRepo, mockClient, mockObsRepo, governance, platformSvc, nil, nil, "auto")
	payloadBytes, _ := json.Marshal(AITagPayload{ImageID: 1, Path: "/single.jpg"})
	err := handler(context.Background(), 1, string(payloadBytes))
	if err == nil {
		t.Fatal("expected upstream failure")
	}
	if len(platformSvc.failed) != 2 {
		t.Fatalf("expected two failure sync calls, got %+v", platformSvc.failed)
	}
	if len(platformSvc.failed[0]) != 1 || platformSvc.failed[0][0] != 2 {
		t.Fatalf("expected invalid payload job synced first, got %+v", platformSvc.failed)
	}
	if len(platformSvc.failed[1]) != 1 || platformSvc.failed[1][0] != 1 {
		t.Fatalf("expected triggering request job synced on AI failure, got %+v", platformSvc.failed)
	}
}

func TestBatchAITagHandler_TriggeringJobPostProcessingFailureReturnsError(t *testing.T) {
	mockJobRepo := &mockJobRepoForAI{AsyncJob: domain.AsyncJob{ID: 1, Type: "ai_tag_generation", Status: "running", Payload: `{"image_id":1,"path":"/single.jpg"}`}}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{result: &ai.TagResult{Tags: []string{"girl", "anime"}, Confidence: 0.9, ModelName: "doubao"}}
	governance := &mockTagGovernanceService{err: errors.New("merge failed")}
	platformSvc := &mockBatchPlatformService{}

	handler := NewBatchAITagJobHandler(mockJobRepo, mockClient, mockObsRepo, governance, platformSvc, nil, nil, "auto")
	payloadBytes, _ := json.Marshal(AITagPayload{ImageID: 1, Path: "/single.jpg"})
	err := handler(context.Background(), 1, string(payloadBytes))
	if err == nil || !strings.Contains(err.Error(), "merge failed") {
		t.Fatalf("expected triggering job failure to return merge error, got %v", err)
	}
	if len(platformSvc.failed) != 1 || platformSvc.failed[0][0] != 1 {
		t.Fatalf("expected failed sync for triggering job, got %+v", platformSvc.failed)
	}
}

func TestBatchAITagHandler_OnlyClaimsSiblingJobsFromSameBatch(t *testing.T) {
	triggerTaskID := int64(101)
	sameBatchTaskID := int64(102)
	otherBatchTaskID := int64(201)
	mockJobRepo := &mockJobRepoForAI{
		AsyncJob: domain.AsyncJob{ID: 1, PlatformTaskID: &triggerTaskID, Type: "ai_tag_generation", Status: "running", Payload: `{"image_id":1,"path":"/trigger.jpg"}`},
		extraJobs: []domain.AsyncJob{
			{ID: 2, PlatformTaskID: &sameBatchTaskID, Type: "ai_tag_generation", Status: "running", Payload: `{"image_id":2,"path":"/same.jpg"}`},
			{ID: 3, PlatformTaskID: &otherBatchTaskID, Type: "ai_tag_generation", Status: "running", Payload: `{"image_id":3,"path":"/other.jpg"}`},
		},
	}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{result: &ai.TagResult{Tags: []string{"girl", "anime"}, Confidence: 0.9, ModelName: "doubao"}}
	governance := &mockTagGovernanceService{}
	taskRepo := &mockPlatformTaskRepo{tasks: map[int64]*domain.PlatformTask{
		triggerTaskID:    {ID: triggerTaskID, BatchID: 1000, TaskType: domain.PlatformTaskTypeAITagGeneration},
		sameBatchTaskID:  {ID: sameBatchTaskID, BatchID: 1000, TaskType: domain.PlatformTaskTypeAITagGeneration},
		otherBatchTaskID: {ID: otherBatchTaskID, BatchID: 2000, TaskType: domain.PlatformTaskTypeAITagGeneration},
	}}

	handler := NewBatchAITagJobHandler(mockJobRepo, mockClient, mockObsRepo, governance, nil, taskRepo, nil, "auto")
	payloadBytes, _ := json.Marshal(AITagPayload{ImageID: 1, Path: "/trigger.jpg"})
	if err := handler(context.Background(), 1, string(payloadBytes)); err != nil {
		t.Fatalf("handler failed: %v", err)
	}
	if mockClient.generateTagsBatchCalls != 1 {
		t.Fatalf("expected one batch call, got %d", mockClient.generateTagsBatchCalls)
	}
	if len(mockClient.lastBatchRequests) != 2 {
		t.Fatalf("expected trigger + same-batch sibling only, got %+v", mockClient.lastBatchRequests)
	}
	if updated, ok := mockJobRepo.updatesByID[3]; !ok || updated.Status != "ready" || updated.StartedAt != nil {
		t.Fatalf("expected other-batch job released back to ready, got %+v", updated)
	}
}

// ========== Mocks ==========

type mockJobRepoForAI struct {
	domain.AsyncJob
	extraJobs   []domain.AsyncJob
	claimCalls  int
	updatesByID map[int64]domain.AsyncJob
}

func (m *mockJobRepoForAI) Save(job *domain.AsyncJob) error {
	m.AsyncJob = *job
	job.ID = 1
	return nil
}
func (m *mockJobRepoForAI) FindByID(id int64) (*domain.AsyncJob, error) {
	if m.AsyncJob.ID == 0 || m.AsyncJob.ID != id {
		return nil, sql.ErrNoRows
	}
	job := m.AsyncJob
	return &job, nil
}
func (m *mockJobRepoForAI) FindByPlatformTaskID(platformTaskID int64) ([]domain.AsyncJob, error) {
	return nil, nil
}
func (m *mockJobRepoForAI) FindByStatus(status string) ([]domain.AsyncJob, error) {
	return nil, nil
}
func (m *mockJobRepoForAI) FindByType(jobType string) ([]domain.AsyncJob, error) {
	return nil, nil
}
func (m *mockJobRepoForAI) Update(job *domain.AsyncJob) error {
	m.AsyncJob = *job
	if m.updatesByID == nil {
		m.updatesByID = make(map[int64]domain.AsyncJob)
	}
	m.updatesByID[job.ID] = *job
	return nil
}
func (m *mockJobRepoForAI) FindRecent(limit int) ([]domain.AsyncJob, error) {
	return nil, nil
}
func (m *mockJobRepoForAI) FindFailed() ([]domain.AsyncJob, error) {
	return nil, nil
}
func (m *mockJobRepoForAI) FindByTypeAndStatus(jobType string, status string) ([]domain.AsyncJob, error) {
	return nil, nil
}
func (m *mockJobRepoForAI) UpdateStatus(id int64, status string, errorMsg *string) error {
	return nil
}
func (m *mockJobRepoForAI) CountByStatus(status string) (int64, error) {
	return 0, nil
}
func (m *mockJobRepoForAI) ResetRunningToReady() (int64, error) {
	return 0, nil
}
func (m *mockJobRepoForAI) CasPendingToRunning(ctx context.Context, jobType string, limit int) ([]domain.AsyncJob, error) {
	return nil, nil
}
func (m *mockJobRepoForAI) FindAndClaimReadyJobs(ctx context.Context, jobType string, limit int) ([]domain.AsyncJob, error) {
	m.claimCalls++
	if len(m.extraJobs) == 0 {
		return nil, nil
	}
	if limit > len(m.extraJobs) {
		limit = len(m.extraJobs)
	}
	claimed := append([]domain.AsyncJob(nil), m.extraJobs[:limit]...)
	m.extraJobs = m.extraJobs[limit:]
	return claimed, nil
}

type mockTagObservationRepo struct {
	savedObservation *domain.TagObservation
}

func (m *mockTagObservationRepo) Save(ctx context.Context, obs *domain.TagObservation) error {
	m.savedObservation = obs
	obs.ID = 1
	return nil
}
func (m *mockTagObservationRepo) FindByImageID(ctx context.Context, imageID int64) ([]*domain.TagObservation, error) {
	return nil, nil
}
func (m *mockTagObservationRepo) FindByProvider(ctx context.Context, provider string, limit int) ([]*domain.TagObservation, error) {
	return nil, nil
}
func (m *mockTagObservationRepo) FindByID(ctx context.Context, id int64) (*domain.TagObservation, error) {
	return nil, nil
}

type mockTagGovernanceService struct {
	called        bool
	imageID       int64
	tags          []string
	observationID int64
	confidence    float64
	err           error
}

func (m *mockTagGovernanceService) MergeTags(ctx context.Context, imageID int64, tags []string, observationID int64, confidence float64) error {
	m.called = true
	m.imageID = imageID
	m.tags = append([]string(nil), tags...)
	m.observationID = observationID
	m.confidence = confidence
	return m.err
}

type mockAIClient struct {
	result                 *ai.TagResult
	err                    error
	lastImageURL           string
	lastBatchRequests      []ai.TagRequest
	generateTagsCalls      int
	generateTagsBatchCalls int
	name                   string
}

type mockAITagPresenceChecker struct {
	hasAITags         bool
	hasAITagsForImage map[int64]bool
	err               error
}

func (m *mockAITagPresenceChecker) HasAITags(ctx context.Context, imageID int64) (bool, error) {
	if m.hasAITagsForImage != nil {
		return m.hasAITagsForImage[imageID], m.err
	}
	return m.hasAITags, m.err
}

type mockBatchPlatformService struct {
	completed [][]int64
	failed    [][]int64
	errors    []string
}

type mockPlatformTaskRepo struct {
	tasks map[int64]*domain.PlatformTask
}

func (m *mockPlatformTaskRepo) FindByID(ctx context.Context, taskID int64) (*domain.PlatformTask, error) {
	if m.tasks == nil {
		return nil, sql.ErrNoRows
	}
	task, ok := m.tasks[taskID]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return task, nil
}

func (m *mockBatchPlatformService) MarkJobsCompleted(ctx context.Context, jobIDs []int64) error {
	m.completed = append(m.completed, append([]int64(nil), jobIDs...))
	return nil
}

func (m *mockBatchPlatformService) MarkJobsFailed(ctx context.Context, jobIDs []int64, errorSync string) error {
	m.failed = append(m.failed, append([]int64(nil), jobIDs...))
	m.errors = append(m.errors, errorSync)
	return nil
}

func (m *mockAIClient) GenerateTagsBatch(ctx context.Context, requests []ai.TagRequest) (*ai.BatchTagResult, error) {
	m.generateTagsBatchCalls++
	m.lastBatchRequests = append([]ai.TagRequest(nil), requests...)
	groups := make([][]string, len(requests))
	for i := range groups {
		groups[i] = m.result.Tags
	}
	return &ai.BatchTagResult{
		Groups:      groups,
		ModelName:   m.result.ModelName,
		RawResponse: m.result.RawResponse,
	}, m.err
}
func (m *mockAIClient) Name() string {
	if m.name != "" {
		return m.name
	}
	return "doubao"
}

func (m *mockAIClient) GenerateTags(ctx context.Context, imageURL, prompt string) (*ai.TagResult, error) {
	m.generateTagsCalls++
	m.lastImageURL = imageURL
	return m.result, m.err
}

func mustOpenAIWorkerDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "ai-worker.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	return db
}

func seedAIWorkerImage(t *testing.T, db *sql.DB, imageID int64) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO images (id, path, filename, source_root, file_size, width, height, format, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, imageID, "/images/test.png", "test.png", "/images", 100, 100, 100, "png", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("seed images: %v", err)
	}
}
