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
	"llm/internal/util"
)

// GameService handles game question generation and result evaluation
type GameService struct {
	ragClient     *client.RAGClient
	openaiService *OpenAIService
	cfg           *config.Config
	questionCache map[string]*models.StoredQuestion
	cacheMutex    sync.RWMutex
	logger        *util.Logger
}

// NewGameService creates a new game service
func NewGameService(cfg *config.Config, ragClient *client.RAGClient, openaiService *OpenAIService) *GameService {
	gs := &GameService{
		ragClient:     ragClient,
		openaiService: openaiService,
		cfg:           cfg,
		questionCache: make(map[string]*models.StoredQuestion),
		logger:        util.NewLogger("GameService"),
	}

	go gs.cleanupCacheRoutine()
	return gs
}

// GenerateQuestion generates a question based on user's conversation history
func (gs *GameService) GenerateQuestion(ctx context.Context, req *models.GameQuestionRequest) (interface{}, error) {
	gs.logger.Start("Generate Question")

	// Fetch latest 20 conversations
	searchResults, err := gs.ragClient.SearchConversations(ctx, "conversation", 20)
	if err != nil {
		gs.logger.Error("Failed to search conversations", err)
		gs.logger.End("Generate Question")
		return nil, fmt.Errorf("insufficient conversation history: %w", err)
	}

	// Check if we have enough conversations
	if len(searchResults) < 5 {
		gs.logger.Error("Insufficient conversations", fmt.Errorf("need at least 5, got %d", len(searchResults)))
		gs.logger.End("Generate Question")
		return nil, fmt.Errorf("insufficient_data: need at least 5 conversations, got %d", len(searchResults))
	}

	// Determine difficulty and select conversation
	difficulty := gs.determineDifficulty(req.DifficultyHint, searchResults)
	selectedConv := gs.selectConversation(searchResults, difficulty)
	topic := gs.extractTopic(selectedConv)

	gs.logger.KeyValue("Difficulty", difficulty, "Topic", topic)

	// Generate question based on type
	var response interface{}
	switch req.QuestionType {
	case util.QuestionTypeFillInBlank:
		response, err = gs.generateFillInTheBlankQuestion(ctx, selectedConv, topic)
	case util.QuestionTypeMultipleChoice:
		response, err = gs.generateMultipleChoiceQuestion(ctx, selectedConv, topic)
	default:
		gs.logger.Error("Invalid question type", fmt.Errorf(req.QuestionType))
		gs.logger.End("Generate Question")
		return nil, fmt.Errorf("invalid_question_type: %s", req.QuestionType)
	}

	if err != nil {
		gs.logger.End("Generate Question")
		return nil, err
	}

	// Cache the question
	gs.cacheQuestion(response)

	gs.logger.Success("Question generated and cached")
	gs.logger.End("Generate Question")
	return response, nil
}

// EvaluateGameResult evaluates a game result and stores the evaluation
func (gs *GameService) EvaluateGameResult(ctx context.Context, req *models.GameResultRequest) (*models.GameResultResponse, error) {
	gs.logger.Start("Evaluate Game Result")

	// Calculate retention score
	retentionScore := gs.calculateRetentionScore(req)
	confidence := gs.determineConfidence(retentionScore)
	recommendation := gs.getRecommendation(retentionScore)

	gs.logger.KeyValue("Retention Score", retentionScore, "Confidence", confidence)

	// Get topic from cached question
	topic := util.DifficultyEasy // Default
	cachedQuestion := gs.getCachedQuestion(req.QuestionID)
	if cachedQuestion != nil && cachedQuestion.Topic != "" {
		topic = cachedQuestion.Topic
	}

	// Save evaluation asynchronously
	go gs.saveEvaluation(context.Background(), req, topic, retentionScore)

	// Suggest next difficulty
	nextDifficulty := gs.suggestNextDifficulty(retentionScore)

	gs.logger.Success("Game result evaluated")
	gs.logger.End("Evaluate Game Result")

	return &models.GameResultResponse{
		ResultID: uuid.New().String(),
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

// ============================================================================
// Helper Methods - Question Generation
// ============================================================================

func (gs *GameService) generateFillInTheBlankQuestion(ctx context.Context, conv models.RAGConversationSearchResult, topic string) (*models.FillInTheBlankQuestionResponse, error) {
	conversationContent := gs.extractConversationContent(conv)
	baseQuestion, err := gs.openaiService.GenerateFillInTheBlankQuestion(ctx, conversationContent, topic)
	if err != nil {
		return nil, err
	}

	return &models.FillInTheBlankQuestionResponse{
		QuestionID:          uuid.New().String(),
		QuestionType:        util.QuestionTypeFillInBlank,
		Question:            baseQuestion.Question,
		Options:             baseQuestion.Options,
		CorrectAnswer:       baseQuestion.CorrectAnswer,
		BasedOnConversation: conv.ConversationID,
		Difficulty:          gs.determineDifficultyFromConversation(conv),
		Metadata: models.QuestionMetadata{
			Topic:                 topic,
			MemoryScore:           conv.Score,
			DaysSinceConversation: int(time.Since(conv.Timestamp).Hours() / 24),
		},
	}, nil
}

func (gs *GameService) generateMultipleChoiceQuestion(ctx context.Context, conv models.RAGConversationSearchResult, topic string) (*models.MultipleChoiceQuestionResponse, error) {
	conversationContent := gs.extractConversationContent(conv)
	baseQuestion, err := gs.openaiService.GenerateMultipleChoiceQuestion(ctx, conversationContent, topic)
	if err != nil {
		return nil, err
	}

	return &models.MultipleChoiceQuestionResponse{
		QuestionID:          uuid.New().String(),
		QuestionType:        util.QuestionTypeMultipleChoice,
		Question:            baseQuestion.Question,
		Options:             baseQuestion.Options,
		CorrectAnswer:       baseQuestion.CorrectAnswer,
		BasedOnConversation: conv.ConversationID,
		Difficulty:          gs.determineDifficultyFromConversation(conv),
		Metadata: models.QuestionMetadata{
			Topic:                 topic,
			MemoryScore:           conv.Score,
			DaysSinceConversation: int(time.Since(conv.Timestamp).Hours() / 24),
		},
	}, nil
}

func (gs *GameService) determineDifficulty(hint string, searchResults []models.RAGConversationSearchResult) string {
	if hint != "" && (hint == util.DifficultyEasy || hint == util.DifficultyMedium || hint == util.DifficultyHard) {
		return hint
	}

	if len(searchResults) == 0 {
		return util.DifficultyEasy
	}

	recentCount := 0
	for _, result := range searchResults {
		if time.Since(result.Timestamp).Hours()/24 < 1 {
			recentCount++
		}
	}

	if recentCount > len(searchResults)/2 {
		return util.DifficultyEasy
	} else if recentCount > 0 {
		return util.DifficultyMedium
	}
	return util.DifficultyHard
}

func (gs *GameService) determineDifficultyFromConversation(conv models.RAGConversationSearchResult) string {
	daysSince := int(time.Since(conv.Timestamp).Hours() / 24)

	if daysSince == 0 {
		return util.DifficultyEasy
	}
	if daysSince <= 7 {
		return util.DifficultyMedium
	}
	return util.DifficultyHard
}

func (gs *GameService) selectConversation(searchResults []models.RAGConversationSearchResult, difficulty string) models.RAGConversationSearchResult {
	if len(searchResults) == 0 {
		return models.RAGConversationSearchResult{}
	}
	return searchResults[0]
}

func (gs *GameService) extractTopic(conv models.RAGConversationSearchResult) string {
	if len(conv.Messages) > 0 {
		content := conv.Messages[0].Content
		if len(content) > 50 {
			return content[:50]
		}
		return content
	}
	return "일반"
}

func (gs *GameService) extractConversationContent(conv models.RAGConversationSearchResult) string {
	var content string
	for _, msg := range conv.Messages {
		if content != "" {
			content += "\n"
		}
		content += msg.Content
	}
	return content
}

// ============================================================================
// Helper Methods - Evaluation
// ============================================================================

func (gs *GameService) calculateRetentionScore(req *models.GameResultRequest) float32 {
	weights := gs.cfg.MemoryEvaluationWeights

	// Correct answer score (50% weight)
	correctScore := float32(0.0)
	if req.IsCorrect {
		correctScore = 1.0
	}

	// Response time score (30% weight) - faster = better
	timeScore := float32(1.0)
	if req.ResponseTimeMs > util.ResponseTimeThreshold {
		timeScore = 0.0
	} else {
		timeScore = float32(float64(util.ResponseTimeThreshold-req.ResponseTimeMs) / float64(util.ResponseTimeThreshold))
	}

	// Recency score (20% weight)
	recencyScore := float32(1.0)

	return weights[0]*correctScore + weights[1]*timeScore + weights[2]*recencyScore
}

func (gs *GameService) determineConfidence(score float32) string {
	if score >= 0.8 {
		return util.ConfidenceHigh
	} else if score >= 0.5 {
		return util.ConfidenceMedium
	}
	return util.ConfidenceLow
}

func (gs *GameService) getRecommendation(score float32) string {
	if score >= 0.9 {
		return "이 주제는 매우 잘 기억하고 있습니다."
	} else if score >= 0.7 {
		return "이 주제는 비교적 잘 기억하고 있습니다."
	} else if score >= 0.5 {
		return "이 주제는 부분적으로 기억하고 있습니다. 복습을 권장합니다."
	}
	return "이 주제는 잘 기억하지 못하고 있습니다. 자주 복습해주세요."
}

func (gs *GameService) suggestNextDifficulty(score float32) string {
	if score >= 0.8 {
		return util.DifficultyHard
	} else if score >= 0.5 {
		return util.DifficultyMedium
	}
	return util.DifficultyEasy
}

func (gs *GameService) cacheQuestion(q interface{}) {
	gs.cacheMutex.Lock()
	defer gs.cacheMutex.Unlock()

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
		ExpiresAt:  time.Now().Add(24 * time.Hour),
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

func (gs *GameService) saveEvaluation(ctx context.Context, req *models.GameResultRequest, topic string, retentionScore float32) {
	gs.logger.Start("Async: Save Evaluation")

	saveReq := &models.RAGConversationSaveRequest{
		ConversationID: fmt.Sprintf("memory_eval_%s", uuid.New().String()),
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

	_, err := gs.ragClient.SaveConversation(ctx, saveReq)
	if err != nil {
		gs.logger.Warn("Failed to save evaluation", err)
	} else {
		gs.logger.Success("Evaluation saved")
	}

	gs.logger.End("Async: Save Evaluation")
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
