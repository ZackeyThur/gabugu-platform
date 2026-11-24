//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type handlerSurveyRepoStub struct {
	surveys   map[int64]*service.Survey
	responses []service.SurveyResponse
	states    map[int64]service.SurveyUserState
}

func (r *handlerSurveyRepoStub) Create(_ context.Context, s *service.Survey) error {
	s.ID = int64(len(r.surveys) + 1)
	cp := *s
	r.surveys[s.ID] = &cp
	return nil
}
func (r *handlerSurveyRepoStub) GetByID(_ context.Context, id int64) (*service.Survey, error) {
	s, ok := r.surveys[id]
	if !ok {
		return nil, service.ErrSurveyNotFound
	}
	cp := *s
	return &cp, nil
}
func (r *handlerSurveyRepoStub) Update(_ context.Context, s *service.Survey) error {
	cp := *s
	r.surveys[s.ID] = &cp
	return nil
}
func (r *handlerSurveyRepoStub) Delete(_ context.Context, id int64) error {
	delete(r.surveys, id)
	return nil
}
func (r *handlerSurveyRepoStub) List(_ context.Context, p pagination.PaginationParams, _ service.SurveyListFilters) ([]service.Survey, *pagination.PaginationResult, error) {
	return nil, &pagination.PaginationResult{Page: p.Page, PageSize: p.PageSize}, nil
}
func (r *handlerSurveyRepoStub) ListActive(_ context.Context, now time.Time) ([]service.Survey, error) {
	out := make([]service.Survey, 0)
	for _, s := range r.surveys {
		if s.IsActiveAt(now) {
			out = append(out, *s)
		}
	}
	return out, nil
}
func (r *handlerSurveyRepoStub) GetUserStates(_ context.Context, _ int64, _ []int64) (map[int64]service.SurveyUserState, error) {
	return r.states, nil
}
func (r *handlerSurveyRepoStub) UpsertUserState(_ context.Context, surveyID, userID int64, status string, snoozeUntil *time.Time, now time.Time) error {
	r.states[surveyID] = service.SurveyUserState{SurveyID: surveyID, UserID: userID, Status: status, SnoozeUntil: snoozeUntil, UpdatedAt: now}
	return nil
}
func (r *handlerSurveyRepoStub) HasUserSubmitted(_ context.Context, surveyID, userID int64) (bool, error) {
	for _, resp := range r.responses {
		if resp.SurveyID == surveyID && resp.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}
func (r *handlerSurveyRepoStub) HasRecentUsage(_ context.Context, _ int64, _ time.Time) (bool, error) {
	return true, nil
}
func (r *handlerSurveyRepoStub) SubmitAndReserveReward(_ context.Context, rec *service.SubmitSurveyRecord) (*service.SubmitSurveyRecordResult, error) {
	resp := service.SurveyResponse{
		ID:              int64(len(r.responses) + 1),
		SurveyID:        rec.SurveyID,
		UserID:          rec.UserID,
		Answers:         rec.Answers,
		DurationSeconds: rec.DurationSeconds,
		RewardStatus:    service.SurveyRewardStatusNone,
		RewardType:      service.SurveyRewardTypeNone,
		SubmittedAt:     rec.SubmittedAt,
	}
	r.responses = append(r.responses, resp)
	return &service.SubmitSurveyRecordResult{Response: &resp}, nil
}
func (r *handlerSurveyRepoStub) UpdateResponseReward(_ context.Context, _ int64, _ string, _ float64, _ string) error {
	return nil
}
func (r *handlerSurveyRepoStub) ReleaseRewardBudget(_ context.Context, _ int64) error { return nil }
func (r *handlerSurveyRepoStub) GetMicroOptionCounts(_ context.Context, surveyID int64, questionID string) (map[string]int64, error) {
	counts := make(map[string]int64)
	for _, resp := range r.responses {
		if resp.SurveyID != surveyID {
			continue
		}
		if ans, ok := resp.Answers[questionID]; ok && ans.ChoiceID != "" {
			counts[ans.ChoiceID]++
		}
	}
	return counts, nil
}
func (r *handlerSurveyRepoStub) ListAllResponsesForStats(_ context.Context, surveyID int64) ([]service.SurveyResponse, error) {
	out := make([]service.SurveyResponse, 0)
	for _, resp := range r.responses {
		if resp.SurveyID == surveyID {
			out = append(out, resp)
		}
	}
	return out, nil
}
func (r *handlerSurveyRepoStub) CountUserStatesByStatus(_ context.Context, _ int64) (int64, int64, error) {
	return 0, 0, nil
}
func (r *handlerSurveyRepoStub) ListResponses(_ context.Context, _ int64, p pagination.PaginationParams, _ string) ([]service.SurveyResponse, *pagination.PaginationResult, error) {
	return r.responses, &pagination.PaginationResult{Total: int64(len(r.responses)), Page: p.Page, PageSize: p.PageSize}, nil
}

type handlerSurveyUserStub struct{ service.UserRepository }

func (*handlerSurveyUserStub) GetByID(_ context.Context, id int64) (*service.User, error) {
	return &service.User{ID: id, Email: "u@example.com", Balance: 20}, nil
}

type handlerSurveySubStub struct{ service.UserSubscriptionRepository }

func (*handlerSurveySubStub) ListActiveByUserID(_ context.Context, _ int64) ([]service.UserSubscription, error) {
	return nil, nil
}

func TestSurveyHandlerListSubmitAndState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &handlerSurveyRepoStub{
		surveys: map[int64]*service.Survey{
			1: {
				ID:          1,
				Title:       "快问",
				Status:      service.SurveyStatusActive,
				DisplayMode: service.SurveyDisplayModeMicro,
				DeliveryPolicy: service.SurveyDeliveryPolicy{
					SnoozeDays:       3,
					ShowMicroResults: true,
				},
				Questions: []service.SurveyQuestion{
					{
						ID:       "poll",
						Type:     service.SurveyQuestionTypeSingle,
						Title:    "选择一项",
						Required: true,
						Options: []service.SurveyQuestionOption{
							{ID: "a", Label: "选项 A"},
							{ID: "b", Label: "选项 B"},
						},
					},
				},
			},
		},
		states: make(map[int64]service.SurveyUserState),
	}
	svc := service.NewSurveyServiceWithIssuer(repo, &handlerSurveyUserStub{}, &handlerSurveySubStub{}, nil)
	h := NewSurveyHandler(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 101})
		c.Next()
	})
	r.GET("/api/v1/surveys/active", h.ListActive)
	r.POST("/api/v1/surveys/:id/submit", h.Submit)
	r.POST("/api/v1/surveys/:id/state", h.RecordState)

	// 1. 拉取活跃问卷
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/surveys/active", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"display_mode":"micro"`)

	// 2. 提交答卷并返回实时投票占比
	body, _ := json.Marshal(map[string]any{
		"answers":          map[string]any{"poll": map[string]any{"choice_id": "a"}},
		"duration_seconds": 5,
	})
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/surveys/1/submit", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())
	require.Contains(t, w2.Body.String(), `"micro_results"`)
	require.Contains(t, w2.Body.String(), `"percent":100`)

	// 3. 再次提交应被 409 拒绝
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/surveys/1/submit", bytes.NewReader(body))
	req3.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusConflict, w3.Code)
}
