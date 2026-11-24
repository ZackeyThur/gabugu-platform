//go:build v0220migration

package repository

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestSub2APIV0220MigrationNumberCollision(t *testing.T) {
	dsn := os.Getenv("SUB2API_V0220_MIGRATION_TEST_DSN")
	if dsn == "" {
		t.Skip("set SUB2API_V0220_MIGRATION_TEST_DSN for an isolated local PostgreSQL database")
	}
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", u.Hostname())
	require.Equal(t, "18620", u.Port())
	require.Equal(t, "/v0220_test", u.Path)
	require.Equal(t, "gabugu_test", u.User.Username())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	schema := "sub2api_v0220_" + strings.ReplaceAll(t.Name(), "/", "_")
	schema = strings.ToLower(schema)
	_, err = db.ExecContext(ctx, "CREATE SCHEMA "+pq.QuoteIdentifier(schema))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DROP SCHEMA "+pq.QuoteIdentifier(schema)+" CASCADE")
	})

	isolatedURL := *u
	query := isolatedURL.Query()
	query.Set("options", "-c search_path="+schema)
	isolatedURL.RawQuery = query.Encode()
	isolatedDB, err := sql.Open("postgres", isolatedURL.String())
	require.NoError(t, err)
	defer isolatedDB.Close()
	_, err = isolatedDB.ExecContext(ctx, `
CREATE TABLE users (id BIGINT PRIMARY KEY);
CREATE TABLE schema_migrations (filename TEXT PRIMARY KEY, checksum TEXT NOT NULL);
`)
	require.NoError(t, err)

	legacyName := "249_group_image_count_billing_auth_cache.sql"
	legacySQL, err := migrations.FS.ReadFile(legacyName)
	require.NoError(t, err)
	legacyChecksum := migrationChecksum(string(legacySQL))
	_, err = isolatedDB.ExecContext(ctx, `INSERT INTO schema_migrations(filename, checksum) VALUES ($1, $2)`, legacyName, legacyChecksum)
	require.NoError(t, err)

	upstreamName := "249_mcp_oauth.sql"
	upstreamSQL, err := migrations.FS.ReadFile(upstreamName)
	require.NoError(t, err)
	fsys := fstest.MapFS{
		legacyName:   {Data: legacySQL},
		upstreamName: {Data: upstreamSQL},
	}
	require.NoError(t, applyMigrationsFS(ctx, isolatedDB, fsys))
	require.NoError(t, applyMigrationsFS(ctx, isolatedDB, fsys))

	var rows int
	require.NoError(t, isolatedDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&rows))
	require.Equal(t, 2, rows)
	var persistedLegacyChecksum string
	require.NoError(t, isolatedDB.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE filename = $1`, legacyName).Scan(&persistedLegacyChecksum))
	require.Equal(t, legacyChecksum, persistedLegacyChecksum)
	var grantsTable bool
	require.NoError(t, isolatedDB.QueryRowContext(ctx, `SELECT to_regclass('mcp_oauth_grants') IS NOT NULL`).Scan(&grantsTable))
	require.True(t, grantsTable)
}
