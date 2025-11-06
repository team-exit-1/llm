package service

import (
	"context"
	"fmt"
	"time"

	"llm/internal/client"
	"llm/internal/models"
	"llm/internal/util"
)

// AnalysisService handles domain analysis and report generation
type AnalysisService struct {
	ragClient     *client.RAGClient
	openaiService *OpenAIService
	logger        *util.Logger
}

// NewAnalysisService creates a new analysis service
func NewAnalysisService(ragClient *client.RAGClient, openaiService *OpenAIService) *AnalysisService {
	return &AnalysisService{
		ragClient:     ragClient,
		openaiService: openaiService,
		logger:        util.NewLogger("AnalysisService"),
	}
}

// ProcessAnalysisRequest processes a domain analysis request
func (as *AnalysisService) ProcessAnalysisRequest(ctx context.Context, req *models.AnalysisRequest) (*models.AnalysisResponse, error) {
	as.logger.Start("Process Analysis Request")

	// Fetch user's conversation history and incorrect quiz attempts in parallel
	conversationChan := make(chan []string, 1)
	incorrectQuizzesChan := make(chan []string, 1)

	// Fetch conversations
	go func() {
		conversations, err := as.fetchConversationHistory(ctx, req.UserID)
		if err != nil {
			as.logger.Warn("Failed to fetch conversations", err)
			conversationChan <- []string{}
		} else {
			conversationChan <- conversations
		}
	}()

	// Fetch incorrect quizzes
	go func() {
		quizzes, err := as.fetchIncorrectQuizzes(ctx, req.UserID)
		if err != nil {
			as.logger.Warn("Failed to fetch incorrect quizzes", err)
			incorrectQuizzesChan <- []string{}
		} else {
			incorrectQuizzesChan <- quizzes
		}
	}()

	conversationHistory := <-conversationChan
	incorrectQuizzes := <-incorrectQuizzesChan

	as.logger.Section("Fetched Data")
	as.logger.KeyValue("Conversations", len(conversationHistory), "Incorrect Quizzes", len(incorrectQuizzes))

	// Step 1: Analyze domains
	as.logger.Section("Step 1: Analyzing Domains")
	domains, err := as.openaiService.AnalyzeDomains(ctx, conversationHistory, incorrectQuizzes)
	if err != nil {
		as.logger.Error("Failed to analyze domains", err)
		as.logger.End("Process Analysis Request")
		return nil, fmt.Errorf("failed to analyze domains: %w", err)
	}

	// Step 2: Generate professional report
	as.logger.Section("Step 2: Generating Professional Report")
	report, err := as.openaiService.GenerateAnalysisReport(ctx, domains)
	if err != nil {
		as.logger.Error("Failed to generate report", err)
		as.logger.End("Process Analysis Request")
		return nil, fmt.Errorf("failed to generate report: %w", err)
	}

	as.logger.Success("Analysis completed successfully")
	as.logger.End("Process Analysis Request")

	return &models.AnalysisResponse{
		UserID:     req.UserID,
		Domains:    domains,
		Report:     report,
		AnalyzedAt: time.Now(),
	}, nil
}

// ============================================================================
// Helper Methods
// ============================================================================

func (as *AnalysisService) fetchConversationHistory(ctx context.Context, userID string) ([]string, error) {
	as.logger.Section("Fetching Conversation History")

	// Fetch all conversations for this user using a broad search query
	results, err := as.ragClient.SearchConversations(ctx, userID, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to search conversations: %w", err)
	}

	conversations := []string{}
	for _, result := range results {
		for _, msg := range result.Messages {
			conversations = append(conversations, msg.Content)
		}
	}

	as.logger.Info("Retrieved %d conversation messages", len(conversations))
	return conversations, nil
}

func (as *AnalysisService) fetchIncorrectQuizzes(ctx context.Context, userID string) ([]string, error) {
	as.logger.Section("Fetching Incorrect Quizzes")

	// Fetch incorrect quiz attempts
	attempts, err := as.ragClient.GetIncorrectQuizAttempts(ctx, userID, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch incorrect quiz attempts: %w", err)
	}

	quizzes := []string{}
	for _, attempt := range attempts.Items {
		quizStr := fmt.Sprintf(
			"[%s] Q: %s | Your Answer: %s | Correct Answer: %s | Topic: %s",
			attempt.Quiz.QuestionType,
			attempt.Quiz.Question,
			attempt.UserAnswer,
			attempt.CorrectAnswer,
			attempt.Quiz.Topic,
		)
		quizzes = append(quizzes, quizStr)
	}

	as.logger.Info("Retrieved %d incorrect quiz attempts", len(quizzes))
	return quizzes, nil
}

// ProcessDomainAnalysisOnly performs domain analysis only (without report generation)
func (as *AnalysisService) ProcessDomainAnalysisOnly(ctx context.Context, req *models.AnalysisRequest) (*models.DomainAnalysisOnlyResponse, error) {
	as.logger.Start("Process Domain Analysis Only")

	// Fetch user's conversation history and incorrect quiz attempts in parallel
	conversationChan := make(chan []string, 1)
	incorrectQuizzesChan := make(chan []string, 1)

	go func() {
		conversations, err := as.fetchConversationHistory(ctx, req.UserID)
		if err != nil {
			as.logger.Warn("Failed to fetch conversations", err)
			conversationChan <- []string{}
		} else {
			conversationChan <- conversations
		}
	}()

	go func() {
		quizzes, err := as.fetchIncorrectQuizzes(ctx, req.UserID)
		if err != nil {
			as.logger.Warn("Failed to fetch incorrect quizzes", err)
			incorrectQuizzesChan <- []string{}
		} else {
			incorrectQuizzesChan <- quizzes
		}
	}()

	conversationHistory := <-conversationChan
	incorrectQuizzes := <-incorrectQuizzesChan

	as.logger.Section("Analyzing Domains")
	domains, err := as.openaiService.AnalyzeDomains(ctx, conversationHistory, incorrectQuizzes)
	if err != nil {
		as.logger.Error("Failed to analyze domains", err)
		as.logger.End("Process Domain Analysis Only")
		return nil, fmt.Errorf("failed to analyze domains: %w", err)
	}

	as.logger.Success("Domain analysis completed")
	as.logger.End("Process Domain Analysis Only")

	return &models.DomainAnalysisOnlyResponse{
		UserID:     req.UserID,
		Domains:    domains,
		AnalyzedAt: time.Now(),
	}, nil
}

// ProcessReportGenerationOnly generates a report from provided domain scores
func (as *AnalysisService) ProcessReportGenerationOnly(ctx context.Context, req *models.ReportGenerationRequest) (string, error) {
	as.logger.Start("Process Report Generation Only")

	// Validate that we have all required domains
	if len(req.Domains) != 4 {
		as.logger.Error("Invalid domain count", fmt.Errorf("expected 4 domains, got %d", len(req.Domains)))
		as.logger.End("Process Report Generation Only")
		return "", fmt.Errorf("invalid_request: expected 4 domains, got %d", len(req.Domains))
	}

	// Extract domain data
	familyScore, familyInsights := 0, []string{}
	lifeEventsScore, lifeEventsInsights := 0, []string{}
	careerScore, careerInsights := 0, []string{}
	hobbiesScore, hobbiesInsights := 0, []string{}

	for _, domain := range req.Domains {
		switch domain.Domain {
		case "family":
			familyScore = domain.Score
			familyInsights = domain.Insights
		case "life_events":
			lifeEventsScore = domain.Score
			lifeEventsInsights = domain.Insights
		case "career":
			careerScore = domain.Score
			careerInsights = domain.Insights
		case "hobbies":
			hobbiesScore = domain.Score
			hobbiesInsights = domain.Insights
		}
	}

	as.logger.Section("Generating Report")
	report, err := as.openaiService.GenerateReportFromDomainScores(
		ctx,
		familyScore, familyInsights,
		lifeEventsScore, lifeEventsInsights,
		careerScore, careerInsights,
		hobbiesScore, hobbiesInsights,
	)
	if err != nil {
		as.logger.Error("Failed to generate report", err)
		as.logger.End("Process Report Generation Only")
		return "", fmt.Errorf("failed to generate report: %w", err)
	}

	as.logger.Success("Report generation completed")
	as.logger.End("Process Report Generation Only")

	return report, nil
}
