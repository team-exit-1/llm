package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/sashabaranov/go-openai"

	"llm/internal/config"
	"llm/internal/models"
	"llm/internal/prompts"
)

// OpenAIService handles interactions with OpenAI API
type OpenAIService struct {
	client      *openai.Client
	model       string
	temperature float32
	maxTokens   int
}

// NewOpenAIService creates a new OpenAI service
func NewOpenAIService(cfg *config.Config) *OpenAIService {
	client := openai.NewClient(cfg.OpenAIAPIKey)
	return &OpenAIService{
		client:      client,
		model:       cfg.OpenAIModel,
		temperature: cfg.OpenAITemperature,
		maxTokens:   cfg.OpenAIMaxTokens,
	}
}

// GenerateChatResponse generates a chat response using OpenAI
func (os *OpenAIService) GenerateChatResponse(ctx context.Context, userMessage string, contextMessages []string) (string, error) {
	// Build messages for OpenAI
	messages := []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: `당신은 사용자의 과거 대화를 바탕으로 도움이 되는 답변을 하는 어시스턴트입니다.
사용자의 이전 대화 내용을 고려하여 일관되고 도움이 되는 답변을 제공하세요.
답변은 친근하고 간결하게 해주세요.`,
		},
	}

	// Add context from previous conversations
	if len(contextMessages) > 0 {
		for _, msg := range contextMessages {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: fmt.Sprintf("[참고 정보] %s", msg),
			})
		}
	}

	// Add user message
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userMessage,
	})

	resp, err := os.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       os.model,
		Messages:    messages,
		Temperature: os.temperature,
		MaxTokens:   os.maxTokens,
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

// GenerateChatResponseWithProfile generates a chat response with user profile information and incorrect quiz attempts
func (os *OpenAIService) GenerateChatResponseWithProfile(ctx context.Context, userMessage string, contextMessages []string, profileInfo *models.PersonalInfoListResponse, incorrectAttempts *models.IncorrectQuizAttemptsResponse) (string, error) {
	log.Printf("\n=== OPENAI SERVICE LOG START ===\n")

	// Build enhanced system prompt with profile context and incorrect attempts
	systemPrompt := prompts.ChatSystemPrompt(profileInfo, incorrectAttempts)

	log.Printf("\n--- System Prompt (LLM Thinking Instructions) ---\n")
	log.Printf("%s\n", systemPrompt)

	// Build messages for OpenAI
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	// Add relevant context from previous conversations with natural integration
	if len(contextMessages) > 0 {
		// Limit to 3 most relevant contexts to keep conversation focused
		contextLimit := 3
		if len(contextMessages) < contextLimit {
			contextLimit = len(contextMessages)
		}

		contextStr := "최근 대화 이력:\n"
		for i := 0; i < contextLimit; i++ {
			contextStr += fmt.Sprintf("- %s\n", contextMessages[i])
		}

		log.Printf("\n--- Context from Previous Conversations ---\n")
		log.Printf("%s\n", contextStr)

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: contextStr,
		})
	} else {
		log.Printf("\n--- Context from Previous Conversations ---\n")
		log.Printf("No context messages available\n")
	}

	// Add user message
	log.Printf("\n--- User Message ---\n")
	log.Printf("%s\n", userMessage)

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userMessage,
	})

	// Log all messages being sent to OpenAI
	log.Printf("\n--- All Messages Sent to OpenAI ---\n")
	messagesJSON, _ := json.MarshalIndent(messages, "", "  ")
	log.Printf("%s\n", string(messagesJSON))

	// Log API request details
	log.Printf("\n--- OpenAI API Request Details ---\n")
	log.Printf("Model: %s\n", os.model)
	log.Printf("Temperature: %.2f\n", os.temperature)
	log.Printf("Max Tokens: %d\n", os.maxTokens)
	log.Printf("Message Count: %d\n", len(messages))

	resp, err := os.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       os.model,
		Messages:    messages,
		Temperature: os.temperature,
		MaxTokens:   os.maxTokens,
	})

	if err != nil {
		log.Printf("ERROR: OpenAI API call failed: %v\n", err)
		log.Printf("=== OPENAI SERVICE LOG END ===\n\n")
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	if len(resp.Choices) == 0 {
		log.Printf("ERROR: No choices returned from OpenAI\n")
		log.Printf("=== OPENAI SERVICE LOG END ===\n\n")
		return "", fmt.Errorf("no choices returned from OpenAI")
	}

	// Log OpenAI response details
	log.Printf("\n--- OpenAI API Response ---\n")
	log.Printf("Response ID: %s\n", resp.ID)
	log.Printf("Model: %s\n", resp.Model)
	log.Printf("Total Tokens: %d (Prompt: %d, Completion: %d)\n",
		resp.Usage.TotalTokens, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
	log.Printf("Finish Reason: %s\n", resp.Choices[0].FinishReason)

	responseContent := resp.Choices[0].Message.Content
	log.Printf("\n--- LLM Response Content ---\n")
	log.Printf("%s\n", responseContent)
	log.Printf("\n=== OPENAI SERVICE LOG END ===\n\n")

	return responseContent, nil
}

// GenerateFillInTheBlankQuestion generates a fill-in-the-blank question with multiple choice options
func (os *OpenAIService) GenerateFillInTheBlankQuestion(ctx context.Context, conversationContent string, topic string) (*models.FillInTheBlankQuestionResponse, error) {
	systemPrompt := prompts.FillInTheBlankQuestionSystemPrompt()
	userPrompt := prompts.FillInTheBlankQuestionUserPrompt(conversationContent, topic)

	resp, err := os.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: os.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		Temperature: 0.7,
		MaxTokens:   200,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate fill-in-the-blank question: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from OpenAI")
	}

	responseContent := resp.Choices[0].Message.Content

	// Parse JSON response
	type QuestionResponse struct {
		Question string `json:"question"`
		Options  []struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"options"`
		CorrectAnswer string `json:"correct_answer"`
	}

	var questionData QuestionResponse
	if err := json.Unmarshal([]byte(responseContent), &questionData); err != nil {
		return nil, fmt.Errorf("failed to parse fill-in-the-blank response: %w", err)
	}

	// Convert to response format
	options := make([]models.QuestionOption, len(questionData.Options))
	for i, opt := range questionData.Options {
		options[i] = models.QuestionOption{ID: opt.ID, Text: opt.Text}
	}

	return &models.FillInTheBlankQuestionResponse{
		Question:      questionData.Question,
		Options:       options,
		CorrectAnswer: questionData.CorrectAnswer,
	}, nil
}

// GenerateMultipleChoiceQuestion generates a multiple choice question
func (os *OpenAIService) GenerateMultipleChoiceQuestion(ctx context.Context, conversationContent string, topic string) (*models.MultipleChoiceQuestionResponse, error) {
	systemPrompt := prompts.MultipleChoiceQuestionSystemPrompt()
	userPrompt := prompts.MultipleChoiceQuestionUserPrompt(conversationContent, topic)

	resp, err := os.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: os.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		Temperature: 0.7,
		MaxTokens:   300,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate multiple choice question: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from OpenAI")
	}

	responseContent := resp.Choices[0].Message.Content

	// Parse JSON response
	type QuestionResponse struct {
		Question string `json:"question"`
		Options  []struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"options"`
		CorrectAnswer string `json:"correct_answer"`
	}

	var questionData QuestionResponse
	if err := json.Unmarshal([]byte(responseContent), &questionData); err != nil {
		return nil, fmt.Errorf("failed to parse multiple choice response: %w", err)
	}

	// Convert to response format
	options := make([]models.QuestionOption, len(questionData.Options))
	for i, opt := range questionData.Options {
		options[i] = models.QuestionOption{ID: opt.ID, Text: opt.Text}
	}

	return &models.MultipleChoiceQuestionResponse{
		Question:      questionData.Question,
		Options:       options,
		CorrectAnswer: questionData.CorrectAnswer,
	}, nil
}

// EvaluateUserResponseQuality evaluates the quality of user's response in conversation
func (os *OpenAIService) EvaluateUserResponseQuality(ctx context.Context, userMessage string, contextMessages []string, profileInfo *models.PersonalInfoListResponse) (int, error) {
	log.Printf("\n=== USER RESPONSE QUALITY EVALUATION ===\n")

	// Build evaluation prompts using prompts package
	systemPrompt := prompts.UserResponseEvaluationSystemPrompt()
	userPrompt := prompts.UserResponseEvaluationUserPrompt(userMessage, contextMessages, profileInfo)

	log.Printf("\n--- Evaluation System Prompt ---\n")
	log.Printf("%s\n", systemPrompt)
	log.Printf("\n--- Evaluation User Prompt ---\n")
	log.Printf("%s\n", userPrompt)

	resp, err := os.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: os.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		Temperature: 0.7,
		MaxTokens:   200,
	})

	if err != nil {
		log.Printf("ERROR: Failed to evaluate user response: %v\n", err)
		return 50, fmt.Errorf("failed to evaluate response: %w", err)
	}

	if len(resp.Choices) == 0 {
		log.Printf("ERROR: No response from OpenAI\n")
		return 50, fmt.Errorf("no response from OpenAI")
	}

	responseContent := resp.Choices[0].Message.Content
	log.Printf("\n--- OpenAI Evaluation Response ---\n")
	log.Printf("%s\n", responseContent)

	// Parse JSON response to extract score
	var evaluationResult struct {
		Score     int    `json:"score"`
		Reasoning string `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(responseContent), &evaluationResult); err != nil {
		log.Printf("WARNING: Failed to parse evaluation JSON, using default score of 50\n")
		log.Printf("Parse error: %v\n", err)
		return 50, nil
	}

	// Ensure score is within 0-100 range
	if evaluationResult.Score < 0 {
		evaluationResult.Score = 0
	} else if evaluationResult.Score > 100 {
		evaluationResult.Score = 100
	}

	log.Printf("\n--- Final Quality Score ---\n")
	log.Printf("Score: %d/100\n", evaluationResult.Score)
	log.Printf("Reasoning: %s\n\n", evaluationResult.Reasoning)

	return evaluationResult.Score, nil
}

// EvaluateMemory evaluates user's memory based on game result
func (os *OpenAIService) EvaluateMemory(ctx context.Context, question string, userAnswer string, isCorrect bool, responseTimeMs int64, topic string) (*models.MemoryEvaluation, error) {
	systemPrompt := prompts.MemoryEvaluationSystemPrompt()
	userPrompt := prompts.MemoryEvaluationUserPrompt(question, userAnswer, isCorrect, responseTimeMs, topic)

	resp, err := os.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: os.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		Temperature: 0.7,
		MaxTokens:   200,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to evaluate memory: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from OpenAI")
	}

	return nil, nil // Return parsed result in actual implementation
}
