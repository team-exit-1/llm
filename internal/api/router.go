package api

import (
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"llm/internal/api/handler"
	"llm/internal/api/middleware"
	"llm/internal/config"
	"llm/internal/service"
)

// Router sets up all API routes
func Router(cfg *config.Config, chatService *service.ChatService, gameService *service.GameService, analysisService *service.AnalysisService) *gin.Engine {
	router := gin.Default()

	// Apply middlewares
	router.Use(middleware.RequestIDMiddleware())

	// Create handlers
	chatHandler := handler.NewChatHandler(chatService)
	gameHandler := handler.NewGameHandler(gameService)
	analysisHandler := handler.NewAnalysisHandler(analysisService)
	healthHandler := handler.NewHealthHandler()

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// Health check
	router.GET("/health", healthHandler.Check)

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

	// Analysis API routes
	analysis := router.Group("/api")
	{
		analysis.POST("/analysis", analysisHandler.ProcessAnalysis)                   // 통합: 도메인 분석 + 리포트
		analysis.POST("/analysis/domains", analysisHandler.ProcessDomainAnalysisOnly) // 도메인 분석만
		analysis.POST("/analysis/report", analysisHandler.ProcessReportGeneration)    // 리포트 생성만
	}

	return router
}
