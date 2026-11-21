"""s4：本地新用户登录 → 新手引导 → 创建密钥。需要本地前端 3000 / 后端 8085 / Postgres 在跑。
测试用户 video-demo@local.test 开头新建、结尾删除（连同它的密钥）。运行：/usr/bin/python3 tools/video/seg_s4.py"""
import os, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from rec import Session
import _local as L

uid = L.setup(balance=1.0)          # 余额 1 刀：像注册时用体验码拿到的
try:
    with Session("s4") as r:
        pg = r.page
        L.install_rename(r.ctx)
        L.login(pg)
        # 模拟「刚注册」：待提示标记 + 没有任何引导进度，进控制台引导自动弹出
        pg.evaluate(f"sessionStorage.setItem('tutorial_welcome_pending','{uid}')")
        pg.goto(L.BASE + "/dashboard"); pg.wait_for_selector(".og-card", timeout=15000); pg.wait_for_timeout(1500)
        r.start()

        # 第 1 拍：引导欢迎卡片自动弹出，点「开始」
        r.beat()
        r.hold(0.3)
        L.click_fast(r, pg.get_by_role("button", name="开始", exact=True), 0.7)
        pg.wait_for_timeout(700)

        # 第 2 拍：选 Claude Code，引导指向侧栏「API 密钥」，点进去
        r.beat()
        r.hold(0.3)
        L.click_fast(r, pg.get_by_role("button", name="Claude Code"), 0.6)
        pg.wait_for_timeout(700); r.hold(0.3)
        L.click_fast(r, pg.locator('[data-tour="sidebar-my-keys"]'), 0.6)
        pg.wait_for_selector('[data-tour="keys-create-btn"]'); pg.wait_for_timeout(600)

        # 第 3 拍：点「创建密钥」，给密钥起名字
        r.beat()
        L.click_fast(r, pg.locator('[data-tour="keys-create-btn"]'), 0.5)
        pg.wait_for_selector('[data-tour="key-form-name"]'); pg.wait_for_timeout(500)
        name = pg.locator('[data-tour="key-form-name"]')
        L.click_fast(r, name, 0.4)
        L.typewrite(r, "我的 Claude Code", 0.8)

        # 第 4 拍：厂商已是 Anthropic（鼠标指一下），再选分组
        r.beat()
        r.move_to(pg.locator('[data-tour="key-form-provider"] .grid > *').first, 0.6)
        r.hold(0.6)
        L.click_fast(r, pg.locator('[data-tour="key-form-group"]'), 0.6)
        pg.wait_for_selector(".select-option"); r.hold(0.5)
        L.click_fast(r, pg.locator(".select-option").first, 0.6)
        pg.wait_for_timeout(300)

        # 第 5 拍：点「创建」，引导浮层淡出，停在列表里出现的新密钥
        r.beat()
        L.click_fast(r, pg.locator('[data-tour="key-form-submit"]'), 0.6)
        pg.wait_for_selector("table tbody tr", timeout=15000); pg.wait_for_timeout(500)
        pg.add_style_tag(content=".og-root{opacity:0!important;transition:opacity .6s;pointer-events:none!important}")
        r.hold(0.8)
        pg.wait_for_timeout(600)
        r.move_to(pg.locator("table tbody tr").first, 0.8, dx=-60)
finally:
    L.cleanup()
