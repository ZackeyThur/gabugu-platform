package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

func TestUserUsageTrendBucketsInRangeTimezone(t *testing.T) {
	for _, zone := range []string{"Asia/Shanghai", "UTC", "America/New_York"} {
		t.Run(zone, func(t *testing.T) {
			loc, err := time.LoadLocation(zone)
			require.NoError(t, err)
			start := time.Date(2026, 11, 9, 0, 0, 0, 0, loc)
			end := start.AddDate(0, 0, 1)
			db, mock := newSQLMock(t)
			repo := &usageLogRepository{sql: db}
			// AT TIME ZONE makes this independent of the database session zone.
			mock.ExpectQuery(`(?s)`+regexp.QuoteMeta(`TO_CHAR(created_at AT TIME ZONE $3, 'YYYY-MM-DD')`)+`.*WHERE created_at >= \$1 AND created_at < \$2\s+AND user_id = \$4`).
				WithArgs(start, end, zone, int64(42)).
				WillReturnRows(sqlmock.NewRows([]string{"date", "requests", "input_tokens", "output_tokens", "cache_creation_tokens", "cache_read_tokens", "total_tokens", "cost", "actual_cost"}).
					AddRow("2026-11-09", 1, 10, 2, 0, 0, 12, 0.2, 0.1))
			points, err := repo.GetUsageTrendWithUsageFilters(context.Background(), start, end, "day", usagestats.UsageLogFilters{UserID: 42})
			require.NoError(t, err)
			require.Len(t, points, 1)
			require.Equal(t, "2026-11-09", points[0].Date)
			require.Equal(t, 0.1, points[0].ActualCost)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserUsageTrendFixedOffsetDoesNotSendUnknownZoneName(t *testing.T) {
	loc := time.FixedZone("custom-test-zone", 8*60*60)
	start := time.Date(2026, 11, 9, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, 1)
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	mock.ExpectQuery(regexp.QuoteMeta(`TO_CHAR(created_at AT TIME ZONE make_interval(secs => $3), 'YYYY-MM-DD')`)).
		WithArgs(start, end, 8*60*60, int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"date", "requests", "input_tokens", "output_tokens", "cache_creation_tokens", "cache_read_tokens", "total_tokens", "cost", "actual_cost"}))
	_, err := repo.GetUsageTrendWithUsageFilters(context.Background(), start, end, "day", usagestats.UsageLogFilters{UserID: 42})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageTrendLegacyCallKeepsSessionTimezoneContract(t *testing.T) {
	start := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	mock.ExpectQuery(regexp.QuoteMeta(`TO_CHAR(created_at, 'YYYY-MM-DD')`)).
		WithArgs(start, end, int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"date", "requests", "input_tokens", "output_tokens", "cache_creation_tokens", "cache_read_tokens", "total_tokens", "cost", "actual_cost"}))
	_, err := repo.GetUsageTrendWithFilters(context.Background(), start, end, "day", 42, 0, 0, 0, "", nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
