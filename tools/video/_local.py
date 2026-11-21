"""本地演示数据：建一个明显的测试用户 video-demo@local.test，结束时连同它名下的密钥 / 使用记录一起删掉。
只动这个用户名下的数据。需要本地 Postgres（读 backend/config.yaml 的密码）。"""
import os, re, subprocess

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
EMAIL = "video-demo@local.test"
PASSWORD = "VideoDemo#2026"
BASE = "http://localhost:3000"

def _env():
    txt = open(os.path.join(ROOT, "backend/config.yaml"), encoding="utf-8").read()
    m = re.search(r"^database:.*?password:\s*\"?([^\"\n]+)\"?", txt, re.S | re.M)
    return {**os.environ, "PGPASSWORD": m.group(1).strip()}

def sql(query, **vars):
    cmd = ["psql", "-h", "127.0.0.1", "-U", "sub2api", "-d", "sub2api", "-v", "ON_ERROR_STOP=1", "-qtA"]
    for k, v in vars.items(): cmd += ["-v", f"{k}={v}"]
    return subprocess.run(cmd + ["-c", query], check=True, capture_output=True, text=True, env=_env()).stdout.strip()

def cleanup():
    uid = sql(f"select id from users where email='{EMAIL}'")
    if not uid: return
    sql(f"delete from usage_logs where user_id={uid}")
    sql(f"delete from api_keys where user_id={uid}")
    sql(f"delete from users where id={uid}")

def setup(balance=1.0):
    """建（或重建）测试用户：余额 balance、没有任何密钥。返回用户 id。"""
    cleanup()
    h = subprocess.check_output(["htpasswd", "-bnBC", "10", "", PASSWORD], text=True).strip().lstrip(":").replace("$2y$", "$2a$", 1)
    sql(f"insert into users (email, password_hash, role, balance, status, username) values ('{EMAIL}', '{h}', 'user', {balance}, 'active', 'demo')")
    return int(sql(f"select id from users where email='{EMAIL}'"))

def login(page):
    page.goto(f"{BASE}/login")
    page.locator('input[type="email"]').fill(EMAIL)
    page.locator('input[type="password"]').fill(PASSWORD)
    page.locator('button[type="submit"]').click()
    page.wait_for_url("**/dashboard", timeout=20000)
    page.wait_for_timeout(1200)

# ---- 画面用的小工具 ----
import json, math, re as _re

# 本地测试分组的名字带「[测试]」，录屏时在浏览器里改成和线上一致的对外名（只改显示，不动数据库）
RENAMES = {
    "[测试] Claude 稳定": "Claude 特惠（Kiro）",
    "不配模型映射：模型列表取 Claude 默认列表。": "性价比之选，适合日常编码",
    "[测试] Claude Max": "Claude 正价满血",
    "高于官方价的分组，用来测试「高于官方价」提示。": "官方正价，稳定优先",
    "模型广场演示": "Claude AWS分组",
    "Default group": "官方满血，稳定优先",
}

def install_rename(ctx):
    def handler(route):
        resp = route.fetch()
        if "json" not in (resp.headers.get("content-type") or ""):
            return route.fulfill(response=resp)
        text = resp.text()
        for a, b in RENAMES.items(): text = text.replace(a, b)
        headers = {k: v for k, v in resp.headers.items() if k.lower() not in ("content-length", "content-encoding")}
        route.fulfill(status=resp.status, headers=headers, body=text)
    ctx.route(_re.compile(r"/api/v1/(groups|keys|usage)"), handler)

def typewrite(r, text, sec):
    """每帧打几个字，总共约 sec 秒。"""
    frames = max(1, int(sec * 10)); k = max(1, math.ceil(len(text) / frames))
    for i in range(0, len(text), k):
        r.page.keyboard.type(text[i:i + k]); r.snap()

def click_fast(r, loc, move=0.5):
    r.move_to(loc, move); r.hold(0.1); loc.click(); r.hold(0.3)

def seed_key(uid, group_id=None, name="我的 Claude Code"):
    """给测试用户建一把密钥（站内默认脱敏显示），返回 key id。"""
    import secrets
    gid = group_id or sql("select id from groups where deleted_at is null and platform='anthropic' and name like '%Claude 稳定' limit 1")
    return int(sql(f"insert into api_keys (user_id, key, name, group_id, status) values ({uid}, 'sk-{secrets.token_hex(32)}', '{name}', {gid}, 'active') returning id").splitlines()[0])

def dismiss_guide(page, uid):
    """把该用户的新手引导标成已关闭（先登录、在站内页面上调用，之后再 goto）。"""
    page.evaluate(f"localStorage.setItem('onboarding_guide_v1:{uid}', JSON.stringify({{status:'dismissed',step:'welcome',client:null,keyId:null}}))")

def seed_usage(uid, key_id, rows):
    """rows: [(分钟前, 输入token, 输出token, 耗时ms)]，模型固定 claude-sonnet-5-5。"""
    gid = sql("select group_id from api_keys where id=%d" % key_id)
    acc = sql("select id from accounts order by id limit 1")
    for i, (mins, tin, tout, ms) in enumerate(rows):
        ic, oc = tin * 0.8 / 1e6, tout * 4 / 1e6
        sql(f"insert into usage_logs (user_id, api_key_id, account_id, request_id, model, input_tokens, output_tokens, input_cost, output_cost, total_cost, actual_cost, group_id, rate_multiplier, stream, duration_ms, first_token_ms, created_at) "
            f"values ({uid}, {key_id}, {acc}, 'video-demo-{key_id}-{i}', 'claude-sonnet-5-5', {tin}, {tout}, {ic:.8f}, {oc:.8f}, {ic+oc:.8f}, {ic+oc:.8f}, {gid}, 0.4, true, {ms}, {ms//6}, now() - interval '{mins} minutes')")
