-- 广场本地测试数据：所有分组、账号名称都以「[测试]」开头，清理见 plaza_cleanup.sql。
-- 账号指向不可达地址 127.0.0.1:9，只用于让分组「有可调度账号」，不能真实调用。
BEGIN;

CREATE TEMP TABLE _plaza_groups (key text, name text, platform text, rate numeric, exclusive bool, description text,
  peak bool, peak_start text, peak_end text, peak_mult numeric, long_ctx bool, mapping jsonb) ON COMMIT DROP;

INSERT INTO _plaza_groups VALUES
 ('low',    '[测试] 低价福利',     'openai',    0.08, false, '福利价，适合日常编程与批量任务。', false, '', '', 1, true,
   '{"gpt-5.5":"gpt-5.5","gpt-5.4":"gpt-5.4","gpt-5.4-mini":"gpt-5.4-mini","gpt-5.3-codex":"gpt-5.3-codex"}'),
 ('pro',    '[测试] GPT Pro 号池', 'openai',    0.25, false, '官方 Pro 账号池，稳定不降智。', false, '', '', 1, true,
   '{"gpt-5.5":"gpt-5.5","gpt-5.4":"gpt-5.4","gpt-5.2":"gpt-5.2"}'),
 ('peak',   '[测试] 高峰加价分组', 'openai',    0.35, false, '晚高峰加价，用来测试高峰提示和关闭长上下文分档。', true, '20:00', '23:00', 1.5, false,
   '{"gpt-5.4":"gpt-5.4","gpt-5.4-mini":"gpt-5.4-mini"}'),
 ('claude', '[测试] Claude 稳定',  'anthropic', 0.40, false, '不配模型映射：模型列表取 Claude 默认列表。', false, '', '', 1, true, NULL),
 ('max',    '[测试] Claude Max',   'anthropic', 1.50, false, '高于官方价的分组，用来测试「高于官方价」提示。', false, '', '', 1, true,
   '{"claude-opus-4-6":"claude-opus-4-6","claude-sonnet-4-6":"claude-sonnet-4-6","claude-haiku-4-5":"claude-haiku-4-5"}'),
 ('grok',   '[测试] Grok 稳定',    'grok',      0.30, false, '不配模型映射：模型列表取 Grok 默认列表。', false, '', '', 1, true, NULL),
 ('vip',    '[测试] 大客户专属',   'anthropic', 1.20, true,  '专属分组：未登录时不应出现。', false, '', '', 1, true,
   '{"claude-opus-4-6":"claude-opus-4-6"}'),
 ('empty',  '[测试] 无账号分组',   'openai',    1.00, false, '没有可调度账号：不应出现。', false, '', '', 1, true, NULL);

CREATE TEMP TABLE _plaza_ids (key text, group_id bigint) ON COMMIT DROP;

WITH ins AS (
  INSERT INTO groups (name, description, platform, rate_multiplier, is_exclusive, peak_rate_enabled, peak_start, peak_end,
                      peak_rate_multiplier, long_context_pricing_enabled, status)
  SELECT name, description, platform, rate, exclusive, peak, peak_start, peak_end, peak_mult, long_ctx, 'active' FROM _plaza_groups
  RETURNING id, name)
INSERT INTO _plaza_ids SELECT g.key, ins.id FROM ins JOIN _plaza_groups g ON g.name = ins.name;

WITH acc AS (
  INSERT INTO accounts (name, platform, type, credentials, extra, status, schedulable)
  SELECT g.name || ' 账号', g.platform, 'apikey',
         jsonb_build_object('api_key', 'sk-plaza-local-test', 'base_url', 'http://127.0.0.1:9')
           || CASE WHEN g.mapping IS NULL THEN '{}'::jsonb ELSE jsonb_build_object('model_mapping', g.mapping) END,
         '{}'::jsonb, 'active', true
  FROM _plaza_groups g WHERE g.key <> 'empty'
  RETURNING id, name)
INSERT INTO account_groups (account_id, group_id)
SELECT acc.id, i.group_id FROM acc
JOIN _plaza_groups g ON acc.name = g.name || ' 账号'
JOIN _plaza_ids i ON i.key = g.key;

-- 未登录也能看广场
UPDATE settings SET value = 'false' WHERE key = 'model_plaza_require_auth';
UPDATE settings SET value = 'true'  WHERE key = 'model_plaza_enabled';

COMMIT;
