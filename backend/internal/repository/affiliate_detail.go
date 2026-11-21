package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *affiliateRepository) GetInviteeTotals(ctx context.Context, inviterID int64, now time.Time) (*service.AffiliateInviteeTotals, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, `
WITH invitees AS (
    SELECT *, $2::timestamptz <= created_at + INTERVAL '30 days' AS within_window,
           qualified_at IS NOT NULL AND qualified_at <= created_at + INTERVAL '30 days'
             AND total_paid_amount >= $3 AND total_consumed_amount >= $5 AS qualified
    FROM user_affiliates WHERE inviter_id = $1
)
SELECT COUNT(*),
       COUNT(*) FILTER (WHERE qualified),
       COALESCE(SUM(total_paid_amount) FILTER (WHERE qualified), 0)::double precision,
       COALESCE(SUM(
         CASE WHEN step1_reward_status = 'pending_review'
                OR (within_window AND (step1_reward_status = 'pending_consume'
                    OR (step1_reward_status = 'none' AND total_paid_amount >= $3 AND total_consumed_amount < $5)))
              THEN $6::numeric ELSE 0 END
         + CASE WHEN step2_reward_status = 'pending_review'
                OR (within_window AND (step2_reward_status = 'pending_consume'
                    OR (step2_reward_status = 'none' AND total_paid_amount >= $4 AND total_consumed_amount < $5)))
              THEN $7::numeric ELSE 0 END
       ), 0)::double precision,
       COALESCE(SUM(
         CASE WHEN step1_reward_status = 'granted' THEN $6::numeric ELSE 0 END
         + CASE WHEN step2_reward_status = 'granted' THEN $7::numeric ELSE 0 END
       ), 0)::double precision,
       COUNT(*) FILTER (WHERE within_window AND total_paid_amount >= $3 AND total_consumed_amount < $5)
FROM invitees`, inviterID, now, service.AffiliateStep1MinPaid, service.AffiliateStep2MinPaid,
		service.AffiliateMinConsumed, service.AffiliateStep1Bonus, service.AffiliateStep2Bonus)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("affiliate totals query returned no row")
	}
	totals := &service.AffiliateInviteeTotals{}
	if err := rows.Scan(&totals.InviteeCount, &totals.QualifiedCount, &totals.QualifiedPaidTotal,
		&totals.PendingStepTotal, &totals.GrantedStepTotal, &totals.RemindCount); err != nil {
		return nil, err
	}
	return totals, rows.Err()
}
