package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type dashboardTimezoneRepoStub struct {
	service.UsageLogRepository
	userID int64
	tz     string
	called bool
}

func (s *dashboardTimezoneRepoStub) GetUserDashboardStatsInTimezone(_ context.Context, userID int64, tz string) (*usagestats.UserDashboardStats, error) {
	s.userID, s.tz, s.called = userID, tz, true
	return &usagestats.UserDashboardStats{TodayActualCost: 0.000123}, nil
}

func TestDashboardStatsForwardsTimezoneAndAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tz := range []string{"Asia/Shanghai", "UTC", "Invalid/Timezone", ""} {
		t.Run(tz, func(t *testing.T) {
			repo := &dashboardTimezoneRepoStub{}
			h := NewUsageHandler(service.NewUsageService(repo, nil, nil, nil), nil, nil, nil)
			router := gin.New()
			router.GET("/api/v1/usage/dashboard/stats", func(c *gin.Context) {
				c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
				h.DashboardStats(c)
			})
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/usage/dashboard/stats?user_id=99&timezone="+url.QueryEscape(tz), nil))
			require.Equal(t, http.StatusOK, rec.Code)
			require.True(t, repo.called)
			require.Equal(t, int64(42), repo.userID)
			require.Equal(t, tz, repo.tz)
			require.Contains(t, rec.Body.String(), `"today_actual_cost":0.000123`)
		})
	}
}

func TestDashboardStatsTimezoneRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &dashboardTimezoneRepoStub{}
	h := NewUsageHandler(service.NewUsageService(repo, nil, nil, nil), nil, nil, nil)
	router := gin.New()
	router.GET("/api/v1/usage/dashboard/stats", h.DashboardStats)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/usage/dashboard/stats?timezone=Asia%2FShanghai", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, repo.called)
}
