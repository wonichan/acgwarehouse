package sidecar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type SidecarClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewSidecarClient(baseURL string) *SidecarClient {
	return &SidecarClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type DetectionImageInput struct {
	ID       int64  `json:"id"`
	Path     string `json:"path"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FileSize int64  `json:"file_size"`
	Format   string `json:"format"`
}

type DetectionRequest struct {
	Threshold int                   `json:"threshold"`
	Images    []DetectionImageInput `json:"images"`
}

type DetectionTaskStatus struct {
	TaskID   string  `json:"task_id"`
	Status   string  `json:"status"`
	Progress float64 `json:"progress"`
	Message  string  `json:"message"`
}

type DetectionResultMember struct {
	ImageID               int64                    `json:"image_id"`
	SHA256                string                   `json:"sha256"`
	PHash                 string                   `json:"phash"`
	Distance              int                      `json:"distance"`
	IsRecommended         bool                     `json:"is_recommended"`
	RecommendationScore   float64                  `json:"recommendation_score"`
	RecommendationReasons []map[string]interface{} `json:"recommendation_reasons"`
}

type DetectionResultGroup struct {
	GroupID       int                     `json:"group_id"`
	RecommendedID int64                   `json:"recommended_id"`
	Members       []DetectionResultMember `json:"members"`
}

type DetectionResult struct {
	Groups            []DetectionResultGroup   `json:"groups"`
	TotalImages       int                      `json:"total_images"`
	TotalGroups       int                      `json:"total_groups"`
	SkippedImages     []map[string]interface{} `json:"skipped_images"`
	ComputationTimeMs int                      `json:"computation_time_ms"`
}

func (c *SidecarClient) SubmitDetection(ctx context.Context, req DetectionRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal detection request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/compute/duplicates/detect", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build detection request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("sidecar detection request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("duplicate detection task already running (HTTP 409): %s", strings.TrimSpace(string(respBody)))
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("sidecar returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result DetectionTaskStatus
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode detection task response: %w", err)
	}

	if result.TaskID == "" {
		return "", fmt.Errorf("sidecar returned empty task id")
	}

	return result.TaskID, nil
}

func (c *SidecarClient) PollProgress(ctx context.Context, taskID string) (*DetectionTaskStatus, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/compute/duplicates/tasks/"+taskID, nil)
	if err != nil {
		return nil, fmt.Errorf("build poll request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sidecar poll request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sidecar returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var status DetectionTaskStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("decode poll response: %w", err)
	}

	return &status, nil
}

func (c *SidecarClient) FetchResults(ctx context.Context, taskID string) (*DetectionResult, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/compute/duplicates/tasks/"+taskID+"/result", nil)
	if err != nil {
		return nil, fmt.Errorf("build result request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sidecar result request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sidecar returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result DetectionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode result response: %w", err)
	}

	return &result, nil
}
