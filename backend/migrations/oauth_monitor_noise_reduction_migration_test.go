package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration234SeparatesOAuthEmailAndDisablesOpsAlertEmail(t *testing.T) {
	content, err := FS.ReadFile("234_oauth_monitor_noise_reduction.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "oauth_account_monitor_email")
	require.Contains(t, sql, "{alert,recipients}")
	require.Contains(t, sql, "WHERE key = 'ops_email_notification_config'")
	require.Contains(t, sql, "jsonb_set(value::jsonb, '{alert,enabled}', 'false'::jsonb")
	require.NotContains(t, sql, "{report,enabled}")
}

func TestMigration234AddsSustainedFailureAndQuotaCycleState(t *testing.T) {
	content, err := FS.ReadFile("234_oauth_monitor_noise_reduction.sql")
	require.NoError(t, err)

	sql := string(content)
	for _, column := range []string{
		"consecutive_failures",
		"last_success_at",
		"last_failure_at",
		"last_error_code",
		"failure_started_at",
		"last_unavailable_notified_at",
		"failure_sequence",
		"last_notified_failure_sequence",
		"quota_alert_active",
	} {
		require.Contains(t, sql, column)
	}
}
