//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestV0220FullMigrationRunnerIsIdempotent(t *testing.T) {
	ctx := context.Background()
	var before int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&before))
	require.Greater(t, before, 0)

	for range 2 {
		require.NoError(t, ApplyMigrations(ctx, integrationDB))
	}
	require.NoError(t, verifyStandbyMigrationsFS(ctx, integrationDB, migrations.FS))

	var after int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&after))
	require.Equal(t, before, after)
	for _, table := range []string{"mcp_oauth_grants", "mcp_oauth_codes", "mcp_oauth_credentials"} {
		var exists bool
		require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT to_regclass($1) IS NOT NULL`, table).Scan(&exists))
		require.True(t, exists, table)
	}
}
