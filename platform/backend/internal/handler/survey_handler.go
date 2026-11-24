package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// SurveyHandler handles user-facing in-app survey endpoints.
type SurveyHandler struct {
	surveyService *service.SurveyService
}

func NewSurveyHandler(surveyService *service.SurveyService) *SurveyHandler {
	return &SurveyHandler{surveyService: surveyService}
}

type submitSurveyRequest struct {
	Answers         service.SurveyAnswers `json:"answers"`
	DurationSeconds int                   `json:"duration_seconds"`
}

type surveyStateRequest struct {
	Action string `json:"action" binding:"required,oneof=snooze dismiss"`
}

// ListActive returns eligible surveys for the current user.
// GET /api/v1/surveys/active
func (h *SurveyHandler) ListActive(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	items, err := h.surveyService.ListActiveForUser(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.UserSurvey, 0, len(items))
	for i := range items {
		out = append(out, *dto.UserSurveyFromService(&items[i]))
	}
	response.Success(c, out)
}

// Submit submits user responses and issues any configured survey reward.
// POST /api/v1/surveys/:id/submit
func (h *SurveyHandler) Submit(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	surveyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || surveyID <= 0 {
		response.BadRequest(c, "Invalid survey ID")
		return
	}

	var req submitSurveyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	out, err := h.surveyService.Submit(c.Request.Context(), subject.UserID, surveyID, &service.SubmitSurveyInput{
		Answers:         req.Answers,
		DurationSeconds: req.DurationSeconds,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.SubmitSurveyResultFromService(out))
}

// RecordState records snooze ("later") or permanent dismissal for a survey.
// POST /api/v1/surveys/:id/state
func (h *SurveyHandler) RecordState(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	surveyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || surveyID <= 0 {
		response.BadRequest(c, "Invalid survey ID")
		return
	}

	var req surveyStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	snoozeUntil, err := h.surveyService.RecordUserAction(c.Request.Context(), subject.UserID, surveyID, req.Action)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"message":      "ok",
		"action":       req.Action,
		"snooze_until": snoozeUntil,
	})
}
