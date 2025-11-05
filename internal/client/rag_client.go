package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"llm/internal/config"
	"llm/internal/models"
)

// RAGClient handles communication with RAG server
type RAGClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewRAGClient creates a new RAG client
func NewRAGClient(cfg *config.Config) *RAGClient {
	return &RAGClient{
		baseURL: cfg.RAGServerURL,
		httpClient: &http.Client{
			Timeout: cfg.RAGServerTimeout,
		},
		timeout: cfg.RAGServerTimeout,
	}
}

// SearchConversations searches for similar conversations in RAG server
func (rc *RAGClient) SearchConversations(ctx context.Context, query string, limit int) ([]models.RAGConversationSearchResult, error) {
	baseURL := fmt.Sprintf("%s/api/rag/conversation/search", rc.baseURL)

	// Build query parameters with proper URL encoding
	params := url.Values{}
	params.Add("query", query)
	params.Add("top_k", fmt.Sprintf("%d", limit))
	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search conversations: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search conversations failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Success bool `json:"success"`
		Data    struct {
			Results []models.RAGConversationSearchResult `json:"results"`
		} `json:"data"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !apiResp.Success {
		if apiResp.Error != nil {
			return nil, fmt.Errorf("search failed: %s - %s", apiResp.Error.Code, apiResp.Error.Message)
		}
		return nil, fmt.Errorf("search failed: unknown error")
	}

	return apiResp.Data.Results, nil
}

// SaveConversation saves a conversation to RAG server
func (rc *RAGClient) SaveConversation(ctx context.Context, req *models.RAGConversationSaveRequest) (string, error) {
	url := fmt.Sprintf("%s/api/rag/conversation/store", rc.baseURL)

	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := rc.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to save conversation: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("save conversation failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Success bool `json:"success"`
		Data    struct {
			ConversationID string `json:"conversation_id"`
		} `json:"data"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !apiResp.Success {
		if apiResp.Error != nil {
			return "", fmt.Errorf("save failed: %s - %s", apiResp.Error.Code, apiResp.Error.Message)
		}
		return "", fmt.Errorf("save failed: unknown error")
	}

	return apiResp.Data.ConversationID, nil
}

// Health checks if RAG server is healthy
func (rc *RAGClient) Health(ctx context.Context) (bool, error) {
	url := fmt.Sprintf("%s/api/rag/health", rc.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check health: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
