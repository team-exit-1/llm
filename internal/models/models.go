package models

import "time"

// ===== Chat Models =====

// ChatRequest represents a chat message request
type ChatRequest struct {
	Message string `json:"message" binding:"required"`
	UserID  string `json:"user_id" binding:"required"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	ConversationID string       `json:"conversation_id"`
	Message        string       `json:"message"`
	Response       string       `json:"response"`
	ContextUsed    ContextUsage `json:"context_used"`
	CreatedAt      time.Time    `json:"created_at"`
}

// ContextUsage represents context information used in response
type ContextUsage struct {
	TotalConversations int     `json:"total_conversations"`
	TopScore           float32 `json:"top_score"`
}

// ===== Game Question Models =====

// GameQuestionRequest represents a request to generate a game question
type GameQuestionRequest struct {
	UserID         string `json:"user_id" binding:"required"`
	QuestionType   string `json:"question_type" binding:"required,oneof=fill_in_blank multiple_choice"`
	DifficultyHint string `json:"difficulty_hint,omitempty"` // easy, medium, hard
}

// GameQuestionResponse represents a game question response (base)
type GameQuestionResponse struct {
	QuestionID          string           `json:"question_id"`
	QuestionType        string           `json:"question_type"`
	Question            string           `json:"question"`
	CorrectAnswer       string           `json:"correct_answer"`
	BasedOnConversation string           `json:"based_on_conversation"`
	Difficulty          string           `json:"difficulty"`
	Metadata            QuestionMetadata `json:"metadata"`
}

// FillInTheBlankQuestionResponse represents a fill-in-the-blank question with multiple choice options
type FillInTheBlankQuestionResponse struct {
	QuestionID          string           `json:"question_id"`
	QuestionType        string           `json:"question_type"` // "fill_in_blank"
	Question            string           `json:"question"`
	Options             []QuestionOption `json:"options"`        // 4 choices to fill in the blank
	CorrectAnswer       string           `json:"correct_answer"` // "A", "B", "C", "D"
	BasedOnConversation string           `json:"based_on_conversation"`
	Difficulty          string           `json:"difficulty"`
	Metadata            QuestionMetadata `json:"metadata"`
}

// MultipleChoiceQuestionResponse represents a multiple choice question
type MultipleChoiceQuestionResponse struct {
	QuestionID          string           `json:"question_id"`
	QuestionType        string           `json:"question_type"` // "multiple_choice"
	Question            string           `json:"question"`
	Options             []QuestionOption `json:"options"`
	CorrectAnswer       string           `json:"correct_answer"` // "A", "B", "C", "D"
	BasedOnConversation string           `json:"based_on_conversation"`
	Difficulty          string           `json:"difficulty"`
	Metadata            QuestionMetadata `json:"metadata"`
}

// QuestionOption represents a single option in multiple choice
type QuestionOption struct {
	ID   string `json:"id"` // "A", "B", "C", "D"
	Text string `json:"text"`
}

// QuestionMetadata represents metadata about the generated question
type QuestionMetadata struct {
	Topic                 string  `json:"topic"`
	MemoryScore           float32 `json:"memory_score"`
	DaysSinceConversation int     `json:"days_since_conversation"`
}

// ===== Game Result Models =====

// GameResultRequest represents game result data
type GameResultRequest struct {
	UserID         string `json:"user_id" binding:"required"`
	QuestionID     string `json:"question_id" binding:"required"`
	UserAnswer     string `json:"user_answer" binding:"required"`
	IsCorrect      bool   `json:"is_correct"`
	ResponseTimeMs int64  `json:"response_time_ms"`
	GameSessionID  string `json:"game_session_id"`
}

// GameResultResponse represents the response after processing game result
type GameResultResponse struct {
	ResultID               string                 `json:"result_id"`
	MemoryEvaluation       MemoryEvaluation       `json:"memory_evaluation"`
	NextQuestionSuggestion NextQuestionSuggestion `json:"next_question_suggestion"`
	StoredAt               time.Time              `json:"stored_at"`
}

// MemoryEvaluation represents user's memory evaluation
type MemoryEvaluation struct {
	Topic          string  `json:"topic"`
	RetentionScore float32 `json:"retention_score"`
	Confidence     string  `json:"confidence"` // "high", "medium", "low"
	Recommendation string  `json:"recommendation"`
}

// NextQuestionSuggestion represents suggestions for the next question
type NextQuestionSuggestion struct {
	Difficulty      string `json:"difficulty"`
	TopicPreference string `json:"topic_preference"`
}

// ===== RAG Server Models =====

// RAGConversationSearchRequest represents a request to search RAG conversations
type RAGConversationSearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// RAGConversationSearchResult represents a search result from RAG
type RAGConversationSearchResult struct {
	ConversationID string       `json:"conversation_id"`
	Score          float32      `json:"score"`
	Timestamp      time.Time    `json:"timestamp"`
	Messages       []RAGMessage `json:"messages"`
}

// RAGMessage represents a message in RAG conversation
type RAGMessage struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

// RAGConversationSaveRequest represents a request to save conversation to RAG
type RAGConversationSaveRequest struct {
	ConversationID string       `json:"conversation_id"`
	Messages       []RAGMessage `json:"messages"`
	Metadata       *RAGMetadata `json:"metadata,omitempty"`
}

// RAGMetadata represents metadata for RAG storage
type RAGMetadata struct {
	Source         string  `json:"source,omitempty"`
	SessionID      string  `json:"session_id,omitempty"`
	Type           string  `json:"type,omitempty"` // "chat", "memory_evaluation", etc.
	RetentionScore float32 `json:"retention_score,omitempty"`
	QuestionID     string  `json:"question_id,omitempty"`
}

// ===== API Response Wrappers =====

// APIResponse represents a standard API response wrapper
type APIResponse struct {
	Success  bool        `json:"success"`
	Data     interface{} `json:"data,omitempty"`
	Error    *ErrorInfo  `json:"error,omitempty"`
	Metadata Metadata    `json:"metadata"`
}

// ErrorInfo represents error details in API response
type ErrorInfo struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Metadata represents response metadata
type Metadata struct {
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id"`
}

// ===== Question Storage Models =====

// StoredQuestion represents a question stored in memory for retrieval
type StoredQuestion struct {
	QuestionID          string
	UserID              string
	QuestionType        string
	Question            string
	CorrectAnswer       string
	BasedOnConversation string
	Difficulty          string
	Topic               string
	GeneratedAt         time.Time
	ExpiresAt           time.Time
}

// RAGConversationInfo represents conversation info from RAG
type RAGConversationInfo struct {
	ID        string
	Question  string
	Answer    string
	CreatedAt time.Time
}

// ===== Personal Info Models =====

// PersonalInfoCreateRequest represents a request to create personal info
type PersonalInfoCreateRequest struct {
	UserID     string `json:"user_id" binding:"required"`
	Content    string `json:"content" binding:"required"`
	Category   string `json:"category" binding:"required"`
	Importance string `json:"importance" binding:"required,oneof=high medium low"`
}

// PersonalInfoUpdateRequest represents a request to update personal info
type PersonalInfoUpdateRequest struct {
	Content    string `json:"content"`
	Category   string `json:"category"`
	Importance string `json:"importance,omitempty"`
}

// PersonalInfoResponse represents a personal info response
type PersonalInfoResponse struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	Content    string `json:"content"`
	Category   string `json:"category"`
	Importance string `json:"importance"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// PersonalInfoListResponse represents a list of personal info items
type PersonalInfoListResponse struct {
	Items  []PersonalInfoResponse `json:"items"`
	Total  int                    `json:"total"`
	UserID string                 `json:"user_id"`
}

// ===== Incorrect Quiz Attempt Models =====

// QuizOption represents a single quiz option
type QuizOption struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// QuizInfo represents quiz information
type QuizInfo struct {
	QuizID              string       `json:"quiz_id"`
	QuestionType        string       `json:"question_type"` // "ox", "multiple_choice"
	Question            string       `json:"question"`
	Options             []QuizOption `json:"options,omitempty"`
	Difficulty          string       `json:"difficulty"`
	Topic               string       `json:"topic"`
	BasedOnConversation string       `json:"based_on_conversation"`
	Category            string       `json:"category"`
	Hint                string       `json:"hint,omitempty"`
}

// IncorrectQuizAttempt represents an incorrect quiz attempt
type IncorrectQuizAttempt struct {
	Correct       bool     `json:"correct"`
	AttemptID     int64    `json:"attempt_id"`
	AttemptOrder  int      `json:"attempt_order"`
	Quiz          QuizInfo `json:"quiz"`
	UserAnswer    string   `json:"user_answer"`
	CorrectAnswer string   `json:"correct_answer"`
	IsCorrect     bool     `json:"is_correct"`
	Score         int      `json:"score"`
	AttemptTime   string   `json:"attempt_time"`
}

// IncorrectQuizAttemptsResponse represents a list of incorrect quiz attempts
type IncorrectQuizAttemptsResponse struct {
	Items  []IncorrectQuizAttempt `json:"items,omitempty"`
	Total  int                    `json:"total,omitempty"`
	UserID string                 `json:"user_id,omitempty"`
}
