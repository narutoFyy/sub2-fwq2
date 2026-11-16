package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type modelRadarRepository struct{ db *sql.DB }

func NewModelRadarRepository(db *sql.DB) service.ModelRadarRepository {
	return &modelRadarRepository{db: db}
}

func (r *modelRadarRepository) ListConfigs(ctx context.Context) ([]*service.ModelRadarConfig, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil model radar repository")
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, group_id, model_id, reasoning_effort, enabled, interval_minutes, visible, next_run_at, last_run_at FROM model_radar_configs ORDER BY group_id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]*service.ModelRadarConfig, 0)
	for rows.Next() {
		item := &service.ModelRadarConfig{}
		var next, last sql.NullTime
		if err := rows.Scan(&item.ID, &item.GroupID, &item.ModelID, &item.ReasoningEffort, &item.Enabled, &item.IntervalMinutes, &item.Visible, &next, &last); err != nil {
			return nil, err
		}
		if next.Valid {
			item.NextRunAt = &next.Time
		}
		if last.Valid {
			item.LastRunAt = &last.Time
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// UpsertConfig 保存管理员可编辑的字段。item.NextRunAt 是「最晚下次时间」：新开启的分组立刻排上，
// 已排的保持原时间、只在比它更晚时提前；关闭的清空。last_run_at 不动。
func (r *modelRadarRepository) UpsertConfig(ctx context.Context, item *service.ModelRadarConfig) (*service.ModelRadarConfig, error) {
	if r == nil || r.db == nil || item == nil || item.GroupID <= 0 {
		return nil, fmt.Errorf("invalid model radar config")
	}
	result := &service.ModelRadarConfig{}
	var next, last sql.NullTime
	err := r.db.QueryRowContext(ctx, `INSERT INTO model_radar_configs (group_id, model_id, reasoning_effort, enabled, interval_minutes, visible, next_run_at)
VALUES ($1,$2,$3,$4,$5,$6, CASE WHEN $4 THEN NOW() END)
ON CONFLICT (group_id) DO UPDATE SET model_id=EXCLUDED.model_id, reasoning_effort=EXCLUDED.reasoning_effort, enabled=EXCLUDED.enabled,
  interval_minutes=EXCLUDED.interval_minutes, visible=EXCLUDED.visible,
  next_run_at=CASE WHEN NOT EXCLUDED.enabled THEN NULL
                   WHEN model_radar_configs.next_run_at IS NULL THEN NOW()
                   ELSE LEAST(model_radar_configs.next_run_at, COALESCE($7::timestamptz, model_radar_configs.next_run_at)) END,
  updated_at=NOW()
RETURNING id, group_id, model_id, reasoning_effort, enabled, interval_minutes, visible, next_run_at, last_run_at`,
		item.GroupID, strings.TrimSpace(item.ModelID), item.ReasoningEffort, item.Enabled, item.IntervalMinutes, item.Visible, item.NextRunAt).
		Scan(&result.ID, &result.GroupID, &result.ModelID, &result.ReasoningEffort, &result.Enabled, &result.IntervalMinutes, &result.Visible, &next, &last)
	if err != nil {
		return nil, err
	}
	if next.Valid {
		result.NextRunAt = &next.Time
	}
	if last.Valid {
		result.LastRunAt = &last.Time
	}
	return result, nil
}

func (r *modelRadarRepository) GetSettings(ctx context.Context) (*service.ModelRadarSettings, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil model radar repository")
	}
	out := service.DefaultModelRadarSettings()
	err := r.db.QueryRowContext(ctx, `SELECT interval_minutes, ciallo_attempts, nerf_confirm_rounds, drawing_enabled, retention_days FROM model_radar_settings WHERE id=1`).
		Scan(&out.IntervalMinutes, &out.CialloAttempts, &out.NerfConfirmRounds, &out.DrawingEnabled, &out.RetentionDays)
	if err == sql.ErrNoRows {
		return &out, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateSettings 保存全局设置；间隔调短时，把已经排得更晚的分组提前到新间隔之内。
func (r *modelRadarRepository) UpdateSettings(ctx context.Context, in service.ModelRadarSettings) (*service.ModelRadarSettings, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil model radar repository")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO model_radar_settings (id, interval_minutes, ciallo_attempts, nerf_confirm_rounds, drawing_enabled, retention_days) VALUES (1,$1,$2,$3,$4,$5)
ON CONFLICT (id) DO UPDATE SET interval_minutes=EXCLUDED.interval_minutes, ciallo_attempts=EXCLUDED.ciallo_attempts, nerf_confirm_rounds=EXCLUDED.nerf_confirm_rounds, drawing_enabled=EXCLUDED.drawing_enabled, retention_days=EXCLUDED.retention_days, updated_at=NOW()`,
		in.IntervalMinutes, in.CialloAttempts, in.NerfConfirmRounds, in.DrawingEnabled, in.RetentionDays); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE model_radar_configs SET next_run_at = NOW() + make_interval(mins => $1::int)
WHERE enabled AND interval_minutes = 0 AND next_run_at > NOW() + make_interval(mins => $1::int)`, in.IntervalMinutes); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &in, nil
}

func (r *modelRadarRepository) ClaimDue(ctx context.Context, groupID int64, now, next time.Time) (bool, error) {
	if r == nil || r.db == nil {
		return false, fmt.Errorf("nil model radar repository")
	}
	res, err := r.db.ExecContext(ctx, `UPDATE model_radar_configs SET next_run_at=$3 WHERE group_id=$1 AND enabled AND next_run_at IS NOT NULL AND next_run_at <= $2`, groupID, now, next)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	return affected > 0, err
}

func (r *modelRadarRepository) MarkRun(ctx context.Context, groupID int64, last, next time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil model radar repository")
	}
	_, err := r.db.ExecContext(ctx, `UPDATE model_radar_configs SET last_run_at=$2, next_run_at=CASE WHEN enabled THEN $3::timestamptz END WHERE group_id=$1`, groupID, last, next)
	return err
}

func (r *modelRadarRepository) DeleteResult(ctx context.Context, resultID int64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil model radar repository")
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM model_radar_results WHERE id=$1`, resultID)
	return err
}

func (r *modelRadarRepository) DeleteGroupResults(ctx context.Context, groupID int64) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil model radar repository")
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM model_radar_results WHERE group_id=$1`, groupID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *modelRadarRepository) DeleteResultsBefore(ctx context.Context, before time.Time) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil model radar repository")
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM model_radar_results WHERE detected_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *modelRadarRepository) ListResults(ctx context.Context, groupID *int64, testType string, limit int) ([]*service.ModelRadarResult, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil model radar repository")
	}
	if limit <= 0 || limit > 1000 {
		limit = 300
	}
	query := `SELECT id, group_id, model_id, test_type, status, response_text, error_message, latency_ms, detected_at, review_status, reviewed_at, reviewed_by FROM model_radar_results WHERE 1=1`
	args := make([]any, 0, 3)
	if groupID != nil {
		args = append(args, *groupID)
		query += fmt.Sprintf(" AND group_id=$%d", len(args))
	}
	if testType != "" {
		args = append(args, testType)
		query += fmt.Sprintf(" AND test_type=$%d", len(args))
	}
	args = append(args, limit)
	query += fmt.Sprintf(" ORDER BY detected_at DESC LIMIT $%d", len(args))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]*service.ModelRadarResult, 0)
	for rows.Next() {
		item := &service.ModelRadarResult{}
		var reviewedAt sql.NullTime
		var reviewedBy sql.NullInt64
		if err := rows.Scan(&item.ID, &item.GroupID, &item.ModelID, &item.TestType, &item.Status, &item.ResponseText, &item.ErrorMessage, &item.LatencyMS, &item.DetectedAt, &item.ReviewStatus, &reviewedAt, &reviewedBy); err != nil {
			return nil, err
		}
		if reviewedAt.Valid {
			item.ReviewedAt = &reviewedAt.Time
		}
		if reviewedBy.Valid {
			item.ReviewedBy = &reviewedBy.Int64
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *modelRadarRepository) CreateResult(ctx context.Context, item *service.ModelRadarResult) (*service.ModelRadarResult, error) {
	if r == nil || r.db == nil || item == nil {
		return nil, fmt.Errorf("invalid model radar result")
	}
	out := &service.ModelRadarResult{}
	var reviewedAt sql.NullTime
	var reviewedBy sql.NullInt64
	err := r.db.QueryRowContext(ctx, `INSERT INTO model_radar_results (group_id, model_id, test_type, status, response_text, error_message, latency_ms, detected_at, review_status) VALUES ($1,$2,$3,$4,$5,$6,$7,COALESCE($8,NOW()),$9) RETURNING id, group_id, model_id, test_type, status, response_text, error_message, latency_ms, detected_at, review_status, reviewed_at, reviewed_by`, item.GroupID, item.ModelID, item.TestType, item.Status, item.ResponseText, item.ErrorMessage, item.LatencyMS, item.DetectedAt, item.ReviewStatus).Scan(&out.ID, &out.GroupID, &out.ModelID, &out.TestType, &out.Status, &out.ResponseText, &out.ErrorMessage, &out.LatencyMS, &out.DetectedAt, &out.ReviewStatus, &reviewedAt, &reviewedBy)
	if err != nil {
		return nil, err
	}
	if reviewedAt.Valid {
		out.ReviewedAt = &reviewedAt.Time
	}
	if reviewedBy.Valid {
		out.ReviewedBy = &reviewedBy.Int64
	}
	return out, nil
}

func (r *modelRadarRepository) ReviewResult(ctx context.Context, resultID int64, status string, reviewerID int64) (*service.ModelRadarResult, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil model radar repository")
	}
	out := &service.ModelRadarResult{}
	var reviewedAt sql.NullTime
	var reviewedBy sql.NullInt64
	err := r.db.QueryRowContext(ctx, `UPDATE model_radar_results SET review_status=$2, status=$3, reviewed_at=NOW(), reviewed_by=$4 WHERE id=$1 AND test_type='drawing' RETURNING id, group_id, model_id, test_type, status, response_text, error_message, latency_ms, detected_at, review_status, reviewed_at, reviewed_by`, resultID, status, status, reviewerID).Scan(&out.ID, &out.GroupID, &out.ModelID, &out.TestType, &out.Status, &out.ResponseText, &out.ErrorMessage, &out.LatencyMS, &out.DetectedAt, &out.ReviewStatus, &reviewedAt, &reviewedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrModelRadarReview
		}
		return nil, err
	}
	if reviewedAt.Valid {
		out.ReviewedAt = &reviewedAt.Time
	}
	if reviewedBy.Valid {
		out.ReviewedBy = &reviewedBy.Int64
	}
	return out, nil
}
