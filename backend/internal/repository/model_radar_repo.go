package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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
	rows, err := r.db.QueryContext(ctx, `SELECT id, group_id, model_id, reasoning_effort, enabled, next_run_at, last_run_at FROM model_radar_configs ORDER BY group_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*service.ModelRadarConfig, 0)
	for rows.Next() {
		item := &service.ModelRadarConfig{}
		var next, last sql.NullTime
		if err := rows.Scan(&item.ID, &item.GroupID, &item.ModelID, &item.ReasoningEffort, &item.Enabled, &next, &last); err != nil {
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

func (r *modelRadarRepository) UpsertConfig(ctx context.Context, item *service.ModelRadarConfig) (*service.ModelRadarConfig, error) {
	if r == nil || r.db == nil || item == nil || item.GroupID <= 0 {
		return nil, fmt.Errorf("invalid model radar config")
	}
	result := &service.ModelRadarConfig{}
	var next, last sql.NullTime
	err := r.db.QueryRowContext(ctx, `INSERT INTO model_radar_configs (group_id, model_id, reasoning_effort, enabled, next_run_at, last_run_at) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (group_id) DO UPDATE SET model_id=EXCLUDED.model_id, reasoning_effort=EXCLUDED.reasoning_effort, enabled=EXCLUDED.enabled, next_run_at=EXCLUDED.next_run_at, last_run_at=EXCLUDED.last_run_at, updated_at=NOW() RETURNING id, group_id, model_id, reasoning_effort, enabled, next_run_at, last_run_at`, item.GroupID, strings.TrimSpace(item.ModelID), item.ReasoningEffort, item.Enabled, item.NextRunAt, item.LastRunAt).Scan(&result.ID, &result.GroupID, &result.ModelID, &result.ReasoningEffort, &result.Enabled, &next, &last)
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
	defer rows.Close()
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
