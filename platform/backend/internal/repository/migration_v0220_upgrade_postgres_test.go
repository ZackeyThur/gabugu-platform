//go:build v0220migration

package repository

import (
	"context"
	"database/sql"
	"io/fs"
	"net/url"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestSub2APIV0220UpgradesV0219SchemaAndPreservesReadsWrites(t *testing.T) {
	dsn := os.Getenv("SUB2API_V0220_MIGRATION_TEST_DSN")
	require.NotEmpty(t, dsn, "set SUB2API_V0220_MIGRATION_TEST_DSN for the isolated local PostgreSQL database")
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", u.Hostname())
	require.Equal(t, "18620", u.Port())
	require.Equal(t, "/v0220_test", u.Path)
	require.Equal(t, "gabugu_test", u.User.Username())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	var publicTables int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pg_tables WHERE schemaname='public'`).Scan(&publicTables))
	require.Zero(t, publicTables, "the dedicated database must be empty before the one-time upgrade rehearsal")

	oldFS := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS, ".")
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "249_mcp_oauth.sql" {
			continue
		}
		data, err := fs.ReadFile(migrations.FS, entry.Name())
		require.NoError(t, err)
		oldFS[entry.Name()] = &fstest.MapFile{Data: data}
	}
	legacyName := "249_group_image_count_billing_auth_cache.sql"
	legacySQL, err := oldFS.ReadFile(legacyName)
	require.NoError(t, err)
	legacyChecksum := migrationChecksum(string(legacySQL))
	require.NoError(t, applyMigrationsFS(ctx, db, oldFS))
	require.NoError(t, verifyStandbyMigrationsFS(ctx, db, oldFS))

	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })
	user, err := client.User.Create().SetEmail("v0220-compat@example.invalid").SetPasswordHash("synthetic").Save(ctx)
	require.NoError(t, err)
	var accountIDs []int64
	defer func() {
		for _, id := range accountIDs {
			_, _ = db.ExecContext(context.Background(), "DELETE FROM scheduler_outbox WHERE account_id = $1", id)
			_, _ = db.ExecContext(context.Background(), "DELETE FROM accounts WHERE id = $1", id)
		}
		_, _ = db.ExecContext(context.Background(), "DELETE FROM users WHERE id = $1", user.ID)
	}()
	_, err = db.ExecContext(ctx, `UPDATE users SET balance = balance + 12.5 WHERE id = $1`, user.ID)
	require.NoError(t, err)
	accounts := newAccountRepositoryWithSQL(client, db, nil)
	makeAccount := func(name string, priority int) *service.Account {
		account := &service.Account{Name: name, Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey,
			Priority: priority, Concurrency: 3, Status: service.StatusActive, Schedulable: true,
			Credentials: map[string]any{"api_key": "synthetic"}, Extra: map[string]any{}}
		require.NoError(t, accounts.Create(ctx, account))
		accountIDs = append(accountIDs, account.ID)
		return account
	}
	a := makeAccount("v0220-order-a", 30)
	b := makeAccount("v0220-order-b", 10)
	require.NoError(t, accounts.ReorderAccounts(ctx, []int64{b.ID, a.ID}))

	require.NoError(t, applyMigrationsFS(ctx, db, migrations.FS))
	require.NoError(t, verifyStandbyMigrationsFS(ctx, db, oldFS), "the v0.19 migration set must remain valid on the upgraded schema")
	require.NoError(t, verifyStandbyMigrationsFS(ctx, db, migrations.FS))
	require.NoError(t, applyMigrationsFS(ctx, db, migrations.FS))
	var storedChecksum string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE filename=$1`, legacyName).Scan(&storedChecksum))
	require.Equal(t, legacyChecksum, storedChecksum)
	ordered, _, err := accounts.List(ctx, pagination.PaginationParams{Page: 1, PageSize: 2, SortOrder: "asc"})
	require.NoError(t, err)
	require.Equal(t, []int64{b.ID, a.ID}, []int64{ordered[0].ID, ordered[1].ID})
	require.NoError(t, db.QueryRowContext(ctx, `SELECT balance FROM users WHERE id=$1`, user.ID).Scan(&user.Balance))
	require.Equal(t, 12.5, user.Balance)
	_, err = db.ExecContext(ctx, `UPDATE users SET balance = balance - 2.5 WHERE id = $1`, user.ID)
	require.NoError(t, err)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT balance FROM users WHERE id=$1`, user.ID).Scan(&user.Balance))
	require.Equal(t, 10.0, user.Balance)
	var mcpTables int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('mcp_oauth_grants','mcp_oauth_codes','mcp_oauth_credentials')`).Scan(&mcpTables))
	require.Equal(t, 3, mcpTables)
}
