//go:build unit

package repository

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"regexp"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

// This test only uses a disposable, separately initialized loopback instance.
func TestV0219UpgradeOnIsolatedPostgres(t *testing.T) {
	if os.Getenv("GABUGU_V0219_ISOLATED_TEST") != "1" {
		t.Skip("requires the dedicated local upgrade verification instance")
	}
	databaseName := os.Getenv("GABUGU_V0219_MIGRATION_DB")
	if databaseName == "" {
		databaseName = "upgrade_verify"
	}
	require.Regexp(t, regexp.MustCompile(`^upgrade_verify(?:_[0-9]+)?$`), databaseName)
	db, err := sql.Open("postgres", "host=127.0.0.1 port=15432 user=candidate_test dbname="+databaseName+" sslmode=disable")
	require.NoError(t, err)
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	var database, user, server string
	var port int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT current_database(), current_user, inet_server_addr()::text, inet_server_port()").Scan(&database, &user, &server, &port))
	require.Equal(t, databaseName, database)
	require.Equal(t, "candidate_test", user)
	require.Equal(t, "127.0.0.1/32", server)
	require.Equal(t, 15432, port)
	var existing int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM pg_tables WHERE schemaname = 'public'").Scan(&existing))
	require.Zero(t, existing, "verification database must be empty")

	added := map[string]bool{
		"246_enable_gemini_image_generation_groups.sql": true,
		"247_scoped_admin_api_keys.sql":                 true,
		"248_group_image_count_billing.sql":             true,
		"249_group_image_count_billing_auth_cache.sql":  true,
	}
	baseline := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS, ".")
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() || added[entry.Name()] {
			continue
		}
		data, err := fs.ReadFile(migrations.FS, entry.Name())
		require.NoError(t, err)
		baseline[entry.Name()] = &fstest.MapFile{Data: data}
	}
	require.NoError(t, applyMigrationsFS(ctx, db, baseline))
	_, err = db.ExecContext(ctx, `INSERT INTO groups (name, platform, allow_image_generation) VALUES
		('verify-gemini-off', 'gemini', false), ('verify-antigravity-off', 'antigravity', false),
		('verify-gemini-on', 'gemini', true), ('verify-openai-off', 'openai', false),
		('verify-composite-off', 'composite', false)`)
	require.NoError(t, err)
	require.ErrorContains(t, verifyStandbyMigrationsFS(ctx, db, migrations.FS), "246_enable_gemini_image_generation_groups.sql is not applied")
	require.NoError(t, applyMigrationsFS(ctx, db, migrations.FS))
	require.NoError(t, verifyStandbyMigrationsFS(ctx, db, migrations.FS))
	require.NoError(t, verifyStandbyMigrationsFS(ctx, db, baseline))

	for name, expected := range map[string]bool{
		"verify-gemini-off": false, "verify-antigravity-off": false, "verify-gemini-on": true,
		"verify-openai-off": false, "verify-composite-off": false,
	} {
		var enabled, countBilling bool
		require.NoError(t, db.QueryRowContext(ctx, "SELECT allow_image_generation, image_count_billing FROM groups WHERE name = $1", name).Scan(&enabled, &countBilling))
		require.Equal(t, expected, enabled, name)
		require.False(t, countBilling, name)
	}
	var migrationCount int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE filename IN
		('246_enable_gemini_image_generation_groups.sql', '247_scoped_admin_api_keys.sql', '248_group_image_count_billing.sql', '249_group_image_count_billing_auth_cache.sql')`).Scan(&migrationCount))
	require.Equal(t, 4, migrationCount)

	// Simulate a pre-upgrade writer changing an existing field after migration.
	_, err = db.ExecContext(ctx, "UPDATE groups SET allow_image_generation = false WHERE name = 'verify-gemini-off'")
	require.NoError(t, err)
	require.NoError(t, applyMigrationsFS(ctx, db, migrations.FS))
	var enabled bool
	require.NoError(t, db.QueryRowContext(ctx, "SELECT allow_image_generation FROM groups WHERE name = 'verify-gemini-off'").Scan(&enabled))
	require.False(t, enabled, "an applied migration must not override a later setting")
}

func TestV0219QuotaOnIsolatedPostgres(t *testing.T) {
	if os.Getenv("GABUGU_V0219_ISOLATED_TEST") != "1" {
		t.Skip("requires the dedicated local upgrade verification instance")
	}
	db, err := sql.Open("postgres", "host=127.0.0.1 port=15434 user=candidate_test dbname=account_order_test sslmode=disable")
	require.NoError(t, err)
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	var accountID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO accounts (name, platform, type, credentials, extra)
		VALUES ('v0219-quota-test', 'gemini', 'service_account', '{}',
		'{"quota_limit":100,"quota_daily_limit":10,"quota_weekly_limit":20}') RETURNING id`).Scan(&accountID)
	require.NoError(t, err)

	// Stale caller hints must fall back to the full SQL and update both periods.
	state, err := incrementUsageBillingAccountQuota(ctx, tx, accountID, 4, false, false)
	require.NoError(t, err)
	require.InDelta(t, 4, state.TotalUsed, 1e-9)
	require.InDelta(t, 4, state.DailyUsed, 1e-9)
	require.InDelta(t, 4, state.WeeklyUsed, 1e-9)
	state, err = incrementUsageBillingAccountQuota(ctx, tx, accountID, 7, false, false)
	require.NoError(t, err)
	require.InDelta(t, 11, state.TotalUsed, 1e-9)
	require.InDelta(t, 11, state.DailyUsed, 1e-9)
	require.InDelta(t, 11, state.WeeklyUsed, 1e-9)
	var events int
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT count(*) FROM scheduler_outbox WHERE account_id = $1", accountID).Scan(&events))
	require.Equal(t, 1, events, "crossing the daily limit must enqueue an account refresh")
	_, err = incrementUsageBillingAccountQuota(ctx, tx, accountID, 1, false, false)
	require.NoError(t, err)
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT count(*) FROM scheduler_outbox WHERE account_id = $1", accountID).Scan(&events))
	require.Equal(t, 1, events, "remaining above the limit must not enqueue another event")
}

// The database contains only a captured production schema and migration ledger,
// with no account, credential, user, payment or usage rows.
func TestV0219RestoredSchemaMigrations(t *testing.T) {
	if os.Getenv("GABUGU_V0219_ISOLATED_TEST") != "1" {
		t.Skip("requires the dedicated local upgrade verification instance")
	}
	db, err := sql.Open("postgres", "host=127.0.0.1 port=15432 user=candidate_test dbname=wallet_upgrade_verify sslmode=disable")
	require.NoError(t, err)
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var users, accounts, keys, orders, usage int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM users), (SELECT count(*) FROM accounts),
		(SELECT count(*) FROM api_keys), (SELECT count(*) FROM payment_orders), (SELECT count(*) FROM usage_logs)`).Scan(&users, &accounts, &keys, &orders, &usage))
	require.Equal(t, []int{0, 0, 0, 0, 0}, []int{users, accounts, keys, orders, usage})
	require.NoError(t, applyMigrationsFS(ctx, db, migrations.FS))
	require.NoError(t, applyMigrationsFS(ctx, db, migrations.FS))
	require.NoError(t, verifyStandbyMigrationsFS(ctx, db, migrations.FS))
}
