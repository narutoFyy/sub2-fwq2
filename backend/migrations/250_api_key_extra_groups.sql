-- 一个 Key 可同时绑定多个分组：group_id 仍是主分组，其余分组按顺序存在 extra_group_ids。
-- 请求按模型所属平台在这些分组里选一个；老 Key 该列为空，行为不变。
ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS extra_group_ids JSONB DEFAULT NULL;

COMMENT ON COLUMN api_keys.extra_group_ids IS '附加分组 ID 列表（JSON 数组，按优先级排序），主分组仍为 group_id';
