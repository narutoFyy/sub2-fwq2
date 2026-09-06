-- OAuth monitor notification isolation and sustained-failure state.
ALTER TABLE oauth_account_monitor_states
    ADD COLUMN IF NOT EXISTS consecutive_failures INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_success_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_failure_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_error_code VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS failure_started_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_unavailable_notified_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS failure_sequence BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_notified_failure_sequence BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS quota_alert_active BOOLEAN NOT NULL DEFAULT FALSE;

INSERT INTO settings (key, value, updated_at)
SELECT
    'oauth_account_monitor_email',
    jsonb_build_object(
        'enabled', COALESCE((value::jsonb #>> '{alert,enabled}')::boolean, FALSE),
        'recipients', COALESCE(value::jsonb #> '{alert,recipients}', '[]'::jsonb)
    )::text,
    NOW()
FROM settings
WHERE key = 'ops_email_notification_config'
ON CONFLICT (key) DO NOTHING;

INSERT INTO settings (key, value, updated_at)
VALUES ('oauth_account_monitor_email', '{"enabled":false,"recipients":[]}', NOW())
ON CONFLICT (key) DO NOTHING;

UPDATE settings
SET value = jsonb_set(value::jsonb, '{alert,enabled}', 'false'::jsonb, TRUE)::text,
    updated_at = NOW()
WHERE key = 'ops_email_notification_config'
  AND COALESCE((value::jsonb #>> '{alert,enabled}')::boolean, FALSE) = TRUE;
