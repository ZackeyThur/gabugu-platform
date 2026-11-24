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

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestSub2APIV0220UpgradesCapturedOldSchema(t *testing.T) {
	dsn := os.Getenv("SUB2API_V0220_SNAPSHOT_TEST_DSN")
	require.NotEmpty(t, dsn, "set SUB2API_V0220_SNAPSHOT_TEST_DSN for the dedicated schema-only snapshot database")
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", u.Hostname())
	require.Equal(t, "15434", u.Port())
	require.Equal(t, "/account_order_test", u.Path)
	require.Equal(t, "candidate_test", u.User.Username())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	var users, accounts, oldLedger int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&users))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM accounts`).Scan(&accounts))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&oldLedger))
	require.Equal(t, []int{0, 0, 0}, []int{users, accounts, oldLedger}, "schema snapshot must remain data-free before synthetic migration ledger setup")

	oldMigrations := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS, ".")
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "249_mcp_oauth.sql" {
			continue
		}
		data, err := migrations.FS.ReadFile(entry.Name())
		require.NoError(t, err)
		oldMigrations[entry.Name()] = &fstest.MapFile{Data: data}
		_, err = db.ExecContext(ctx, `INSERT INTO schema_migrations(filename, checksum) VALUES ($1, $2)`, entry.Name(), migrationChecksum(string(data)))
		require.NoError(t, err)
	}
	require.Greater(t, len(oldMigrations), 300)
	_, err = db.ExecContext(ctx, `INSERT INTO atlas_schema_revisions(version, description, type, applied, total, executed_at, execution_time, hash)
		VALUES ('v0.2.19-schema-snapshot', 'synthetic prior Atlas baseline for schema-only snapshot', 1, 0, 0, NOW(), 0, 'schema-only')`)
	require.NoError(t, err)
	require.NoError(t, verifyStandbyMigrationsFS(ctx, db, oldMigrations))
	require.NoError(t, applyMigrationsFS(ctx, db, migrations.FS))
	require.NoError(t, verifyStandbyMigrationsFS(ctx, db, oldMigrations))
	require.NoError(t, verifyStandbyMigrationsFS(ctx, db, migrations.FS))
	require.NoError(t, applyMigrationsFS(ctx, db, migrations.FS))
	var newLedger int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&newLedger))
	require.Equal(t, len(oldMigrations)+1, newLedger)
	var mcpTables int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('mcp_oauth_grants','mcp_oauth_codes','mcp_oauth_credentials')`).Scan(&mcpTables))
	require.Equal(t, 3, mcpTables)
}
