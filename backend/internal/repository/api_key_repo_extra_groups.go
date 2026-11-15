package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"

	"entgo.io/ent/dialect"
)

// api_keys.extra_group_ids 是迁移 250 加的 JSONB 列（主分组仍在 group_id），
// 不在 ent schema 里，所以这里用原生 SQL 读写。
// 仅 Postgres（生产与集成测试）有该列；SQLite 单测由 ent 自动建表、没有这一列，
// 一律按「没有附加分组」处理。

func (r *apiKeyRepository) isPostgres() bool {
	return r.client != nil && r.client.Driver().Dialect() == dialect.Postgres
}

// setExtraGroupIDs 覆盖写入一把 Key 的附加分组；空列表写 NULL。
func (r *apiKeyRepository) setExtraGroupIDs(ctx context.Context, keyID int64, ids []int64) error {
	if !r.isPostgres() {
		if len(ids) > 0 {
			return fmt.Errorf("api key repository: extra groups require PostgreSQL")
		}
		return nil
	}
	if r.sql == nil {
		return fmt.Errorf("api key repository: sql executor is not configured")
	}
	var value any
	if len(ids) > 0 {
		raw, err := json.Marshal(ids)
		if err != nil {
			return err
		}
		value = string(raw)
	}
	query := `UPDATE api_keys SET extra_group_ids = $1 WHERE id = $2`
	if value != nil {
		query = `UPDATE api_keys SET extra_group_ids = $1::jsonb WHERE id = $2`
	}
	_, err := r.sql.ExecContext(ctx, query, value, keyID)
	return err
}

// loadExtraGroupIDs 批量读取附加分组。
func (r *apiKeyRepository) loadExtraGroupIDs(ctx context.Context, keyIDs []int64) (map[int64][]int64, error) {
	out := make(map[int64][]int64)
	if len(keyIDs) == 0 || r.sql == nil || !r.isPostgres() {
		return out, nil
	}
	query := `SELECT id, extra_group_ids FROM api_keys WHERE id = ANY($1::bigint[]) AND extra_group_ids IS NOT NULL`
	args := []any{pq.Array(keyIDs)}
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id int64
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			return nil, err
		}
		if len(raw) == 0 {
			continue
		}
		var ids []int64
		if err := json.Unmarshal(raw, &ids); err != nil {
			return nil, fmt.Errorf("decode extra_group_ids of api key %d: %w", id, err)
		}
		if len(ids) > 0 {
			out[id] = ids
		}
	}
	return out, rows.Err()
}

func (r *apiKeyRepository) attachExtraGroupIDs(ctx context.Context, key *service.APIKey) error {
	if key == nil {
		return nil
	}
	m, err := r.loadExtraGroupIDs(ctx, []int64{key.ID})
	if err != nil {
		return err
	}
	key.ExtraGroupIDs = m[key.ID]
	return nil
}

func (r *apiKeyRepository) attachExtraGroupIDsToList(ctx context.Context, keys []service.APIKey) error {
	if len(keys) == 0 {
		return nil
	}
	ids := make([]int64, len(keys))
	for i := range keys {
		ids[i] = keys[i].ID
	}
	m, err := r.loadExtraGroupIDs(ctx, ids)
	if err != nil {
		return err
	}
	for i := range keys {
		keys[i].ExtraGroupIDs = m[keys[i].ID]
	}
	return nil
}

// attachExtraGroups 按附加分组 ID 的顺序加载分组对象（认证路径，用认证投影）。
// 已删除的分组会被跳过，不会让整把 Key 失效。
func (r *apiKeyRepository) attachExtraGroups(ctx context.Context, key *service.APIKey) error {
	if key == nil || len(key.ExtraGroupIDs) == 0 {
		return nil
	}
	query := r.client.Group.Query().Where(group.IDIn(key.ExtraGroupIDs...))
	selectAuthGroupFields(query)
	rows, err := query.All(ctx)
	if err != nil {
		return err
	}
	byID := make(map[int64]*service.Group, len(rows))
	for _, row := range rows {
		byID[row.ID] = groupEntityToService(row)
	}
	key.ExtraGroups = make([]*service.Group, 0, len(key.ExtraGroupIDs))
	for _, id := range key.ExtraGroupIDs {
		if g, ok := byID[id]; ok {
			key.ExtraGroups = append(key.ExtraGroups, g)
		}
	}
	return nil
}

// listKeysByExtraGroupID 返回把 groupID 绑成附加分组的 Key 字符串，用于分组变更时失效认证缓存。
func (r *apiKeyRepository) listKeysByExtraGroupID(ctx context.Context, groupID int64) ([]string, error) {
	if r.sql == nil || !r.isPostgres() {
		return nil, nil
	}
	rows, err := r.sql.QueryContext(ctx,
		`SELECT key FROM api_keys WHERE deleted_at IS NULL AND extra_group_ids @> $1::jsonb`,
		fmt.Sprintf("[%d]", groupID))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}
