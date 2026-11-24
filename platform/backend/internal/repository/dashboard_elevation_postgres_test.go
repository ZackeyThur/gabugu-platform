//go:build dashboard_pg_integration

package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func openDashboardPGTestDB(t *testing.T) (*sql.DB, *dbent.Client) {
	t.Helper()
	dsn := os.Getenv("DASHBOARD_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set DASHBOARD_TEST_POSTGRES_DSN to run real PostgreSQL dashboard integration tests")
	}
	require.NoError(t, timezone.Init("Asia/Shanghai"))

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.PingContext(context.Background()))

	entClient := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = entClient.Close() })
	return db, entClient
}

func TestDashboardElevationPostgresMigrationsStatsPlatformFilterAndPosters(t *testing.T) {
	ctx := context.Background()
	db, entClient := openDashboardPGTestDB(t)

	// 1. Apply full migration chain twice to verify Migration 254 & idempotency.
	require.NoError(t, ApplyMigrations(ctx, db))
	require.NoError(t, ApplyMigrations(ctx, db))

	var migrationCount int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE filename = '254_add_announcement_dashboard_poster.sql'`).Scan(&migrationCount)
	require.NoError(t, err)
	require.Equal(t, 1, migrationCount)

	// Clean any leftover rows from previous runs on the same test DB.
	_, err = db.ExecContext(ctx, `TRUNCATE usage_logs, api_keys, accounts, groups, announcements, users RESTART IDENTITY CASCADE`)
	require.NoError(t, err)

	// 2. Announcement repository round-trip for show_on_dashboard and dashboard_link.
	annRepo := NewAnnouncementRepository(entClient)
	annSvc := service.NewAnnouncementService(annRepo, nil, nil, nil)
	created, err := annSvc.Create(ctx, &service.CreateAnnouncementInput{
		Title:           "Spring Poster",
		Content:         "Promo body\n\n<!-- gabugu-announcement-images:v1 -->\n<img src=\"/brand/poster.webp\" alt=\"Poster\">\n<!-- /gabugu-announcement-images -->",
		Status:          service.AnnouncementStatusActive,
		NotifyMode:      service.AnnouncementNotifyModeSilent,
		ShowOnDashboard: true,
		DashboardLink:   "/purchase?amount=100",
	})
	require.NoError(t, err)
	require.True(t, created.ShowOnDashboard)
	require.Equal(t, "/purchase?amount=100", created.DashboardLink)

	fetched, err := annRepo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.True(t, fetched.ShowOnDashboard)
	require.Equal(t, "/purchase?amount=100", fetched.DashboardLink)

	// Toggle off & clear link via Update.
	showFalse := false
	emptyLink := ""
	updated, err := annSvc.Update(ctx, created.ID, &service.UpdateAnnouncementInput{
		ShowOnDashboard: &showFalse,
		DashboardLink:   &emptyLink,
	})
	require.NoError(t, err)
	require.False(t, updated.ShowOnDashboard)
	require.Equal(t, "", updated.DashboardLink)

	fetched2, err := annRepo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.False(t, fetched2.ShowOnDashboard)
	require.Equal(t, "", fetched2.DashboardLink)

	// 3. Seed users, groups, accounts, api_keys, and mixed usage_logs.
	var emptyUserID, activeUserID int64
	err = db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, role, status, balance, concurrency)
		VALUES ('empty_dash@example.com', 'hash', 'user', 'active', 50, 5)
		RETURNING id
	`).Scan(&emptyUserID)
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, role, status, balance, concurrency)
		VALUES ('active_dash@example.com', 'hash', 'user', 'active', 100, 5)
		RETURNING id
	`).Scan(&activeUserID)
	require.NoError(t, err)

	usageRepo := &usageLogRepository{client: entClient, sql: db}

	// 3a. Zero-usage user: verify NO NULL scan errors on empty aggregates.
	emptyStats, err := usageRepo.GetUserDashboardStatsInTimezone(ctx, emptyUserID, "Asia/Shanghai")
	require.NoError(t, err)
	require.Equal(t, int64(0), emptyStats.TotalRequests)
	require.Equal(t, int64(0), emptyStats.TodayRequests)
	require.Equal(t, 0.0, emptyStats.TodayAverageDurationMs)
	require.Equal(t, 0.0, emptyStats.Recent7dBalanceActualCost)
	require.Equal(t, 0.0, emptyStats.YesterdaySameTimeActualCost)
	require.Equal(t, int64(0), emptyStats.Recent5mRequests)

	// Seed groups: concrete anthropic, composite, empty-platform (if allowed)
	var grpAnthropicID, grpCompositeID int64
	err = db.QueryRowContext(ctx, `
		INSERT INTO groups (name, platform, status)
		VALUES ('anthropic-pro', 'anthropic', 'active')
		RETURNING id
	`).Scan(&grpAnthropicID)
	require.NoError(t, err)
	err = db.QueryRowContext(ctx, `
		INSERT INTO groups (name, platform, status)
		VALUES ('composite-mix', 'composite', 'active')
		RETURNING id
	`).Scan(&grpCompositeID)
	require.NoError(t, err)

	// Seed accounts: openai & gemini
	var accOpenAIID, accGeminiID int64
	err = db.QueryRowContext(ctx, `
		INSERT INTO accounts (name, platform, type, credentials, status)
		VALUES ('acc-openai', 'openai', 'apikey', '{}'::jsonb, 'active')
		RETURNING id
	`).Scan(&accOpenAIID)
	require.NoError(t, err)
	err = db.QueryRowContext(ctx, `
		INSERT INTO accounts (name, platform, type, credentials, status)
		VALUES ('acc-gemini', 'gemini', 'apikey', '{}'::jsonb, 'active')
		RETURNING id
	`).Scan(&accGeminiID)
	require.NoError(t, err)

	// Seed API keys: one active, one soft-deleted.
	var keyActiveID, keyDeletedID int64
	err = db.QueryRowContext(ctx, `
		INSERT INTO api_keys (user_id, key, name, status)
		VALUES ($1, 'sk-dash-active-1', 'active-cli', 'active')
		RETURNING id
	`, activeUserID).Scan(&keyActiveID)
	require.NoError(t, err)
	err = db.QueryRowContext(ctx, `
		INSERT INTO api_keys (user_id, key, name, status, deleted_at)
		VALUES ($1, 'sk-dash-deleted-2', 'retired-cli', 'inactive', NOW())
		RETURNING id
	`, activeUserID).Scan(&keyDeletedID)
	require.NoError(t, err)

	now := time.Now()
	todayStart, tomorrowStart := userDashboardDayRange(now, "Asia/Shanghai")
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	yesterdaySameTime := userDashboardYesterdaySameTime(now, todayStart, yesterdayStart)

	insertLog := func(createdAt time.Time, apiKeyID int64, accountID int64, groupID *int64, model string, inTok, outTok, cCreate, cRead int, totalCost, actualCost float64, billingType int16, durationMs *int) {
		_, execErr := db.ExecContext(ctx, `
			INSERT INTO usage_logs (
				user_id, api_key_id, account_id, group_id, model,
				input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
				input_cost, output_cost, cache_creation_cost, cache_read_cost,
				total_cost, actual_cost, rate_multiplier, billing_type, duration_ms, created_at
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9,
				0, 0, 0, 0,
				$10, $11, 1.0, $12, $13, $14
			)
		`, activeUserID, apiKeyID, accountID, groupID, model,
			inTok, outTok, cCreate, cRead,
			totalCost, actualCost, billingType, durationMs, createdAt)
		require.NoError(t, execErr)
	}

	dur400 := 400
	dur600 := 600
	// Row 1: Today (2m ago), billing_type=0 (balance), anthropic group + openai account => effective platform MUST be anthropic
	insertLog(now.Add(-2*time.Minute), keyActiveID, accOpenAIID, &grpAnthropicID, "claude-sonnet-4-6", 1000, 500, 100, 400, 2.0, 1.5, 0, &dur400)
	// Row 2: Today (1m ago), billing_type=1 (subscription), composite group + openai account => effective platform MUST be openai, duration NULL
	insertLog(now.Add(-1*time.Minute), keyDeletedID, accOpenAIID, &grpCompositeID, "gpt-5.4", 800, 200, 0, 0, 3.0, 2.5, 1, nil)
	// Row 3: Today (start + 1s), billing_type=0 (balance), NULL group + gemini account => effective platform MUST be gemini
	insertLog(todayStart.Add(1*time.Second), keyActiveID, accGeminiID, nil, "gemini-3.8-pro", 500, 500, 0, 0, 1.0, 0.5, 0, &dur600)

	// Row 4: Yesterday BEFORE yesterdaySameTime (if yesterdaySameTime > yesterdayStart+2s, use yesterdayStart+1s)
	yEarly := yesterdayStart.Add(1 * time.Second)
	expectYEarlyInSameTime := yEarly.Before(yesterdaySameTime)
	insertLog(yEarly, keyActiveID, accOpenAIID, &grpAnthropicID, "claude-sonnet-4-6", 100, 100, 0, 0, 5.0, 3.0, 0, &dur400)

	// Row 5: Yesterday LATE (23:59:00), billing_type=1 (subscription) -> in yesterday total & recent 7d subscription, NOT in same-time unless now is after 23:59
	yLate := todayStart.Add(-1 * time.Minute)
	expectYLateInSameTime := yLate.Before(yesterdaySameTime)
	insertLog(yLate, keyActiveID, accOpenAIID, &grpCompositeID, "gpt-5.4", 200, 200, 0, 0, 10.0, 8.0, 1, &dur400)

	// Row 6: 3 days ago, billing_type=0 (balance) -> in recent 7d balance ONLY
	insertLog(todayStart.AddDate(0, 0, -3).Add(2*time.Hour), keyActiveID, accOpenAIID, &grpAnthropicID, "claude-sonnet-4-6", 100, 50, 0, 0, 6.0, 4.0, 0, &dur400)

	// Row 7: 8 days ago, billing_type=0 (balance) -> OUTSIDE recent 7d window
	insertLog(todayStart.AddDate(0, 0, -8).Add(2*time.Hour), keyActiveID, accOpenAIID, &grpAnthropicID, "claude-sonnet-4-6", 100, 50, 0, 0, 50.0, 40.0, 0, &dur400)

	// 4. Verify GetUserDashboardStatsInTimezone against real PG16!
	dashStats, err := usageRepo.GetUserDashboardStatsInTimezone(ctx, activeUserID, "Asia/Shanghai")
	require.NoError(t, err)

	require.Equal(t, int64(2), dashStats.TotalAPIKeys+1) // only non-deleted key counted -> TotalAPIKeys == 1
	require.Equal(t, int64(1), dashStats.TotalAPIKeys)
	require.Equal(t, int64(1), dashStats.ActiveAPIKeys)
	require.Equal(t, int64(3), dashStats.TodayRequests)
	require.InDelta(t, 4.5, dashStats.TodayActualCost, 1e-6)
	require.InDelta(t, 2.0, dashStats.TodayBalanceActualCost, 1e-6)      // 1.5 + 0.5
	require.InDelta(t, 2.5, dashStats.TodaySubscriptionActualCost, 1e-6) // 2.5
	require.InDelta(t, 500.0, dashStats.TodayAverageDurationMs, 1e-6)    // AVG(400, NULL, 600) = 500!

	require.Equal(t, int64(2), dashStats.YesterdayRequests)
	require.InDelta(t, 11.0, dashStats.YesterdayActualCost, 1e-6) // 3.0 + 8.0
	if expectYEarlyInSameTime && !expectYLateInSameTime {
		require.Equal(t, int64(1), dashStats.YesterdaySameTimeRequests)
		require.InDelta(t, 3.0, dashStats.YesterdaySameTimeActualCost, 1e-6)
	}

	// Recent 7 complete days [today-7d, today): balance = 3.0 (yesterday) + 4.0 (3d ago) = 7.0; subscription = 8.0 (yesterday)
	require.InDelta(t, 7.0, dashStats.Recent7dBalanceActualCost, 1e-6)
	require.InDelta(t, 8.0, dashStats.Recent7dSubscriptionActualCost, 1e-6)

	// Recent 5m performance stats: 2 rows within 5m => rpm=0, rpm_exact=0.4, recent_5m_requests>=2
	require.GreaterOrEqual(t, dashStats.Recent5mRequests, int64(2))
	require.InDelta(t, float64(dashStats.Recent5mRequests)/5.0, dashStats.RpmExact, 1e-9)

	// 5. Verify platform filter semantics across ALL 6 repository methods!
	rangeStart := todayStart
	rangeEnd := tomorrowStart

	// 5a. ListWithFilters
	openaiLogs, _, err := usageRepo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, usagestats.UsageLogFilters{
		UserID:    activeUserID,
		Platform:  "openai",
		StartTime: &rangeStart,
		EndTime:   &rangeEnd,
	})
	require.NoError(t, err)
	require.Len(t, openaiLogs, 1)
	require.Equal(t, "gpt-5.4", openaiLogs[0].Model)

	anthropicLogs, _, err := usageRepo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, usagestats.UsageLogFilters{
		UserID:    activeUserID,
		Platform:  "anthropic",
		StartTime: &rangeStart,
		EndTime:   &rangeEnd,
	})
	require.NoError(t, err)
	require.Len(t, anthropicLogs, 1)
	require.Equal(t, "claude-sonnet-4-6", anthropicLogs[0].Model)

	geminiLogs, _, err := usageRepo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 20}, usagestats.UsageLogFilters{
		UserID:    activeUserID,
		Platform:  "gemini",
		StartTime: &rangeStart,
		EndTime:   &rangeEnd,
	})
	require.NoError(t, err)
	require.Len(t, geminiLogs, 1)
	require.Equal(t, "gemini-3.8-pro", geminiLogs[0].Model)

	// 5b. GetStatsWithFilters
	openaiStats, err := usageRepo.GetStatsWithFilters(ctx, usagestats.UsageLogFilters{
		UserID:    activeUserID,
		Platform:  "openai",
		StartTime: &rangeStart,
		EndTime:   &rangeEnd,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), openaiStats.TotalRequests)
	require.InDelta(t, 2.5, openaiStats.TotalActualCost, 1e-6)

	// 5c. GetUsageTrendWithUsageFilters (hourly & daily)
	hourlyTrend, err := usageRepo.GetUsageTrendWithUsageFilters(ctx, rangeStart, rangeEnd, "hour", usagestats.UsageLogFilters{
		UserID:   activeUserID,
		Platform: "anthropic",
	})
	require.NoError(t, err)
	require.Len(t, hourlyTrend, 1)
	require.Equal(t, int64(1), hourlyTrend[0].Requests)
	require.Regexp(t, `^\d{4}-\d{2}-\d{2} \d{2}:00$`, hourlyTrend[0].Date)

	// 5d. GetModelStatsWithUsageFiltersBySource
	modelStats, err := usageRepo.GetModelStatsWithUsageFiltersBySource(ctx, rangeStart, rangeEnd, usagestats.UsageLogFilters{
		UserID:   activeUserID,
		Platform: "gemini",
	}, usagestats.ModelSourceRequested)
	require.NoError(t, err)
	require.Len(t, modelStats, 1)
	require.Equal(t, "gemini-3.8-pro", modelStats[0].Model)

	// 5e. GetGroupStatsWithUsageFilters
	groupStats, err := usageRepo.GetGroupStatsWithUsageFilters(ctx, rangeStart, rangeEnd, usagestats.UsageLogFilters{
		UserID:   activeUserID,
		Platform: "openai",
	})
	require.NoError(t, err)
	require.Len(t, groupStats, 1)
	require.Equal(t, grpCompositeID, groupStats[0].GroupID)
	require.Equal(t, "composite-mix", groupStats[0].GroupName)

	// 5f. GetAPIKeyStatsWithUsageFilters (active + soft-deleted keys)
	keyStats, err := usageRepo.GetAPIKeyStatsWithUsageFilters(ctx, rangeStart, rangeEnd, usagestats.UsageLogFilters{
		UserID: activeUserID,
	})
	require.NoError(t, err)
	require.Len(t, keyStats, 2)
	// Ordered by actual_cost DESC: retired-cli ($2.5) first, active-cli ($2.0) second.
	require.Equal(t, keyDeletedID, keyStats[0].APIKeyID)
	require.Equal(t, "retired-cli", keyStats[0].KeyName)
	require.True(t, keyStats[0].Deleted)
	require.InDelta(t, 2.5, keyStats[0].ActualCost, 1e-6)

	require.Equal(t, keyActiveID, keyStats[1].APIKeyID)
	require.Equal(t, "active-cli", keyStats[1].KeyName)
	require.False(t, keyStats[1].Deleted)
	require.InDelta(t, 2.0, keyStats[1].ActualCost, 1e-6)

	// 6. Concurrent fan-out (mirrors DashboardSnapshotV2 errgroup) against real PG16 connection pool.
	usageSvc := service.NewUsageService(usageRepo, nil, nil, nil)
	snapFilters := usagestats.UsageLogFilters{UserID: activeUserID}
	var (
		cTrend   []usagestats.TrendDataPoint
		cModels  []usagestats.ModelStat
		cGroups  []usagestats.GroupStat
		cAPIKeys []usagestats.APIKeyStat
	)
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		var e error
		cTrend, e = usageSvc.GetUsageTrendWithFilters(egCtx, rangeStart, rangeEnd, "hour", snapFilters)
		return e
	})
	eg.Go(func() error {
		var e error
		cModels, e = usageSvc.GetModelStatsWithFiltersBySource(egCtx, rangeStart, rangeEnd, snapFilters, usagestats.ModelSourceRequested)
		return e
	})
	eg.Go(func() error {
		var e error
		cGroups, e = usageSvc.GetGroupStatsWithFilters(egCtx, rangeStart, rangeEnd, snapFilters)
		return e
	})
	eg.Go(func() error {
		var e error
		cAPIKeys, e = usageSvc.GetAPIKeyStatsWithFilters(egCtx, rangeStart, rangeEnd, snapFilters)
		return e
	})
	require.NoError(t, eg.Wait())
	require.NotEmpty(t, cTrend)
	require.Len(t, cModels, 3)
	require.Len(t, cGroups, 3) // grpAnthropicID, grpCompositeID, unassigned (0)
	require.Len(t, cAPIKeys, 2)
}
