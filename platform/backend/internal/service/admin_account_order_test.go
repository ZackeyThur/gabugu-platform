package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type accountOrderRepoStub struct {
	ids []int64
	err error
}

func (s *accountOrderRepoStub) ReorderAccounts(_ context.Context, ids []int64) error {
	s.ids = append([]int64(nil), ids...)
	return s.err
}

func TestValidateAccountOrderIDs(t *testing.T) {
	for _, ids := range [][]int64{nil, {}, {0}, {-1}, {1, 1}, make([]int64, MaxAccountReorderIDs+1)} {
		require.ErrorIs(t, ValidateAccountOrderIDs(ids), ErrAccountOrderInvalid)
	}
	ids := make([]int64, MaxAccountReorderIDs)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	require.NoError(t, ValidateAccountOrderIDs(ids))
}

func TestAdminReorderAccountsValidatesAndPassesOrder(t *testing.T) {
	repo := &accountOrderRepoStub{}
	svc := &adminServiceImpl{accountOrderRepo: repo}
	require.ErrorIs(t, svc.ReorderAccounts(context.Background(), []int64{1, 1}), ErrAccountOrderInvalid)
	require.Nil(t, repo.ids)
	require.NoError(t, svc.ReorderAccounts(context.Background(), []int64{8, 3, 5}))
	require.Equal(t, []int64{8, 3, 5}, repo.ids)
	repo.err = ErrAccountOrderConflict
	require.ErrorIs(t, svc.ReorderAccounts(context.Background(), []int64{8}), ErrAccountOrderConflict)
}
