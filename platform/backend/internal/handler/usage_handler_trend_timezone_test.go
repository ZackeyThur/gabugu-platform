package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	appTimezone "github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type userTrendTimezoneRepoStub struct {
	service.UsageLogRepository
	start, end time.Time
	filters    usagestats.UsageLogFilters
}

func (s *userTrendTimezoneRepoStub) GetUsageTrendWithUsageFilters(_ context.Context, start, end time.Time, _ string, filters usagestats.UsageLogFilters) ([]usagestats.TrendDataPoint, error) {
	s.start, s.end, s.filters = start, end, filters
	return []usagestats.TrendDataPoint{}, nil
}

func (s *userTrendTimezoneRepoStub) GetModelStatsWithUsageFiltersBySource(_ context.Context, start, end time.Time, filters usagestats.UsageLogFilters, _ string) ([]usagestats.ModelStat, error) {
	s.start, s.end, s.filters = start, end, filters
	return []usagestats.ModelStat{}, nil
}

func TestDashboardAnalysisRangeRetainsResolvedTimezone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previous := appTimezone.Name()
	require.NoError(t, appTimezone.Init("UTC"))
	t.Cleanup(func() { require.NoError(t, appTimezone.Init(previous)) })
	for _, tc := range []struct {
		query, wantZone, wantStartUTC string
	}{
		{"Asia/Shanghai", "Asia/Shanghai", "2026-11-08T16:00:00Z"},
		{"UTC", "UTC", "2026-11-09T00:00:00Z"},
		{"Invalid/Timezone", "UTC", "2026-11-09T00:00:00Z"},
		{"", "UTC", "2026-11-09T00:00:00Z"},
	} {
		for _, endpoint := range []string{"trend", "models"} {
			t.Run(tc.query+"/"+endpoint, func(t *testing.T) {
				repo := &userTrendTimezoneRepoStub{}
				h := NewUsageHandler(service.NewUsageService(repo, nil, nil, nil), nil, nil, nil)
				router := gin.New()
				router.GET("/analysis", func(c *gin.Context) {
					c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
					if endpoint == "trend" {
						h.DashboardTrend(c)
					} else {
						h.DashboardModels(c)
					}
				})
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/analysis?start_date=2026-11-09&end_date=2026-11-09&timezone="+url.QueryEscape(tc.query), nil))
				require.Equal(t, http.StatusOK, rec.Code)
				require.Equal(t, tc.wantZone, repo.start.Location().String())
				require.Equal(t, tc.wantStartUTC, repo.start.UTC().Format(time.RFC3339))
				require.Equal(t, repo.start.AddDate(0, 0, 1), repo.end)
				require.Equal(t, int64(42), repo.filters.UserID)
			})
		}
	}
}
