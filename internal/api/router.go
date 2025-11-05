package api

import (
	"github.com/gin-gonic/gin"

	"llm/internal/api/handler"
	"llm/internal/api/middleware"
	"llm/internal/config"
	"llm/internal/service"
)

// Router sets up all API routes
func Router(cfg *config.Config, chatService *service.ChatService, gameService *service.GameService) *gin.Engine {
	router := gin.Default()

	// Apply middlewares
	router.Use(middleware.RequestIDMiddleware())

	// Create handlers
	chatHandler := handler.NewChatHandler(chatService)
	gameHandler := handler.NewGameHandler(gameService)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"service": "llm-server",
		})
	})

	// Chat API routes
	chat := router.Group("/api")
	{
		chat.POST("/chat", chatHandler.Handle)
	}

	// Game API routes
	game := router.Group("/api/game")
	{
		game.POST("/question", gameHandler.GenerateQuestion)
		game.POST("/result", gameHandler.EvaluateResult)
	}

	return router
}