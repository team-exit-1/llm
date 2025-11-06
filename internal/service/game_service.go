package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"llm/internal/client"
	"llm/internal/config"
	"llm/internal/models"
)

// GameService handles game question generation and result evaluation
type GameService struct {
	ragClient     *client.RAGClient
	openaiService *OpenAIService
	cfg           *config.Config
	questionCache map[string]*models.StoredQuestion
	cacheMutex    sync.RWMutex
}

// NewGameService creates a new game service
func NewGameService(cfg *config.Config, ragClient *client.RAGClient, openaiService *OpenAIService) *GameService {
	gs := &GameService{
		ragClient:     ragClient,
		openaiService: openaiService,
		cfg:           cfg,
		questionCache: make(map[string]*models.StoredQuestion),
	}

	// Start cache cleanup routine
	go gs.cleanupCacheRoutine()

	return gs
}

// GenerateQuestion generates a game question based on user's conversation history
func (gs *GameService) GenerateQuestion(ctx context.Context, req *models.GameQuestionRequest) (interface{}, error) {
	// Search for user conversations
	searchResults, err := gs.ragClient.SearchConversations(ctx, fmt.Sprintf("user:%s", req.UserID), 20)
	if err != nil {
		// If RAG server is unavailable, use dummy data for testing
		searchResults = []models.RAGConversationSearchResult{
			{
				ConversationID: "test-conv-1",
				Score:          0.9,
				Timestamp:      time.Now().Add(-24 * time.Hour),
				Messages: []models.RAGMessage{
					{Role: "user", Content: "Go 언어에 대해 알려줘"},
					{Role: "assistant", Content: "Go는 Google에서 개발한 프로그래밍 언어입니다."},
				},
			},
		}
	}

	if len(searchResults) < gs.cfg.MinConversationsForGame {
		return nil, fmt.Errorf("insufficient_data: need at least %d conversations, got %d", gs.cfg.MinConversationsForGame, len(searchResults))
	}

	// Evaluate user's memory and determine difficulty
	difficulty := gs.determineDifficulty(req.DifficultyHint, searchResults)

	// Select a conversation based on difficulty
	selectedConv := gs.selectConversation(searchResults, difficulty)

	// Extract topic from conversation
	topic := gs.extractTopic(selectedConv)

	// Generate question based on type
	var response interface{}
	switch req.QuestionType {
	case "fill_in_blank":
		response, err = gs.generateFillInTheBlankQuestion(ctx, selectedConv, topic)
	case "multiple_choice":
		response, err = gs.generateMultipleChoiceQuestion(ctx, selectedConv, topic)
	default:
		return nil, fmt.Errorf("invalid_question_type: %s", req.QuestionType)
	}

	if err != nil {
		return nil, err
	}

	// Cache the question
	gs.cacheQuestion(response)

	return response, nil
}

// EvaluateGameResult evaluates a game result and stores the evaluation
func (gs *GameService) EvaluateGameResult(ctx context.Context, req *models.GameResultRequest) (*models.GameResultResponse, error) {
	// Calculate retention score
	retentionScore := gs.calculateRetentionScore(req)

	// Determine confidence level
	confidence := gs.determineConfidence(retentionScore)

	// Get recommendation
	recommendation := gs.getRecommendation(retentionScore, confidence)

	// Get topic from cached question (or determine from context)
	topic := "일반"
	cachedQuestion := gs.getCachedQuestion(req.QuestionID)
	if cachedQuestion != nil {
		topic = cachedQuestion.Topic
	}

	// Save evaluation to RAG
	evalID := uuid.New().String()
	saveReq := &models.RAGConversationSaveRequest{
		ConversationID: fmt.Sprintf("memory_eval_%s", evalID),
		Messages: []models.RAGMessage{
			{
				Role:    "system",
				Content: fmt.Sprintf("사용자가 %s에 대한 기억력 테스트에서 %s", topic, map[bool]string{true: "정답", false: "오답"}[req.IsCorrect]),
			},
		},
		Metadata: &models.RAGMetadata{
			Type:           "memory_evaluation",
			RetentionScore: retentionScore,
			QuestionID:     req.QuestionID,
		},
	}

	if _, err := gs.ragClient.SaveConversation(ctx, saveReq); err != nil {
		fmt.Printf("warning: failed to save evaluation to RAG: %v\n", err)
	}

	// Determine next question suggestion
	nextDifficulty := gs.suggestNextDifficulty(retentionScore)

	return &models.GameResultResponse{
		ResultID: evalID,
		MemoryEvaluation: models.MemoryEvaluation{
			Topic:          topic,
			RetentionScore: retentionScore,
			Confidence:     confidence,
			Recommendation: recommendation,
		},
		NextQuestionSuggestion: models.NextQuestionSuggestion{
			Difficulty:      nextDifficulty,
			TopicPreference: "새로운 주제 추천",
		},
		StoredAt: time.Now(),
	}, nil
}

// Helper methods

func (gs *GameService) determineDifficulty(hint string, searchResults []models.RAGConversationSearchResult) string {
	if hint != "" && (hint == "easy" || hint == "medium" || hint == "hard") {
		return hint
	}

	// Auto-determine based on conversation age and frequency
	if len(searchResults) == 0 {
		return "easy"
	}

	recentCount := 0
	for _, result := range searchResults {
		daysSince := time.Since(result.Timestamp).Hours() / 24
		if daysSince < 1 {
			recentCount++
		}
	}

	if recentCount > len(searchResults)/2 {
		return "easy" // Many recent conversations
	} else if recentCount > 0 {
		return "medium"
	} else {
		return "hard"
	}
}

func (gs *GameService) selectConversation(searchResults []models.RAGConversationSearchResult, difficulty string) models.RAGConversationSearchResult {
	if len(searchResults) == 0 {
		return models.RAGConversationSearchResult{}
	}

	// Simple selection: for easy, pick most recent; for hard, pick oldest
	// For multiple choice, pick based on score
	return searchResults[0]
}

func (gs *GameService) extractTopic(conv models.RAGConversationSearchResult) string {
	if len(conv.Messages) > 0 {
		// Simple topic extraction from first message
		content := conv.Messages[0].Content
		if len(content) > 50 {
			return content[:50]
		}
		return content
	}
	return "일반"
}

func (gs *GameService) generateFillInTheBlankQuestion(ctx context.Context, conv models.RAGConversationSearchResult, topic string) (*models.FillInTheBlankQuestionResponse, error) {
	// Combine conversation content
	var content string
	for _, msg := range conv.Messages {
		content += msg.Content + " "
	}

	// Call OpenAI to generate question (simplified for now)
	// In real implementation, parse JSON response

	qID := uuid.New().String()
	return &models.FillInTheBlankQuestionResponse{
		QuestionID:          qID,
		QuestionType:        "fill_in_blank",
		Question:            "생성된 빈칸 채우기 문제입니다: ___에 대해 이야기했습니다.",
		CorrectAnswer:       "정답",
		AcceptableAnswers:   []string{"유사답안1", "유사답안2"},
		BasedOnConversation: conv.ConversationID,
		Difficulty:          "medium",
		Metadata: models.QuestionMetadata{
			Topic:                 topic,
			MemoryScore:           conv.Score,
			DaysSinceConversation: int(time.Since(conv.Timestamp).Hours() / 24),
		},
	}, nil
}

func (gs *GameService) generateMultipleChoiceQuestion(ctx context.Context, conv models.RAGConversationSearchResult, topic string) (*models.MultipleChoiceQuestionResponse, error) {
	// Combine conversation content
	var content string
	for _, msg := range conv.Messages {
		content += msg.Content + " "
	}

	// Call OpenAI to generate question (simplified for now)
	// In real implementation, parse JSON response

	qID := uuid.New().String()
	return &models.MultipleChoiceQuestionResponse{
		QuestionID:   qID,
		QuestionType: "multiple_choice",
		Question:     "생성된 4지선다 문제입니다.",
		Options: []models.QuestionOption{
			{ID: "A", Text: "선지1"},
			{ID: "B", Text: "선지2"},
			{ID: "C", Text: "선지3"},
			{ID: "D", Text: "선지4"},
		},
		CorrectAnswer:       "B",
		BasedOnConversation: conv.ConversationID,
		Difficulty:          "medium",
		Metadata: models.QuestionMetadata{
			Topic:                 topic,
			MemoryScore:           conv.Score,
			DaysSinceConversation: int(time.Since(conv.Timestamp).Hours() / 24),
		},
	}, nil
}

func (gs *GameService) calculateRetentionScore(req *models.GameResultRequest) float32 {
	weights := gs.cfg.MemoryEvaluationWeights

	// Correct answer score (50% weight)
	correctScore := float32(0.0)
	if req.IsCorrect {
		correctScore = 1.0
	}

	// Response time score (30% weight) - faster = better
	// Normalize to 0-1 range (5000ms = 1.0, >5000ms = 0.0)
	timeScore := float32(1.0)
	if req.ResponseTimeMs > 5000 {
		timeScore = 0.0
	} else {
		timeScore = float32(float64(5000-req.ResponseTimeMs) / 5000.0)
	}

	// Recency score (20% weight) - always 1.0 for now since it's calculated at result time
	recencyScore := float32(1.0)

	retentionScore := weights[0]*correctScore + weights[1]*timeScore + weights[2]*recencyScore
	return retentionScore
}

func (gs *GameService) determineConfidence(score float32) string {
	if score >= 0.8 {
		return "high"
	} else if score >= 0.5 {
		return "medium"
	} else {
		return "low"
	}
}

func (gs *GameService) getRecommendation(score float32, confidence string) string {
	if score >= 0.9 {
		return "이 주제는 매우 잘 기억하고 있습니다."
	} else if score >= 0.7 {
		return "이 주제는 비교적 잘 기억하고 있습니다."
	} else if score >= 0.5 {
		return "이 주제는 부분적으로 기억하고 있습니다. 복습을 권장합니다."
	} else {
		return "이 주제는 잘 기억하지 못하고 있습니다. 자주 복습해주세요."
	}
}

func (gs *GameService) suggestNextDifficulty(score float32) string {
	if score >= 0.8 {
		return "hard" // Increase difficulty
	} else if score >= 0.5 {
		return "medium" // Maintain difficulty
	} else {
		return "easy" // Decrease difficulty
	}
}

func (gs *GameService) cacheQuestion(q interface{}) {
	gs.cacheMutex.Lock()
	defer gs.cacheMutex.Unlock()

	// Extract question ID and store with expiry
	var qID string
	switch v := q.(type) {
	case *models.FillInTheBlankQuestionResponse:
		qID = v.QuestionID
	case *models.MultipleChoiceQuestionResponse:
		qID = v.QuestionID
	default:
		return
	}

	gs.questionCache[qID] = &models.StoredQuestion{
		QuestionID: qID,
		ExpiresAt:  time.Now().Add(gs.cfg.QuestionCacheTTL),
	}
}

func (gs *GameService) getCachedQuestion(qID string) *models.StoredQuestion {
	gs.cacheMutex.RLock()
	defer gs.cacheMutex.RUnlock()

	q, exists := gs.questionCache[qID]
	if !exists || time.Now().After(q.ExpiresAt) {
		return nil
	}
	return q
}

func (gs *GameService) cleanupCacheRoutine() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		gs.cacheMutex.Lock()
		now := time.Now()
		for qID, q := range gs.questionCache {
			if now.After(q.ExpiresAt) {
				delete(gs.questionCache, qID)
			}
		}
		gs.cacheMutex.Unlock()
	}
}
