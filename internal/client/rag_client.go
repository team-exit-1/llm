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

// CreatePersonalInfo creates a new personal information entry
func (rc *RAGClient) CreatePersonalInfo(ctx context.Context, req *models.PersonalInfoCreateRequest) (string, error) {
	url := fmt.Sprintf("%s/api/rag/personal-info", rc.baseURL)

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
		return "", fmt.Errorf("failed to create personal info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("create personal info failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Success bool `json:"success"`
		Data    struct {
			PersonalInfo struct {
				ID string `json:"id"`
			} `json:"personal_info"`
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
			return "", fmt.Errorf("create failed: %s - %s", apiResp.Error.Code, apiResp.Error.Message)
		}
		return "", fmt.Errorf("create failed: unknown error")
	}

	return apiResp.Data.PersonalInfo.ID, nil
}

// GetPersonalInfoByUser retrieves all personal information for a user
func (rc *RAGClient) GetPersonalInfoByUser(ctx context.Context, userID string) (*models.PersonalInfoListResponse, error) {
	url := fmt.Sprintf("%s/api/rag/personal-info/user/%s", rc.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get personal info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get personal info failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Success bool `json:"success"`
		Data    struct {
			PersonalInfoList models.PersonalInfoListResponse `json:"personal_info_list"`
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
			return nil, fmt.Errorf("get failed: %s - %s", apiResp.Error.Code, apiResp.Error.Message)
		}
		return nil, fmt.Errorf("get failed: unknown error")
	}

	return &apiResp.Data.PersonalInfoList, nil
}

// GetIncorrectQuizAttempts retrieves incorrect quiz attempts for a user
func (rc *RAGClient) GetIncorrectQuizAttempts(ctx context.Context, userID string, limit int) (*models.IncorrectQuizAttemptsResponse, error) {
	url := fmt.Sprintf("%s/api/rag/quiz-attempts/incorrect?user_id=%s&limit=%d", rc.baseURL, userID, limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get incorrect attempts: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get incorrect attempts failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Status string `json:"status"`
		Code   int    `json:"code"`
		Data   struct {
			Items  []models.IncorrectQuizAttempt `json:"items,omitempty"`
			Total  int                           `json:"total,omitempty"`
			UserID string                        `json:"user_id,omitempty"`
		} `json:"data"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.Code != http.StatusOK && apiResp.Code != 0 {
		if apiResp.Error != nil {
			return nil, fmt.Errorf("get failed: %s - %s", apiResp.Error.Code, apiResp.Error.Message)
		}
		return nil, fmt.Errorf("get failed: status code %d", apiResp.Code)
	}

	return &models.IncorrectQuizAttemptsResponse{
		Items:  apiResp.Data.Items,
		Total:  apiResp.Data.Total,
		UserID: apiResp.Data.UserID,
	}, nil
}
