package util

// Log message constants
const (
	LogStart    = "=== %s START ==="
	LogEnd      = "=== %s END ===\n"
	LogSection  = "--- %s ---"
	LogError    = "ERROR: %v"
	LogWarning  = "WARNING: %v"
)

// Service constants
const (
	MinRetentionScore       = 0.0
	MaxRetentionScore       = 1.0
	ResponseTimeThreshold   = 5000 // milliseconds
	ConversationCacheTTL    = 1 // hour
	QuestionCacheTTL        = 24 // hours
)

// Difficulty levels
const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)

// Confidence levels
const (
	ConfidenceHigh   = "high"
	ConfidenceMedium = "medium"
	ConfidenceLow    = "low"
)

// Question types
const (
	QuestionTypeFillInBlank  = "fill_in_blank"
	QuestionTypeMultipleChoice = "multiple_choice"
)

// Response score defaults
const (
	DefaultResponseScore = 50
	MinScore            = 0
	MaxScore            = 100
)