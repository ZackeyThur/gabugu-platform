//go:build unit

package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

// Run only against the schema-only disposable upgrade verification database.
// Function replacement, fixtures, and outbox writes all roll back together.
func TestImageCountBillingInvalidationOnIsolatedPostgres(t *testing.T) {
	if os.Getenv("GABUGU_V0219_ISOLATED_TEST") != "1" {
		t.Skip("requires the dedicated local upgrade verification instance")
	}
	db, err := sql.Open("postgres", "host=127.0.0.1 port=15432 user=candidate_test dbname=wallet_upgrade_verify sslmode=disable connect_timeout=5")
	require.NoError(t, err)
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var database, user, server string
	var port int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT current_database(), current_user, inet_server_addr()::text, inet_server_port()").Scan(&database, &user, &server, &port))
	require.Equal(t, "wallet_upgrade_verify", database)
	require.Equal(t, "candidate_test", user)
	require.Equal(t, "127.0.0.1/32", server)
	require.Equal(t, 15432, port)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	// Establish the column as well as the trigger without depending on another
	// test having upgraded this schema-only database first.
	for _, name := range []string{"248_group_image_count_billing.sql", "249_group_image_count_billing_auth_cache.sql"} {
		migration, err := fs.ReadFile(migrations.FS, name)
		require.NoError(t, err)
		_, err = tx.ExecContext(ctx, string(migration))
		require.NoError(t, err)
	}

	suffix := time.Now().UnixNano()
	var groupID, otherGroupID, userID int64
	require.NoError(t, tx.QueryRowContext(ctx, "INSERT INTO groups (name, platform) VALUES ($1, 'openai') RETURNING id", fmt.Sprintf("image-billing-trigger-%d", suffix)).Scan(&groupID))
	require.NoError(t, tx.QueryRowContext(ctx, "INSERT INTO groups (name, platform) VALUES ($1, 'openai') RETURNING id", fmt.Sprintf("image-billing-other-%d", suffix)).Scan(&otherGroupID))
	require.NoError(t, tx.QueryRowContext(ctx, "INSERT INTO users (email, password_hash) VALUES ($1, 'synthetic-test-only') RETURNING id", fmt.Sprintf("image-billing-%d@example.invalid", suffix)).Scan(&userID))
	var cacheKeys []string
	for i, fixture := range []struct {
		groupID int64
		status  string
		deleted bool
		empty   bool
	}{
		{groupID: groupID, status: "active"},
		{groupID: groupID, status: "disabled"}, // Existing trigger includes disabled keys too.
		{groupID: groupID, status: "active", deleted: true},
		{groupID: otherGroupID, status: "active"},
		{groupID: groupID, status: "active", empty: true},
	} {
		key := fmt.Sprintf("sk-image-billing-%d-%d", suffix, i)
		if fixture.empty {
			key = ""
		}
		sum := sha256.Sum256([]byte(key))
		cacheKeys = append(cacheKeys, hex.EncodeToString(sum[:]))
		_, err := tx.ExecContext(ctx, `INSERT INTO api_keys (user_id, group_id, key, name, status, deleted_at)
			VALUES ($1, $2, $3, 'synthetic-image-billing', $4, CASE WHEN $5 THEN NOW() ELSE NULL END)`, userID, fixture.groupID, key, fixture.status, fixture.deleted)
		require.NoError(t, err)
	}
	clear := func(t *testing.T) {
		t.Helper()
		for _, key := range cacheKeys {
			_, err := tx.ExecContext(ctx, "DELETE FROM auth_cache_invalidation_outbox WHERE cache_key = $1", key)
			require.NoError(t, err)
		}
	}
	assertCounts := func(t *testing.T, expected int) {
		t.Helper()
		for i, key := range cacheKeys {
			var count int
			require.NoError(t, tx.QueryRowContext(ctx, "SELECT count(*) FROM auth_cache_invalidation_outbox WHERE cache_key = $1", key).Scan(&count))
			want := 0
			if i < 2 {
				want = expected
			}
			require.Equal(t, want, count, "fixture %d: bound nondeleted nonempty keys only", i)
		}
	}
	clear(t)
	for _, test := range []struct {
		name   string
		update string
		count  int
	}{
		{"enable image count billing only", "image_count_billing = true", 1},
		{"same image count billing value", "image_count_billing = true", 0},
		{"disable image count billing only", "image_count_billing = false", 1},
		{"cosmetic change", "name = name || '-renamed'", 0},
		{"image permission", "allow_image_generation = NOT allow_image_generation", 1},
		{"status", "status = 'disabled'", 1},
		{"exclusive", "is_exclusive = NOT is_exclusive", 1},
		{"platform", "platform = 'anthropic'", 1},
		{"subscription type", "subscription_type = 'subscription'", 1},
		{"rate multiplier", "rate_multiplier = rate_multiplier + 0.1", 1},
		{"peak enabled", "peak_rate_enabled = NOT peak_rate_enabled", 1},
		{"peak start", "peak_start = '08:17'", 1},
		{"peak end", "peak_end = '09:23'", 1},
		{"peak multiplier", "peak_rate_multiplier = peak_rate_multiplier + 0.1", 1},
		{"profit enabled", "profit_control_enabled = NOT profit_control_enabled", 1},
		{"profit margin", "profit_min_margin = profit_min_margin + 0.01", 1},
		{"profit buffer", "profit_safety_buffer = profit_safety_buffer + 0.01", 1},
		{"same watched values", "profit_min_margin = profit_min_margin, image_count_billing = image_count_billing", 0},
		{"soft delete", "deleted_at = NOW()", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			clear(t)
			_, err := tx.ExecContext(ctx, "UPDATE groups SET "+test.update+" WHERE id = $1", groupID)
			require.NoError(t, err)
			assertCounts(t, test.count)
		})
	}
}
