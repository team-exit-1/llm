package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"

	"llm/internal/config"
	"llm/internal/models"
	"llm/internal/prompts"
	"llm/internal/util"
)

// OpenAIService handles all interactions with OpenAI API
type OpenAIService struct {
	client      *openai.Client
	model       string
	temperature float32
	maxTokens   int
	logger      *util.Logger
}

// NewOpenAIService creates a new OpenAI service instance
func NewOpenAIService(cfg *config.Config) *OpenAIService {
	return &OpenAIService{
		client:      openai.NewClient(cfg.OpenAIAPIKey),
		model:       cfg.OpenAIModel,
		temperature: cfg.OpenAITemperature,
		maxTokens:   cfg.OpenAIMaxTokens,
		logger:      util.NewLogger("OpenAIService"),
	}
}

// GenerateChatResponse generates a simple chat response
func (os *OpenAIService) GenerateChatResponse(ctx context.Context, userMessage string, contextMessages []string) (string, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: `당신은 사용자의 과거 대화를 바탕으로 도움이 되는 답변을 하는 어시스턴트입니다.`,
		},
	}

	// Add context from previous conversations
	for _, msg := range contextMessages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: fmt.Sprintf("[참고 정보] %s", msg),
		})
	}

	// Add user message
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userMessage,
	})

	content, err := os.callOpenAI(ctx, messages)
	if err != nil {
		return "", err
	}

	return content, nil
}

// GenerateChatResponseWithProfile generates a response with user profile and incorrect attempts
func (os *OpenAIService) GenerateChatResponseWithProfile(ctx context.Context, userMessage string, contextMessages []string, profileInfo *models.PersonalInfoListResponse, incorrectAttempts *models.IncorrectQuizAttemptsResponse) (string, error) {
	os.logger.Start("Chat Response Generation")

	systemPrompt := prompts.ChatSystemPrompt(profileInfo, incorrectAttempts)
	os.logger.Section("System Prompt")
	os.logger.Info(systemPrompt)

	// Build messages
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
	}

	// Add context
	if len(contextMessages) > 0 {
		contextLimit := 3
		if len(contextMessages) < contextLimit {
			contextLimit = len(contextMessages)
		}

		contextStr := "최근 대화 이력:\n"
		for i := 0; i < contextLimit; i++ {
			contextStr += fmt.Sprintf("- %s\n", contextMessages[i])
		}

		os.logger.Section("Context")
		os.logger.Info(contextStr)

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: contextStr,
		})
	}

	// Add user message
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userMessage,
	})

	os.logger.Section("Calling OpenAI")
	content, err := os.callOpenAI(ctx, messages)
	if err != nil {
		os.logger.Error("Failed to generate response", err)
		os.logger.End("Chat Response Generation")
		return "", err
	}

	os.logger.Success("Response generated")
	os.logger.End("Chat Response Generation")
	return content, nil
}

// GenerateFillInTheBlankQuestion generates a fill-in-the-blank question
func (os *OpenAIService) GenerateFillInTheBlankQuestion(ctx context.Context, conversationContent string, topic string) (*models.FillInTheBlankQuestionResponse, error) {
	os.logger.Start("Fill-in-the-blank Question Generation")

	systemPrompt := prompts.FillInTheBlankQuestionSystemPrompt()
	userPrompt := prompts.FillInTheBlankQuestionUserPrompt(conversationContent, topic)

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: userPrompt},
	}

	content, err := os.callOpenAI(ctx, messages)
	if err != nil {
		os.logger.Error("Failed to generate question", err)
		os.logger.End("Fill-in-the-blank Question Generation")
		return nil, err
	}

	// Parse response
	questionData, err := os.parseQuestionResponse(content)
	if err != nil {
		os.logger.Error("Failed to parse response", err)
		os.logger.End("Fill-in-the-blank Question Generation")
		return nil, err
	}

	// Convert options to models.QuestionOption
	options := make([]models.QuestionOption, len(questionData.Options))
	for i, opt := range questionData.Options {
		options[i] = models.QuestionOption{ID: opt.ID, Text: opt.Text}
	}

	response := &models.FillInTheBlankQuestionResponse{
		Question:      questionData.Text,
		Options:       options,
		CorrectAnswer: questionData.CorrectAnswer,
	}

	os.logger.Success("Question generated")
	os.logger.End("Fill-in-the-blank Question Generation")
	return response, nil
}

// GenerateMultipleChoiceQuestion generates a multiple choice question
func (os *OpenAIService) GenerateMultipleChoiceQuestion(ctx context.Context, conversationContent string, topic string) (*models.MultipleChoiceQuestionResponse, error) {
	os.logger.Start("Multiple Choice Question Generation")

	systemPrompt := prompts.MultipleChoiceQuestionSystemPrompt()
	userPrompt := prompts.MultipleChoiceQuestionUserPrompt(conversationContent, topic)

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: userPrompt},
	}

	content, err := os.callOpenAI(ctx, messages)
	if err != nil {
		os.logger.Error("Failed to generate question", err)
		os.logger.End("Multiple Choice Question Generation")
		return nil, err
	}

	// Parse response
	questionData, err := os.parseQuestionResponse(content)
	if err != nil {
		os.logger.Error("Failed to parse response", err)
		os.logger.End("Multiple Choice Question Generation")
		return nil, err
	}

	// Convert options to models.QuestionOption
	options := make([]models.QuestionOption, len(questionData.Options))
	for i, opt := range questionData.Options {
		options[i] = models.QuestionOption{ID: opt.ID, Text: opt.Text}
	}

	response := &models.MultipleChoiceQuestionResponse{
		Question:      questionData.Text,
		Options:       options,
		CorrectAnswer: questionData.CorrectAnswer,
	}

	os.logger.Success("Question generated")
	os.logger.End("Multiple Choice Question Generation")
	return response, nil
}

// EvaluateUserResponseQuality evaluates the quality of a user's response
func (os *OpenAIService) EvaluateUserResponseQuality(ctx context.Context, userMessage string, contextMessages []string, profileInfo *models.PersonalInfoListResponse) (int, error) {
	os.logger.Start("User Response Quality Evaluation")

	systemPrompt := prompts.UserResponseEvaluationSystemPrompt()
	userPrompt := prompts.UserResponseEvaluationUserPrompt(userMessage, contextMessages, profileInfo)

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: userPrompt},
	}

	content, err := os.callOpenAI(ctx, messages)
	if err != nil {
		os.logger.Error("Failed to evaluate response", err)
		os.logger.End("User Response Quality Evaluation")
		return util.DefaultResponseScore, err
	}

	// Parse response
	var evalResult struct {
		Score     int    `json:"score"`
		Reasoning string `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(content), &evalResult); err != nil {
		os.logger.Warn("Failed to parse evaluation response, using default score", err)
		os.logger.End("User Response Quality Evaluation")
		return util.DefaultResponseScore, nil
	}

	// Clamp score
	if evalResult.Score < util.MinScore {
		evalResult.Score = util.MinScore
	} else if evalResult.Score > util.MaxScore {
		evalResult.Score = util.MaxScore
	}

	os.logger.KeyValue("Score", evalResult.Score, "Reasoning", evalResult.Reasoning)
	os.logger.End("User Response Quality Evaluation")
	return evalResult.Score, nil
}

// EvaluateMemory evaluates user's memory based on game result
func (os *OpenAIService) EvaluateMemory(ctx context.Context, question string, userAnswer string, isCorrect bool, responseTimeMs int64, topic string) (*models.MemoryEvaluation, error) {
	os.logger.Start("Memory Evaluation")

	systemPrompt := prompts.MemoryEvaluationSystemPrompt()
	userPrompt := prompts.MemoryEvaluationUserPrompt(question, userAnswer, isCorrect, responseTimeMs, topic)

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: userPrompt},
	}

	content, err := os.callOpenAI(ctx, messages)
	if err != nil {
		os.logger.Error("Failed to evaluate memory", err)
		os.logger.End("Memory Evaluation")
		return nil, err
	}

	os.logger.Info("Memory evaluation completed")
	os.logger.End("Memory Evaluation")

	// Parse response
	var evalResult struct {
		RetentionScore float32 `json:"retention_score"`
		Confidence     string  `json:"confidence"`
		Recommendation string  `json:"recommendation"`
	}

	if err := json.Unmarshal([]byte(content), &evalResult); err != nil {
		return nil, fmt.Errorf("failed to parse memory evaluation: %w", err)
	}

	return &models.MemoryEvaluation{
		Topic:          topic,
		RetentionScore: evalResult.RetentionScore,
		Confidence:     evalResult.Confidence,
		Recommendation: evalResult.Recommendation,
	}, nil
}

// callOpenAI makes a call to OpenAI API with given messages
func (os *OpenAIService) callOpenAI(ctx context.Context, messages []openai.ChatCompletionMessage) (string, error) {
	resp, err := os.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       os.model,
		Messages:    messages,
		Temperature: os.temperature,
		MaxTokens:   os.maxTokens,
	})

	if err != nil {
		return "", fmt.Errorf("openai api call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from openai")
	}

	return resp.Choices[0].Message.Content, nil
}

// parseQuestionResponse parses OpenAI's question response JSON
func (os *OpenAIService) parseQuestionResponse(content string) (Question, error) {
	var response Question

	if err := json.Unmarshal([]byte(content), &response); err != nil {
		return Question{}, fmt.Errorf("failed to parse question response json: %w", err)
	}

	if response.Text == "" || len(response.Options) == 0 || response.CorrectAnswer == "" {
		return Question{}, fmt.Errorf("incomplete question response from openai")
	}

	return response, nil
}

// Question represents a parsed question response
type Question struct {
	Text    string `json:"question"`
	Options []struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	} `json:"options"`
	CorrectAnswer string `json:"correct_answer"`
}
