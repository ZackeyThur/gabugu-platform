package service

// Transaction-only routines from the deployed Gabugu wallet. No grant workers or old pricing are enabled.
import (
	"context"
	"database/sql"
	"errors"
	"math"
)

var (
	ErrSourceWalletInsufficient   = errors.New("insufficient spendable wallet balance")
	ErrSourceWalletConflict       = errors.New("wallet request fingerprint conflict")
	ErrSourceWalletBudgetExceeded = errors.New("actual usage exceeds reserved budget")
	ErrSourceWalletState          = errors.New("wallet reservation cannot change terminal state")
)

type SourceWalletReserveInput struct {
	UserID                 int64
	RequestID, Fingerprint string
	MaxAmountUSD           float64
}
type SourceWalletResult struct {
	ReservationID       string  `json:"reservation_id"`
	Status              string  `json:"status"`
	AvailableBalanceUSD float64 `json:"available_balance_usd"`
	ReservedUSD         float64 `json:"reserved_usd"`
	ActualUSD           float64 `json:"actual_usd"`
	RewardUSD           float64 `json:"reward_usd"`
	CashUSD             float64 `json:"cash_usd"`
	LegacyUSD           float64 `json:"legacy_usd"`
}

func validWalletAmount(v float64) bool {
	return v >= 0 && !math.IsNaN(v) && !math.IsInf(v, 0) && v < 1e10
}
func ReserveSourceWalletTx(ctx context.Context, tx *sql.Tx, in SourceWalletReserveInput) (*SourceWalletResult, error) {
	var balance float64
	if err := tx.QueryRowContext(ctx, `SELECT balance FROM users WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, in.UserID).Scan(&balance); err != nil {
		return nil, err
	}
	var fp, state string
	var max, actual sql.NullFloat64
	err := tx.QueryRowContext(ctx, `SELECT fingerprint,state,maximum,actual FROM source_wallet_reservations WHERE user_id=$1 AND request_id=$2`, in.UserID, in.RequestID).Scan(&fp, &state, &max, &actual)
	if err == nil {
		if fp != in.Fingerprint || max.Float64 != QuantizeUsageBillingAmount(in.MaxAmountUSD) {
			return nil, ErrSourceWalletConflict
		}
		return walletResultTx(ctx, tx, in.UserID, in.RequestID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	amount := QuantizeUsageBillingAmount(in.MaxAmountUSD)
	if amount <= 0 {
		return nil, errors.New("reserve amount is below wallet precision")
	}
	if balance < amount {
		return nil, ErrSourceWalletInsufficient
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO source_wallet_reservations(user_id,request_id,fingerprint,maximum,state) VALUES($1,$2,$3,$4,'reserved')`, in.UserID, in.RequestID, in.Fingerprint, amount); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,remaining FROM source_wallet_lots WHERE user_id=$1 AND remaining>0 AND revoked_at IS NULL
 ORDER BY CASE WHEN kind IN('gift','reward') THEN 0 WHEN kind='cash' THEN 1 ELSE 2 END,expires_at ASC NULLS LAST,id FOR UPDATE`, in.UserID)
	if err != nil {
		return nil, err
	}
	type part struct {
		id  int64
		amt float64
	}
	var parts []part
	need := amount
	for rows.Next() {
		var id int64
		var a float64
		if err = rows.Scan(&id, &a); err != nil {
			rows.Close()
			return nil, err
		}
		take := math.Min(a, need)
		if take > 0 {
			parts = append(parts, part{id, take})
			need = QuantizeUsageBillingAmount(need - take)
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if need > 0 {
		return nil, errors.New("wallet source mismatch")
	}
	for _, p := range parts {
		if _, err = tx.ExecContext(ctx, `UPDATE source_wallet_lots SET remaining=remaining-$1,held=held+$1 WHERE id=$2`, p.amt, p.id); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO source_wallet_reservation_lots(user_id,request_id,lot_id,amount) VALUES($1,$2,$3,$4)`, in.UserID, in.RequestID, p.id, p.amt); err != nil {
			return nil, err
		}
	}
	if err = managedWalletTx(ctx, tx); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE users SET balance=balance-$1,frozen_balance=COALESCE(frozen_balance,0)+$1,updated_at=NOW() WHERE id=$2`, amount, in.UserID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "SELECT set_config('gabugu.wallet_managed','',true)"); err != nil {
		return nil, err
	}
	return walletResultTx(ctx, tx, in.UserID, in.RequestID)
}
func managedWalletTx(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `SELECT set_config('gabugu.wallet_managed','1',true)`)
	return err
}
func walletResultTx(ctx context.Context, tx *sql.Tx, uid int64, id string) (*SourceWalletResult, error) {
	out := &SourceWalletResult{ReservationID: id}
	err := tx.QueryRowContext(ctx, `SELECT r.state,u.balance,CASE WHEN r.state='reserved' THEN r.maximum ELSE 0 END,COALESCE(r.actual,0),
 COALESCE(SUM(CASE WHEN l.kind IN('gift','reward') THEN CASE WHEN r.state='reserved' THEN p.amount ELSE p.charged END ELSE 0 END),0),
 COALESCE(SUM(CASE WHEN l.kind='cash' THEN CASE WHEN r.state='reserved' THEN p.amount ELSE p.charged END ELSE 0 END),0),
 COALESCE(SUM(CASE WHEN l.kind='legacy' THEN CASE WHEN r.state='reserved' THEN p.amount ELSE p.charged END ELSE 0 END),0)
 FROM source_wallet_reservations r JOIN users u ON u.id=r.user_id
 LEFT JOIN source_wallet_reservation_lots p ON p.user_id=r.user_id AND p.request_id=r.request_id
 LEFT JOIN source_wallet_lots l ON l.id=p.lot_id WHERE r.user_id=$1 AND r.request_id=$2 GROUP BY r.user_id,r.request_id,u.id`, uid, id).
		Scan(&out.Status, &out.AvailableBalanceUSD, &out.ReservedUSD, &out.ActualUSD, &out.RewardUSD, &out.CashUSD, &out.LegacyUSD)
	return out, err
}
func SettleSourceWalletTx(ctx context.Context, tx *sql.Tx, uid int64, id string, actual float64, fp string, release bool) (*SourceWalletResult, error) {
	if !validWalletAmount(actual) {
		return nil, errors.New("invalid actual wallet amount")
	}
	actual = QuantizeUsageBillingAmount(actual)
	var balance float64
	if err := tx.QueryRowContext(ctx, `SELECT balance FROM users WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, uid).Scan(&balance); err != nil {
		return nil, err
	}
	var storedFP, state string
	var maximum float64
	var previous sql.NullFloat64
	if err := tx.QueryRowContext(ctx, `SELECT fingerprint,state,maximum,actual FROM source_wallet_reservations WHERE user_id=$1 AND request_id=$2 FOR UPDATE`, uid, id).Scan(&storedFP, &state, &maximum, &previous); err != nil {
		return nil, err
	}
	if storedFP != fp {
		return nil, ErrSourceWalletConflict
	}
	terminal := "settled"
	if release {
		terminal = "released"
	}
	if state != "reserved" {
		if state != terminal || (!release && previous.Float64 != actual) {
			return nil, ErrSourceWalletState
		}
		return walletResultTx(ctx, tx, uid, id)
	}
	if actual > maximum {
		return nil, ErrSourceWalletBudgetExceeded
	}
	rows, err := tx.QueryContext(ctx, `SELECT p.lot_id,p.amount-p.refunded,l.kind,l.revoked_at FROM source_wallet_reservation_lots p JOIN source_wallet_lots l ON l.id=p.lot_id
 WHERE p.user_id=$1 AND p.request_id=$2 ORDER BY CASE WHEN l.kind IN('gift','reward') THEN 0 WHEN l.kind='cash' THEN 1 ELSE 2 END,l.expires_at ASC NULLS LAST,l.id FOR UPDATE OF l,p`, uid, id)
	if err != nil {
		return nil, err
	}
	type part struct {
		id      int64
		amt     float64
		kind    string
		revoked sql.NullTime
	}
	var parts []part
	for rows.Next() {
		var p part
		if err = rows.Scan(&p.id, &p.amt, &p.kind, &p.revoked); err != nil {
			rows.Close()
			return nil, err
		}
		parts = append(parts, p)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	availableHold := 0.0
	for _, p := range parts {
		availableHold += p.amt
	}
	if actual > QuantizeUsageBillingAmount(availableHold) {
		return nil, ErrSourceWalletBudgetExceeded
	}
	need := actual
	removed := 0.0
	for _, p := range parts {
		charge := math.Min(need, p.amt)
		back := QuantizeUsageBillingAmount(p.amt - charge)
		need = QuantizeUsageBillingAmount(need - charge)
		if p.revoked.Valid {
			removed += back
			back = 0
		}
		invalid := 0.0
		if p.kind == "cash" && p.revoked.Valid {
			invalid = charge
		}
		if _, err = tx.ExecContext(ctx, `UPDATE source_wallet_lots SET held=held-$1,remaining=remaining+$2,spent=spent+$3,invalid_spent=invalid_spent+$4 WHERE id=$5`, p.amt, back, charge, invalid, p.id); err != nil {
			return nil, err
		}
		if p.revoked.Valid && p.kind == "reward" {
			if _, err = tx.ExecContext(ctx, `UPDATE referral_enrollments e SET unrecouped=l.spent+l.held FROM source_wallet_lots l WHERE e.reward_lot_id=l.id AND l.id=$1 AND e.state='revoked'`, p.id); err != nil {
				return nil, err
			}
		}
		if _, err = tx.ExecContext(ctx, `UPDATE source_wallet_reservation_lots SET charged=$1 WHERE user_id=$2 AND request_id=$3 AND lot_id=$4`, charge, uid, id, p.id); err != nil {
			return nil, err
		}
		if charge > 0 {
			if _, err = tx.ExecContext(ctx, `INSERT INTO source_wallet_events(user_id,request_id,action,lot_id,amount,cash_qualifying) VALUES($1,$2,'settle',$3,$4,$5)`, uid, id, p.id, charge, p.kind == "cash" && !p.revoked.Valid); err != nil {
				return nil, err
			}
		}
	}
	if need > 0 {
		return nil, errors.New("wallet reservation source mismatch")
	}
	if err = managedWalletTx(ctx, tx); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE users SET balance=balance+$1,frozen_balance=frozen_balance-$2,updated_at=NOW() WHERE id=$3`, QuantizeUsageBillingAmount(availableHold-actual-removed), availableHold, uid); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE source_wallet_reservations SET state=$1,actual=$2,settled_at=NOW() WHERE user_id=$3 AND request_id=$4`, terminal, actual, uid, id); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "SELECT set_config('gabugu.wallet_managed','',true)"); err != nil {
		return nil, err
	}
	return walletResultTx(ctx, tx, uid, id)
}
