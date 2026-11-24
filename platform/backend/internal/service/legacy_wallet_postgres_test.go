//go:build legacy_wallet_integration

package service

import (
	"context"
	"database/sql"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"fmt"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

// Fixtures and service operations share one transaction, rolled back by each test.
type legacyWalletPGUserRepo struct {
	UserRepository
	client *dbent.Client
}

func (r *legacyWalletPGUserRepo) DeductAvailableBalance(ctx context.Context, uid int64, amount float64) (float64, error) {
	client := r.client
	if tx := dbent.TxFromContext(ctx); tx != nil {
		client = tx.Client()
	}
	rows, err := client.QueryContext(ctx, `WITH target AS(SELECT id,balance FROM users WHERE id=$2 AND deleted_at IS NULL FOR UPDATE),updated AS(UPDATE users u SET balance=target.balance-LEAST($1,GREATEST(target.balance,0)) FROM target WHERE u.id=target.id RETURNING target.balance-u.balance AS deducted)SELECT deducted FROM updated`, amount, uid)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, fmt.Errorf("missing user")
	}
	var got float64
	err = rows.Scan(&got)
	return got, err
}
func legacyWalletTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("GABUGU_LEGACY_WALLET_TEST_DSN")
	if dsn == "" {
		t.Skip("requires disposable restored Gabugu DB")
	}
	t.Setenv("GABUGU_LEGACY_WALLET_COMPAT", "true")
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	require.NoError(t, ValidateLegacyWalletCompat(context.Background(), db))
	return db
}
func TestLegacyWalletPostgresCreditUsageRefundRestoration(t *testing.T) {
	db := legacyWalletTestDB(t)
	ctx := context.Background()
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer tx.Rollback()
	ctx = dbent.NewTxContext(ctx, tx)
	client = tx.Client()
	u, err := client.User.Create().SetEmail(fmt.Sprintf("legacy-compat-%d@example.invalid", time.Now().UnixNano())).SetPasswordHash("synthetic").SetBalance(0).Save(ctx)
	require.NoError(t, err)
	o, err := client.PaymentOrder.Create().SetUserID(u.ID).SetUserEmail(u.Email).SetUserName("compat-test").SetAmount(12).SetBonusAmount(2).SetPayAmount(10).SetFeeRate(0).SetRechargeCode(fmt.Sprintf("COMPAT-%d", time.Now().UnixNano())).SetOutTradeNo(fmt.Sprintf("compat_%d", time.Now().UnixNano())).SetPaymentType(payment.TypeAlipay).SetPaymentTradeNo("synthetic-no-provider").SetOrderType(payment.OrderTypeBalance).SetStatus(OrderStatusCompleted).SetExpiresAt(time.Now().Add(time.Hour)).SetClientIP("127.0.0.1").SetSrcHost("example.invalid").Save(ctx)
	require.NoError(t, err)
	require.NoError(t, setLegacyPaymentWalletCredit(withLegacyPaymentWallet(ctx, o), client))
	_, err = client.ExecContext(ctx, "UPDATE users SET balance=balance+12 WHERE id=$1", u.ID)
	require.NoError(t, err)
	require.NoError(t, recordLegacyPaymentWalletReceipt(withLegacyPaymentWallet(ctx, o), client))
	assertReceipt := func(wantOrder int64) {
		rows, e := client.QueryContext(ctx, "SELECT order_id,principal,historical FROM referral_first_recharges WHERE user_id=$1", u.ID)
		require.NoError(t, e)
		defer rows.Close()
		require.True(t, rows.Next())
		var orderID int64
		var principal float64
		var historical bool
		require.NoError(t, rows.Scan(&orderID, &principal, &historical))
		require.Equal(t, wantOrder, orderID)
		require.Equal(t, 10.0, principal)
		require.True(t, historical)
	}
	assertReceipt(o.ID)
	laterOrder, err := client.PaymentOrder.Create().SetUserID(u.ID).SetUserEmail(u.Email).SetUserName("compat-later").SetAmount(20).SetPayAmount(20).SetFeeRate(0).SetRechargeCode(fmt.Sprintf("LATER-%d", time.Now().UnixNano())).SetOutTradeNo(fmt.Sprintf("later_%d", time.Now().UnixNano())).SetPaymentType(payment.TypeAlipay).SetPaymentTradeNo("synthetic-no-provider").SetOrderType(payment.OrderTypeBalance).SetStatus(OrderStatusCompleted).SetExpiresAt(time.Now().Add(time.Hour)).SetClientIP("127.0.0.1").SetSrcHost("example.invalid").Save(ctx)
	require.NoError(t, err)
	require.NoError(t, recordLegacyPaymentWalletReceipt(withLegacyPaymentWallet(ctx, laterOrder), client))
	assertReceipt(o.ID)
	// Prove a later SQL failure rolls credit, lots and receipt back together.
	// The savepoint restores the original receipt and excludes the later credit.
	_, err = client.ExecContext(ctx, "SAVEPOINT receipt_failure")
	require.NoError(t, err)
	_, err = client.ExecContext(ctx, "DELETE FROM referral_first_recharges WHERE user_id=$1", u.ID)
	require.NoError(t, err)
	require.NoError(t, setLegacyPaymentWalletCredit(withLegacyPaymentWallet(ctx, laterOrder), client))
	_, err = client.ExecContext(ctx, "UPDATE users SET balance=balance+20 WHERE id=$1", u.ID)
	require.NoError(t, err)
	require.NoError(t, recordLegacyPaymentWalletReceipt(withLegacyPaymentWallet(ctx, laterOrder), client))
	_, err = client.ExecContext(ctx, "UPDATE source_wallet_lots SET held=-1 WHERE user_id=$1", u.ID)
	require.Error(t, err)
	_, err = client.ExecContext(ctx, "ROLLBACK TO SAVEPOINT receipt_failure")
	require.NoError(t, err)
	assertReceipt(o.ID)
	rowsReceipt, err := client.QueryContext(ctx, "SELECT u.balance,(SELECT COUNT(*) FROM source_wallet_lots l WHERE l.user_id=u.id AND l.payment_order_id=$2) FROM users u WHERE u.id=$1", u.ID, laterOrder.ID)
	require.NoError(t, err)
	require.True(t, rowsReceipt.Next())
	var receiptBalance float64
	var laterLots int
	require.NoError(t, rowsReceipt.Scan(&receiptBalance, &laterLots))
	rowsReceipt.Close()
	require.Equal(t, 12.0, receiptBalance)
	require.Zero(t, laterLots)
	snapshot := func() (balance, cash, gift, spent float64) {
		rows, e := client.QueryContext(ctx, `SELECT u.balance,COALESCE(SUM(l.remaining)FILTER(WHERE l.kind='cash'),0),COALESCE(SUM(l.remaining)FILTER(WHERE l.kind='gift'),0),COALESCE(SUM(l.spent)FILTER(WHERE l.kind='cash'),0) FROM users u LEFT JOIN source_wallet_lots l ON l.user_id=u.id WHERE u.id=$1 GROUP BY u.id`, u.ID)
		require.NoError(t, e)
		defer rows.Close()
		require.True(t, rows.Next())
		require.NoError(t, rows.Scan(&balance, &cash, &gift, &spent))
		return
	}
	balance, cash, gift, spent := snapshot()
	require.Equal(t, 12.0, balance)
	require.Equal(t, 10.0, cash)
	require.Equal(t, 2.0, gift)
	require.Zero(t, spent)
	// A failed refund spanning gift and principal restores the exact original lots.
	preSvc := &PaymentService{entClient: client, userRepo: &legacyWalletPGUserRepo{client: client}}
	prePlan := &RefundPlan{OrderID: o.ID, Order: o, RefundAmount: 11, BalanceToDeduct: 11, DeductBalance: true, DeductionType: payment.DeductionTypeBalance}
	got, err := preSvc.deductSourceRefund(ctx, prePlan)
	require.NoError(t, err)
	require.Equal(t, 11.0, got)
	balance, cash, gift, _ = snapshot()
	require.Equal(t, 1.0, balance)
	require.Zero(t, cash)
	require.Equal(t, 1.0, gift)
	require.NoError(t, preSvc.restoreSourceRefund(ctx, prePlan))
	balance, cash, gift, _ = snapshot()
	require.Equal(t, 12.0, balance)
	require.Equal(t, 10.0, cash)
	require.Equal(t, 2.0, gift)
	// Same transaction-local metadata as the native UsageBillingRepository hook.
	_, err = client.ExecContext(ctx, `SELECT set_config('gabugu.wallet_credit','{"operation":"usage"}',true),set_config('gabugu.wallet_request','compat-usage',true)`)
	require.NoError(t, err)
	_, err = client.ExecContext(ctx, "UPDATE users SET balance=balance-3 WHERE id=$1", u.ID)
	require.NoError(t, err)
	balance, cash, gift, spent = snapshot()
	require.Equal(t, 9.0, balance)
	require.Equal(t, 9.0, cash)
	require.Zero(t, gift)
	require.Equal(t, 1.0, spent)
	rows, err := client.QueryContext(ctx, "SELECT COALESCE(SUM(amount)FILTER(WHERE cash_qualifying),0) FROM source_wallet_events WHERE user_id=$1 AND request_id='compat-usage'", u.ID)
	require.NoError(t, err)
	require.True(t, rows.Next())
	var qualified float64
	require.NoError(t, rows.Scan(&qualified))
	rows.Close()
	require.Equal(t, 1.0, qualified)
	svc := &PaymentService{entClient: client, userRepo: &legacyWalletPGUserRepo{client: client}}
	plan := &RefundPlan{OrderID: o.ID, Order: o, RefundAmount: 4, BalanceToDeduct: 4, DeductBalance: true, DeductionType: payment.DeductionTypeBalance}
	for i := 0; i < 2; i++ {
		deducted, e := svc.deductSourceRefund(ctx, plan)
		require.NoError(t, e)
		require.Equal(t, 4.0, deducted)
	}
	balance, cash, gift, spent = snapshot()
	require.Equal(t, 5.0, balance)
	require.Equal(t, 5.0, cash)
	require.Zero(t, gift)
	require.Equal(t, 1.0, spent)
	require.NoError(t, svc.restoreSourceRefund(ctx, plan))
	require.NoError(t, svc.restoreSourceRefund(ctx, plan))
	balance, cash, _, _ = snapshot()
	require.Equal(t, 9.0, balance)
	require.Equal(t, 9.0, cash)
	_, err = client.ExecContext(ctx, "SAVEPOINT compat_failure")
	require.NoError(t, err)
	_, err = svc.deductSourceRefund(ctx, plan)
	require.NoError(t, err)
	_, err = client.ExecContext(ctx, "UPDATE source_wallet_lots SET held=-1 WHERE user_id=$1", u.ID)
	require.Error(t, err)
	_, err = client.ExecContext(ctx, "ROLLBACK TO SAVEPOINT compat_failure")
	require.NoError(t, err)
	balance, cash, _, _ = snapshot()
	require.Equal(t, 9.0, balance)
	require.Equal(t, 9.0, cash)
	_, err = svc.deductSourceRefund(ctx, plan)
	require.NoError(t, err)
	require.NoError(t, svc.finalizeSourceRefundTx(ctx, client, plan))
	rows, err = client.QueryContext(ctx, "SELECT state FROM source_wallet_refund_attempts WHERE order_id=$1", o.ID)
	require.NoError(t, err)
	require.True(t, rows.Next())
	var state string
	require.NoError(t, rows.Scan(&state))
	rows.Close()
	require.Equal(t, "completed", state)
	require.Error(t, svc.restoreSourceRefund(ctx, plan))
	// Refund finalization removes only this order's exact held slices. A later
	// release observes refunded slices and cannot restore the same money again.
	var lot int64
	rows, err = client.QueryContext(ctx, "SELECT id FROM source_wallet_lots WHERE user_id=$1 AND payment_order_id=$2 AND kind='cash'", u.ID, o.ID)
	require.NoError(t, err)
	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&lot))
	rows.Close()
	_, err = client.ExecContext(ctx, "INSERT INTO source_wallet_reservations(user_id,request_id,fingerprint,maximum,state)VALUES($1,'forced-held','forced-held',5,'reserved')", u.ID)
	require.NoError(t, err)
	_, err = client.ExecContext(ctx, "INSERT INTO source_wallet_reservation_lots(user_id,request_id,lot_id,amount)VALUES($1,'forced-held',$2,5)", u.ID, lot)
	require.NoError(t, err)
	_, err = client.ExecContext(ctx, "UPDATE source_wallet_lots SET remaining=remaining-5,held=held+5 WHERE id=$1", lot)
	require.NoError(t, err)
	_, err = client.ExecContext(ctx, "SELECT set_config('gabugu.wallet_managed','1',true)")
	require.NoError(t, err)
	_, err = client.ExecContext(ctx, "UPDATE users SET balance=balance-5,frozen_balance=frozen_balance+5 WHERE id=$1", u.ID)
	require.NoError(t, err)
	_, err = client.ExecContext(ctx, "SELECT set_config('gabugu.wallet_managed','',true)")
	require.NoError(t, err)
	plan.RefundAmount = 12
	require.NoError(t, svc.finalizeSourceRefundTx(ctx, client, plan))
	require.NoError(t, svc.finalizeSourceRefundTx(ctx, client, plan))
	var frozen, refunded float64
	rows, err = client.QueryContext(ctx, "SELECT u.frozen_balance,p.refunded FROM users u JOIN source_wallet_reservation_lots p ON p.user_id=u.id WHERE u.id=$1 AND p.request_id='forced-held'", u.ID)
	require.NoError(t, err)
	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&frozen, &refunded))
	rows.Close()
	require.Zero(t, frozen)
	require.Equal(t, 5.0, refunded)

}
func TestLegacyWalletPostgresBatchPreservesHistoricalReservation(t *testing.T) {
	db := legacyWalletTestDB(t)
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	var uid int64
	err = tx.QueryRowContext(ctx, "INSERT INTO users(email,password_hash,balance) VALUES($1,'synthetic',0) RETURNING id", fmt.Sprintf("legacy-batch-%d@example.invalid", time.Now().UnixNano())).Scan(&uid)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `SELECT set_config('gabugu.wallet_credit','{"cash":10,"gift":2,"source":"compat-batch"}',true)`)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, "UPDATE users SET balance=balance+12 WHERE id=$1", uid)
	require.NoError(t, err)
	historical := SourceWalletReserveInput{UserID: uid, RequestID: "historical-playground", Fingerprint: "historical-playground", MaxAmountUSD: 0.14}
	_, err = ReserveSourceWalletTx(ctx, tx, historical)
	require.NoError(t, err)
	batch := SourceWalletReserveInput{UserID: uid, RequestID: "new-native-batch", Fingerprint: "new-native-batch", MaxAmountUSD: 8}
	_, err = ReserveSourceWalletTx(ctx, tx, batch)
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		result, e := SettleSourceWalletTx(ctx, tx, uid, batch.RequestID, 3, batch.Fingerprint, false)
		require.NoError(t, e)
		require.Equal(t, "settled", result.Status)
	}
	var balance, frozen, held float64
	var state string
	err = tx.QueryRowContext(ctx, `SELECT u.balance,u.frozen_balance,r.state,COALESCE(SUM(p.amount-p.refunded),0) FROM users u JOIN source_wallet_reservations r ON r.user_id=u.id LEFT JOIN source_wallet_reservation_lots p ON p.user_id=r.user_id AND p.request_id=r.request_id WHERE u.id=$1 AND r.request_id=$2 GROUP BY u.id,r.user_id,r.request_id`, uid, historical.RequestID).Scan(&balance, &frozen, &state, &held)
	require.NoError(t, err)
	require.InDelta(t, 8.86, balance, 1e-8)
	require.InDelta(t, 0.14, frozen, 1e-8)
	require.Equal(t, "reserved", state)
	require.InDelta(t, 0.14, held, 1e-8)
	var lotsBalance, lotsHeld float64
	err = tx.QueryRowContext(ctx, "SELECT SUM(remaining),SUM(held) FROM source_wallet_lots WHERE user_id=$1", uid).Scan(&lotsBalance, &lotsHeld)
	require.NoError(t, err)
	require.InDelta(t, balance, lotsBalance, 1e-8)
	require.InDelta(t, frozen, lotsHeld, 1e-8)
}
