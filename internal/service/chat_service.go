package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"llm/internal/client"
	"llm/internal/config"
	"llm/internal/models"
	"llm/internal/util"
)

// ChatService handles chat functionality
type ChatService struct {
	ragClient     *client.RAGClient
	openaiService *OpenAIService
	cfg           *config.Config
	logger        *util.Logger
}

// NewChatService creates a new chat service
func NewChatService(cfg *config.Config, ragClient *client.RAGClient, openaiService *OpenAIService) *ChatService {
	return &ChatService{
		ragClient:     ragClient,
		openaiService: openaiService,
		cfg:           cfg,
		logger:        util.NewLogger("ChatService"),
	}
}

// ProcessChat processes a user chat message and returns a response
func (cs *ChatService) ProcessChat(ctx context.Context, req *models.ChatRequest) (*models.ChatResponse, error) {
	cs.logger.Start("Process Chat")

	// Parallel fetch: conversations, profile, and incorrect attempts
	searchRes := cs.fetchConversations(ctx, req)
	profileRes := cs.fetchUserProfile(ctx, req)
	incorrectAttemptsRes := cs.fetchIncorrectAttempts(ctx, req)

	// Validate search results
	if searchRes.err != nil {
		cs.logger.Error("Failed to search conversations", searchRes.err)
		cs.logger.End("Process Chat")
		return nil, fmt.Errorf("failed to search conversations: %w", searchRes.err)
	}

	// Log fetched data
	cs.logFetchedData(searchRes, profileRes, incorrectAttemptsRes)

	// Build context from search results
	contextMessages := cs.extractContextMessages(searchRes.results)
	maxScore := cs.extractMaxScore(searchRes.results)

	// Extract profile and incorrect attempts
	var profileInfo *models.PersonalInfoListResponse
	if profileRes.err == nil && profileRes.profile != nil {
		profileInfo = profileRes.profile
	}

	var incorrectAttempts *models.IncorrectQuizAttemptsResponse
	if incorrectAttemptsRes.err == nil && incorrectAttemptsRes.attempts != nil {
		incorrectAttempts = incorrectAttemptsRes.attempts
	}

	// Generate response
	cs.logger.Section("Generating Response")
	response, err := cs.openaiService.GenerateChatResponseWithProfile(ctx, req.Message, contextMessages, profileInfo, incorrectAttempts)
	if err != nil {
		cs.logger.Error("Failed to generate response", err)
		cs.logger.End("Process Chat")
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	// Create conversation ID
	conversationID := uuid.New().String()

	// Evaluate user response and save asynchronously
	go cs.evaluateAndSave(context.Background(), req, response, conversationID, contextMessages, profileInfo)

	cs.logger.Success("Chat processed successfully")
	cs.logger.End("Process Chat")

	return &models.ChatResponse{
		ConversationID: conversationID,
		Message:        req.Message,
		Response:       response,
		ContextUsed: models.ContextUsage{
			TotalConversations: len(searchRes.results),
			TopScore:           maxScore,
		},
		CreatedAt: time.Now(),
	}, nil
}

// ============================================================================
// Helper Methods - Fetching
// ============================================================================

type searchResult struct {
	results []*models.RAGConversationSearchResult
	err     error
}

type profileResult struct {
	profile *models.PersonalInfoListResponse
	err     error
}

type incorrectAttemptsResult struct {
	attempts *models.IncorrectQuizAttemptsResponse
	err      error
}

func (cs *ChatService) fetchConversations(ctx context.Context, req *models.ChatRequest) searchResult {
	results := make(chan searchResult, 1)
	go func() {
		rag, err := cs.ragClient.SearchConversations(ctx, req.Message, 5)
		results <- searchResult{results: cs.convertToPointers(rag), err: err}
	}()
	return <-results
}

func (cs *ChatService) fetchUserProfile(ctx context.Context, req *models.ChatRequest) profileResult {
	results := make(chan profileResult, 1)
	go func() {
		profile, err := cs.ragClient.GetPersonalInfoByUser(ctx, req.UserID)
		results <- profileResult{profile: profile, err: err}
	}()
	return <-results
}

func (cs *ChatService) fetchIncorrectAttempts(ctx context.Context, req *models.ChatRequest) incorrectAttemptsResult {
	results := make(chan incorrectAttemptsResult, 1)
	go func() {
		attempts, err := cs.ragClient.GetIncorrectQuizAttempts(ctx, req.UserID, 5)
		results <- incorrectAttemptsResult{attempts: attempts, err: err}
	}()
	return <-results
}

// ============================================================================
// Helper Methods - Data Processing
// ============================================================================

func (cs *ChatService) logFetchedData(searchRes searchResult, profileRes profileResult, incorrectAttemptsRes incorrectAttemptsResult) {
	cs.logger.Section("RAG Conversation Search Results")
	cs.logger.KeyValue("Total conversations", len(searchRes.results))

	if profileRes.err == nil && profileRes.profile != nil {
		cs.logger.Section("Personal Info")
		cs.logger.Info("Profile info found: %d items", len(profileRes.profile.Items))
	}

	if incorrectAttemptsRes.err == nil && incorrectAttemptsRes.attempts != nil {
		cs.logger.Section("Incorrect Quiz Attempts")
		cs.logger.Info("Found %d incorrect attempts", len(incorrectAttemptsRes.attempts.Items))
	}
}

func (cs *ChatService) extractContextMessages(results []*models.RAGConversationSearchResult) []string {
	contextMessages := []string{}
	for _, result := range results {
		if result == nil {
			continue
		}
		for _, msg := range result.Messages {
			if msg.Role == "user" || msg.Role == "assistant" {
				contextMessages = append(contextMessages, msg.Content)
			}
		}
	}
	return contextMessages
}

func (cs *ChatService) extractMaxScore(results []*models.RAGConversationSearchResult) float32 {
	maxScore := float32(0.0)
	for _, result := range results {
		if result != nil && result.Score > maxScore {
			maxScore = result.Score
		}
	}
	return maxScore
}

// ============================================================================
// Helper Methods - Async Processing
// ============================================================================

func (cs *ChatService) evaluateAndSave(ctx context.Context, req *models.ChatRequest, response, conversationID string, contextMessages []string, profileInfo *models.PersonalInfoListResponse) {
	cs.logger.Start("Async: Evaluate and Save")

	// Evaluate user response quality
	responseScore := util.DefaultResponseScore
	score, err := cs.openaiService.EvaluateUserResponseQuality(ctx, req.Message, contextMessages, profileInfo)
	if err != nil {
		cs.logger.Warn("Failed to evaluate response quality, using default", err)
	} else {
		responseScore = score
	}

	// Save conversation to RAG
	saveReq := &models.RAGConversationSaveRequest{
		ConversationID: conversationID,
		Messages: []models.RAGMessage{
			{Role: "user", Content: req.Message},
			{Role: "assistant", Content: response},
		},
		Metadata: &models.RAGMetadata{
			Source:            "llm_chat",
			SessionID:         req.UserID,
			Type:              "chat",
			ConversationScore: responseScore,
		},
	}

	_, err = cs.ragClient.SaveConversation(ctx, saveReq)
	if err != nil {
		cs.logger.Warn("Failed to save conversation", err)
	} else {
		cs.logger.Success(fmt.Sprintf("Conversation saved with quality score: %d/100", responseScore))
	}

	cs.logger.End("Async: Evaluate and Save")
}

// ============================================================================
// Utility Methods
// ============================================================================

func (cs *ChatService) convertToPointers(results []models.RAGConversationSearchResult) []*models.RAGConversationSearchResult {
	pointers := make([]*models.RAGConversationSearchResult, len(results))
	for i := range results {
		pointers[i] = &results[i]
	}
	return pointers
}
