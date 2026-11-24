package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestUserUsageTrendShanghaiDaysWithUTCPostgres(t *testing.T) {
	// Opt in only for a disposable local instance. A temporary table shadows
	// usage_logs and the transaction is rolled back; no application schema runs.
	dsn := os.Getenv("GABUGU_DASHBOARD_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("set GABUGU_DASHBOARD_TEST_PG_DSN for disposable local PostgreSQL")
	}
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	_, err = tx.ExecContext(ctx, `SET LOCAL TIME ZONE 'UTC'`)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `CREATE TEMP TABLE usage_logs (
		user_id BIGINT NOT NULL, model TEXT NOT NULL, requested_model TEXT,
		created_at TIMESTAMPTZ NOT NULL,
		input_tokens INTEGER DEFAULT 10, output_tokens INTEGER DEFAULT 0,
		cache_creation_tokens INTEGER DEFAULT 0, cache_read_tokens INTEGER DEFAULT 0,
		total_cost NUMERIC(20,10) DEFAULT 2, actual_cost NUMERIC(20,10) DEFAULT 1,
		account_stats_cost NUMERIC(20,10), account_rate_multiplier NUMERIC(10,4)
	) ON COMMIT DROP`)
	require.NoError(t, err)
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	start := time.Date(2026, 11, 9, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, 1)
	for _, instant := range []time.Time{start.Add(-time.Microsecond), start, start.Add(30 * time.Minute), end.Add(-time.Microsecond), end} {
		_, err = tx.ExecContext(ctx, `INSERT INTO usage_logs (user_id, model, created_at) VALUES (42, 'gpt-5.5', $1)`, instant)
		require.NoError(t, err)
	}
	// A different user inside the range must not contribute to either result.
	_, err = tx.ExecContext(ctx, `INSERT INTO usage_logs (user_id, model, created_at) VALUES (99, 'gpt-5.5', $1)`, start)
	require.NoError(t, err)
	var oldBuckets int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(DISTINCT TO_CHAR(created_at, 'YYYY-MM-DD'))
		FROM usage_logs WHERE user_id = 42 AND created_at >= $1 AND created_at < $2`, start, end).Scan(&oldBuckets)
	require.NoError(t, err)
	require.Equal(t, 2, oldBuckets, "the old session-timezone grouping splits this one Shanghai day")
	repo := &usageLogRepository{sql: tx}
	filters := usagestats.UsageLogFilters{UserID: 42, ModelFilterSource: usagestats.ModelSourceRequested}
	trend, err := repo.GetUsageTrendWithUsageFilters(ctx, start, end, "day", filters)
	require.NoError(t, err)
	require.Len(t, trend, 1)
	require.Equal(t, "2026-11-09", trend[0].Date)
	require.Equal(t, int64(3), trend[0].Requests)
	require.Equal(t, 3.0, trend[0].ActualCost)
	models, err := repo.GetModelStatsWithUsageFiltersBySource(ctx, start, end, filters, usagestats.ModelSourceRequested)
	require.NoError(t, err)
	require.Len(t, models, 1)
	require.Equal(t, trend[0].ActualCost, models[0].ActualCost)
	require.Equal(t, trend[0].Requests, models[0].Requests)
}
