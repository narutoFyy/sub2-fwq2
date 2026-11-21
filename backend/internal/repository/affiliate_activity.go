package repository

import (
	"context"
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// PrepareAffiliateUserForRequest reads the authoritative balance after expiring
// trial credit, and records activity even for an older, uninvited inviter.
func (r *userRepository) PrepareAffiliateUserForRequest(ctx context.Context, userID int64, clientIP string) (*service.AffiliateUserAdmission, error) {
	cleanIP, _ := service.ComputeIPSubnet(clientIP)
	result := &service.AffiliateUserAdmission{}
	// The common case is a single indexed read. Only first activity, IP changes
	// or an actually due expiry require a transaction and reward-scope lock.
	q := clientFromContext(ctx, r.client)
	rows, err := q.QueryContext(ctx, `SELECT u.balance::double precision, ua.user_id,
        COALESCE(ua.last_active_ip, ''),
        COALESCE(ua.trial_bonus_status = 'active' AND ua.trial_expires_at <= NOW(), false),
        COALESCE(ua.trial_bonus_status = 'expired' AND ua.trial_expired_amount > 0, false)
        FROM users u LEFT JOIN user_affiliates ua ON ua.user_id = u.id
        WHERE u.id = $1 AND u.deleted_at IS NULL`, userID)
	if err != nil {
		return nil, err
	}
	if !rows.Next() {
		err := rows.Err()
		_ = rows.Close()
		if err != nil {
			return nil, err
		}
		return nil, service.ErrUserNotFound
	}
	var affiliateID sql.NullInt64
	var lastIP string
	var expiryDue bool
	if err := rows.Scan(&result.Balance, &affiliateID, &lastIP, &expiryDue, &result.TrialExpired); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if affiliateID.Valid && !expiryDue && (cleanIP == "" || cleanIP == lastIP) {
		return result, nil
	}
	repo := &affiliateRepository{client: r.client}
	err = repo.withTx(ctx, func(txCtx context.Context, q *dbent.Client) error {
		if err := lockAffiliateRewardScopeTx(txCtx, q, userID); err != nil {
			return err
		}
		if _, err := ensureUserAffiliateWithClient(txCtx, q, userID); err != nil {
			return err
		}
		expired, err := expireUserTrialBonusTx(txCtx, q, userID)
		if err != nil {
			return err
		}
		result.ExpiredAmount = expired
		result.TrialExpired = result.TrialExpired || expired > 0
		if cleanIP != "" {
			if _, err := q.ExecContext(txCtx, `UPDATE user_affiliates SET last_active_ip = $1, updated_at = NOW() WHERE user_id = $2 AND last_active_ip IS DISTINCT FROM $1::varchar`, cleanIP, userID); err != nil {
				return err
			}
		}
		result.Balance, err = queryUserBalance(txCtx, q, userID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *affiliateRepository) RecordRegistrationIP(ctx context.Context, userID int64, clientIP string) error {
	cleanIP, subnet := service.ComputeIPSubnet(clientIP)
	if userID <= 0 || cleanIP == "" {
		return nil
	}
	return r.withTx(ctx, func(txCtx context.Context, q *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, q, userID); err != nil {
			return err
		}
		_, err := q.ExecContext(txCtx, `UPDATE user_affiliates SET register_ip = $1, register_ip_subnet = $2, updated_at = NOW() WHERE user_id = $3 AND COALESCE(register_ip, '') = ''`, cleanIP, subnet, userID)
		return err
	})
}

// ExpiredAffiliateTrialAdmission is also checked by per-turn billing eligibility:
// a WebSocket connection can outlive its initial HTTP authentication.
func (r *userRepository) ExpiredAffiliateTrialAdmission(ctx context.Context, userID int64) (*service.AffiliateUserAdmission, error) {
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `SELECT user_id FROM user_affiliates WHERE user_id = $1 AND ((trial_bonus_status = 'active' AND trial_expires_at <= NOW()) OR (trial_bonus_status = 'expired' AND trial_expired_amount > 0))`, userID)
	if err != nil {
		return nil, err
	}
	due := rows.Next()
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if !due {
		return nil, nil
	}
	return r.PrepareAffiliateUserForRequest(ctx, userID, "")
}
