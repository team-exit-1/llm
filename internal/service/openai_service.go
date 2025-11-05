package service

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"

	"llm/internal/config"
	"llm/internal/models"
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

// GenerateOXQuestion generates an OX quiz question
func (os *OpenAIService) GenerateOXQuestion(ctx context.Context, conversationContent string, topic string) (*models.OXQuestionResponse, error) {
	systemPrompt := `과거 대화 내용을 바탕으로 사용자의 기억력을 테스트하는 OX 문제를 생성하세요.
생성한 문제는 다음 JSON 형식으로 반환하세요:
{
  "question": "문제 내용",
  "correct_answer": "O 또는 X"
}

주의: JSON만 반환하고 다른 텍스트는 포함하지 마세요.`

	userPrompt := fmt.Sprintf(`대화 내용: %s

주제: %s

위 대화를 바탕으로 OX 문제를 1개 생성하세요.`, conversationContent, topic)

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
		return nil, fmt.Errorf("failed to generate OX question: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from OpenAI")
	}

	return nil, nil // Return parsed result in actual implementation
}

// GenerateMultipleChoiceQuestion generates a multiple choice question
func (os *OpenAIService) GenerateMultipleChoiceQuestion(ctx context.Context, conversationContent string, topic string) (*models.MultipleChoiceQuestionResponse, error) {
	systemPrompt := `과거 대화 내용을 바탕으로 사용자의 기억력을 테스트하는 4지선다 문제를 생성하세요.
생성한 문제는 다음 JSON 형식으로 반환하세요:
{
  "question": "문제 내용",
  "options": [
    {"id": "A", "text": "보기1"},
    {"id": "B", "text": "보기2"},
    {"id": "C", "text": "보기3"},
    {"id": "D", "text": "보기4"}
  ],
  "correct_answer": "A, B, C, D 중 하나"
}

주의: JSON만 반환하고 다른 텍스트는 포함하지 마세요.`

	userPrompt := fmt.Sprintf(`대화 내용: %s

주제: %s

위 대화를 바탕으로 4지선다 문제를 1개 생성하세요.`, conversationContent, topic)

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

	return nil, nil // Return parsed result in actual implementation
}

// EvaluateMemory evaluates user's memory based on game result
func (os *OpenAIService) EvaluateMemory(ctx context.Context, question string, userAnswer string, isCorrect bool, responseTimeMs int64, topic string) (*models.MemoryEvaluation, error) {
	systemPrompt := `사용자의 게임 결과를 분석하여 해당 주제에 대한 기억 정도를 평가하세요.
반환 형식:
{
  "retention_score": 0.0~1.0 범위의 점수,
  "confidence": "high, medium, low 중 하나",
  "recommendation": "평가 설명"
}`

	userPrompt := fmt.Sprintf(`
문제: %s
사용자 답변: %s
정답 여부: %v
응답 시간(ms): %d
주제: %s

위 정보를 바탕으로 사용자의 기억 정도를 평가하세요.`, question, userAnswer, isCorrect, responseTimeMs, topic)

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