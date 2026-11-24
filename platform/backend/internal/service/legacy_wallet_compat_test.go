//go:build unit

package service

import (
	"context"
	"encoding/json"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLegacyWalletCompatDisabledNeedsNoLegacyDatabase(t *testing.T) {
	t.Setenv("GABUGU_LEGACY_WALLET_COMPAT", "")
	require.NoError(t, ValidateLegacyWalletCompat(context.Background(), nil))
	require.NoError(t, SetLegacyUsageWalletMetadata(context.Background(), nil, "req"))
	require.NoError(t, setLegacyPaymentWalletCredit(context.Background(), nil))
}
func TestLegacyWalletCompatPaymentCashGiftMetadata(t *testing.T) {
	raw, err := legacyPaymentWalletMetadata(&dbent.PaymentOrder{ID: 17, Amount: 12, BonusAmount: 2, OrderType: payment.OrderTypeBalance})
	require.NoError(t, err)
	var meta map[string]any
	require.NoError(t, json.Unmarshal(raw, &meta))
	require.Equal(t, float64(10), meta["cash"])
	require.Equal(t, float64(2), meta["gift"])
	require.Equal(t, float64(17), meta["order_id"])
	require.Equal(t, "payment:17", meta["source"])
}

func TestLegacyWalletCompatSurveyGiftMetadata(t *testing.T) {
	raw, err := legacyGiftWalletMetadata(legacyGiftWalletMeta{Amount: 0.5, Source: "survey:42"})
	require.NoError(t, err)
	var meta map[string]any
	require.NoError(t, json.Unmarshal(raw, &meta))
	require.Equal(t, float64(0), meta["cash"])
	require.Equal(t, float64(0.5), meta["gift"])
	require.Equal(t, "survey:42", meta["source"])
}

type legacyWalletFinalizationCacheSpy struct {
	APIKeyAuthCacheInvalidator
	users []int64
}

func (s *legacyWalletFinalizationCacheSpy) InvalidateAuthCacheByUserID(_ context.Context, uid int64) {
	s.users = append(s.users, uid)
}
func TestLegacyWalletFinalizationInvalidatesHeldZeroDeductionAndInviter(t *testing.T) {
	t.Setenv("GABUGU_LEGACY_WALLET_COMPAT", "true")
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	spy := &legacyWalletFinalizationCacheSpy{}
	svc := &PaymentService{entClient: client, redeemService: &RedeemService{authCacheInvalidator: spy}}
	mock.ExpectQuery("SELECT inviter_user_id FROM referral_enrollments").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"inviter_user_id"}).AddRow(11))
	svc.invalidateFinalizedRefundBalanceCache(context.Background(), &RefundPlan{Order: &dbent.PaymentOrder{UserID: 7}, DeductionType: payment.DeductionTypeBalance, deductionHeld: true, BalanceToDeduct: 0})
	require.Equal(t, []int64{7, 11}, spy.users)
	require.NoError(t, mock.ExpectationsWereMet())
}
func TestLegacyWalletDisabledKeepsNativeFinalizationInvalidation(t *testing.T) {
	t.Setenv("GABUGU_LEGACY_WALLET_COMPAT", "")
	spy := &legacyWalletFinalizationCacheSpy{}
	svc := &PaymentService{redeemService: &RedeemService{authCacheInvalidator: spy}}
	plan := &RefundPlan{Order: &dbent.PaymentOrder{UserID: 7}, DeductionType: payment.DeductionTypeBalance, deductionHeld: true, BalanceToDeduct: 0}
	svc.invalidateFinalizedRefundBalanceCache(context.Background(), plan)
	require.Empty(t, spy.users)
	plan.deductionHeld = false
	plan.BalanceToDeduct = 1
	svc.invalidateFinalizedRefundBalanceCache(context.Background(), plan)
	require.Equal(t, []int64{7}, spy.users)
}
