package repository

import (
	"context"
	"database/sql"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// RecordInviteeConsumption is also used by non-billing callers. Unified usage billing
// calls recordInviteeConsumptionTx inside its own transaction and idempotency claim.
func (r *affiliateRepository) RecordInviteeConsumption(ctx context.Context, inviteeID int64, consumedAmount float64, clientIP string) (*service.AffiliateRewardEvalOutcome, error) {
	if inviteeID <= 0 || consumedAmount <= 0 {
		return nil, nil
	}
	outcome := &service.AffiliateRewardEvalOutcome{InviteeID: inviteeID}
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if err := lockAffiliateRewardScopeTx(txCtx, txClient, inviteeID); err != nil {
			return err
		}
		if _, err := expireUserTrialBonusTx(txCtx, txClient, inviteeID); err != nil {
			return err
		}
		return recordInviteeConsumptionTx(txCtx, txClient, inviteeID, consumedAmount, clientIP, outcome)
	})
	if err != nil {
		return nil, err
	}
	return outcome, nil
}

// Caller must acquire the affiliate reward scope before balance/affiliate row locks.
func recordInviteeConsumptionTx(txCtx context.Context, txClient affiliateQueryExecer, inviteeID int64, consumedAmount float64, clientIP string, outcome *service.AffiliateRewardEvalOutcome) error {
	clientIP, _ = service.ComputeIPSubnet(clientIP)
	rows, err := txClient.QueryContext(txCtx, `
SELECT inviter_id,
       COALESCE(total_paid_amount, 0)::double precision,
       COALESCE(total_consumed_amount, 0)::double precision,
       COALESCE(trial_bonus_status, 'none'),
       COALESCE(trial_bonus_amount, 0)::double precision,
       COALESCE(step1_reward_status, 'none'),
       COALESCE(step2_reward_status, 'none')
FROM user_affiliates
WHERE user_id = $1
FOR UPDATE`, inviteeID)
	if err != nil {
		return fmt.Errorf("lock user_affiliates on consumption: %w", err)
	}
	if !rows.Next() {
		_ = rows.Close()
		return nil
	}
	var inviterID sql.NullInt64
	var totalPaid, totalConsumed, trialBonusAmount float64
	var trialStatus, step1Status, step2Status string
	if err := rows.Scan(
		&inviterID,
		&totalPaid,
		&totalConsumed,
		&trialStatus,
		&trialBonusAmount,
		&step1Status,
		&step2Status,
	); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	newConsumed := totalConsumed + consumedAmount
	newTrialStatus := trialStatus
	if trialStatus == "active" && newConsumed >= trialBonusAmount {
		newTrialStatus = "used"
	}

	if _, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET total_consumed_amount = total_consumed_amount + $1,
    trial_bonus_status = $2,
    last_active_ip = CASE WHEN $3 <> '' THEN $3 ELSE last_active_ip END,
    updated_at = NOW()
WHERE user_id = $4`, consumedAmount, newTrialStatus, clientIP, inviteeID); err != nil {
		return fmt.Errorf("update invitee consumption: %w", err)
	}

	if !inviterID.Valid || inviterID.Int64 <= 0 {
		return nil
	}
	if totalPaid < service.AffiliateStep1MinPaid && step1Status != "pending_consume" && step2Status != "pending_consume" {
		return nil
	}
	outcome.InviterID = inviterID.Int64
	return evaluateInviteeAndLadderRewardsTx(txCtx, txClient, inviteeID, outcome)
}
