"""s4b：新手引导的「先把软件装好」页，指到 Node.js 的下载按钮（只指、不点，不会真的下载）。
需要本地前端 3000 / 后端 8085 / Postgres 在跑。运行：/usr/bin/python3 tools/video/seg_s4b.py（抖音版加 VARIANT=dy）"""
import os, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from rec import Session
import _local as L

uid = L.setup(balance=1.0)
try:
    kid = L.seed_key(uid)
    with Session("s4b") as r:
        pg = r.page
        L.install_rename(r.ctx)
        L.login(pg)
        # 进度直接放在「问装了吗」这一步（Key 已建好）
        pg.evaluate(f"localStorage.setItem('onboarding_guide_v1:{uid}', JSON.stringify({{status:'active',step:'ccs',client:'claude-code',keyId:{kid},history:[]}}))")
        pg.goto(L.BASE + "/keys"); pg.wait_for_selector(".og-root", timeout=15000); pg.wait_for_timeout(1800)
        r.start()

        # 第 1 拍：「都装好了吗？」→ 点「还没有，带我装」
        r.beat()
        r.hold(0.5)
        L.click_fast(r, pg.get_by_role("button", name="还没有，带我装"), 0.7)
        pg.wait_for_selector("text=安装 Node.js", timeout=15000); pg.wait_for_timeout(1200)   # 等 Node 版本信息加载

        # 第 2 拍：下载页，鼠标从上往下扫一遍
        r.beat()
        r.move(640, 160, 0.6)
        r.move(640, 330, 0.8)

        # 第 3 拍：指到 Node.js 的下载按钮并框住（不点击）
        r.beat()
        node = pg.locator("a, button").filter(has_text="Node 22").first
        r.move_to(node, 0.7)
        r.highlight(node, pad=5)

        # 第 4 拍：框改到标题「已装 22 或更高版本可跳过」
        r.beat()
        r.clear_highlight()
        title = pg.get_by_text("安装 Node.js").first
        r.move_to(title, 0.6, dx=60)
        r.highlight(title.locator("xpath=.."), pad=6)
finally:
    L.cleanup()
