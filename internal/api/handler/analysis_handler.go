package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"llm/internal/models"
	"llm/internal/service"
)

// AnalysisHandler handles domain analysis API requests
type AnalysisHandler struct {
	analysisService *service.AnalysisService
}

// NewAnalysisHandler creates a new analysis handler
func NewAnalysisHandler(analysisService *service.AnalysisService) *AnalysisHandler {
	return &AnalysisHandler{
		analysisService: analysisService,
	}
}

// ProcessAnalysis handles analysis requests
// @Summary Process domain analysis
// @Description Analyze user's conversation history and incorrect quizzes across 4 domains (family, life events, career, hobbies)
// @Tags Analysis
// @Accept json
// @Produce json
// @Param request body models.AnalysisRequest true "Analysis request"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /api/analysis [post]
func (h *AnalysisHandler) ProcessAnalysis(c *gin.Context) {
	var req models.AnalysisRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request format", err.Error())
		return
	}

	if req.UserID == "" {
		h.respondError(c, http.StatusBadRequest, "INVALID_USER_ID", "User ID cannot be empty", nil)
		return
	}

	resp, err := h.analysisService.ProcessAnalysisRequest(c.Request.Context(), &req)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "ANALYSIS_FAILED", "Failed to process analysis", err.Error())
		return
	}

	h.respondSuccess(c, http.StatusOK, resp)
}

// Helper methods

func (h *AnalysisHandler) respondSuccess(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, models.APIResponse{
		Success: true,
		Data:    data,
		Metadata: models.Metadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: c.GetString("request_id"),
		},
	})
}

func (h *AnalysisHandler) respondError(c *gin.Context, statusCode int, code string, message string, details interface{}) {
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
