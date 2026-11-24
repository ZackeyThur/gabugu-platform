package repository

import (
	"context"
	"errors"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestReorderAccountsAtomicDisplayOnly(t *testing.T) {
	for _, tc := range []struct {
		name    string
		rows    int64
		execErr error
		want    error
	}{
		{name: "success", rows: 3},
		{name: "missing_or_deleted_id_rolls_back", rows: 2, want: service.ErrAccountOrderConflict},
		{name: "query_error_rolls_back", execErr: errors.New("database unavailable"), want: errors.New("database unavailable")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
			t.Cleanup(func() { _ = client.Close() })
			mock.ExpectBegin()
			// An exact write shape rejects priority/runtime changes or extra cache/outbox writes.
			exec := mock.ExpectExec(`(?s)^UPDATE accounts AS a\s+SET display_order = input.position\s+FROM unnest\(\$1::bigint\[\]\) WITH ORDINALITY AS input\(id, position\)\s+WHERE a.id = input.id AND a.deleted_at IS NULL$`).WithArgs("{8,3,5}")
			if tc.execErr != nil {
				exec.WillReturnError(tc.execErr)
			} else {
				exec.WillReturnResult(sqlmock.NewResult(0, tc.rows))
			}
			if tc.want == nil {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}
			repo := newAccountRepositoryWithSQL(client, db, nil)
			err = repo.ReorderAccounts(context.Background(), []int64{8, 3, 5})
			if tc.want == nil {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.want.Error())
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestReorderAccountsRejectsInvalidBeforeDatabase(t *testing.T) {
	repo := &accountRepository{}
	for _, ids := range [][]int64{nil, {0}, {-1}, {2, 2}, make([]int64, service.MaxAccountReorderIDs+1)} {
		require.ErrorIs(t, repo.ReorderAccounts(context.Background(), ids), service.ErrAccountOrderInvalid)
	}
}

func TestAccountListDisplayOrderSQL(t *testing.T) {
	for _, sortBy := range []string{"", "display_order"} {
		for _, order := range []string{"asc", "desc"} {
			s := entsql.Dialect(dialect.Postgres).Select("*").From(entsql.Table("accounts"))
			for _, apply := range accountListOrder(pagination.PaginationParams{SortBy: sortBy, SortOrder: order}) {
				apply(s)
			}
			query, args := s.Query()
			require.Empty(t, args)
			require.Contains(t, query, `CASE WHEN "accounts"."display_order" > 0 THEN 0 ELSE 1 END ASC`)
			if order == "desc" {
				require.Contains(t, query, `"accounts"."display_order" DESC`)
			} else {
				require.NotContains(t, query, `"accounts"."display_order" DESC`)
			}
			require.Contains(t, query, `"accounts"."name" ASC, "accounts"."id" ASC`)
		}
	}
}
