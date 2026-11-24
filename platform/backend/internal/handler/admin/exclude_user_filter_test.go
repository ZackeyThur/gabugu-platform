package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type excludeUserUsageCapture struct {
	service.UsageLogRepository
	listFilters    usagestats.UsageLogFilters
	statsFilters   usagestats.UsageLogFilters
	statsCalls     atomic.Int32
	trendFilters   usagestats.UsageLogFilters
	trendCalls     atomic.Int32
	modelFilters   usagestats.UsageLogFilters
	modelCalls     atomic.Int32
	groupFilters   usagestats.UsageLogFilters
	groupCalls     atomic.Int32
	breakdownDim   usagestats.UserBreakdownDimension
	breakdownCalls atomic.Int32
}

func (s *excludeUserUsageCapture) ListWithFilters(_ context.Context, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]service.UsageLog, *pagination.PaginationResult, error) {
	s.listFilters = filters
	return []service.UsageLog{}, &pagination.PaginationResult{Total: 0, Page: params.Page, PageSize: params.PageSize}, nil
}

func (s *excludeUserUsageCapture) GetStatsWithFilters(_ context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	s.statsCalls.Add(1)
	s.statsFilters = filters
	return &usagestats.UsageStats{TotalRequests: filters.ExcludeUserID}, nil
}

func (s *excludeUserUsageCapture) GetUsageTrendWithUsageFilters(_ context.Context, _, _ time.Time, _ string, filters usagestats.UsageLogFilters) ([]usagestats.TrendDataPoint, error) {
	s.trendCalls.Add(1)
	s.trendFilters = filters
	return []usagestats.TrendDataPoint{{Date: "2026-11-20", Requests: filters.ExcludeUserID}}, nil
}

func (s *excludeUserUsageCapture) GetModelStatsWithUsageFiltersBySource(_ context.Context, _, _ time.Time, filters usagestats.UsageLogFilters, _ string) ([]usagestats.ModelStat, error) {
	s.modelCalls.Add(1)
	s.modelFilters = filters
	return []usagestats.ModelStat{{Model: "m", Requests: filters.ExcludeUserID}}, nil
}

func (s *excludeUserUsageCapture) GetGroupStatsWithUsageFilters(_ context.Context, _, _ time.Time, filters usagestats.UsageLogFilters) ([]usagestats.GroupStat, error) {
	s.groupCalls.Add(1)
	s.groupFilters = filters
	return []usagestats.GroupStat{{GroupID: 1, Requests: filters.ExcludeUserID}}, nil
}

func (s *excludeUserUsageCapture) GetUserBreakdownStats(_ context.Context, _, _ time.Time, dim usagestats.UserBreakdownDimension, _ int) ([]usagestats.UserBreakdownItem, error) {
	s.breakdownCalls.Add(1)
	s.breakdownDim = dim
	return []usagestats.UserBreakdownItem{}, nil
}

type excludeUserOpsCaptureRepo struct {
	service.OpsRepository
	lastFilter *service.OpsErrorLogFilter
}

func (r *excludeUserOpsCaptureRepo) ListErrorLogs(_ context.Context, filter *service.OpsErrorLogFilter) (*service.OpsErrorLogList, error) {
	r.lastFilter = filter
	return &service.OpsErrorLogList{Errors: []*service.OpsErrorLog{}, Total: 0, Page: filter.Page, PageSize: filter.PageSize}, nil
}

func newExcludeUserTestRouter(usageRepo *excludeUserUsageCapture, opsRepo *excludeUserOpsCaptureRepo, withHeaderAdmin bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if withHeaderAdmin {
		router.Use(func(c *gin.Context) {
			if raw := c.GetHeader("X-Test-Admin-ID"); raw != "" {
				var id int64
				for _, ch := range raw {
					id = id*10 + int64(ch-'0')
				}
				c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: id})
			}
			c.Next()
		})
	}
	usageSvc := service.NewUsageService(usageRepo, nil, nil, nil)
	usageH := NewUsageHandler(usageSvc, nil, nil, nil)
	dashSvc := service.NewDashboardService(usageRepo, nil, nil, nil)
	dashH := NewDashboardHandler(dashSvc, nil)
	opsSvc := service.NewOpsService(opsRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	opsH := NewOpsHandler(opsSvc)

	router.GET("/admin/usage", usageH.List)
	router.GET("/admin/usage/stats", usageH.Stats)
	router.GET("/admin/dashboard/trend", dashH.GetUsageTrend)
	router.GET("/admin/dashboard/models", dashH.GetModelStats)
	router.GET("/admin/dashboard/groups", dashH.GetGroupStats)
	router.GET("/admin/dashboard/snapshot-v2", dashH.GetSnapshotV2)
	router.GET("/admin/dashboard/user-breakdown", dashH.GetUserBreakdown)
	router.GET("/admin/ops/errors", opsH.GetErrorLogs)
	return router
}

func TestExcludeSelfResolvesCurrentAdminAndIsolatesCaches(t *testing.T) {
	t.Cleanup(func() {
		usageStatsCache = newSnapshotCache(30 * time.Second)
		resetDashboardReadCachesForTest()
	})
	usageStatsCache = newSnapshotCache(30 * time.Second)
	resetDashboardReadCachesForTest()

	usageRepo := &excludeUserUsageCapture{}
	opsRepo := &excludeUserOpsCaptureRepo{}
	router := newExcludeUserTestRouter(usageRepo, opsRepo, true)

	doReq := func(path string, adminID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if adminID != "" {
			req.Header.Set("X-Test-Admin-ID", adminID)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	// Admin 11 with exclude_self=true -> ExcludeUserID=11 across all endpoints
	require.Equal(t, http.StatusOK, doReq("/admin/usage?exclude_self=true", "11").Code)
	require.Equal(t, int64(11), usageRepo.listFilters.ExcludeUserID)

	require.Equal(t, http.StatusOK, doReq("/admin/usage/stats?exclude_self=true&start_date=2026-11-19&end_date=2026-11-20", "11").Code)
	require.Equal(t, int64(11), usageRepo.statsFilters.ExcludeUserID)

	require.Equal(t, http.StatusOK, doReq("/admin/dashboard/models?exclude_self=true&start_date=2026-11-19&end_date=2026-11-20", "11").Code)
	require.Equal(t, int64(11), usageRepo.modelFilters.ExcludeUserID)

	require.Equal(t, http.StatusOK, doReq("/admin/dashboard/groups?exclude_self=true&start_date=2026-11-19&end_date=2026-11-20", "11").Code)
	require.Equal(t, int64(11), usageRepo.groupFilters.ExcludeUserID)

	require.Equal(t, http.StatusOK, doReq("/admin/dashboard/trend?exclude_self=true&start_date=2026-11-19&end_date=2026-11-20&granularity=day", "11").Code)
	require.Equal(t, int64(11), usageRepo.trendFilters.ExcludeUserID)

	require.Equal(t, http.StatusOK, doReq("/admin/dashboard/snapshot-v2?exclude_self=true&start_date=2026-11-19&end_date=2026-11-20&granularity=day&include_stats=false&include_trend=true&include_model_stats=true&include_group_stats=true", "11").Code)
	require.Equal(t, int64(11), usageRepo.trendFilters.ExcludeUserID)

	require.Equal(t, http.StatusOK, doReq("/admin/dashboard/user-breakdown?exclude_self=true&start_date=2026-11-19&end_date=2026-11-20", "11").Code)
	require.Equal(t, int64(11), usageRepo.breakdownDim.ExcludeUserID)

	require.Equal(t, http.StatusOK, doReq("/admin/ops/errors?exclude_self=true&view=all", "11").Code)
	require.NotNil(t, opsRepo.lastFilter)
	require.NotNil(t, opsRepo.lastFilter.ExcludeUserID)
	require.Equal(t, int64(11), *opsRepo.lastFilter.ExcludeUserID)

	// Admin 22 with exclude_self=true must NOT hit Admin 11's cached stats or snapshot-v2
	statsCallsBefore := usageRepo.statsCalls.Load()
	require.Equal(t, http.StatusOK, doReq("/admin/usage/stats?exclude_self=true&start_date=2026-11-19&end_date=2026-11-20", "22").Code)
	require.Equal(t, statsCallsBefore+1, usageRepo.statsCalls.Load())
	require.Equal(t, int64(22), usageRepo.statsFilters.ExcludeUserID)

	trendCallsBefore := usageRepo.trendCalls.Load()
	modelCallsBefore := usageRepo.modelCalls.Load()
	groupCallsBefore := usageRepo.groupCalls.Load()
	require.Equal(t, http.StatusOK, doReq("/admin/dashboard/snapshot-v2?exclude_self=true&start_date=2026-11-19&end_date=2026-11-20&granularity=day&include_stats=false&include_trend=true&include_model_stats=true&include_group_stats=true", "22").Code)
	require.Equal(t, trendCallsBefore+1, usageRepo.trendCalls.Load())
	require.Equal(t, modelCallsBefore+1, usageRepo.modelCalls.Load())
	require.Equal(t, groupCallsBefore+1, usageRepo.groupCalls.Load())
	require.Equal(t, int64(22), usageRepo.trendFilters.ExcludeUserID)

	// Explicit exclude_user_id overrides exclude_self=true
	require.Equal(t, http.StatusOK, doReq("/admin/usage?exclude_self=true&exclude_user_id=77", "11").Code)
	require.Equal(t, int64(77), usageRepo.listFilters.ExcludeUserID)

	// Toggling off (exclude_self=false) clears exclusion and misses cache
	statsCallsBefore = usageRepo.statsCalls.Load()
	require.Equal(t, http.StatusOK, doReq("/admin/usage/stats?exclude_self=false&start_date=2026-11-19&end_date=2026-11-20", "22").Code)
	require.Equal(t, statsCallsBefore+1, usageRepo.statsCalls.Load())
	require.Equal(t, int64(0), usageRepo.statsFilters.ExcludeUserID)
}

func TestExcludeSelfValidationAndAuthGuard(t *testing.T) {
	usageRepo := &excludeUserUsageCapture{}
	opsRepo := &excludeUserOpsCaptureRepo{}
	router := newExcludeUserTestRouter(usageRepo, opsRepo, false)

	for _, path := range []string{
		"/admin/usage?exclude_self=true",
		"/admin/usage/stats?exclude_self=true",
		"/admin/dashboard/trend?exclude_self=true",
		"/admin/dashboard/models?exclude_self=true",
		"/admin/dashboard/groups?exclude_self=true",
		"/admin/dashboard/snapshot-v2?exclude_self=true",
		"/admin/dashboard/user-breakdown?exclude_self=true",
		"/admin/ops/errors?exclude_self=true",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusUnauthorized, rec.Code, path)
	}

	for _, path := range []string{
		"/admin/usage?exclude_self=maybe",
		"/admin/usage?exclude_user_id=-1",
		"/admin/usage?exclude_user_id=abc",
		"/admin/ops/errors?exclude_user_id=0",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code, path)
	}
}

func TestSnapshotV2RejectsExcludeWithUnsupportedComponents(t *testing.T) {
	usageRepo := &excludeUserUsageCapture{}
	router := newExcludeUserTestRouter(usageRepo, &excludeUserOpsCaptureRepo{}, true)
	do := func(query string) int {
		req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/snapshot-v2?"+query, nil)
		req.Header.Set("X-Test-Admin-ID", "11")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}
	// include_stats 默认为 true，未显式关闭时必须拒绝
	require.Equal(t, http.StatusBadRequest, do("exclude_self=true&include_trend=true"))
	require.Equal(t, http.StatusBadRequest, do("exclude_self=true&include_stats=false&include_users_trend=true"))
	// 使用记录页的支持组合必须通过
	require.Equal(t, http.StatusOK, do("exclude_self=true&include_stats=false&include_users_trend=false&include_trend=true&include_model_stats=true&include_group_stats=true&start_date=2026-11-19&end_date=2026-11-20"))
}

func TestExcludeSelfAppliesToRequestErrors(t *testing.T) {
	usageRepo := &excludeUserUsageCapture{}
	opsRepo := &excludeUserOpsCaptureRepo{}
	gin.SetMode(gin.TestMode)
	router := newExcludeUserTestRouter(usageRepo, opsRepo, true)
	router.GET("/admin/ops/request-errors", NewOpsHandler(service.NewOpsService(opsRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)).ListRequestErrors)
	req := httptest.NewRequest(http.MethodGet, "/admin/ops/request-errors?exclude_self=true", nil)
	req.Header.Set("X-Test-Admin-ID", "11")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, opsRepo.lastFilter)
	require.NotNil(t, opsRepo.lastFilter.ExcludeUserID)
	require.Equal(t, int64(11), *opsRepo.lastFilter.ExcludeUserID)
}
