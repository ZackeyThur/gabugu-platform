package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"os"
	"strings"
)

// LegacyWalletCompatEnabled preserves source provenance only for migrated Gabugu databases.
func LegacyWalletCompatEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("GABUGU_LEGACY_WALLET_COMPAT")), "true")
}

// ValidateLegacyWalletCompat fails startup before serving if the explicitly enabled old schema is incomplete.
func ValidateLegacyWalletCompat(ctx context.Context, db *sql.DB) error {
	if !LegacyWalletCompatEnabled() {
		return nil
	}
	var ok bool
	err := db.QueryRowContext(ctx, `SELECT to_regclass('public.source_wallet_lots') IS NOT NULL AND to_regclass('public.source_wallet_events') IS NOT NULL AND to_regclass('public.source_wallet_reservations') IS NOT NULL AND to_regclass('public.source_wallet_reservation_lots') IS NOT NULL AND to_regclass('public.source_wallet_refund_attempts') IS NOT NULL AND to_regclass('public.referral_enrollments') IS NOT NULL AND to_regclass('public.referral_first_recharges') IS NOT NULL AND to_regprocedure('public.referral_restore_reward_debit(bigint,numeric)') IS NOT NULL AND to_regprocedure('public.referral_adjust_first_recharge(bigint)') IS NOT NULL AND EXISTS(SELECT 1 FROM pg_trigger WHERE tgrelid='public.users'::regclass AND tgname='source_wallet_balance_update' AND tgenabled='O')`).Scan(&ok)
	if err != nil {
		return fmt.Errorf("legacy wallet schema check: %w", err)
	}
	if !ok {
		return errors.New("GABUGU_LEGACY_WALLET_COMPAT requires existing Gabugu source-wallet schema and enabled trigger")
	}
	return nil
}

type legacyPaymentWalletKey struct{}
type legacyGiftWalletKey struct{}

type legacyGiftWalletMeta struct {
	Amount float64
	Source string
}

func withLegacyPaymentWallet(ctx context.Context, o *dbent.PaymentOrder) context.Context {
	if !LegacyWalletCompatEnabled() {
		return ctx
	}
	return context.WithValue(ctx, legacyPaymentWalletKey{}, o)
}

func withLegacyGiftWallet(ctx context.Context, amount float64, source string) context.Context {
	if !LegacyWalletCompatEnabled() {
		return ctx
	}
	return context.WithValue(ctx, legacyGiftWalletKey{}, legacyGiftWalletMeta{
		Amount: QuantizeUsageBillingAmount(amount),
		Source: strings.TrimSpace(source),
	})
}

func legacyPaymentWalletMetadata(o *dbent.PaymentOrder) ([]byte, error) {
	principal := paymentOrderAmountWithoutBonus(o)
	return json.Marshal(map[string]any{"cash": principal, "gift": QuantizeUsageBillingAmount(o.Amount - principal), "order_id": o.ID, "source": fmt.Sprintf("payment:%d", o.ID)})
}

func legacyGiftWalletMetadata(m legacyGiftWalletMeta) ([]byte, error) {
	return json.Marshal(map[string]any{"cash": 0, "gift": QuantizeUsageBillingAmount(m.Amount), "source": m.Source})
}

// The payment metadata is installed on the same Ent transaction that claims the redeem code and credits the user.
func setLegacyPaymentWalletCredit(ctx context.Context, client *dbent.Client) error {
	if !LegacyWalletCompatEnabled() {
		return nil
	}
	if o, ok := ctx.Value(legacyPaymentWalletKey{}).(*dbent.PaymentOrder); ok && o != nil {
		meta, err := legacyPaymentWalletMetadata(o)
		if err != nil {
			return err
		}
		_, err = client.ExecContext(ctx, "SELECT set_config('gabugu.wallet_credit',$1,true)", string(meta))
		return err
	}
	if g, ok := ctx.Value(legacyGiftWalletKey{}).(legacyGiftWalletMeta); ok && g.Amount > 0 {
		meta, err := legacyGiftWalletMetadata(g)
		if err != nil {
			return err
		}
		_, err = client.ExecContext(ctx, "SELECT set_config('gabugu.wallet_credit',$1,true)", string(meta))
		return err
	}
	return nil
}

// SetLegacyUsageWalletMetadata shares the native billing/dedup transaction; the old trigger consumes this once.
func SetLegacyUsageWalletMetadata(ctx context.Context, tx *sql.Tx, request string) error {
	if !LegacyWalletCompatEnabled() {
		return nil
	}
	_, err := tx.ExecContext(ctx, `SELECT set_config('gabugu.wallet_credit','{"operation":"usage"}',true),set_config('gabugu.wallet_request',$1,true)`, request)
	return err
}

// recordLegacyPaymentWalletReceipt consumes first-recharge eligibility without
// granting an old-product reward. It must commit with the actual payment credit.
// Migration 247 defines user_id as the receipt primary key; later orders cannot
// replace the first one. No referral_record_first_recharge function is invoked.
func recordLegacyPaymentWalletReceipt(ctx context.Context, client *dbent.Client) error {
	o, ok := ctx.Value(legacyPaymentWalletKey{}).(*dbent.PaymentOrder)
	if !LegacyWalletCompatEnabled() || !ok || o == nil {
		return nil
	}
	if dbent.TxFromContext(ctx) == nil {
		return errors.New("legacy payment receipt requires credit transaction")
	}
	_, err := client.ExecContext(ctx, `INSERT INTO referral_first_recharges(user_id,order_id,principal,historical) VALUES($1,$2,$3,TRUE) ON CONFLICT(user_id) DO NOTHING`, o.UserID, o.ID, paymentOrderAmountWithoutBonus(o))
	return err
}
