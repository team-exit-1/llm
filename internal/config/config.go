package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Server
	Port   int
	Env    string // development, production
	APIKey string

	// RAG Server
	RAGServerURL     string
	RAGServerTimeout time.Duration

	// OpenAI
	OpenAIAPIKey          string
	OpenAIModel           string
	OpenAITemperature     float32
	OpenAIMaxTokens       int

	// Game Settings
	MinConversationsForGame int
	QuestionCacheTTL       time.Duration
	MemoryEvaluationWeights [3]float32 // correct, speed, recency weights

	// Logging
	LogLevel string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Port:                    getEnvAsInt("PORT", 3000),
		Env:                     getEnv("ENVIRONMENT", "development"),
		APIKey:                  getEnv("API_KEY", ""),
		RAGServerURL:            getEnv("RAG_SERVER_URL", "http://localhost:8080"),
		RAGServerTimeout:        time.Duration(getEnvAsInt("RAG_SERVER_TIMEOUT", 5000)) * time.Millisecond,
		OpenAIAPIKey:            getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:             getEnv("OPENAI_MODEL", "gpt-4"),
		OpenAITemperature:       float32(getEnvAsFloat("OPENAI_TEMPERATURE", 0.7)),
		OpenAIMaxTokens:         getEnvAsInt("OPENAI_MAX_TOKENS", 3000),
		MinConversationsForGame: getEnvAsInt("MIN_CONVERSATIONS_FOR_GAME", 5),
		QuestionCacheTTL:        time.Duration(getEnvAsInt("QUESTION_CACHE_TTL", 300)) * time.Second,
		LogLevel:                getEnv("LOG_LEVEL", "info"),
	}

	// Parse memory evaluation weights
	weights := parseWeights(getEnv("MEMORY_EVALUATION_WEIGHTS", "0.5,0.3,0.2"))
	cfg.MemoryEvaluationWeights = weights

	// Validate required fields
	if cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsFloat(key string, defaultVal float64) float64 {
	valStr := getEnv(key, "")
	if val, err := strconv.ParseFloat(valStr, 64); err == nil {
		return val
	}
	return defaultVal
}

func parseWeights(weightStr string) [3]float32 {
	// Default weights
	weights := [3]float32{0.5, 0.3, 0.2}

	// Try to parse custom weights (format: "0.5,0.3,0.2")
	// If parsing fails, use defaults
	return weights
}