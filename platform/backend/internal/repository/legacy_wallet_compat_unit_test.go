//go:build unit

package repository

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLegacyWalletUsageMetadataSharesNativeDedupTransaction(t *testing.T) {
	t.Setenv("GABUGU_LEGACY_WALLET_COMPAT", "true")
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := &usageBillingRepository{db: db}
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO usage_billing_dedup").WithArgs("legacy-usage", int64(7), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery("SELECT request_fingerprint").WithArgs("legacy-usage", int64(7)).WillReturnRows(sqlmock.NewRows([]string{"request_fingerprint"}))
	mock.ExpectExec("SELECT set_config").WithArgs("legacy-usage").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(conditionalBalanceDeductSQL).WithArgs(3.0, int64(9)).WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(9.0))
	mock.ExpectCommit()
	result, err := repo.Apply(context.Background(), &service.UsageBillingCommand{RequestID: "legacy-usage", UserID: 9, APIKeyID: 7, BalanceCost: 3})
	require.NoError(t, err)
	require.True(t, result.Applied)
	require.NoError(t, mock.ExpectationsWereMet())
}
func TestLegacyWalletMetadataFailureRollsBackNativeDedup(t *testing.T) {
	t.Setenv("GABUGU_LEGACY_WALLET_COMPAT", "true")
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := &usageBillingRepository{db: db}
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO usage_billing_dedup").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery("SELECT request_fingerprint").WillReturnRows(sqlmock.NewRows([]string{"request_fingerprint"}))
	mock.ExpectExec("SELECT set_config").WillReturnError(errors.New("metadata failed"))
	mock.ExpectRollback()
	_, err = repo.Apply(context.Background(), &service.UsageBillingCommand{RequestID: "metadata-error", UserID: 9, APIKeyID: 7, BalanceCost: 3})
	require.ErrorContains(t, err, "metadata failed")
	require.NoError(t, mock.ExpectationsWereMet())
}
