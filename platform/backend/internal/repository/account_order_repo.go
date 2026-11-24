package repository

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// ReorderAccounts changes presentation order only. It deliberately bypasses
// account update hooks, scheduler outbox/cache writes, credentials and runtime state.
func (r *accountRepository) ReorderAccounts(ctx context.Context, ids []int64) error {
	if err := service.ValidateAccountOrderIDs(ids); err != nil {
		return err
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `UPDATE accounts AS a
SET display_order = input.position
FROM unnest($1::bigint[]) WITH ORDINALITY AS input(id, position)
WHERE a.id = input.id AND a.deleted_at IS NULL`, pq.Array(ids))
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return service.ErrAccountOrderConflict
	}
	return tx.Commit()
}
