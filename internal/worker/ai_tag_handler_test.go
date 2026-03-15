package worker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/wonichan/acgwarehouse-backend/internal/ai"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
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
	RegisterAITagHandler(manager, mockClient, mockObsRepo)

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
	RegisterAITagHandler(manager, mockClient, mockObsRepo)

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
	RegisterAITagHandler(manager, mockClient, mockObsRepo)

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

func TestAITagHandler_InvalidPayload(t *testing.T) {
	// Test: 处理无效 payload
	mockJobRepo := &mockJobRepoForAI{}
	mockObsRepo := &mockTagObservationRepo{}
	mockClient := &mockAIClient{}

	manager := NewManager(mockJobRepo)
	RegisterAITagHandler(manager, mockClient, mockObsRepo)

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
	RegisterAITagHandler(manager, mockClient, mockObsRepo)

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
func (m *mockJobRepoForAI) Update(job *domain.AsyncJob) error {
	m.AsyncJob = *job
	return nil
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
