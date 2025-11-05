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
	ragClient      *client.RAGClient
	openaiService  *OpenAIService
	cfg            *config.Config
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
	// Search for similar conversations in RAG
	searchResults, err := cs.ragClient.SearchConversations(ctx, req.Message, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to search conversations: %w", err)
	}

	// Build context from search results
	contextMessages := []string{}
	maxScore := float32(0.0)

	for _, result := range searchResults {
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

	// Generate response using OpenAI
	response, err := cs.openaiService.GenerateChatResponse(ctx, req.Message, contextMessages)
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
			TotalConversations: len(searchResults),
			TopScore:           maxScore,
		},
		CreatedAt: time.Now(),
	}, nil
}