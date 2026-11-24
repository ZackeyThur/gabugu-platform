package service

import (
	"context"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const MaxAccountReorderIDs = 10000

var (
	ErrAccountOrderInvalid  = infraerrors.BadRequest("ACCOUNT_ORDER_INVALID", "ids must contain 1 to 10000 unique positive account IDs")
	ErrAccountOrderConflict = infraerrors.Conflict("ACCOUNT_ORDER_CONFLICT", "some accounts no longer exist; reload the account list")
)

func ValidateAccountOrderIDs(ids []int64) error {
	if len(ids) == 0 || len(ids) > MaxAccountReorderIDs {
		return ErrAccountOrderInvalid
	}
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return ErrAccountOrderInvalid
		}
		if _, exists := seen[id]; exists {
			return ErrAccountOrderInvalid
		}
		seen[id] = struct{}{}
	}
	return nil
}

func (s *adminServiceImpl) ReorderAccounts(ctx context.Context, ids []int64) error {
	if err := ValidateAccountOrderIDs(ids); err != nil {
		return err
	}
	return s.accountOrderRepo.ReorderAccounts(ctx, ids)
}
