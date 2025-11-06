package util

import (
	"encoding/json"
	"fmt"
)

// QuestionResponse represents the JSON structure returned from OpenAI for question generation
type QuestionResponse struct {
	Question      string      `json:"question"`
	Options       []Option    `json:"options"`
	CorrectAnswer string      `json:"correct_answer"`
}

// Option represents a single question option
type Option struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// EvaluationResponse represents the JSON structure returned from OpenAI for evaluation
type EvaluationResponse struct {
	Score     int    `json:"score"`
	Reasoning string `json:"reasoning"`
}

// MemoryEvaluationResponse represents the JSON structure returned from OpenAI for memory evaluation
type MemoryEvaluationResponse struct {
	RetentionScore float32 `json:"retention_score"`
	Confidence     string  `json:"confidence"`
	Recommendation string  `json:"recommendation"`
}

// ParseQuestionResponse parses OpenAI response into QuestionResponse
func ParseQuestionResponse(content string) (*QuestionResponse, error) {
	var response QuestionResponse
	if err := json.Unmarshal([]byte(content), &response); err != nil {
		return nil, fmt.Errorf("failed to parse question response: %w", err)
	}

	// Validate
	if response.Question == "" {
		return nil, fmt.Errorf("question response missing question field")
	}
	if len(response.Options) == 0 {
		return nil, fmt.Errorf("question response missing options")
	}
	if response.CorrectAnswer == "" {
		return nil, fmt.Errorf("question response missing correct_answer field")
	}

	return &response, nil
}

// ParseEvaluationResponse parses OpenAI response into EvaluationResponse
func ParseEvaluationResponse(content string) (*EvaluationResponse, error) {
	var response EvaluationResponse
	if err := json.Unmarshal([]byte(content), &response); err != nil {
		return nil, fmt.Errorf("failed to parse evaluation response: %w", err)
	}

	// Validate and clamp score
	if response.Score < 0 {
		response.Score = 0
	} else if response.Score > 100 {
		response.Score = 100
	}

	return &response, nil
}

// ParseMemoryEvaluationResponse parses OpenAI response into MemoryEvaluationResponse
func ParseMemoryEvaluationResponse(content string) (*MemoryEvaluationResponse, error) {
	var response MemoryEvaluationResponse
	if err := json.Unmarshal([]byte(content), &response); err != nil {
		return nil, fmt.Errorf("failed to parse memory evaluation response: %w", err)
	}

	// Validate and clamp score
	if response.RetentionScore < 0 {
		response.RetentionScore = 0
	} else if response.RetentionScore > 1 {
		response.RetentionScore = 1
	}

	return &response, nil
}