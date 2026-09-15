package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

const openAILowCostProbeBudgetAdvisoryLockID int64 = 748802330233

type openAILowCostProbeRepository struct {
	db *sql.DB
}

func NewOpenAILowCostProbeRepository(db *sql.DB) service.OpenAILowCostProbeStateRepository {
	return &openAILowCostProbeRepository{db: db}
}

func (r *openAILowCostProbeRepository) ListStates(ctx context.Context) ([]service.OpenAILowCostProbeState, error) {
	return r.listStates(ctx, 0)
}

func (r *openAILowCostProbeRepository) ListStatesByGroup(ctx context.Context, groupID int64) ([]service.OpenAILowCostProbeState, error) {
	if groupID <= 0 {
		return []service.OpenAILowCostProbeState{}, nil
	}
	return r.listStates(ctx, groupID)
}

func (r *openAILowCostProbeRepository) listStates(ctx context.Context, groupID int64) ([]service.OpenAILowCostProbeState, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil OpenAI low-cost probe repository")
	}
	query := `
SELECT
  s.group_id,
  s.account_id,
  s.status,
  s.last_probe_at,
  s.last_success_at,
  s.last_error,
  s.latency_ms,
  s.consecutive_failures,
  s.consecutive_successes,
  s.disabled_until,
  s.estimated_spend_usd,
  s.updated_at,
  COALESCE(g.name, ''),
  COALESCE(a.name, ''),
  COALESCE(g.rate_multiplier, 0),
  COALESCE(a.rate_multiplier, 1)
FROM openai_low_cost_probe_states s
JOIN groups g ON g.id = s.group_id
JOIN accounts a ON a.id = s.account_id`
	args := make([]any, 0, 1)
	if groupID > 0 {
		query += " WHERE s.group_id = $1"
		args = append(args, groupID)
	}
	query += " ORDER BY s.group_id, s.account_id"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	states := make([]service.OpenAILowCostProbeState, 0)
	for rows.Next() {
		var state service.OpenAILowCostProbeState
		var lastProbe, lastSuccess, disabledUntil sql.NullTime
		var latency sql.NullInt64
		if err := rows.Scan(
			&state.GroupID,
			&state.AccountID,
			&state.Status,
			&lastProbe,
			&lastSuccess,
			&state.LastError,
			&latency,
			&state.ConsecutiveFailures,
			&state.ConsecutiveSuccesses,
			&disabledUntil,
			&state.EstimatedSpendUSD,
			&state.UpdatedAt,
			&state.GroupName,
			&state.AccountName,
			&state.GroupRateMultiplier,
			&state.AccountRateMultiplier,
		); err != nil {
			return nil, err
		}
		if lastProbe.Valid {
			value := lastProbe.Time
			state.LastProbeAt = &value
		}
		if lastSuccess.Valid {
			value := lastSuccess.Time
			state.LastSuccessAt = &value
		}
		if latency.Valid {
			value := latency.Int64
			state.LatencyMS = &value
		}
		if disabledUntil.Valid {
			value := disabledUntil.Time
			state.DisabledUntil = &value
		}
		states = append(states, state)
	}
	return states, rows.Err()
}

func (r *openAILowCostProbeRepository) UpsertState(ctx context.Context, state *service.OpenAILowCostProbeState) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil OpenAI low-cost probe repository")
	}
	if state == nil || state.GroupID <= 0 || state.AccountID <= 0 {
		return fmt.Errorf("invalid OpenAI low-cost probe state")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO openai_low_cost_probe_states (
  group_id, account_id, status, last_probe_at, last_success_at, last_error,
  latency_ms, consecutive_failures, consecutive_successes, disabled_until,
  estimated_spend_usd, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW())
ON CONFLICT (group_id, account_id) DO UPDATE SET
  status = EXCLUDED.status,
  last_probe_at = EXCLUDED.last_probe_at,
  last_success_at = EXCLUDED.last_success_at,
  last_error = EXCLUDED.last_error,
  latency_ms = EXCLUDED.latency_ms,
  consecutive_failures = EXCLUDED.consecutive_failures,
  consecutive_successes = EXCLUDED.consecutive_successes,
  disabled_until = EXCLUDED.disabled_until,
  estimated_spend_usd = EXCLUDED.estimated_spend_usd,
  updated_at = NOW()`,
		state.GroupID,
		state.AccountID,
		state.Status,
		state.LastProbeAt,
		state.LastSuccessAt,
		state.LastError,
		state.LatencyMS,
		state.ConsecutiveFailures,
		state.ConsecutiveSuccesses,
		state.DisabledUntil,
		state.EstimatedSpendUSD,
	)
	return err
}

func (r *openAILowCostProbeRepository) DeleteOutsideScope(ctx context.Context, scopes []service.OpenAILowCostProbeScope) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil OpenAI low-cost probe repository")
	}
	if len(scopes) == 0 {
		_, err := r.db.ExecContext(ctx, `DELETE FROM openai_low_cost_probe_states`)
		return err
	}
	groupIDs := make([]int64, 0, len(scopes))
	accountIDs := make([]int64, 0, len(scopes))
	for _, scope := range scopes {
		groupIDs = append(groupIDs, scope.GroupID)
		accountIDs = append(accountIDs, scope.AccountID)
	}
	_, err := r.db.ExecContext(ctx, `
DELETE FROM openai_low_cost_probe_states AS state
WHERE NOT EXISTS (
  SELECT 1
  FROM unnest($1::bigint[], $2::bigint[]) AS wanted(group_id, account_id)
  WHERE wanted.group_id = state.group_id AND wanted.account_id = state.account_id
)`, pq.Array(groupIDs), pq.Array(accountIDs))
	return err
}

func (r *openAILowCostProbeRepository) ReserveBudget(ctx context.Context, now time.Time, accountID int64, estimatedSpendUSD, maxSpendUSD float64) (float64, bool, error) {
	if r == nil || r.db == nil {
		return 0, false, fmt.Errorf("nil OpenAI low-cost probe repository")
	}
	if estimatedSpendUSD <= 0 || maxSpendUSD <= 0 {
		used, err := r.GetBudgetUsage(ctx, now)
		return used, false, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, openAILowCostProbeBudgetAdvisoryLockID); err != nil {
		return 0, false, err
	}
	var used float64
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(SUM(estimated_spend_usd), 0)
FROM openai_low_cost_probe_budget_ledger
WHERE created_at > $1 - INTERVAL '24 hours'`, now).Scan(&used); err != nil {
		return 0, false, err
	}
	if used+estimatedSpendUSD > maxSpendUSD+1e-12 {
		if err := tx.Commit(); err != nil {
			return 0, false, err
		}
		return used, false, nil
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO openai_low_cost_probe_budget_ledger (account_id, estimated_spend_usd, created_at)
VALUES ($1, $2, $3)`, accountID, estimatedSpendUSD, now); err != nil {
		return 0, false, err
	}
	if err := tx.Commit(); err != nil {
		return 0, false, err
	}
	return used + estimatedSpendUSD, true, nil
}

func (r *openAILowCostProbeRepository) GetBudgetUsage(ctx context.Context, now time.Time) (float64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil OpenAI low-cost probe repository")
	}
	var used float64
	err := r.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(estimated_spend_usd), 0)
FROM openai_low_cost_probe_budget_ledger
WHERE created_at > $1 - INTERVAL '24 hours'`, now).Scan(&used)
	return used, err
}
