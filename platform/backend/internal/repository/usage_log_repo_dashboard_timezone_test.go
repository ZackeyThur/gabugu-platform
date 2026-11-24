//go:build unit

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestUserDashboardDayRange(t *testing.T) {
	useGroupUsageRepositoryTestTimezone(t, "Asia/Shanghai")
	for _, tc := range []struct {
		name, now, timezone, start, end string
	}{
		{"before Shanghai midnight", "2026-11-08T15:59:59Z", "Asia/Shanghai", "2026-11-07T16:00:00Z", "2026-11-08T16:00:00Z"},
		{"at Shanghai midnight", "2026-11-08T16:00:00Z", "Asia/Shanghai", "2026-11-08T16:00:00Z", "2026-11-09T16:00:00Z"},
		{"UTC same instant", "2026-11-08T16:00:00Z", "UTC", "2026-11-08T00:00:00Z", "2026-11-09T00:00:00Z"},
		{"missing retains server timezone", "2026-11-08T16:00:00Z", "", "2026-11-08T16:00:00Z", "2026-11-09T16:00:00Z"},
		{"invalid retains server timezone", "2026-11-08T16:00:00Z", "Invalid/Timezone", "2026-11-08T16:00:00Z", "2026-11-09T16:00:00Z"},
		{"DST spring calendar day is 23 hours", "2026-03-08T12:00:00Z", "America/New_York", "2026-03-08T05:00:00Z", "2026-03-09T04:00:00Z"},
		{"DST autumn calendar day is 25 hours", "2026-11-01T12:00:00Z", "America/New_York", "2026-11-01T04:00:00Z", "2026-11-02T05:00:00Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, tc.now)
			require.NoError(t, err)
			start, end := userDashboardDayRange(now, tc.timezone)
			require.Equal(t, tc.start, start.UTC().Format(time.RFC3339))
			require.Equal(t, tc.end, end.UTC().Format(time.RFC3339))
		})
	}
}

func TestUserDashboardYesterdaySameTime(t *testing.T) {
	useGroupUsageRepositoryTestTimezone(t, "Asia/Shanghai")
	now, err := time.Parse(time.RFC3339, "2026-11-08T06:30:00Z") // 14:30 Asia/Shanghai
	require.NoError(t, err)
	today, _ := userDashboardDayRange(now, "Asia/Shanghai")
	yesterday := today.AddDate(0, 0, -1)
	cutoff := userDashboardYesterdaySameTime(now, today, yesterday)
	require.Equal(t, "2026-11-07T06:30:00Z", cutoff.UTC().Format(time.RFC3339))

	// DST autumn 25-hour day: clamp cutoff so it never crosses into today.
	dstNow, err := time.Parse(time.RFC3339, "2026-11-02T04:30:00Z") // 24h30m after 2026-11-01 00:00 EDT
	require.NoError(t, err)
	dstToday, _ := userDashboardDayRange(dstNow.Add(-1*time.Hour), "America/New_York")
	dstYesterday := dstToday.AddDate(0, 0, -1)
	dstCutoff := userDashboardYesterdaySameTime(dstToday.Add(24*time.Hour+30*time.Minute), dstToday, dstYesterday)
	require.True(t, !dstCutoff.After(dstToday))
}

func TestGetUserDashboardStatsTimezoneSharesBoundariesAndUserScope(t *testing.T) {
	useGroupUsageRepositoryTestTimezone(t, "Asia/Shanghai")
	for _, tz := range []string{"Asia/Shanghai", "UTC", "America/New_York", "", "Invalid/Timezone"} {
		t.Run(tz, func(t *testing.T) {
			db, mock := newSQLMock(t)
			repo := &usageLogRepository{sql: db}
			start, end := userDashboardDayRange(time.Now(), tz)
			yesterdayStart := start.AddDate(0, 0, -1)
			recent7dStart := start.AddDate(0, 0, -7)
			const userID int64 = 42
			mock.ExpectQuery(`SELECT COUNT\(\*\) FROM api_keys WHERE user_id = \$1 AND deleted_at IS NULL`).
				WithArgs(userID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
			mock.ExpectQuery(`SELECT COUNT\(\*\) FROM api_keys WHERE user_id = \$1 AND status = \$2 AND deleted_at IS NULL`).
				WithArgs(userID, "active").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(`(?s)COUNT\(\*\) as total_requests,.*SUM\(total_cost\).*SUM\(actual_cost\).*FROM usage_logs\s+WHERE user_id = \$1\s*$`).
				WithArgs(userID).WillReturnRows(sqlmock.NewRows([]string{"requests", "input", "output", "creation", "read", "cost", "actual", "duration"}).AddRow(3, 10, 20, 30, 40, 0.9, 0.45, 12))
			mock.ExpectQuery(`(?s)today_requests,.*today_average_duration_ms,.*today_balance_actual_cost,.*today_subscription_actual_cost,.*yesterday_requests,.*yesterday_same_time_actual_cost,.*recent_7d_balance_actual_cost,.*recent_7d_subscription_actual_cost.*WHERE user_id = \$1 AND created_at >= \$2 AND created_at < \$3`).
				WithArgs(userID, recent7dStart, end, start, yesterdayStart, sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{
					"today_requests", "today_input", "today_output", "today_creation", "today_read",
					"today_cost", "today_actual", "today_avg_duration_ms",
					"today_balance_actual", "today_sub_actual",
					"yesterday_requests", "yesterday_tokens", "yesterday_cost", "yesterday_actual",
					"yesterday_same_time_requests", "yesterday_same_time_actual",
					"recent_7d_balance_actual", "recent_7d_sub_actual",
				}).AddRow(
					1, 1, 2, 3, 4,
					0.3, 0.15, 420.5,
					0.10, 0.05,
					4, 40, 0.6, 0.30,
					2, 0.12,
					7.0, 14.0,
				))
			mock.ExpectQuery(`(?s)COUNT\(\*\) as request_count,.*WHERE created_at >= \$1 AND user_id = \$2`).
				WithArgs(sqlmock.AnyArg(), userID).WillReturnRows(sqlmock.NewRows([]string{"requests", "tokens"}).AddRow(3, 50))
			// All three platform "today" aggregates must share the exact same range.
			mock.ExpectQuery(`(?s)COUNT\(\*\) FILTER \(WHERE ul.created_at >= \$2 AND ul.created_at < \$3\) as today_requests,.*FILTER \(WHERE ul.created_at >= \$2 AND ul.created_at < \$3\).*as today_tokens,.*FILTER \(WHERE ul.created_at >= \$2 AND ul.created_at < \$3\).*as today_actual_cost.*WHERE ul.user_id = \$1\s+AND ul.actual_cost > 0`).
				WithArgs(userID, start, end).WillReturnRows(sqlmock.NewRows([]string{"platform", "requests", "tokens", "actual", "today_requests", "today_tokens", "today_actual"}).AddRow("openai", 3, 100, 0.45, 1, 10, 0.15))

			var stats *UserDashboardStats
			var err error
			if tz == "" {
				stats, err = repo.GetUserDashboardStats(context.Background(), userID)
			} else {
				stats, err = repo.GetUserDashboardStatsInTimezone(context.Background(), userID, tz)
			}
			require.NoError(t, err)
			require.Equal(t, 0.9, stats.TotalCost)
			require.Equal(t, 0.45, stats.TotalActualCost)
			require.Equal(t, 0.3, stats.TodayCost)
			require.Equal(t, 0.15, stats.TodayActualCost)
			require.Equal(t, 420.5, stats.TodayAverageDurationMs)
			require.Equal(t, 0.10, stats.TodayBalanceActualCost)
			require.Equal(t, 0.05, stats.TodaySubscriptionActualCost)
			require.Equal(t, int64(4), stats.YesterdayRequests)
			require.Equal(t, int64(40), stats.YesterdayTokens)
			require.Equal(t, 0.6, stats.YesterdayCost)
			require.Equal(t, 0.30, stats.YesterdayActualCost)
			require.Equal(t, int64(2), stats.YesterdaySameTimeRequests)
			require.Equal(t, 0.12, stats.YesterdaySameTimeActualCost)
			require.Equal(t, 7.0, stats.Recent7dBalanceActualCost)
			require.Equal(t, 14.0, stats.Recent7dSubscriptionActualCost)
			require.Equal(t, int64(0), stats.Rpm)
			require.InDelta(t, 0.6, stats.RpmExact, 1e-9)
			require.InDelta(t, 10.0, stats.TpmExact, 1e-9)
			require.Equal(t, int64(3), stats.Recent5mRequests)
			require.Len(t, stats.ByPlatform, 1)
			require.Equal(t, stats.TodayActualCost, stats.ByPlatform[0].TodayActualCost)
			require.Equal(t, stats.TotalActualCost, stats.ByPlatform[0].TotalActualCost)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
