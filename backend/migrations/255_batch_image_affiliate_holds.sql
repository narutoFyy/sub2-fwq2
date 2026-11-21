-- Track actual trial-credit use separately from paid consumption and reservations.
-- NULL is only a migration marker, so replay never resets live accounting.
ALTER TABLE user_affiliates ADD COLUMN IF NOT EXISTS trial_consumed_amount DECIMAL(20,8);
UPDATE user_affiliates
SET trial_consumed_amount = LEAST(GREATEST(trial_bonus_amount - trial_expired_amount, 0), GREATEST(total_consumed_amount, 0))
WHERE trial_consumed_amount IS NULL;
ALTER TABLE user_affiliates ALTER COLUMN trial_consumed_amount SET DEFAULT 0;
ALTER TABLE user_affiliates ALTER COLUMN trial_consumed_amount SET NOT NULL;

CREATE TABLE IF NOT EXISTS usage_billing_batch_image_holds (
    batch_id VARCHAR(64) PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id BIGINT NOT NULL,
    hold_amount DECIMAL(20,10) NOT NULL CHECK (hold_amount >= 0),
    trial_reserved_amount DECIMAL(20,8) NOT NULL DEFAULT 0 CHECK (trial_reserved_amount >= 0 AND trial_reserved_amount <= hold_amount),
    client_ip VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL CHECK (status IN ('reserved', 'captured', 'released')),
    actual_amount DECIMAL(20,10) NOT NULL DEFAULT 0 CHECK (actual_amount >= 0 AND actual_amount <= hold_amount),
    credited_inviter_id BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_batch_image_holds_reserved_user
    ON usage_billing_batch_image_holds (user_id) WHERE status = 'reserved';

-- Existing jobs are not proof that a hold succeeded: require the durable billing
-- claim (including its archive). Preserve terminal claims even when the job's
-- post-billing status update failed. This does not backfill historical rewards.
WITH claims AS NOT MATERIALIZED (
    SELECT request_id, api_key_id FROM usage_billing_dedup
    UNION ALL
    SELECT request_id, api_key_id FROM usage_billing_dedup_archive
), proven AS (
    SELECT j.*, CASE
        WHEN EXISTS (SELECT 1 FROM claims c WHERE c.request_id = 'batch_image_capture:' || j.batch_id AND c.api_key_id = j.api_key_id) THEN 'captured'
        WHEN EXISTS (SELECT 1 FROM claims c WHERE c.request_id = 'batch_image_release:' || j.batch_id AND c.api_key_id = j.api_key_id) THEN 'released'
        ELSE 'reserved' END AS hold_status
    FROM batch_image_jobs j
    WHERE j.api_key_id IS NOT NULL
      AND COALESCE(j.hold_amount, j.estimated_cost) > 0
      AND EXISTS (SELECT 1 FROM claims c WHERE c.request_id = 'batch_image_hold:' || j.batch_id AND c.api_key_id = j.api_key_id)
), new_holds AS (
    SELECT p.*, COALESCE(p.hold_amount, p.estimated_cost) AS hold_value
    FROM proven p JOIN users u ON u.id = p.user_id
    WHERE NOT EXISTS (SELECT 1 FROM usage_billing_batch_image_holds existing WHERE existing.batch_id = p.batch_id)
), eligible AS (

-- Legacy money had no per-hold provenance. Allocate only the remaining trial
-- principal, after recorded consumption and expiry, to eligible open holds in
-- creation order. Never allocate already-clawed-back money. Existing assignments
-- are included so replay cannot assign the same principal a second time.
    SELECT h.batch_id, h.hold_value, h.user_id,
           GREATEST(ua.trial_bonus_amount - ua.trial_consumed_amount - ua.trial_expired_amount
               - COALESCE((SELECT SUM(prior.trial_reserved_amount) FROM usage_billing_batch_image_holds prior WHERE prior.user_id = h.user_id AND prior.status = 'reserved'), 0), 0) AS remaining,
           COALESCE(SUM(h.hold_value) OVER (PARTITION BY h.user_id ORDER BY h.created_at, h.batch_id ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING), 0) AS preceding
    FROM new_holds h
    JOIN user_affiliates ua ON ua.user_id = h.user_id
    WHERE h.hold_status = 'reserved' AND ua.trial_bonus_status IN ('active', 'expired')
      AND (ua.trial_granted_at IS NULL OR h.created_at >= ua.trial_granted_at)
      AND (ua.trial_expires_at IS NULL OR h.created_at < ua.trial_expires_at)
)
INSERT INTO usage_billing_batch_image_holds (batch_id, user_id, api_key_id, hold_amount, trial_reserved_amount, status, actual_amount, created_at, updated_at)
SELECT h.batch_id, h.user_id, h.api_key_id, h.hold_value,
       COALESCE(TRUNC(LEAST(e.hold_value, GREATEST(e.remaining - e.preceding, 0)), 8), 0), h.hold_status,
       CASE WHEN h.hold_status = 'captured' THEN LEAST(GREATEST(COALESCE(h.actual_cost, 0), 0), h.hold_value) ELSE 0 END,
       h.created_at, h.updated_at
FROM new_holds h LEFT JOIN eligible e ON e.batch_id = h.batch_id
ON CONFLICT (batch_id) DO NOTHING;
