package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/ai"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

func TestRegisterAITagHandler_Registration(t *testing.T) {
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
	RegisterAITagHandler(manager, mockClient, mockObsRepo, &mockTagGovernanceService{}, nil)

	// 验证处理器已注册
	if _, ok := manager.handlers["ai_tag_generation"]; !ok {
		t.Error("expected ai_tag_generation handler to be registered")
	}
}

func TestAITagHandler_ParsesPayload(t *testing.T) {
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
	RegisterAITagHandler(manager, mockClient, mockObsRepo, governance, nil)

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

func TestAITagHandler_SavesObservation(t *testing.T) {
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
	RegisterAITagHandler(manager, mockClient, mockObsRepo, &mockTagGovernanceService{}, nil)

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
	if obs.Provider != "mock" {
		t.Errorf("expected provider 'mock', got %s", obs.Provider)
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

func TestAITagHandler_PersistsPendingImageTagsForReview(t *testing.T) {
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

	manager := NewManager(&mockJobRepoForAI{})
	RegisterAITagHandler(manager, mockClient, obsRepo, governance, imageTagRepo)

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

func TestAITagHandler_RemovesRejectedAITagsBeforeSavingFreshResults(t *testing.T) {
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

	manager := NewManager(&mockJobRepoForAI{})
	RegisterAITagHandler(manager, mockClient, obsRepo, governance, imageTagRepo)

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

func TestAITagHandler_SkipsWhenImageAlreadyHasAITags(t *testing.T) {
	mockJobRepo := &mockJobRepoForAI{}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{
		result: &ai.TagResult{
			Tags:       []string{"blue hair", "girl", "outdoors"},
			Confidence: 0.95,
			ModelName:  "doubao-vision-pro",
		},
	}

	manager := NewManager(mockJobRepo)
	governance := &mockTagGovernanceService{}
	aiTagChecker := &mockAITagPresenceChecker{hasAITags: true}
	RegisterAITagHandler(manager, mockClient, mockObsRepo, governance, aiTagChecker)

	payloadBytes, _ := json.Marshal(AITagPayload{ImageID: 789, Path: "/test/image.png"})
	handler := manager.handlers["ai_tag_generation"]
	if err := handler(context.Background(), 3, string(payloadBytes)); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if mockClient.lastImageURL != "" {
		t.Fatalf("expected AI client not to be called, got %q", mockClient.lastImageURL)
	}
	if mockObsRepo.savedObservation != nil {
		t.Fatal("expected observation not to be saved when AI tags already exist")
	}
	if governance.called {
		t.Fatal("expected governance merge not to be called when AI tags already exist")
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

func TestAITagHandler_InvalidPayload(t *testing.T) {
	// Test: 处理无效 payload
	mockJobRepo := &mockJobRepoForAI{}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{}

	manager := NewManager(mockJobRepo)
	RegisterAITagHandler(manager, mockClient, mockObsRepo, &mockTagGovernanceService{}, nil)

	handler := manager.handlers["ai_tag_generation"]
	err := handler(context.Background(), 1, "invalid json")
	if err == nil {
		t.Error("expected error for invalid payload, got nil")
	}
}

func TestAITagHandler_AIServiceError(t *testing.T) {
	// Test: AI 服务错误处理
	mockJobRepo := &mockJobRepoForAI{}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{
		err: context.DeadlineExceeded,
	}

	manager := NewManager(mockJobRepo)
	RegisterAITagHandler(manager, mockClient, mockObsRepo, &mockTagGovernanceService{}, nil)

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

func TestAITagHandler_FailsFastWhenGovernanceNil(t *testing.T) {
	mockJobRepo := &mockJobRepoForAI{}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{
		result: &ai.TagResult{
			Tags:       []string{"girl", "anime"},
			Confidence: 0.88,
			ModelName:  "qwen-vl-max",
		},
	}

	manager := NewManager(mockJobRepo)
	RegisterAITagHandler(manager, mockClient, mockObsRepo, nil, nil)

	payload := AITagPayload{ImageID: 1, Path: "/test.jpg"}
	payloadBytes, _ := json.Marshal(payload)

	handler := manager.handlers["ai_tag_generation"]
	err := handler(context.Background(), 1, string(payloadBytes))
	if err == nil {
		t.Fatal("expected error when governance is nil")
	}

	if mockClient.lastImageURL != "" {
		t.Fatal("expected AI client not to be called when governance is nil")
	}

	if mockObsRepo.savedObservation != nil {
		t.Fatal("expected observation not to be saved when governance is nil")
	}
}

// ========== Mocks ==========

type mockJobRepoForAI struct {
	domain.AsyncJob
}

func (m *mockJobRepoForAI) Save(job *domain.AsyncJob) error {
	m.AsyncJob = *job
	job.ID = 1
	return nil
}
func (m *mockJobRepoForAI) FindByID(id int64) (*domain.AsyncJob, error) { return &m.AsyncJob, nil }
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
	result       *ai.TagResult
	err          error
	lastImageURL string
}

type mockAITagPresenceChecker struct {
	hasAITags bool
	err       error
}

func (m *mockAITagPresenceChecker) HasAITags(ctx context.Context, imageID int64) (bool, error) {
	return m.hasAITags, m.err
}

func (m *mockAIClient) Name() string {
	return "mock"
}

func (m *mockAIClient) GenerateTags(ctx context.Context, imageURL, prompt string) (*ai.TagResult, error) {
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
