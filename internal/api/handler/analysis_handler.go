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

// ProcessDomainAnalysisOnly handles domain analysis only requests (without report)
// @Summary Process domain analysis only
// @Description Analyze user's conversation history and incorrect quizzes across 4 domains without generating a report
// @Tags Analysis
// @Accept json
// @Produce json
// @Param request body models.AnalysisRequest true "Analysis request (user_id required)"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /api/analysis/domains [post]
func (h *AnalysisHandler) ProcessDomainAnalysisOnly(c *gin.Context) {
	var req models.AnalysisRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request format", err.Error())
		return
	}

	if req.UserID == "" {
		h.respondError(c, http.StatusBadRequest, "INVALID_USER_ID", "User ID cannot be empty", nil)
		return
	}

	resp, err := h.analysisService.ProcessDomainAnalysisOnly(c.Request.Context(), &req)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "DOMAIN_ANALYSIS_FAILED", "Failed to process domain analysis", err.Error())
		return
	}

	h.respondSuccess(c, http.StatusOK, resp)
}

// ProcessReportGeneration handles report generation from domain scores
// @Summary Generate professional report from domain scores
// @Description Generate a professional markdown report based on provided domain analysis scores
// @Tags Analysis
// @Accept json
// @Produce json
// @Param request body models.ReportGenerationRequest true "Report generation request (4 domains required)"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /api/analysis/report [post]
func (h *AnalysisHandler) ProcessReportGeneration(c *gin.Context) {
	var req models.ReportGenerationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request format", err.Error())
		return
	}

	if len(req.Domains) != 4 {
		h.respondError(c, http.StatusBadRequest, "INVALID_DOMAINS", "Exactly 4 domains are required", nil)
		return
	}

	report, err := h.analysisService.ProcessReportGenerationOnly(c.Request.Context(), &req)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "REPORT_GENERATION_FAILED", "Failed to generate report", err.Error())
		return
	}

	response := models.ReportGenerationResponse{
		Report:      report,
		GeneratedAt: time.Now(),
	}

	h.respondSuccess(c, http.StatusOK, response)
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
