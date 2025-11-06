// @title LLM Server API
// @version 1.0
// @description LLM Server for RAG-based Chat and Game Question Generation
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host refo-llm-hackerton.dsmhs.kr
// @basePath /
// @schemes https http

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "llm/docs"
	"llm/internal/api"
	"llm/internal/client"
	"llm/internal/config"
	"llm/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting LLM Server on port %d", cfg.Port)

	// Initialize RAG client
	ragClient := client.NewRAGClient(cfg)

	// Check RAG server health
	ctx, cancel := context.WithTimeout(context.Background(), cfg.RAGServerTimeout)
	defer cancel()

	if healthy, err := ragClient.Health(ctx); !healthy || err != nil {
		log.Printf("Warning: RAG server health check failed: %v", err)
	} else {
		log.Println("RAG server is healthy")
	}

	// Initialize OpenAI service
	openaiService := service.NewOpenAIService(cfg)

	// Initialize services
	chatService := service.NewChatService(cfg, ragClient, openaiService)
	gameService := service.NewGameService(cfg, ragClient, openaiService)

	// Setup router
	router := api.Router(cfg, chatService, gameService)

	// Start server in a goroutine
	addr := fmt.Sprintf(":%d", cfg.Port)
	go func() {
		if err := router.Run(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("LLM Server running on %s", addr)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down LLM server...")
}