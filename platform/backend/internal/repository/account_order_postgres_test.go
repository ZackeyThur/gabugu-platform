package repository

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// This opt-in real PostgreSQL test accepts only this task's dedicated local DB.
// It never falls back to application configuration or any existing database.
func TestAccountDisplayOrderPostgres(t *testing.T) {
	dsn := os.Getenv("ACCOUNT_ORDER_TEST_DSN")
	if dsn == "" {
		t.Skip("set ACCOUNT_ORDER_TEST_DSN for isolated PostgreSQL validation")
	}
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", u.Hostname())
	require.Equal(t, "15434", u.Port())
	require.Equal(t, "/account_order_test", u.Path)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.PingContext(ctx))
	require.NoError(t, ApplyMigrations(ctx, db))
	// Re-running the embedded migration chain must be safe.
	require.NoError(t, ApplyMigrations(ctx, db))
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	repo := newAccountRepositoryWithSQL(client, db, nil)
	makeAccount := func(name string, priority int) *service.Account {
		a := &service.Account{Name: name, Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey,
			Priority: priority, DisplayOrder: 99, Concurrency: 3, Status: service.StatusActive, Schedulable: true,
			Credentials: map[string]any{"api_key": "local-test-only"}, Extra: map[string]any{"marker": "keep"}}
		require.NoError(t, repo.Create(ctx, a))
		t.Cleanup(func() {
			_, _ = db.ExecContext(context.Background(), "DELETE FROM scheduler_outbox WHERE account_id = $1", a.ID)
			_, _ = db.ExecContext(context.Background(), "DELETE FROM accounts WHERE id = $1", a.ID)
		})
		require.Zero(t, a.DisplayOrder, "new accounts must start unarranged")
		return a
	}
	a := makeAccount("rank-a", 29)
	b := makeAccount("rank-z", 10)
	pendingZ := makeAccount("pending-z", 50)
	pendingA := makeAccount("pending-a", 70)
	deleted := makeAccount("deleted", 80)
	_, err = db.ExecContext(ctx, "UPDATE accounts SET deleted_at = NOW() WHERE id = $1", deleted.ID)
	require.NoError(t, err)
	snapshot := func(id int64) string {
		var value string
		require.NoError(t, db.QueryRowContext(ctx, "SELECT (to_jsonb(accounts) - 'display_order')::text FROM accounts WHERE id = $1", id).Scan(&value))
		return value
	}
	beforeA, beforeB := snapshot(a.ID), snapshot(b.ID)
	var beforeOutbox int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM scheduler_outbox").Scan(&beforeOutbox))
	require.NoError(t, repo.ReorderAccounts(ctx, []int64{b.ID, a.ID}))
	require.Equal(t, beforeA, snapshot(a.ID))
	require.Equal(t, beforeB, snapshot(b.ID))
	var afterOutbox int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM scheduler_outbox").Scan(&afterOutbox))
	require.Equal(t, beforeOutbox, afterOutbox)
	// A separate repository/client reads the same persisted order, as another device does.
	readerClient := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	reader := newAccountRepositoryWithSQL(readerClient, db, nil)
	for _, tc := range []struct {
		page  int
		order string
		ids   []int64
	}{
		{1, "asc", []int64{b.ID, a.ID}}, {2, "asc", []int64{pendingA.ID, pendingZ.ID}},
		{1, "desc", []int64{a.ID, b.ID}}, {2, "desc", []int64{pendingA.ID, pendingZ.ID}},
	} {
		accounts, result, err := reader.List(ctx, pagination.PaginationParams{Page: tc.page, PageSize: 2, SortOrder: tc.order})
		require.NoError(t, err)
		require.EqualValues(t, 4, result.Total)
		require.Len(t, accounts, 2)
		require.Equal(t, tc.ids, []int64{accounts[0].ID, accounts[1].ID})
	}
	for _, absent := range []int64{deleted.ID, 9223372036854775807} {
		require.ErrorIs(t, repo.ReorderAccounts(ctx, []int64{a.ID, absent}), service.ErrAccountOrderConflict)
		stored, err := repo.GetByID(ctx, a.ID)
		require.NoError(t, err)
		require.Equal(t, 2, stored.DisplayOrder, "partial order must roll back")
	}
	// A normal settings save does not overwrite a newer presentation order.
	stored, err := repo.GetByID(ctx, a.ID)
	require.NoError(t, err)
	stored.DisplayOrder = 0 // simulate a stale editor snapshot
	stored.Name = "renamed"
	require.NoError(t, repo.Update(ctx, stored))
	stored, err = repo.GetByID(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, 2, stored.DisplayOrder)
	require.Equal(t, 29, stored.Priority)
	accounts, _, err := repo.List(ctx, pagination.PaginationParams{Page: 1, PageSize: 10, SortBy: "priority"})
	require.NoError(t, err)
	require.Equal(t, b.ID, accounts[0].ID, "scheduler priority remains an independent sort")
}
