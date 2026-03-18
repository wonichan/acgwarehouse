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
	RegisterAITagHandler(manager, mockClient, mockObsRepo, &mockTagGovernanceService{})

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
	RegisterAITagHandler(manager, mockClient, mockObsRepo, governance)

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
	RegisterAITagHandler(manager, mockClient, mockObsRepo, &mockTagGovernanceService{})

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
	RegisterAITagHandler(manager, mockClient, obsRepo, governance)

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
		if imageTag.SourceObservationID == nil || *imageTag.SourceObservationID != observations[0].ID {
			t.Fatalf("image tag source observation = %v, want %d", imageTag.SourceObservationID, observations[0].ID)
		}
	}
}

func TestAITagHandler_InvalidPayload(t *testing.T) {
	// Test: 处理无效 payload
	mockJobRepo := &mockJobRepoForAI{}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{}

	manager := NewManager(mockJobRepo)
	RegisterAITagHandler(manager, mockClient, mockObsRepo, &mockTagGovernanceService{})

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
	RegisterAITagHandler(manager, mockClient, mockObsRepo, &mockTagGovernanceService{})

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
func (m *mockJobRepoForAI) UpdateStatus(id int64, status string, errorMsg *string) error {
	return nil
}
func (m *mockJobRepoForAI) CountByStatus(status string) (int64, error) {
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

func (m *mockAIClient) Name() string {
	return "mock"
}

func (m *mockAIClient) GenerateTags(ctx interface{}, imageURL, prompt string) (*ai.TagResult, error) {
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
