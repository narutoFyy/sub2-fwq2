package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func qualifiedAffiliateTotalsTx(ctx context.Context, q affiliateQueryExecer, inviterID int64) (int, float64, error) {
	rows, err := q.QueryContext(ctx, `SELECT COUNT(*)::integer, COALESCE(SUM(total_paid_amount), 0)::double precision
FROM user_affiliates WHERE inviter_id = $1 AND qualified_at IS NOT NULL
AND qualified_at <= created_at + INTERVAL '30 days' AND total_paid_amount >= $2 AND total_consumed_amount >= $3`, inviterID, service.AffiliateStep1MinPaid, service.AffiliateMinConsumed)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()
	var count int
	var paid float64
	if rows.Next() {
		if err := rows.Scan(&count, &paid); err != nil {
			return 0, 0, err
		}
	}
	return count, paid, rows.Err()
}

func lockAffiliateGrantScopeTx(ctx context.Context, q affiliateQueryExecer, grantID int64) error {
	rows, err := q.QueryContext(ctx, `SELECT inviter_id, invitee_id FROM affiliate_reward_grants WHERE id = $1`, grantID)
	if err != nil {
		return err
	}
	var inviterID int64
	var inviteeID sql.NullInt64
	if !rows.Next() {
		_ = rows.Close()
		return service.ErrAffiliateRewardGrantNotFound
	}
	if err := rows.Scan(&inviterID, &inviteeID); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	return lockAffiliateUsersTx(ctx, q, inviterID, inviteeID.Int64)
}

func affiliateGrantStillEligibleTx(ctx context.Context, q affiliateQueryExecer, grantID, inviterID int64, inviteeID sql.NullInt64, rewardType string, tierTarget int) (bool, error) {
	if rewardType == "step1" || rewardType == "step2" {
		if !inviteeID.Valid {
			return false, nil
		}
		threshold := service.AffiliateStep1MinPaid
		if rewardType == "step2" {
			threshold = service.AffiliateStep2MinPaid
		}
		rows, err := q.QueryContext(ctx, `SELECT EXISTS (
SELECT 1 FROM user_affiliates ua JOIN affiliate_reward_grants g ON g.id = $1
WHERE ua.user_id = $2 AND ua.inviter_id = $3 AND ua.total_paid_amount >= $4 AND ua.total_consumed_amount >= $5
AND ua.qualified_at IS NOT NULL AND ua.qualified_at <= ua.created_at + INTERVAL '30 days'
AND g.created_at <= ua.created_at + INTERVAL '30 days')`, grantID, inviteeID.Int64, inviterID, threshold, service.AffiliateMinConsumed)
		if err != nil {
			return false, err
		}
		defer rows.Close()
		var eligible bool
		if rows.Next() {
			if err := rows.Scan(&eligible); err != nil {
				return false, err
			}
		}
		return eligible, rows.Err()
	}
	count, paid, err := qualifiedAffiliateTotalsTx(ctx, q, inviterID)
	if err != nil {
		return false, err
	}
	for _, tier := range service.DefaultAffiliateLadderTiers() {
		if tier.TierTarget == tierTarget && rewardType == fmt.Sprintf("tier_%d", tierTarget) {
			return count >= tierTarget && paid >= tier.RequiresMinTotalPaid, nil
		}
	}
	return false, nil
}

// revokeAffiliateGrantTx cancels pending awards and recovers granted awards exactly once.
func revokeAffiliateGrantTx(ctx context.Context, q affiliateQueryExecer, operationID string, orderID int64) error {
	rows, err := q.QueryContext(ctx, `SELECT inviter_id, invitee_id, reward_type, amount::double precision, status
FROM affiliate_reward_grants WHERE operation_id = $1 FOR UPDATE`, operationID)
	if err != nil {
		return err
	}
	if !rows.Next() {
		return rows.Close()
	}
	var inviterID int64
	var inviteeID sql.NullInt64
	var rewardType, status string
	var amount float64
	if err := rows.Scan(&inviterID, &inviteeID, &rewardType, &amount, &status); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if status != "granted" && status != "pending_review" {
		return nil
	}
	if _, err := q.ExecContext(ctx, `UPDATE affiliate_reward_grants SET status = 'clawed_back', review_reason = 'refund_no_longer_eligible', updated_at = NOW() WHERE operation_id = $1`, operationID); err != nil {
		return err
	}
	if status != "granted" {
		return nil
	}
	if _, err := q.ExecContext(ctx, `UPDATE users SET balance = balance - $1, updated_at = NOW() WHERE id = $2`, amount, inviterID); err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, `INSERT INTO user_affiliate_ledger (user_id, action, amount, source_user_id, source_order_id, operation_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`, inviterID, "clawback_"+rewardType, -amount, nullableInt64Arg(nullableInt64Ptr(inviteeID)), orderID, "reward_clawback:"+operationID)
	return err
}

// ClawbackOrderAffiliateRewards accepts the cumulative refunded paid amount for an order.
// An order watermark makes retry and incremental partial refunds idempotent in one transaction.
func (r *affiliateRepository) ClawbackOrderAffiliateRewards(ctx context.Context, orderID, inviteeID int64, refundedPaidAmount float64) ([]int64, error) {
	if orderID <= 0 || inviteeID <= 0 || refundedPaidAmount <= 0 || math.IsNaN(refundedPaidAmount) || math.IsInf(refundedPaidAmount, 0) {
		return nil, nil
	}
	affectedUsers := make([]int64, 0, 2)
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if err := lockAffiliateRewardScopeTx(txCtx, txClient, inviteeID); err != nil {
			return err
		}
		orderRows, err := txClient.QueryContext(txCtx, `SELECT GREATEST(amount - CASE WHEN order_type = 'balance' THEN COALESCE(bonus_amount, 0) ELSE 0 END, 0)::double precision,
COALESCE(affiliate_refunded_paid_amount, 0)::double precision FROM payment_orders WHERE id = $1 AND user_id = $2 FOR UPDATE`, orderID, inviteeID)
		if err != nil {
			return err
		}
		if !orderRows.Next() {
			_ = orderRows.Close()
			return fmt.Errorf("affiliate refund order %d not found for user %d", orderID, inviteeID)
		}
		var originalPaid, alreadyRefunded float64
		if err := orderRows.Scan(&originalPaid, &alreadyRefunded); err != nil {
			_ = orderRows.Close()
			return err
		}
		if err := orderRows.Close(); err != nil {
			return err
		}
		cumulative := math.Round(math.Min(refundedPaidAmount, originalPaid)*1e8) / 1e8
		deltaPaid := math.Round((cumulative-alreadyRefunded)*1e8) / 1e8
		if originalPaid <= 0 || deltaPaid <= 0 {
			return nil
		}
		if _, err := txClient.ExecContext(txCtx, `UPDATE payment_orders SET affiliate_refunded_paid_amount = $1 WHERE id = $2`, cumulative, orderID); err != nil {
			return err
		}
		rows, err := txClient.QueryContext(txCtx, `SELECT inviter_id, total_paid_amount::double precision, trial_bonus_amount::double precision,
trial_expired_amount::double precision, trial_bonus_status FROM user_affiliates WHERE user_id = $1 FOR UPDATE`, inviteeID)
		if err != nil {
			return err
		}
		if !rows.Next() {
			return rows.Close()
		}
		var inviterID sql.NullInt64
		var totalPaid, trialAmount, trialExpired float64
		var trialStatus string
		if err := rows.Scan(&inviterID, &totalPaid, &trialAmount, &trialExpired, &trialStatus); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		newPaid := math.Max(0, totalPaid-deltaPaid)
		if _, err := txClient.ExecContext(txCtx, `UPDATE user_affiliates SET total_paid_amount = $1,
qualified_at = CASE WHEN $1::numeric < $3::numeric THEN NULL ELSE qualified_at END,
updated_at = NOW() WHERE user_id = $2`, newPaid, inviteeID, service.AffiliateStep1MinPaid); err != nil {
			return err
		}
		affectedUsers = append(affectedUsers, inviteeID)
		// Abnormal refunds recover the invitee gift too, including already-used gift value.
		// Previously expired value was already removed and must not be deducted twice.
		if trialAmount > 0 && trialStatus != "clawed_back" {
			reclaim := math.Max(0, trialAmount-trialExpired)
			if reclaim > 0 {
				if _, err := txClient.ExecContext(txCtx, `UPDATE users SET balance = balance - $1, updated_at = NOW() WHERE id = $2`, reclaim, inviteeID); err != nil {
					return err
				}
				if _, err := txClient.ExecContext(txCtx, `INSERT INTO user_affiliate_ledger (user_id, action, amount, source_order_id, operation_id, created_at, updated_at)
VALUES ($1, 'clawback_trial', $2, $3, $4, NOW(), NOW())`, inviteeID, -reclaim, orderID, fmt.Sprintf("trial_clawback:%d", inviteeID)); err != nil {
					return err
				}
			}
			if _, err := txClient.ExecContext(txCtx, `UPDATE user_affiliates SET trial_bonus_status = 'clawed_back', updated_at = NOW() WHERE user_id = $1`, inviteeID); err != nil {
				return err
			}
		}
		if !inviterID.Valid || inviterID.Int64 <= 0 {
			return nil
		}
		affectedUsers = append(affectedUsers, inviterID.Int64)
		// Thaw under the same affiliate-row lock before deciding which wallet pays the refund.
		if _, err := thawFrozenQuotaTx(txCtx, txClient, inviterID.Int64); err != nil {
			return err
		}
		rebateRows, err := txClient.QueryContext(txCtx, `SELECT id, amount::double precision, frozen_until FROM user_affiliate_ledger
WHERE user_id = $1 AND source_order_id = $2 AND action = 'accrue' ORDER BY id FOR UPDATE`, inviterID.Int64, orderID)
		if err != nil {
			return err
		}
		type rebateEntry struct {
			id          int64
			amount      float64
			frozenUntil sql.NullTime
		}
		var entries []rebateEntry
		for rebateRows.Next() {
			var e rebateEntry
			if err := rebateRows.Scan(&e.id, &e.amount, &e.frozenUntil); err != nil {
				_ = rebateRows.Close()
				return err
			}
			entries = append(entries, e)
		}
		if err := rebateRows.Close(); err != nil {
			return err
		}
		for _, entry := range entries {
			// Subtract rounded cumulative targets, avoiding drift across many small refunds.
			claw := math.Round(entry.amount*cumulative/originalPaid*1e8)/1e8 - math.Round(entry.amount*alreadyRefunded/originalPaid*1e8)/1e8
			claw = math.Round(claw*1e8) / 1e8
			if claw <= 0 {
				continue
			}
			opID := fmt.Sprintf("clawback_rebate:%d:%d:%.8f", orderID, entry.id, cumulative)
			var frozenUntil any
			if entry.frozenUntil.Valid {
				frozenUntil = entry.frozenUntil.Time
			}
			if _, err := txClient.ExecContext(txCtx, `INSERT INTO user_affiliate_ledger
(user_id, action, amount, source_user_id, source_order_id, operation_id, frozen_until, created_at, updated_at)
VALUES ($1, 'clawback_rebate', $2, $3, $4, $5, $6, NOW(), NOW())`, inviterID.Int64, -claw, inviteeID, orderID, opID, frozenUntil); err != nil {
				return err
			}
			if entry.frozenUntil.Valid {
				if _, err := txClient.ExecContext(txCtx, `UPDATE user_affiliates SET aff_frozen_quota = GREATEST(aff_frozen_quota - $1, 0),
aff_history_quota = GREATEST(aff_history_quota - $1, 0), updated_at = NOW() WHERE user_id = $2`, claw, inviterID.Int64); err != nil {
					return err
				}
			} else {
				quotaRows, err := txClient.QueryContext(txCtx, `SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1 FOR UPDATE`, inviterID.Int64)
				if err != nil {
					return err
				}
				var quota float64
				if quotaRows.Next() {
					if err := quotaRows.Scan(&quota); err != nil {
						_ = quotaRows.Close()
						return err
					}
				}
				if err := quotaRows.Close(); err != nil {
					return err
				}
				fromQuota := math.Min(math.Max(0, quota), claw)
				if _, err := txClient.ExecContext(txCtx, `UPDATE user_affiliates SET aff_quota = GREATEST(aff_quota - $1, 0), aff_history_quota = GREATEST(aff_history_quota - $2, 0), updated_at = NOW() WHERE user_id = $3`, fromQuota, claw, inviterID.Int64); err != nil {
					return err
				}
				if claw > fromQuota {
					if _, err := txClient.ExecContext(txCtx, `UPDATE users SET balance = balance - $1, updated_at = NOW() WHERE id = $2`, claw-fromQuota, inviterID.Int64); err != nil {
						return err
					}
				}
			}
		}
		for _, step := range []struct {
			name    string
			minPaid float64
		}{{"step1", service.AffiliateStep1MinPaid}, {"step2", service.AffiliateStep2MinPaid}} {
			if newPaid >= step.minPaid {
				continue
			}
			if err := revokeAffiliateGrantTx(txCtx, txClient, fmt.Sprintf("%s:%d", step.name, inviteeID), orderID); err != nil {
				return err
			}
			// Includes pending_consume, which has no grant row yet.
			if _, err := txClient.ExecContext(txCtx, `UPDATE user_affiliates SET `+step.name+`_reward_status = CASE WHEN `+step.name+`_reward_status IN ('granted','pending_review','pending_consume') THEN 'clawed_back' ELSE `+step.name+`_reward_status END, updated_at = NOW() WHERE user_id = $1`, inviteeID); err != nil {
				return err
			}
		}
		count, paid, err := qualifiedAffiliateTotalsTx(txCtx, txClient, inviterID.Int64)
		if err != nil {
			return err
		}
		for _, tier := range service.DefaultAffiliateLadderTiers() {
			if count >= tier.TierTarget && paid >= tier.RequiresMinTotalPaid {
				continue
			}
			if err := revokeAffiliateGrantTx(txCtx, txClient, fmt.Sprintf("tier:%d:%d", inviterID.Int64, tier.TierTarget), orderID); err != nil {
				return err
			}
		}
		return nil
	})
	return affectedUsers, err
}
