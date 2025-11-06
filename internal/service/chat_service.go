package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	// Parallel fetch: search for similar conversations, get user profile, and get incorrect quiz attempts
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

	searchChan := make(chan searchResult, 1)
	profileChan := make(chan profileResult, 1)
	incorrectAttemptsChan := make(chan incorrectAttemptsResult, 1)

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

	// Get incorrect quiz attempts
	go func() {
		attempts, err := cs.ragClient.GetIncorrectQuizAttempts(ctx, req.UserID, 5)
		incorrectAttemptsChan <- incorrectAttemptsResult{attempts: attempts, err: err}
	}()

	// Wait for all results
	searchRes := <-searchChan
	profileRes := <-profileChan
	incorrectAttemptsRes := <-incorrectAttemptsChan

	log.Printf("\n=== CHAT SERVICE LOG START ===\n")
	log.Printf("User ID: %s\n", req.UserID)
	log.Printf("User Message: %s\n", req.Message)

	if searchRes.err != nil {
		log.Printf("ERROR: Failed to search conversations: %v\n", searchRes.err)
		return nil, fmt.Errorf("failed to search conversations: %w", searchRes.err)
	}

	// Log RAG conversation search results
	log.Printf("\n--- RAG Conversation Search Results ---\n")
	if len(searchRes.results) > 0 {
		searchResJSON, _ := json.MarshalIndent(searchRes.results, "", "  ")
		log.Printf("Total conversations found: %d\n", len(searchRes.results))
		log.Printf("Details:\n%s\n", string(searchResJSON))
	} else {
		log.Printf("No conversations found\n")
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

	log.Printf("Context messages extracted: %d\n", len(contextMessages))
	log.Printf("Max relevance score: %.4f\n", maxScore)

	// Extract profile information
	var profileInfo *models.PersonalInfoListResponse
	if profileRes.err != nil {
		log.Printf("WARNING: Failed to get personal info: %v\n", profileRes.err)
	} else if profileRes.profile != nil {
		profileInfo = profileRes.profile
		log.Printf("\n--- Personal Info (Core Server) ---\n")
		profileJSON, _ := json.MarshalIndent(profileInfo, "", "  ")
		log.Printf("%s\n", string(profileJSON))
	} else {
		log.Printf("No personal info found\n")
	}

	// Extract incorrect quiz attempts
	var incorrectAttempts *models.IncorrectQuizAttemptsResponse
	if incorrectAttemptsRes.err != nil {
		log.Printf("WARNING: Failed to get incorrect attempts: %v\n", incorrectAttemptsRes.err)
	} else if incorrectAttemptsRes.attempts != nil {
		incorrectAttempts = incorrectAttemptsRes.attempts
		log.Printf("\n--- Incorrect Quiz Attempts ---\n")
		incorrectJSON, _ := json.MarshalIndent(incorrectAttempts, "", "  ")
		log.Printf("%s\n", string(incorrectJSON))
	} else {
		log.Printf("No incorrect attempts found\n")
	}

	// Generate response using OpenAI with profile, context, and incorrect attempts
	log.Printf("\n--- Calling OpenAI API ---\n")
	response, err := cs.openaiService.GenerateChatResponseWithProfile(ctx, req.Message, contextMessages, profileInfo, incorrectAttempts)
	if err != nil {
		log.Printf("ERROR: Failed to generate response: %v\n", err)
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	log.Printf("\n--- OpenAI Response ---\n")
	log.Printf("%s\n", response)
	log.Printf("\n=== CHAT SERVICE LOG END ===\n\n")

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
