package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// SurveyHandler handles admin survey CRUD, stats, and response inspection.
type SurveyHandler struct {
	surveyService *service.SurveyService
}

func NewSurveyHandler(surveyService *service.SurveyService) *SurveyHandler {
	return &SurveyHandler{surveyService: surveyService}
}

type CreateSurveyRequest struct {
	Title               string                        `json:"title" binding:"required"`
	Description         string                        `json:"description"`
	Status              string                        `json:"status" binding:"omitempty,oneof=draft active archived"`
	DisplayMode         string                        `json:"display_mode" binding:"omitempty,oneof=corner modal micro"`
	Questions           []service.SurveyQuestion      `json:"questions" binding:"required"`
	Targeting           service.AnnouncementTargeting `json:"targeting"`
	DeliveryPolicy      *service.SurveyDeliveryPolicy `json:"delivery_policy"`
	RewardType          string                        `json:"reward_type" binding:"omitempty,oneof=none balance redeem_code"`
	RewardValue         float64                       `json:"reward_value"`
	RewardBudgetLimit   *int                          `json:"reward_budget_limit"`
	RewardCodeValidDays int                           `json:"reward_code_valid_days"`
	StartsAt            *int64                        `json:"starts_at"`
	EndsAt              *int64                        `json:"ends_at"`
}

type UpdateSurveyRequest struct {
	Title               *string                        `json:"title"`
	Description         *string                        `json:"description"`
	Status              *string                        `json:"status" binding:"omitempty,oneof=draft active archived"`
	DisplayMode         *string                        `json:"display_mode" binding:"omitempty,oneof=corner modal micro"`
	Questions           *[]service.SurveyQuestion      `json:"questions"`
	Targeting           *service.AnnouncementTargeting `json:"targeting"`
	DeliveryPolicy      *service.SurveyDeliveryPolicy  `json:"delivery_policy"`
	RewardType          *string                        `json:"reward_type" binding:"omitempty,oneof=none balance redeem_code"`
	RewardValue         *float64                       `json:"reward_value"`
	RewardBudgetLimit   *int                           `json:"reward_budget_limit"` // 0 clears limit
	RewardCodeValidDays *int                           `json:"reward_code_valid_days"`
	StartsAt            *int64                         `json:"starts_at"` // 0 clears
	EndsAt              *int64                         `json:"ends_at"`   // 0 clears
}

// List handles listing surveys with filters.
// GET /api/v1/admin/surveys
func (h *SurveyHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	status := strings.TrimSpace(c.Query("status"))
	search := strings.TrimSpace(c.Query("search"))
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")
	if len(search) > 200 {
		search = search[:200]
	}

	params := pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	items, paginationResult, err := h.surveyService.List(
		c.Request.Context(),
		params,
		service.SurveyListFilters{Status: status, Search: search},
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.Survey, 0, len(items))
	for i := range items {
		out = append(out, *dto.SurveyFromService(&items[i]))
	}
	response.Paginated(c, out, paginationResult.Total, page, pageSize)
}

// GetByID returns one survey configuration.
// GET /api/v1/admin/surveys/:id
func (h *SurveyHandler) GetByID(c *gin.Context) {
	surveyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || surveyID <= 0 {
		response.BadRequest(c, "Invalid survey ID")
		return
	}

	item, err := h.surveyService.GetByID(c.Request.Context(), surveyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.SurveyFromService(item))
}

// Create creates a new survey.
// POST /api/v1/admin/surveys
func (h *SurveyHandler) Create(c *gin.Context) {
	var req CreateSurveyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	var budgetLimit *int
	if req.RewardBudgetLimit != nil && *req.RewardBudgetLimit > 0 {
		v := *req.RewardBudgetLimit
		budgetLimit = &v
	}

	input := &service.CreateSurveyInput{
		Title:               req.Title,
		Description:         req.Description,
		Status:              req.Status,
		DisplayMode:         req.DisplayMode,
		Questions:           req.Questions,
		Targeting:           req.Targeting,
		DeliveryPolicy:      req.DeliveryPolicy,
		RewardType:          req.RewardType,
		RewardValue:         req.RewardValue,
		RewardBudgetLimit:   budgetLimit,
		RewardCodeValidDays: req.RewardCodeValidDays,
		ActorID:             &subject.UserID,
	}
	if req.StartsAt != nil && *req.StartsAt > 0 {
		t := time.Unix(*req.StartsAt, 0)
		input.StartsAt = &t
	}
	if req.EndsAt != nil && *req.EndsAt > 0 {
		t := time.Unix(*req.EndsAt, 0)
		input.EndsAt = &t
	}

	created, err := h.surveyService.Create(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.SurveyFromService(created))
}

// Update updates an existing survey.
// PUT /api/v1/admin/surveys/:id
func (h *SurveyHandler) Update(c *gin.Context) {
	surveyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || surveyID <= 0 {
		response.BadRequest(c, "Invalid survey ID")
		return
	}

	var req UpdateSurveyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	input := &service.UpdateSurveyInput{
		Title:               req.Title,
		Description:         req.Description,
		Status:              req.Status,
		DisplayMode:         req.DisplayMode,
		Questions:           req.Questions,
		Targeting:           req.Targeting,
		DeliveryPolicy:      req.DeliveryPolicy,
		RewardType:          req.RewardType,
		RewardValue:         req.RewardValue,
		RewardCodeValidDays: req.RewardCodeValidDays,
		ActorID:             &subject.UserID,
	}

	if req.RewardBudgetLimit != nil {
		if *req.RewardBudgetLimit <= 0 {
			var cleared *int = nil
			input.RewardBudgetLimit = &cleared
		} else {
			v := *req.RewardBudgetLimit
			ptr := &v
			input.RewardBudgetLimit = &ptr
		}
	}
	if req.StartsAt != nil {
		if *req.StartsAt == 0 {
			var cleared *time.Time = nil
			input.StartsAt = &cleared
		} else {
			t := time.Unix(*req.StartsAt, 0)
			ptr := &t
			input.StartsAt = &ptr
		}
	}
	if req.EndsAt != nil {
		if *req.EndsAt == 0 {
			var cleared *time.Time = nil
			input.EndsAt = &cleared
		} else {
			t := time.Unix(*req.EndsAt, 0)
			ptr := &t
			input.EndsAt = &ptr
		}
	}

	updated, err := h.surveyService.Update(c.Request.Context(), surveyID, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.SurveyFromService(updated))
}

// Delete removes a survey and cascades responses/states.
// DELETE /api/v1/admin/surveys/:id
func (h *SurveyHandler) Delete(c *gin.Context) {
	surveyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || surveyID <= 0 {
		response.BadRequest(c, "Invalid survey ID")
		return
	}

	if err := h.surveyService.Delete(c.Request.Context(), surveyID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Survey deleted successfully"})
}

// GetStats returns aggregated statistics for a survey.
// GET /api/v1/admin/surveys/:id/stats
func (h *SurveyHandler) GetStats(c *gin.Context) {
	surveyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || surveyID <= 0 {
		response.BadRequest(c, "Invalid survey ID")
		return
	}

	stats, err := h.surveyService.GetStats(c.Request.Context(), surveyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

// ListResponses returns paginated individual survey responses.
// GET /api/v1/admin/surveys/:id/responses
func (h *SurveyHandler) ListResponses(c *gin.Context) {
	surveyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || surveyID <= 0 {
		response.BadRequest(c, "Invalid survey ID")
		return
	}

	page, pageSize := response.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 200 {
		search = search[:200]
	}

	items, paginationResult, err := h.surveyService.ListResponses(
		c.Request.Context(),
		surveyID,
		pagination.PaginationParams{Page: page, PageSize: pageSize},
		search,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.SurveyResponseItem, 0, len(items))
	for i := range items {
		out = append(out, *dto.SurveyResponseItemFromService(&items[i]))
	}
	response.Paginated(c, out, paginationResult.Total, page, pageSize)
}
