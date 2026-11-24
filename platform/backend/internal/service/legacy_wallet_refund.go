package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"time"
)

func (s *PaymentService) withSourceRefundTx(ctx context.Context, fn func(context.Context, *dbent.Client) error) error {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return fn(ctx, tx.Client())
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ctx = dbent.NewTxContext(ctx, tx)
	if err = fn(ctx, tx.Client()); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *PaymentService) deductSourceRefund(ctx context.Context, p *RefundPlan) (float64, error) {
	if !LegacyWalletCompatEnabled() {
		return s.deductAvailableBalance(ctx, p.Order.UserID, p.BalanceToDeduct)
	}
	var deducted float64
	err := s.withSourceRefundTx(ctx, func(txCtx context.Context, client *dbent.Client) error {
		if err := lockReferralRefundUsers(txCtx, client, p.Order.UserID); err != nil {
			return err
		}
		// Serialize all user funds before source and request locks.
		rows, err := client.QueryContext(txCtx, "SELECT id FROM users WHERE id=$1 FOR UPDATE", p.Order.UserID)
		if err != nil {
			return err
		}
		for rows.Next() {
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		rowsAttempt, err := client.QueryContext(txCtx, "SELECT deducted,state FROM source_wallet_refund_attempts WHERE order_id=$1 FOR UPDATE", p.OrderID)
		if err != nil {
			return err
		}
		var prior string
		if rowsAttempt.Next() {
			if err = rowsAttempt.Scan(&deducted, &prior); err != nil {
				rowsAttempt.Close()
				return err
			}
		}
		err = rowsAttempt.Err()
		rowsAttempt.Close()
		if err != nil {
			return err
		}
		if prior == "deducted" {
			return nil
		}
		if prior == "completed" {
			return errors.New("legacy refund attempt already completed")
		}
		operation := fmt.Sprintf("refund:%d:%d", p.OrderID, time.Now().UnixNano())
		meta, _ := json.Marshal(map[string]any{"operation": "refund", "order_id": p.OrderID})
		if _, err = client.ExecContext(txCtx, "SELECT set_config('gabugu.wallet_credit',$1,true),set_config('gabugu.wallet_request',$2,true)", string(meta), operation); err != nil {
			return err
		}
		deducted, err = s.deductAvailableBalance(txCtx, p.Order.UserID, p.BalanceToDeduct)
		if err != nil {
			return err
		}
		_, err = client.ExecContext(txCtx, `INSERT INTO source_wallet_refund_attempts(order_id,user_id,operation_id,deducted,state)
   VALUES($1,$2,$3,$4,'deducted') ON CONFLICT(order_id) DO UPDATE SET operation_id=EXCLUDED.operation_id,deducted=EXCLUDED.deducted,state='deducted',updated_at=NOW()`, p.OrderID, p.Order.UserID, operation, deducted)
		return err
	})
	return deducted, err
}
func (s *PaymentService) restoreSourceRefund(ctx context.Context, p *RefundPlan) error {
	if !LegacyWalletCompatEnabled() {
		return s.userRepo.UpdateBalance(ctx, p.Order.UserID, p.BalanceToDeduct)
	}
	return s.withSourceRefundTx(ctx, func(txCtx context.Context, client *dbent.Client) error {
		if err := lockReferralRefundUsers(txCtx, client, p.Order.UserID); err != nil {
			return err
		}
		rows, err := client.QueryContext(txCtx, "SELECT id FROM users WHERE id=$1 FOR UPDATE", p.Order.UserID)
		if err != nil {
			return err
		}
		for rows.Next() {
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		rows, err = client.QueryContext(txCtx, "SELECT operation_id,deducted,state FROM source_wallet_refund_attempts WHERE order_id=$1 FOR UPDATE", p.OrderID)
		if err != nil {
			return err
		}
		if !rows.Next() {
			rows.Close()
			return errors.New("missing source refund attempt")
		}
		var operation, state string
		var amount float64
		if err = rows.Scan(&operation, &amount, &state); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		if state == "restored" {
			return nil
		}
		if state != "deducted" {
			return errors.New("legacy wallet refund attempt is not restorable")
		}
		var restored, revokedReward float64
		sourceRows, err := client.QueryContext(txCtx, `SELECT COALESCE(SUM(e.amount) FILTER(WHERE l.revoked_at IS NULL AND NOT EXISTS(SELECT 1 FROM referral_enrollments r WHERE r.reward_lot_id=l.id AND r.reward_kind='first_recharge')),0),
			COALESCE(SUM(e.amount) FILTER(WHERE l.revoked_at IS NOT NULL AND l.kind='reward' AND NOT EXISTS(SELECT 1 FROM referral_enrollments r WHERE r.reward_lot_id=l.id AND r.reward_kind='first_recharge')),0)
			FROM source_wallet_events e JOIN source_wallet_lots l ON l.id=e.lot_id
			WHERE e.user_id=$1 AND e.request_id=$2 AND e.action='refund_debit'`, p.Order.UserID, operation)
		if err != nil {
			return err
		}
		if !sourceRows.Next() {
			sourceRows.Close()
			return errors.New("refund source restoration query missing")
		}
		err = sourceRows.Scan(&restored, &revokedReward)
		sourceRows.Close()
		if err != nil {
			return err
		}
		if restored > amount {
			return errors.New("refund source restoration exceeds deduction")
		}
		if _, err = client.ExecContext(txCtx, `UPDATE source_wallet_lots l SET remaining=l.remaining+e.amount
   FROM (SELECT lot_id,SUM(amount) amount FROM source_wallet_events WHERE user_id=$1 AND request_id=$2 AND action='refund_debit' GROUP BY lot_id) e WHERE l.id=e.lot_id AND l.revoked_at IS NULL
   AND NOT EXISTS(SELECT 1 FROM referral_enrollments r WHERE r.reward_lot_id=l.id AND r.reward_kind='first_recharge')`, p.Order.UserID, operation); err != nil {
			return err
		}
		firstRows, err := client.QueryContext(txCtx, `SELECT e.lot_id,SUM(e.amount) FROM source_wallet_events e
 JOIN referral_enrollments r ON r.reward_lot_id=e.lot_id AND r.reward_kind='first_recharge'
 WHERE e.user_id=$1 AND e.request_id=$2 AND e.action='refund_debit' GROUP BY e.lot_id ORDER BY e.lot_id`, p.Order.UserID, operation)
		if err != nil {
			return err
		}
		type restorePart struct {
			lot    int64
			amount float64
		}
		var parts []restorePart
		for firstRows.Next() {
			var v restorePart
			if err = firstRows.Scan(&v.lot, &v.amount); err != nil {
				firstRows.Close()
				return err
			}
			parts = append(parts, v)
		}
		err = firstRows.Err()
		firstRows.Close()
		if err != nil {
			return err
		}
		for _, v := range parts {
			rs, e := client.QueryContext(txCtx, "SELECT referral_restore_reward_debit($1,$2)", v.lot, v.amount)
			if e != nil {
				return e
			}
			var back float64
			if !rs.Next() {
				rs.Close()
				return errors.New("missing reward restoration result")
			}
			e = rs.Scan(&back)
			rs.Close()
			if e != nil {
				return e
			}
			restored += back
		}
		if restored > amount {
			return errors.New("refund source restoration exceeds deduction")
		}
		if revokedReward > 0 {
			if _, err = client.ExecContext(txCtx, `UPDATE referral_enrollments r SET unrecouped=GREATEST(unrecouped-e.amount,0)
			FROM(SELECT lot_id,SUM(amount) amount FROM source_wallet_events WHERE user_id=$1 AND request_id=$2 AND action='refund_debit' GROUP BY lot_id)e
			WHERE r.reward_lot_id=e.lot_id AND r.state='revoked'`, p.Order.UserID, operation); err != nil {
				return err
			}
		}
		// Overdraft/unknown residual was never classified as principal. All deducted
		// available funds must be represented by the source events.
		if _, err = client.ExecContext(txCtx, "SELECT set_config('gabugu.wallet_managed','1',true)"); err != nil {
			return err
		}
		if _, err = client.ExecContext(txCtx, "UPDATE users SET balance=balance+$1,updated_at=NOW() WHERE id=$2", restored, p.Order.UserID); err != nil {
			return err
		}
		if _, err = client.ExecContext(txCtx, "SELECT set_config('gabugu.wallet_managed','',true)"); err != nil {
			return err
		}
		_, err = client.ExecContext(txCtx, "UPDATE source_wallet_refund_attempts SET state='restored',updated_at=NOW() WHERE order_id=$1", p.OrderID)
		return err
	})
}
func (s *PaymentService) finalizeSourceRefundTx(ctx context.Context, client *dbent.Client, p *RefundPlan) error {
	if !LegacyWalletCompatEnabled() || !p.DeductBalance || p.DeductionType != "balance" {
		return nil
	}
	if p.Order == nil {
		return errors.New("refund source order missing")
	}
	if err := refundSourceHeldTx(ctx, client, p); err != nil {
		return err
	}
	// Normal refund of unspent funds does not undo already-delivered consumption.
	// Forced/chargeback refunds cap eligible spending at remaining paid principal.
	if _, err := client.ExecContext(ctx, `UPDATE source_wallet_lots SET refunded=GREATEST(refunded,
  LEAST(original_amount,$1::numeric*original_amount/NULLIF($2::numeric,0))) WHERE user_id=$3 AND payment_order_id=$4`, p.RefundAmount, p.Order.Amount, p.Order.UserID, p.OrderID); err != nil {
		return err
	}
	if _, err := client.ExecContext(ctx, "UPDATE source_wallet_refund_attempts SET state='completed',updated_at=NOW() WHERE order_id=$1", p.OrderID); err != nil {
		return err
	}
	return revokeIneligibleReferralTx(ctx, client, p.Order.UserID)
}
func refundSourceHeldTx(ctx context.Context, client *dbent.Client, p *RefundPlan) error {
	// Force refunds can have no available balance because the order's principal
	// is reserved. Remove only the unpaid refund remainder from that order's
	// exact held slices; cancellation can return only their unrefunded remainder.
	var already float64
	rows, err := client.QueryContext(ctx, "SELECT COALESCE(SUM(amount),0) FROM source_wallet_events WHERE user_id=$1 AND request_id=$2 AND action='refund_held'", p.Order.UserID, fmt.Sprintf("refund-final:%d", p.OrderID))
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&already)
	}
	rows.Close()
	if err != nil {
		return err
	}
	need := QuantizeUsageBillingAmount(p.RefundAmount - p.BalanceToDeduct - already)
	if need <= 0 {
		return nil
	}
	rows, err = client.QueryContext(ctx, `SELECT a.user_id,a.request_id,a.lot_id,a.amount-a.refunded
	 FROM source_wallet_reservation_lots a JOIN source_wallet_reservations r USING(user_id,request_id)
	 JOIN source_wallet_lots l ON l.id=a.lot_id WHERE l.user_id=$1 AND l.payment_order_id=$2
	 AND r.state='reserved' AND a.amount>a.refunded ORDER BY l.id,r.request_id FOR UPDATE OF a,l,r`, p.Order.UserID, p.OrderID)
	if err != nil {
		return err
	}
	type slice struct {
		uid, lot int64
		request  string
		amount   float64
	}
	var slices []slice
	for rows.Next() {
		var v slice
		if err = rows.Scan(&v.uid, &v.request, &v.lot, &v.amount); err != nil {
			rows.Close()
			return err
		}
		slices = append(slices, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	removed := 0.0
	for _, v := range slices {
		if need <= 0 {
			break
		}
		take := v.amount
		if take > need {
			take = need
		}
		need = QuantizeUsageBillingAmount(need - take)
		removed += take
		if _, err = client.ExecContext(ctx, "UPDATE source_wallet_reservation_lots SET refunded=refunded+$1 WHERE user_id=$2 AND request_id=$3 AND lot_id=$4", take, v.uid, v.request, v.lot); err != nil {
			return err
		}
		if _, err = client.ExecContext(ctx, "UPDATE source_wallet_lots SET held=held-$1 WHERE id=$2", take, v.lot); err != nil {
			return err
		}
		if _, err = client.ExecContext(ctx, "INSERT INTO source_wallet_events(user_id,request_id,action,lot_id,amount) VALUES($1,$2,'refund_held',$3,$4)", p.Order.UserID, fmt.Sprintf("refund-final:%d", p.OrderID), v.lot, take); err != nil {
			return err
		}
	}
	// The provider may return after a cancellation released the reservation.
	// Still remove the same order's now-available funds under the user lock.
	availableRemoved := 0.0
	if need > 0 {
		rows, err = client.QueryContext(ctx, "SELECT id,remaining FROM source_wallet_lots WHERE user_id=$1 AND payment_order_id=$2 AND remaining>0 ORDER BY id FOR UPDATE", p.Order.UserID, p.OrderID)
		if err != nil {
			return err
		}
		type availablePart struct {
			id     int64
			amount float64
		}
		var available []availablePart
		for rows.Next() {
			var v availablePart
			if err = rows.Scan(&v.id, &v.amount); err != nil {
				rows.Close()
				return err
			}
			available = append(available, v)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, v := range available {
			if need <= 0 {
				break
			}
			take := v.amount
			if take > need {
				take = need
			}
			need = QuantizeUsageBillingAmount(need - take)
			availableRemoved += take
			if _, err = client.ExecContext(ctx, "UPDATE source_wallet_lots SET remaining=remaining-$1 WHERE id=$2", take, v.id); err != nil {
				return err
			}
			if _, err = client.ExecContext(ctx, "INSERT INTO source_wallet_events(user_id,request_id,action,lot_id,amount) VALUES($1,$2,'refund_held',$3,$4)", p.Order.UserID, fmt.Sprintf("refund-final:%d", p.OrderID), v.id, take); err != nil {
				return err
			}
		}
	}
	if removed > 0 || availableRemoved > 0 {
		if _, err = client.ExecContext(ctx, "SELECT set_config('gabugu.wallet_managed','1',true)"); err != nil {
			return err
		}
		if _, err = client.ExecContext(ctx, "UPDATE users SET frozen_balance=frozen_balance-$1,balance=balance-$2 WHERE id=$3", QuantizeUsageBillingAmount(removed), QuantizeUsageBillingAmount(availableRemoved), p.Order.UserID); err != nil {
			return err
		}
	}
	if _, err = client.ExecContext(ctx, "SELECT set_config('gabugu.wallet_managed','',true)"); err != nil {
		return err
	}
	return nil
}
func lockReferralRefundUsers(ctx context.Context, client *dbent.Client, uid int64) error {
	rows, err := client.QueryContext(ctx, `SELECT id FROM users WHERE id=$1 OR id=(SELECT inviter_user_id FROM referral_enrollments WHERE invitee_user_id=$1) ORDER BY id FOR UPDATE`, uid)
	if err != nil {
		return err
	}
	locked := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		locked[id] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	// A binding that committed while this statement waited on the invitee row
	// was not visible in its initial snapshot. Roll back and retry, never acquire
	// its inviter out of lock order or grant under a partially locked owner set.
	rows, err = client.QueryContext(ctx, "SELECT inviter_user_id FROM referral_enrollments WHERE invitee_user_id=$1", uid)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		var inviter int64
		if err = rows.Scan(&inviter); err != nil {
			return err
		}
		if !locked[inviter] {
			return errors.New("referral binding changed while locking wallets; retry transaction")
		}
	}
	return rows.Err()
}
func revokeIneligibleReferralTx(ctx context.Context, client *dbent.Client, uid int64) error {
	// New first-recharge rewards are proportional and recoup consumed rewards as
	// an immediate source debt. The legacy consumption promise remains unchanged.
	if _, err := client.ExecContext(ctx, "SELECT referral_adjust_first_recharge($1)", uid); err != nil {
		return err
	}
	rows, err := client.QueryContext(ctx, `SELECT inviter_user_id,state,threshold_usd,reward_usd,reward_lot_id,referral_principal_spent(invitee_user_id)
 FROM referral_enrollments WHERE invitee_user_id=$1 AND reward_kind='consumption' FOR UPDATE`, uid)
	if err != nil {
		return err
	}
	if !rows.Next() {
		rows.Close()
		return nil
	}
	var inviter, lot int64
	var state string
	var threshold, reward, paid float64
	var lotptr *int64
	if err = rows.Scan(&inviter, &state, &threshold, &reward, &lotptr, &paid); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if paid >= threshold || state != "granted" {
		return nil
	}
	if lotptr == nil {
		return errors.New("granted reward source missing")
	}
	lot = *lotptr
	var remaining float64
	rs, err := client.QueryContext(ctx, "SELECT remaining FROM source_wallet_lots WHERE id=$1 FOR UPDATE", lot)
	if err != nil {
		return err
	}
	if !rs.Next() {
		rs.Close()
		return errors.New("reward lot missing")
	}
	err = rs.Scan(&remaining)
	rs.Close()
	if err != nil {
		return err
	}
	if _, err = client.ExecContext(ctx, "UPDATE source_wallet_lots SET remaining=0,revoked_at=NOW() WHERE id=$1", lot); err != nil {
		return err
	}
	if _, err = client.ExecContext(ctx, "SELECT set_config('gabugu.wallet_managed','1',true)"); err != nil {
		return err
	}
	if _, err = client.ExecContext(ctx, "UPDATE users SET balance=balance-$1,updated_at=NOW() WHERE id=$2", remaining, inviter); err != nil {
		return err
	}
	if _, err = client.ExecContext(ctx, "SELECT set_config('gabugu.wallet_managed','',true)"); err != nil {
		return err
	}
	if _, err = client.ExecContext(ctx, "UPDATE referral_enrollments SET state='revoked',unrecouped=GREATEST($1::numeric-$2::numeric,0) WHERE invitee_user_id=$3", reward, remaining, uid); err != nil {
		return err
	}
	_, err = client.ExecContext(ctx, "INSERT INTO referral_operations(operation_id,invitee_user_id,action,amount) VALUES($1,$2,'revoke',$3) ON CONFLICT DO NOTHING", fmt.Sprintf("revoke:%d", uid), uid, remaining)
	return err
}
