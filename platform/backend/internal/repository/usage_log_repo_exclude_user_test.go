package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

func TestUsageLogRepositoryListWithFiltersExcludeUserID(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	filters := usagestats.UsageLogFilters{
		ExcludeUserID: 99,
		ExactTotal:    true,
	}

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM usage_logs WHERE \(user_id IS NULL OR user_id <> \$1\)`).
		WithArgs(int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectQuery(`SELECT .* FROM usage_logs WHERE \(user_id IS NULL OR user_id <> \$1\) ORDER BY id DESC LIMIT \$2 OFFSET \$3`).
		WithArgs(int64(99), 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	logs, page, err := repo.ListWithFilters(context.Background(), pagination.PaginationParams{Page: 1, PageSize: 20}, filters)
	require.NoError(t, err)
	require.Empty(t, logs)
	require.NotNil(t, page)
	require.Equal(t, int64(0), page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetStatsWithFiltersExcludeUserID(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	filters := usagestats.UsageLogFilters{ExcludeUserID: 99}

	mock.ExpectQuery(`(?s)FROM usage_logs\s+WHERE \(user_id IS NULL OR user_id <> \$1\).*GROUP BY GROUPING SETS`).
		WithArgs(int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{
			"inbound_grouped", "upstream_grouped", "inbound_endpoint", "upstream_endpoint",
			"requests", "input_tokens", "output_tokens", "cache_creation_tokens", "cache_read_tokens",
			"cost", "actual_cost", "account_cost", "avg_duration_ms",
		}).AddRow(1, 1, nil, nil, int64(3), int64(10), int64(20), int64(0), int64(0), 1.0, 0.8, 0.7, 50.0))

	stats, err := repo.GetStatsWithFilters(context.Background(), filters)
	require.NoError(t, err)
	require.Equal(t, int64(3), stats.TotalRequests)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryTrendModelGroupBreakdownExcludeUserID(t *testing.T) {
	start := time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	t.Run("trend bypasses preaggregates and filters raw usage_logs", func(t *testing.T) {
		db, mock := newSQLMock(t)
		repo := &usageLogRepository{sql: db}
		mock.ExpectQuery(`(?s)FROM usage_logs\s+WHERE created_at >= \$1 AND created_at < \$2 AND \(user_id IS NULL OR user_id <> \$3\)`).
			WithArgs(start, end, int64(99)).
			WillReturnRows(sqlmock.NewRows([]string{"date", "requests", "input_tokens", "output_tokens", "cache_creation_tokens", "cache_read_tokens", "total_tokens", "cost", "actual_cost"}))

		_, err := repo.GetUsageTrendWithUsageFilters(context.Background(), start, end, "day", usagestats.UsageLogFilters{ExcludeUserID: 99})
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("model stats excludes user_id", func(t *testing.T) {
		db, mock := newSQLMock(t)
		repo := &usageLogRepository{sql: db}
		mock.ExpectQuery(`(?s)FROM usage_logs\s+WHERE created_at >= \$1 AND created_at < \$2 AND \(user_id IS NULL OR user_id <> \$3\)`).
			WithArgs(start, end, int64(99)).
			WillReturnRows(sqlmock.NewRows([]string{"model", "requests", "input_tokens", "output_tokens", "cache_creation_tokens", "cache_read_tokens", "total_tokens", "cost", "actual_cost", "account_cost"}))

		_, err := repo.GetModelStatsWithUsageFiltersBySource(context.Background(), start, end, usagestats.UsageLogFilters{ExcludeUserID: 99}, usagestats.ModelSourceRequested)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("group stats excludes ul.user_id", func(t *testing.T) {
		db, mock := newSQLMock(t)
		repo := &usageLogRepository{sql: db}
		mock.ExpectQuery(`(?s)FROM usage_logs ul\s+LEFT JOIN groups g ON g\.id = ul\.group_id\s+WHERE ul\.created_at >= \$1 AND ul\.created_at < \$2 AND \(ul\.user_id IS NULL OR ul\.user_id <> \$3\)`).
			WithArgs(start, end, int64(99)).
			WillReturnRows(sqlmock.NewRows([]string{"group_id", "group_name", "requests", "total_tokens", "cost", "actual_cost", "account_cost"}))

		_, err := repo.GetGroupStatsWithUsageFilters(context.Background(), start, end, usagestats.UsageLogFilters{ExcludeUserID: 99})
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user breakdown excludes ul.user_id before limit", func(t *testing.T) {
		db, mock := newSQLMock(t)
		repo := &usageLogRepository{sql: db}
		mock.ExpectQuery(regexp.QuoteMeta("WHERE ul.created_at >= $1 AND ul.created_at < $2 AND (ul.user_id IS NULL OR ul.user_id <> $3)")+`.*GROUP BY ul\.user_id, u\.email ORDER BY actual_cost DESC LIMIT 50`).
			WithArgs(start, end, int64(99)).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "requests", "input_tokens", "output_tokens", "cache_tokens", "total_tokens", "cost", "actual_cost", "account_cost"}))

		_, err := repo.GetUserBreakdownStats(context.Background(), start, end, usagestats.UserBreakdownDimension{ExcludeUserID: 99}, 50)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
