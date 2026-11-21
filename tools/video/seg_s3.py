"""s3：线上注册页，只在表单里打字，绝不提交。运行：/usr/bin/python3 tools/video/seg_s3.py
安全：用 route 拦掉注册 / 发验证码类接口（即使误触也发不出去）；全程不按回车、不点「继续」。"""
import math, os, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from rec import Session, FPS, DY

BASE = "https://yf-mail.com"
EMAIL, PASSWORD, PROMO = "demo@example.com", "Demo#12345", "STONE-BILI"

def block(route):
    route.abort()

def typewrite(r, text, sec):
    """每帧打几个字，总共约 sec 秒（从不输入换行）。"""
    frames = max(1, int(sec * FPS)); k = max(1, math.ceil(len(text) / frames))
    for i in range(0, len(text), k):
        r.page.keyboard.type(text[i:i + k]); r.snap()

def click_fast(r, loc, move=0.5):
    r.move_to(loc, move); r.hold(0.1); loc.click(); r.hold(0.3)

with Session("s3") as r:
    pg = r.page
    r.ctx.route("**/api/v1/auth/register*", block)
    r.ctx.route("**/api/v1/auth/send-verify-code*", block)
    r.ctx.route("**/api/v1/auth/login*", block)
    pg.goto(BASE + "/", wait_until="networkidle"); pg.wait_for_timeout(1500)
    r.start()

    # 第 1 拍：首页 → 点右上角「登录」→ 登录页点「注册」→ 注册页，填邮箱和密码
    r.beat()
    r.hold(0.2)
    click_fast(r, pg.locator('header a[href="/login"], nav a[href="/login"]').first)
    pg.wait_for_url("**/login", timeout=15000); pg.wait_for_selector('a[href="/register"]'); pg.wait_for_timeout(500)
    click_fast(r, pg.locator('a[href="/register"]').first)
    pg.wait_for_url("**/register", timeout=15000); pg.wait_for_selector("#email"); pg.wait_for_timeout(600)
    click_fast(r, pg.locator("#email"), 0.5)
    typewrite(r, EMAIL, 0.8)
    click_fast(r, pg.locator("#password"), 0.4)
    typewrite(r, PASSWORD, 0.6)

    if DY:        # 抖音版：只演示邮箱 + 密码，不出现优惠码；一拍到底
        click_fast(r, pg.locator("#confirmPassword"), 0.4)
        typewrite(r, PASSWORD, 0.5)
        r.hold(0.8)
    else:
        # 第 2 拍：确认密码，再点「优惠码」输入 STONE-BILI
        r.beat()
        click_fast(r, pg.locator("#confirmPassword"), 0.4)
        typewrite(r, PASSWORD, 0.5)
        click_fast(r, pg.locator("#promo_code"), 0.5)
        typewrite(r, PROMO, 1.0)

        # 第 3 拍：等「有效！注册后将获得 … 赠送余额」出现，框住它
        r.beat()
        msg = pg.get_by_text("有效！注册后将获得").first
        for _ in range(40):          # 最多等 4 秒，等待期间照常出帧
            if msg.count() and msg.is_visible(): break
            r.snap()
        box = msg.locator("xpath=ancestor::div[1]")
        r.move(box.bounding_box()["x"] + 60, box.bounding_box()["y"] + 50, 0.5)
        r.highlight(box, pad=4)
