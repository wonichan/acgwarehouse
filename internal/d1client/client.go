package d1client

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

type D1Response struct {
	Success bool             `json:"success"`
	Meta    map[string]any   `json:"meta"`
	Results []map[string]any `json:"results"`
	Error   string           `json:"error,omitempty"`
}

type MutateResponse struct {
	Success bool             `json:"success"`
	Meta    map[string]any   `json:"meta"`
	Results []map[string]any `json:"results"`
	Error   string           `json:"error,omitempty"`
}

type QueryRequest struct {
	SQL    string `json:"sql"`
	Params []any  `json:"params,omitempty"`
}

type MutateRequest struct {
	SQL        string            `json:"sql,omitempty"`
	Params     []any             `json:"params,omitempty"`
	Statements []MutateStatement `json:"statements,omitempty"`
}

type MutateStatement struct {
	SQL    string `json:"sql"`
	Params []any  `json:"params,omitempty"`
}

type Client struct {
	baseURL    string
	apiKey     string
	readOnly   bool
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func NewClientWithAPIKey(baseURL, apiKey string) *Client {
	return NewClientWithAPIKeyAndReadOnly(baseURL, apiKey, false)
}

func NewClientWithAPIKeyAndReadOnly(baseURL, apiKey string, readOnly bool) *Client {
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		apiKey:   apiKey,
		readOnly: readOnly,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Query(ctx context.Context, sql string, params ...any) ([]map[string]any, error) {
	if err := validateReadOnly(sql); err != nil {
		return nil, err
	}

	resp, err := c.doQueryRequest(ctx, sql, params)
	if err != nil {
		return nil, fmt.Errorf("d1 query: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("d1 query failed: %s", resp.Error)
	}
	return resp.Results, nil
}

func (c *Client) QueryOne(ctx context.Context, sql string, params ...any) (map[string]any, error) {
	rows, err := c.Query(ctx, sql, params...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (c *Client) QueryCount(ctx context.Context, sql string, params ...any) (int64, error) {
	row, err := c.QueryOne(ctx, sql, params...)
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, nil
	}
	for _, v := range row {
		switch n := v.(type) {
		case float64:
			return int64(n), nil
		case int64:
			return n, nil
		case json.Number:
			i, err := n.Int64()
			if err != nil {
				return 0, fmt.Errorf("d1: parse count: %w", err)
			}
			return i, nil
		}
	}
	return 0, fmt.Errorf("d1: count query returned no numeric value")
}

func (c *Client) Exec(ctx context.Context, sql string, params ...any) error {
	if c.readOnly {
		return fmt.Errorf("d1 readonly mode rejects mutate query")
	}
	resp, err := c.doMutateRequest(ctx, MutateRequest{SQL: sql, Params: params})
	if err != nil {
		return fmt.Errorf("d1 exec: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("d1 exec failed: %s", resp.Error)
	}
	return nil
}

func (c *Client) ExecReturningID(ctx context.Context, sql string, params ...any) (int64, error) {
	if c.readOnly {
		return 0, fmt.Errorf("d1 readonly mode rejects mutate query")
	}
	resp, err := c.doMutateRequest(ctx, MutateRequest{SQL: sql, Params: params})
	if err != nil {
		return 0, fmt.Errorf("d1 exec returning id: %w", err)
	}
	if !resp.Success {
		return 0, fmt.Errorf("d1 exec returning id failed: %s", resp.Error)
	}

	if resp.Meta != nil {
		if lastRowID, ok := resp.Meta["last_row_id"]; ok {
			switch v := lastRowID.(type) {
			case float64:
				return int64(v), nil
			case int64:
				return v, nil
			case json.Number:
				return v.Int64()
			}
		}
		if changes, ok := resp.Meta["changes"]; ok {
			switch v := changes.(type) {
			case float64:
				if v == 0 {
					return 0, nil
				}
			case int64:
				if v == 0 {
					return 0, nil
				}
			}
		}
	}

	if len(resp.Results) > 0 {
		if id, ok := resp.Results[0]["id"]; ok {
			switch v := id.(type) {
			case float64:
				return int64(v), nil
			case int64:
				return v, nil
			}
		}
	}

	return 0, nil
}

func (c *Client) ExecBatch(ctx context.Context, statements []MutateStatement) error {
	if c.readOnly {
		return fmt.Errorf("d1 readonly mode rejects mutate query")
	}
	resp, err := c.doMutateRequest(ctx, MutateRequest{Statements: statements})
	if err != nil {
		return fmt.Errorf("d1 exec batch: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("d1 exec batch failed: %s", resp.Error)
	}
	return nil
}

func (c *Client) doQueryRequest(ctx context.Context, sql string, params []any) (*D1Response, error) {
	body := QueryRequest{
		SQL:    sql,
		Params: params,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/query", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("d1 query: unauthorized (check API key)")
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("d1 api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var d1Resp D1Response
	if err := json.NewDecoder(resp.Body).Decode(&d1Resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &d1Resp, nil
}

func (c *Client) doMutateRequest(ctx context.Context, reqBody MutateRequest) (*MutateResponse, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/mutate", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("d1 mutate: unauthorized (check API key)")
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("d1 api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var mutResp MutateResponse
	if err := json.NewDecoder(resp.Body).Decode(&mutResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &mutResp, nil
}

func validateReadOnly(sql string) error {
	normalized := strings.TrimSpace(strings.ToUpper(sql))
	if strings.HasPrefix(normalized, "SELECT") || isReadOnlyWithQuery(normalized) {
		return nil
	}
	forbidden := []string{"WITH", "INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE", "REPLACE", "ATTACH", "DETACH"}
	for _, prefix := range forbidden {
		if strings.HasPrefix(normalized, prefix) {
			return fmt.Errorf("d1: write operation not allowed: %s", prefix)
		}
	}
	return fmt.Errorf("d1: only SELECT queries are allowed")
}

func isReadOnlyWithQuery(normalized string) bool {
	if !strings.HasPrefix(normalized, "WITH") {
		return false
	}
	forbidden := []string{"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE", "REPLACE", "ATTACH", "DETACH"}
	for _, keyword := range forbidden {
		if containsSQLKeyword(normalized, keyword) {
			return false
		}
	}
	return strings.Contains(normalized, "SELECT")
}

func containsSQLKeyword(sql, keyword string) bool {
	fields := strings.FieldsFunc(sql, func(r rune) bool {
		return r < 'A' || r > 'Z'
	})
	for _, field := range fields {
		if field == keyword {
			return true
		}
	}
	return false
}
