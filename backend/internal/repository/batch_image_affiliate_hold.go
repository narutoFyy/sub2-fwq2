package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type affiliateTrialBalance struct {
	amount, consumed, expired, reserved float64
	status                              string
	due                                 bool
}

func affiliateTrialBalanceTx(ctx context.Context, q affiliateQueryExecer, userID int64) (*affiliateTrialBalance, error) {
	rows, err := q.QueryContext(ctx, `SELECT trial_bonus_amount::double precision, trial_consumed_amount::double precision,
trial_expired_amount::double precision, trial_bonus_status, COALESCE(trial_expires_at <= NOW(), false),
COALESCE((SELECT SUM(h.trial_reserved_amount) FROM usage_billing_batch_image_holds h WHERE h.user_id = ua.user_id AND h.status = 'reserved'), 0)::double precision
FROM user_affiliates ua WHERE user_id = $1 FOR UPDATE`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	s := &affiliateTrialBalance{}
	if err := rows.Scan(&s.amount, &s.consumed, &s.expired, &s.status, &s.due, &s.reserved); err != nil {
		return nil, err
	}
	return s, rows.Err()
}

func (s *affiliateTrialBalance) available() float64 {
	if s == nil || s.status != "active" || s.due {
		return 0
	}
	return math.Max(0, s.amount-s.consumed-s.expired-s.reserved)
}

type batchImageAffiliateHold struct {
	userID, apiKeyID, inviterID int64
	amount, trial, actual       float64
	status, clientIP            string
}

func getBatchImageAffiliateHoldTx(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*batchImageAffiliateHold, error) {
	h := &batchImageAffiliateHold{}
	err := tx.QueryRowContext(ctx, `SELECT user_id, api_key_id, hold_amount::double precision, trial_reserved_amount::double precision,
status, actual_amount::double precision, client_ip, credited_inviter_id
FROM usage_billing_batch_image_holds WHERE batch_id = $1 FOR UPDATE`, cmd.BatchID).
		Scan(&h.userID, &h.apiKeyID, &h.amount, &h.trial, &h.status, &h.actual, &h.clientIP, &h.inviterID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if h.userID != cmd.UserID || h.apiKeyID != cmd.APIKeyID || math.Abs(h.amount-cmd.HoldAmount) > 1e-8 {
		return nil, fmt.Errorf("batch image hold ownership or amount mismatch: batch=%s", cmd.BatchID)
	}
	return h, nil
}

func replayBatchImageHoldResultTx(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	h, err := getBatchImageAffiliateHoldTx(ctx, tx, cmd)
	if err != nil {
		return nil, err
	}
	result := &service.BatchImageBalanceHoldResult{}
	if h != nil {
		// A capture may commit before its worker updates the job/cache. Preserve
		// the credited inviter on retry, while financial side effects stay deduped.
		result.AffiliateCreditedInviterID = h.inviterID
	}
	return result, nil
}

// All callers hold the user/inviter reward scopes. The row is the authoritative
// reservation, not the user's shared frozen pool or the asynchronous job status.
func applyAffiliateBatchImageHoldTx(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand, operation string) (*service.BatchImageBalanceHoldResult, error) {
	if cmd.UserID <= 0 || cmd.APIKeyID <= 0 || cmd.BatchID == "" || cmd.HoldAmount < 0 || cmd.ActualAmount < 0 ||
		math.IsNaN(cmd.HoldAmount) || math.IsInf(cmd.HoldAmount, 0) || math.IsNaN(cmd.ActualAmount) || math.IsInf(cmd.ActualAmount, 0) {
		return nil, fmt.Errorf("invalid batch image hold command")
	}
	if cmd.ActualAmount-cmd.HoldAmount > 1e-8 {
		return nil, service.ErrBatchImageSettlementCostExceedsHold
	}
	h, err := getBatchImageAffiliateHoldTx(ctx, tx, cmd)
	if err != nil {
		return nil, err
	}
	if h != nil && h.status != "reserved" {
		if operation == "capture" && (h.status != "captured" || math.Abs(h.actual-cmd.ActualAmount) > 1e-8) {
			return nil, fmt.Errorf("batch image hold already %s: batch=%s", h.status, cmd.BatchID)
		}
		// Releasing a captured batch is a safe no-op, never a withdrawal from
		// some other batch's frozen balance.
		return &service.BatchImageBalanceHoldResult{AffiliateCreditedInviterID: h.inviterID}, nil
	}
	if operation == "reserve" && h != nil {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	if cmd.HoldAmount == 0 && cmd.ActualAmount == 0 {
		return &service.BatchImageBalanceHoldResult{Applied: true}, nil
	}
	if operation != "reserve" && h == nil {
		if operation == "release" {
			claimed, err := batchImageHoldClaimExists(ctx, tx, service.BatchImageHoldRequestID(cmd.BatchID), cmd.APIKeyID)
			if err != nil {
				return nil, err
			}
			if !claimed {
				return &service.BatchImageBalanceHoldResult{}, nil
			}
		}
		return nil, fmt.Errorf("batch image hold record missing: batch=%s", cmd.BatchID)
	}

	expired, err := expireUserTrialBonusTx(ctx, tx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	trial, err := affiliateTrialBalanceTx(ctx, tx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	if operation == "reserve" {
		result, err := reserveUsageBillingBatchImageBalance(ctx, tx, cmd)
		if err != nil {
			return nil, err
		}
		reservedTrial := math.Floor(math.Min(cmd.HoldAmount, trial.available())*1e8+1e-7) / 1e8
		cleanIP, _ := service.ComputeIPSubnet(cmd.ClientIP)
		_, err = tx.ExecContext(ctx, `INSERT INTO usage_billing_batch_image_holds
(batch_id, user_id, api_key_id, hold_amount, trial_reserved_amount, client_ip, status)
VALUES ($1, $2, $3, $4, $5, $6, 'reserved')`, cmd.BatchID, cmd.UserID, cmd.APIKeyID, cmd.HoldAmount, reservedTrial, cleanIP)
		if err != nil {
			return nil, err
		}
		result.Applied = true
		result.TrialBonusExpiredAmount = expired
		return result, nil
	}

	actual, usedTrial := 0.0, 0.0
	status := "released"
	if operation == "capture" {
		actual = math.Min(cmd.ActualAmount, h.amount)
		status = "captured"
		if trial != nil && trial.status != "clawed_back" {
			usedTrial = math.Min(actual, h.trial)
		}
	}
	// Refund clawback has already recovered the whole remaining gift from the
	// available wallet, possibly creating debt; do not recover it twice here.
	expiredReturn := 0.0
	if trial != nil && trial.status != "clawed_back" && trial.due {
		expiredReturn = math.Max(0, h.trial-usedTrial)
	}

	financial := *cmd
	financial.HoldAmount = h.amount
	financial.ActualAmount = actual
	var result *service.BatchImageBalanceHoldResult
	if operation == "capture" {
		result, err = captureUsageBillingBatchImageBalance(ctx, tx, &financial)
	} else {
		result, err = releaseRecordedUsageBillingBatchImageBalance(ctx, tx, &financial)
	}
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE usage_billing_batch_image_holds SET status = $1, actual_amount = $2, updated_at = NOW() WHERE batch_id = $3`, status, actual, cmd.BatchID); err != nil {
		return nil, err
	}
	if expiredReturn > 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE users SET balance = balance - $1, updated_at = NOW() WHERE id = $2`, expiredReturn, cmd.UserID); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE user_affiliates SET trial_expired_amount = trial_expired_amount + $1, trial_bonus_status = 'expired', updated_at = NOW() WHERE user_id = $2`, expiredReturn, cmd.UserID); err != nil {
			return nil, err
		}
		if err := recordTrialExpiryLedgerTx(ctx, tx, cmd.UserID, expiredReturn, "batch_image_trial_expire:"+cmd.BatchID); err != nil {
			return nil, err
		}
		if result.NewBalance != nil {
			*result.NewBalance -= expiredReturn
		}
	}
	if operation == "capture" && actual > 0 {
		outcome := &service.AffiliateRewardEvalOutcome{InviteeID: cmd.UserID}
		// Use the trusted IP persisted when the batch was accepted. Workers do
		// not have an HTTP client, and later requests must not replace provenance.
		if err := recordInviteeConsumptionWithTrialTx(ctx, tx, cmd.UserID, actual, h.clientIP, &usedTrial, outcome); err != nil {
			return nil, err
		}
		if outcome.InviterID > 0 && (outcome.Step1Granted > 0 || outcome.Step2Granted > 0 || outcome.TierBonusGranted > 0) {
			result.AffiliateCreditedInviterID = outcome.InviterID
			if _, err := tx.ExecContext(ctx, `UPDATE usage_billing_batch_image_holds SET credited_inviter_id = $1 WHERE batch_id = $2`, outcome.InviterID, cmd.BatchID); err != nil {
				return nil, err
			}
		}
	}
	result.Applied = true
	result.TrialBonusExpiredAmount = expired + expiredReturn
	return result, nil
}

func recordTrialExpiryLedgerTx(ctx context.Context, q affiliateQueryExecer, userID int64, amount float64, operationID string) error {
	snapshot, err := queryAffiliateTransferSnapshot(ctx, q, userID)
	if err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, `INSERT INTO user_affiliate_ledger (user_id, action, amount, operation_id, balance_after,
aff_quota_after, aff_frozen_quota_after, aff_history_quota_after, created_at, updated_at)
VALUES ($1, 'trial_expire', $2, $3, $4, $5, $6, $7, NOW(), NOW())
ON CONFLICT (operation_id) WHERE operation_id IS NOT NULL DO NOTHING`, userID, amount, operationID,
		snapshot.BalanceAfter, snapshot.AvailableQuotaAfter, snapshot.FrozenQuotaAfter, snapshot.HistoryQuotaAfter)
	return err
}
