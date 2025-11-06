package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"llm/internal/client"
	"llm/internal/config"
	"llm/internal/models"
)

// ChatService handles chat functionality
type ChatService struct {
	ragClient     *client.RAGClient
	openaiService *OpenAIService
	cfg           *config.Config
}

// NewChatService creates a new chat service
func NewChatService(cfg *config.Config, ragClient *client.RAGClient, openaiService *OpenAIService) *ChatService {
	return &ChatService{
		ragClient:     ragClient,
		openaiService: openaiService,
		cfg:           cfg,
	}
}

// ProcessChat processes a user chat message and returns a response
func (cs *ChatService) ProcessChat(ctx context.Context, req *models.ChatRequest) (*models.ChatResponse, error) {
	// Parallel fetch: search for similar conversations and get user profile
	type searchResult struct {
		results []*models.RAGConversationSearchResult
		err     error
	}
	type profileResult struct {
		profile *models.PersonalInfoListResponse
		err     error
	}

	searchChan := make(chan searchResult, 1)
	profileChan := make(chan profileResult, 1)

	// Search for similar conversations
	go func() {
		results, err := cs.ragClient.SearchConversations(ctx, req.Message, 5)
		searchChan <- searchResult{results: convertToPointers(results), err: err}
	}()

	// Get user profile information
	go func() {
		profile, err := cs.ragClient.GetPersonalInfoByUser(ctx, req.UserID)
		profileChan <- profileResult{profile: profile, err: err}
	}()

	// Wait for both results
	searchRes := <-searchChan
	profileRes := <-profileChan

	if searchRes.err != nil {
		return nil, fmt.Errorf("failed to search conversations: %w", searchRes.err)
	}

	// Build context from search results
	contextMessages := []string{}
	maxScore := float32(0.0)

	for _, result := range searchRes.results {
		if result == nil {
			continue
		}
		// Extract the question/answer from messages
		for _, msg := range result.Messages {
			if msg.Role == "user" || msg.Role == "assistant" {
				contextMessages = append(contextMessages, msg.Content)
			}
		}
		if result.Score > maxScore {
			maxScore = result.Score
		}
	}

	// Extract profile information
	var profileInfo *models.PersonalInfoListResponse
	if profileRes.err == nil {
		profileInfo = profileRes.profile
	}

	// Generate response using OpenAI with profile and context
	response, err := cs.openaiService.GenerateChatResponseWithProfile(ctx, req.Message, contextMessages, profileInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	// Create conversation ID
	conversationID := uuid.New().String()

	// Save conversation to RAG asynchronously
	go func() {
		saveReq := &models.RAGConversationSaveRequest{
			ConversationID: conversationID,
			Messages: []models.RAGMessage{
				{
					Role:    "user",
					Content: req.Message,
				},
				{
					Role:    "assistant",
					Content: response,
				},
			},
			Metadata: &models.RAGMetadata{
				Source:    "llm_chat",
				SessionID: req.UserID,
				Type:      "chat",
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := cs.ragClient.SaveConversation(ctx, saveReq); err != nil {
			fmt.Printf("warning: failed to save conversation to RAG: %v\n", err)
		}
	}()

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

// Helper function to convert slice to pointer slice
func convertToPointers(results []models.RAGConversationSearchResult) []*models.RAGConversationSearchResult {
	pointers := make([]*models.RAGConversationSearchResult, len(results))
	for i := range results {
		pointers[i] = &results[i]
	}
	return pointers
}
