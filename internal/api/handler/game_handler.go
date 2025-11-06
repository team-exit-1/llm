package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"llm/internal/models"
	"llm/internal/service"
)

// GameHandler handles game API requests
type GameHandler struct {
	gameService *service.GameService
}

// NewGameHandler creates a new game handler
func NewGameHandler(gameService *service.GameService) *GameHandler {
	return &GameHandler{
		gameService: gameService,
	}
}

// GenerateQuestion handles game question generation
// @Summary Generate a game question
// @Description Generate an OX or multiple choice question based on user's conversation history
// @Tags Game
// @Accept json
// @Produce json
// @Param request body models.GameQuestionRequest true "Question generation request"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 422 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /api/game/question [post]
func (h *GameHandler) GenerateQuestion(c *gin.Context) {
	var req models.GameQuestionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_GAME_REQUEST", "Invalid request format", err.Error())
		return
	}

	resp, err := h.gameService.GenerateQuestion(c.Request.Context(), &req)
	if err != nil {
		// Handle specific errors
		errMsg := err.Error()
		statusCode := http.StatusInternalServerError
		errCode := "INTERNAL_ERROR"

		if errMsg == "invalid_question_type" {
			statusCode = http.StatusBadRequest
			errCode = "INVALID_QUESTION_TYPE"
		} else if len(errMsg) > 18 && errMsg[:18] == "insufficient_data:" {
			statusCode = http.StatusUnprocessableEntity
			errCode = "INSUFFICIENT_DATA"
		}

		h.respondError(c, statusCode, errCode, errMsg, nil)
		return
	}

	h.respondSuccess(c, http.StatusOK, resp)
}

// EvaluateResult handles game result evaluation
// @Summary Evaluate game result
// @Description Evaluate user's game result and store the evaluation
// @Tags Game
// @Accept json
// @Produce json
// @Param request body models.GameResultRequest true "Game result"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /api/game/result [post]
func (h *GameHandler) EvaluateResult(c *gin.Context) {
	var req models.GameResultRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_GAME_RESULT", "Invalid request format", err.Error())
		return
	}

	resp, err := h.gameService.EvaluateGameResult(c.Request.Context(), &req)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to evaluate result", err.Error())
		return
	}

	h.respondSuccess(c, http.StatusOK, resp)
}

// Helper methods

func (h *GameHandler) respondSuccess(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, models.APIResponse{
		Success: true,
		Data:    data,
		Metadata: models.Metadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: c.GetString("request_id"),
		},
	})
}

func (h *GameHandler) respondError(c *gin.Context, statusCode int, code string, message string, details interface{}) {
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
