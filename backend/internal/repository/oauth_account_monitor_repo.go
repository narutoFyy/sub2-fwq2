package repository

import (
	"context"
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type oauthAccountMonitorRepository struct{ db *sql.DB }

func NewOAuthAccountMonitorRepository(db *sql.DB) service.OAuthAccountMonitorStateRepository {
	return &oauthAccountMonitorRepository{db: db}
}

func (r *oauthAccountMonitorRepository) GetStates(ctx context.Context, accountIDs []int64) (map[int64]*service.OAuthAccountMonitorState, error) {
	result := make(map[int64]*service.OAuthAccountMonitorState)
	if r == nil || r.db == nil || len(accountIDs) == 0 {
		return result, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT account_id,last_checked_at,health_status,COALESCE(error_message,''),primary_remaining_percent,secondary_remaining_percent,quota_status,last_condition_key,last_notified_at,consecutive_failures,last_success_at,last_failure_at,COALESCE(last_error_code,''),failure_started_at,last_unavailable_notified_at,failure_sequence,last_notified_failure_sequence,quota_alert_active,updated_at FROM oauth_account_monitor_states WHERE account_id = ANY($1)`, pq.Array(accountIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		state := &service.OAuthAccountMonitorState{}
		var checked, notified, lastSuccess, lastFailure, failureStarted, unavailableNotified sql.NullTime
		var errMsg, condition sql.NullString
		var primary, secondary sql.NullFloat64
		if err := rows.Scan(&state.AccountID, &checked, &state.HealthStatus, &errMsg, &primary, &secondary, &state.QuotaStatus, &condition, &notified, &state.ConsecutiveFailures, &lastSuccess, &lastFailure, &state.LastErrorCode, &failureStarted, &unavailableNotified, &state.FailureSequence, &state.LastNotifiedFailureSequence, &state.QuotaAlertActive, &state.UpdatedAt); err != nil {
			return nil, err
		}
		if checked.Valid {
			state.LastCheckedAt = &checked.Time
		}
		state.ErrorMessage = errMsg.String
		if primary.Valid {
			v := primary.Float64
			state.PrimaryRemainingPercent = &v
		}
		if secondary.Valid {
			v := secondary.Float64
			state.SecondaryRemainingPercent = &v
		}
		state.LastConditionKey = condition.String
		if notified.Valid {
			state.LastNotifiedAt = &notified.Time
		}
		if lastSuccess.Valid {
			state.LastSuccessAt = &lastSuccess.Time
		}
		if lastFailure.Valid {
			state.LastFailureAt = &lastFailure.Time
		}
		if failureStarted.Valid {
			state.FailureStartedAt = &failureStarted.Time
		}
		if unavailableNotified.Valid {
			state.LastUnavailableNotifiedAt = &unavailableNotified.Time
		}
		result[state.AccountID] = state
	}
	return result, rows.Err()
}

func (r *oauthAccountMonitorRepository) UpsertState(ctx context.Context, state *service.OAuthAccountMonitorState) error {
	if r == nil || r.db == nil || state == nil {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO oauth_account_monitor_states (account_id,last_checked_at,health_status,error_message,primary_remaining_percent,secondary_remaining_percent,quota_status,last_condition_key,last_notified_at,consecutive_failures,last_success_at,last_failure_at,last_error_code,failure_started_at,last_unavailable_notified_at,failure_sequence,last_notified_failure_sequence,quota_alert_active,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,NOW()) ON CONFLICT (account_id) DO UPDATE SET last_checked_at=EXCLUDED.last_checked_at,health_status=EXCLUDED.health_status,error_message=EXCLUDED.error_message,primary_remaining_percent=EXCLUDED.primary_remaining_percent,secondary_remaining_percent=EXCLUDED.secondary_remaining_percent,quota_status=EXCLUDED.quota_status,last_condition_key=EXCLUDED.last_condition_key,last_notified_at=EXCLUDED.last_notified_at,consecutive_failures=EXCLUDED.consecutive_failures,last_success_at=EXCLUDED.last_success_at,last_failure_at=EXCLUDED.last_failure_at,last_error_code=EXCLUDED.last_error_code,failure_started_at=EXCLUDED.failure_started_at,last_unavailable_notified_at=EXCLUDED.last_unavailable_notified_at,failure_sequence=EXCLUDED.failure_sequence,last_notified_failure_sequence=EXCLUDED.last_notified_failure_sequence,quota_alert_active=EXCLUDED.quota_alert_active,updated_at=NOW()`, state.AccountID, state.LastCheckedAt, state.HealthStatus, state.ErrorMessage, state.PrimaryRemainingPercent, state.SecondaryRemainingPercent, state.QuotaStatus, state.LastConditionKey, state.LastNotifiedAt, state.ConsecutiveFailures, state.LastSuccessAt, state.LastFailureAt, state.LastErrorCode, state.FailureStartedAt, state.LastUnavailableNotifiedAt, state.FailureSequence, state.LastNotifiedFailureSequence, state.QuotaAlertActive)
	return err
}
