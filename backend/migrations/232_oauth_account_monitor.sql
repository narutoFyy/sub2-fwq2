-- OAuth account monitoring runtime state.
CREATE TABLE IF NOT EXISTS oauth_account_monitor_states (
    account_id BIGINT PRIMARY KEY,
    last_checked_at TIMESTAMPTZ,
    health_status VARCHAR(32) NOT NULL DEFAULT 'unknown',
    error_message TEXT,
    primary_remaining_percent DOUBLE PRECISION,
    secondary_remaining_percent DOUBLE PRECISION,
    quota_status VARCHAR(32) NOT NULL DEFAULT 'unknown',
    last_condition_key TEXT NOT NULL DEFAULT '',
    last_notified_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oauth_account_monitor_states_checked
    ON oauth_account_monitor_states (last_checked_at DESC);
