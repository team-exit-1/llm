package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"llm/internal/models"
	"llm/internal/service"
)

// ChatHandler handles chat API requests
type ChatHandler struct {
	chatService *service.ChatService
}

// NewChatHandler creates a new chat handler
func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
	}
}

// Handle handles chat requests
// @Summary Process chat message
// @Description Send a message and get a response based on conversation history
// @Tags chat
// @Accept json
// @Produce json
// @Param request body models.ChatRequest true "Chat request"
// @Success 200 {object} models.APIResponse{data=models.ChatResponse}
// @Failure 400 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /api/chat [post]
func (h *ChatHandler) Handle(c *gin.Context) {
	var req models.ChatRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_MESSAGE", "Invalid request format", err.Error())
		return
	}

	if req.Message == "" {
		h.respondError(c, http.StatusBadRequest, "INVALID_MESSAGE", "Message cannot be empty", nil)
		return
	}

	resp, err := h.chatService.ProcessChat(c.Request.Context(), &req)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to process chat", err.Error())
		return
	}

	h.respondSuccess(c, http.StatusOK, resp)
}

// Helper methods

func (h *ChatHandler) respondSuccess(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, models.APIResponse{
		Success: true,
		Data:    data,
		Metadata: models.Metadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: c.GetString("request_id"),
		},
	})
}

func (h *ChatHandler) respondError(c *gin.Context, statusCode int, code string, message string, details interface{}) {
	c.JSON(statusCode, models.APIResponse{
		Success: false,
		Error: &models.ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
		Metadata: models.Metadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: c.GetString("request_id"),
		},
	})
}