#!/usr/bin/env python3
"""图文教程截图：用 Playwright（系统 Chrome）在本地开发环境里截真实页面，并量出要标注的元素位置。

准备：
  1. 本地后端（8085）和前端（3000）都在跑；
  2. 运行 tools/tutorial-shots/run.sh（它会导入 seed.sql、跑本脚本、再还原数据）。

产物：
  frontend/public/tutorials/<语言>/<id>.jpg   本站页面截图（中/英各一套）
  frontend/public/tutorials/shared/<id>.jpg   不分语言的截图（小铺页面；小铺会拦截脚本浏览器，
                                              这部分是在 Claude 桌面端内置浏览器里手动截的，脚本不会动它）
  frontend/src/features/tutorials/shots.json  每张图的尺寸和标注框（百分比），教程页据此画编号圈

只截我们自己的页面。
"""
from __future__ import annotations

import json
import os
import subprocess
import sys

from playwright.sync_api import Page, sync_playwright

BASE = os.environ.get("SHOTS_BASE", "http://localhost:3000")
SHOP = "https://wzyp.cn/shop/PL2VA4CE"
EMAIL = "shots@local.test"
PASSWORD = os.environ["SHOTS_PASSWORD"]
ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
OUT = os.path.join(ROOT, "frontend", "public", "tutorials")
META = os.path.join(ROOT, "frontend", "src", "features", "tutorials", "shots.json")
VIEW = {"width": 1100, "height": 800}
DEMO_CODE = "DEMO-7K2Q-9XPA-4M8D"

PSQL = ["psql", "-h", "127.0.0.1", "-U", "sub2api", "-d", "sub2api", "-v", "ON_ERROR_STOP=1", "-qtA", "-c"]


def sql(query: str) -> str:
    return subprocess.run(PSQL + [query], check=True, capture_output=True, text=True).stdout.strip()


meta: dict[str, dict[str, dict]] = {"zh": {}, "en": {}, "shared": {}}


def box(page: Page, selector: str):
    """元素的页面坐标（含滚动偏移）；找不到返回 None。"""
    loc = page.locator(selector).first
    if loc.count() == 0:
        return None
    return loc.bounding_box()


def shoot(page: Page, bucket: str, shot_id: str, region: str | dict, marks: list[tuple[int | None, str]], pad: int = 0) -> None:
    """截 region（选择器或 {x,y,width,height}）并记录 marks 里每个元素相对截图的百分比位置。"""
    page.wait_for_timeout(500)
    rect = box(page, region) if isinstance(region, str) else region
    if rect is None:
        raise RuntimeError(f"{bucket}/{shot_id}: region {region!r} not found")
    x = max(0, rect["x"] - pad)
    y = max(0, rect["y"] - pad)
    w = min(page.viewport_size["width"] - x, rect["width"] + 2 * pad)
    h = min(page.viewport_size["height"] - y, rect["height"] + 2 * pad)
    folder = os.path.join(OUT, bucket)
    os.makedirs(folder, exist_ok=True)
    path = os.path.join(folder, f"{shot_id}.jpg")
    page.screenshot(path=path, type="jpeg", quality=84, clip={"x": x, "y": y, "width": w, "height": h})
    found = []
    for n, selector in marks:
        b = box(page, selector)
        if b is None:
            raise RuntimeError(f"{bucket}/{shot_id}: mark {selector!r} not found")
        found.append(
            {
                **({"n": n} if n is not None else {}),
                "x": round((b["x"] - x) / w * 100, 2),
                "y": round((b["y"] - y) / h * 100, 2),
                "w": round(b["width"] / w * 100, 2),
                "h": round(b["height"] / h * 100, 2),
            }
        )
    meta[bucket][shot_id] = {"src": f"/tutorials/{bucket}/{shot_id}.jpg", "w": round(w * 1.5), "h": round(h * 1.5), "css": round(w), "marks": found}
    print(f"  {bucket}/{shot_id}  {round(w)}x{round(h)}  marks={len(found)}")


def login(page: Page) -> None:
    page.goto(f"{BASE}/login")
    page.locator('input[type="email"]').fill(EMAIL)
    page.locator('input[type="password"]').fill(PASSWORD)
    page.locator('button[type="submit"]').click()
    page.wait_for_url("**/dashboard", timeout=15000)
    page.wait_for_timeout(800)


def capture_site(browser, locale: str, uid: int) -> None:
    print(f"== {locale}")
    ctx = browser.new_context(viewport=VIEW, device_scale_factor=1.5, locale="zh-CN" if locale == "zh" else "en-US")
    ctx.add_init_script(
        "localStorage.setItem('theme','dark');"
        f"localStorage.setItem('sub2api_locale','{locale}');"
        f"localStorage.setItem('onboarding_guide_v1:{uid}', JSON.stringify({{status:'dismissed',step:'welcome',client:null,keyId:null}}));"
    )
    page = ctx.new_page()
    login(page)

    # 1) 侧栏：「购买套餐」和「兑换」（同一块区域，标注不同的菜单项）
    redeem_link = 'aside a[href="/redeem"]'
    buy_link = 'aside a[href^="https://wzyp.cn"]'
    top = box(page, redeem_link)["y"]
    side = box(page, "aside")
    bottom = box(page, buy_link)
    side_region = {"x": side["x"], "y": top - 14, "width": side["width"], "height": bottom["y"] + bottom["height"] - top + 40}
    shoot(page, locale, "sidebar-buy", side_region, [(None, buy_link)])
    shoot(page, locale, "sidebar-redeem", side_region, [(None, redeem_link)])

    # 2) 兑换页：粘贴兑换码 → 兑换成功
    page.goto(f"{BASE}/redeem")
    page.wait_for_selector("#code")
    page.locator("#code").fill(DEMO_CODE)
    page.evaluate("document.querySelector('[data-tour=\"redeem-form\"]').closest('.card').setAttribute('data-shot', 'redeem-card')")
    shoot(
        page, locale, "redeem-form", '[data-shot="redeem-card"]',
        [(1, "#code"), (2, '[data-tour="redeem-form"] button[type="submit"]')],
        pad=18,
    )
    page.locator('[data-tour="redeem-form"] button[type="submit"]').click()
    page.wait_for_timeout(1800)
    shoot(page, locale, "redeem-ok", {"x": 300, "y": 90, "width": 760, "height": 640}, [])

    # 3) 创建密钥：每个工具各一遍
    for client, provider, group in (("claude-code", "anthropic", "Claude 经济"), ("codex", "openai", "GPT 经济")):
        page.goto(f"{BASE}/keys")
        page.wait_for_selector('[data-tour="keys-create-btn"]')
        if client == "claude-code":
            shoot(page, locale, "keys-create", {"x": 256, "y": 70, "width": VIEW["width"] - 256, "height": 190}, [(None, '[data-tour="keys-create-btn"]')])
        page.locator('[data-tour="keys-create-btn"]').click()
        page.wait_for_selector('[data-tour="key-form-name"]')
        page.locator('[data-tour="key-form-name"]').fill(client)
        page.locator(f'label:has(input[value="{provider}"])').click()
        page.wait_for_timeout(300)
        page.locator('[data-tour="key-form-group"]').click()
        page.get_by_text(group, exact=False).first.click()
        page.wait_for_timeout(400)
        shoot(
            page, locale, f"keys-form-{client}", ".modal-content",
            [
                (1, '[data-tour="key-form-name"]'),
                (2, '[data-tour="key-form-provider"] .grid'),
                (3, '[data-tour="key-form-group"]'),
                (4, '[data-tour="key-form-submit"]'),
            ],
            pad=16,
        )
        page.locator('[data-tour="key-form-submit"]').click()
        page.wait_for_selector('[data-tour="key-import-ccs"]')
        page.wait_for_timeout(900)
        shoot(page, locale, f"keys-import-{client}", {"x": 256, "y": 150, "width": VIEW["width"] - 256, "height": 290}, [(None, '[data-tour="key-import-ccs"]')])

        if client == "codex":
            key_id = sql(f"select id from api_keys where user_id={uid} order by id desc limit 1")
            sql(
                "insert into usage_logs (user_id, api_key_id, account_id, model, input_tokens, output_tokens, total_cost, actual_cost, input_cost, output_cost) "
                f"values ({uid}, {key_id}, 1, 'gpt-5.5', 21, 6, 0.0001, 0.0001, 0.00005, 0.00005)"
            )
            page.goto(f"{BASE}/usage")
            page.wait_for_selector("table tbody tr", timeout=15000)
            # 页面上有好几张图表里的小表格，使用记录是最后一张
            page.evaluate(
                """() => {
                  const t = [...document.querySelectorAll('table')].pop();
                  t.setAttribute('data-shot', 'records');
                  t.querySelector('tbody tr').setAttribute('data-shot', 'record-row');
                  t.scrollIntoView({block: 'center'});
                }"""
            )
            page.wait_for_timeout(600)
            table = box(page, '[data-shot="records"]')
            shoot(
                page, locale, "usage",
                {"x": 256, "y": max(0, table["y"] - 60), "width": VIEW["width"] - 256, "height": min(VIEW["height"] - max(0, table["y"] - 60), table["height"] + 90)},
                [(None, '[data-shot="record-row"]')],
            )
        sql(f"delete from usage_logs where user_id={uid}")
        sql(f"delete from api_keys where user_id={uid}")
    ctx.close()


def main() -> int:
    uid = int(sql("select id from users where email='shots@local.test'"))
    with sync_playwright() as p:
        browser = p.chromium.launch(channel="chrome", headless=True)
        for locale in ("zh", "en"):
            capture_site(browser, locale, uid)
        browser.close()
    # 小铺的截图（shared）不是脚本截的，保留 shots.json 里已有的 shared 部分
    previous = json.load(open(META)) if os.path.exists(META) else {}
    meta["shared"] = previous.get("shared", {})
    json.dump(meta, open(META, "w"), ensure_ascii=False, indent=2)
    print(f"wrote {META}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
