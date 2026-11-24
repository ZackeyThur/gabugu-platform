package admin

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

type announcementRepoCapture struct {
	service.AnnouncementRepository
	listParams pagination.PaginationParams
	saved      *service.Announcement
}

func (r *announcementRepoCapture) Create(_ context.Context, a *service.Announcement) error {
	a.ID = 42
	cp := *a
	r.saved = &cp
	return nil
}

func (r *announcementRepoCapture) Update(_ context.Context, a *service.Announcement) error {
	cp := *a
	r.saved = &cp
	return nil
}

func (r *announcementRepoCapture) List(ctx context.Context, params pagination.PaginationParams, filters service.AnnouncementListFilters) ([]service.Announcement, *pagination.PaginationResult, error) {
	r.listParams = params
	return []service.Announcement{}, &pagination.PaginationResult{
		Total:    0,
		Page:     params.Page,
		PageSize: params.PageSize,
		Pages:    0,
	}, nil
}

func (r *announcementRepoCapture) GetByID(ctx context.Context, id int64) (*service.Announcement, error) {
	return &service.Announcement{
		ID:        id,
		Title:     "announcement",
		Content:   "content",
		Status:    service.AnnouncementStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

type announcementUserRepoCapture struct {
	service.UserRepository
	listParams pagination.PaginationParams
}

func (r *announcementUserRepoCapture) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters service.UserListFilters) ([]service.User, *pagination.PaginationResult, error) {
	r.listParams = params
	return []service.User{}, &pagination.PaginationResult{
		Total:    0,
		Page:     params.Page,
		PageSize: params.PageSize,
		Pages:    0,
	}, nil
}

type announcementReadRepoCapture struct {
	service.AnnouncementReadRepository
}

func (r *announcementReadRepoCapture) GetReadMapByUsers(ctx context.Context, announcementID int64, userIDs []int64) (map[int64]time.Time, error) {
	return map[int64]time.Time{}, nil
}

type announcementUserSubRepoCapture struct {
	service.UserSubscriptionRepository
}

func newAnnouncementSortTestRouter(announcementRepo *announcementRepoCapture, userRepo *announcementUserRepoCapture) *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := service.NewAnnouncementService(
		announcementRepo,
		&announcementReadRepoCapture{},
		userRepo,
		&announcementUserSubRepoCapture{},
	)
	handler := NewAnnouncementHandler(svc)
	router := gin.New()
	router.GET("/admin/announcements", handler.List)
	router.POST("/admin/announcements", func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 1})
		handler.Create(c)
	})
	router.PUT("/admin/announcements/:id", func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 1})
		handler.Update(c)
	})
	router.GET("/admin/announcements/:id/read-status", handler.ListReadStatus)
	return router
}

func TestAdminAnnouncementListSortParams(t *testing.T) {
	announcementRepo := &announcementRepoCapture{}
	userRepo := &announcementUserRepoCapture{}
	router := newAnnouncementSortTestRouter(announcementRepo, userRepo)

	req := httptest.NewRequest(http.MethodGet, "/admin/announcements?sort_by=title&sort_order=ASC", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "title", announcementRepo.listParams.SortBy)
	require.Equal(t, "ASC", announcementRepo.listParams.SortOrder)
}

func TestAdminAnnouncementListSortDefaults(t *testing.T) {
	announcementRepo := &announcementRepoCapture{}
	userRepo := &announcementUserRepoCapture{}
	router := newAnnouncementSortTestRouter(announcementRepo, userRepo)

	req := httptest.NewRequest(http.MethodGet, "/admin/announcements", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "created_at", announcementRepo.listParams.SortBy)
	require.Equal(t, "desc", announcementRepo.listParams.SortOrder)
}

func TestAdminAnnouncementReadStatusSortParams(t *testing.T) {
	announcementRepo := &announcementRepoCapture{}
	userRepo := &announcementUserRepoCapture{}
	router := newAnnouncementSortTestRouter(announcementRepo, userRepo)

	req := httptest.NewRequest(http.MethodGet, "/admin/announcements/1/read-status?sort_by=balance&sort_order=DESC", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "balance", userRepo.listParams.SortBy)
	require.Equal(t, "DESC", userRepo.listParams.SortOrder)
}

func TestAdminAnnouncementReadStatusSortDefaults(t *testing.T) {
	announcementRepo := &announcementRepoCapture{}
	userRepo := &announcementUserRepoCapture{}
	router := newAnnouncementSortTestRouter(announcementRepo, userRepo)

	req := httptest.NewRequest(http.MethodGet, "/admin/announcements/1/read-status", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "email", userRepo.listParams.SortBy)
	require.Equal(t, "asc", userRepo.listParams.SortOrder)
}

func TestAdminAnnouncementCreateAndUpdateDashboardPosterFields(t *testing.T) {
	announcementRepo := &announcementRepoCapture{}
	userRepo := &announcementUserRepoCapture{}
	router := newAnnouncementSortTestRouter(announcementRepo, userRepo)

	createBody := bytes.NewBufferString(`{"title":"海报活动","content":"正文","status":"active","notify_mode":"silent","show_on_dashboard":true,"dashboard_link":"/purchase?amount=100"}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/announcements", createBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, announcementRepo.saved)
	require.True(t, announcementRepo.saved.ShowOnDashboard)
	require.Equal(t, "/purchase?amount=100", announcementRepo.saved.DashboardLink)

	var resp struct {
		Data struct {
			ShowOnDashboard bool   `json:"show_on_dashboard"`
			DashboardLink   string `json:"dashboard_link"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.True(t, resp.Data.ShowOnDashboard)
	require.Equal(t, "/purchase?amount=100", resp.Data.DashboardLink)

	badBody := bytes.NewBufferString(`{"show_on_dashboard":true,"dashboard_link":"javascript:alert(1)"}`)
	badReq := httptest.NewRequest(http.MethodPut, "/admin/announcements/42", badBody)
	badReq.Header.Set("Content-Type", "application/json")
	badRec := httptest.NewRecorder()
	router.ServeHTTP(badRec, badReq)
	require.Equal(t, http.StatusBadRequest, badRec.Code)
}
