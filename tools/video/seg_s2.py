"""s2：公开广场价格。线上只读，不登录。运行：/usr/bin/python3 tools/video/seg_s2.py"""
import os, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from rec import Session

BASE = "https://yf-mail.com"

with Session("s2") as r:
    pg = r.page
    pg.goto(BASE + "/", wait_until="networkidle"); pg.wait_for_timeout(1500)
    r.start()

    # 第 1 拍：首页停一下，鼠标移到导航「广场」并点进去
    r.beat()
    r.hold(0.8)
    nav = pg.locator('a[href="/plaza"]').first
    r.click(nav, 0.9)
    pg.wait_for_url("**/plaza", timeout=15000); pg.wait_for_selector(".plaza-tabs button", timeout=15000)
    pg.wait_for_timeout(800)

    # 第 2 拍：广场页，鼠标扫过分组标签，标签条缓缓向右滚（让 Claude 特惠露出来）
    r.beat()
    r.move(640, 404, 0.8)
    tab = pg.locator(".plaza-tabs button", has_text="Claude 特惠")
    target = pg.evaluate("""() => { const t=[...document.querySelectorAll('.plaza-tabs button')].find(b=>b.innerText.includes('Claude 特惠'));
        const s=document.querySelector('.plaza-tabs'); return Math.max(0, t.offsetLeft - 120 - (s.offsetLeft||0)); }""")
    steps = 14
    for i in range(1, steps + 1):
        pg.evaluate("([i,steps,t]) => { document.querySelector('.plaza-tabs').scrollTo({left: t*i/steps, behavior: 'instant'}) }", [i, steps, target]); r.snap()
    r.hold(0.3)

    # 第 3 拍：点「Claude 特惠（Kiro）」标签
    r.beat()
    r.click(tab, 0.8)
    pg.wait_for_timeout(600)

    # 第 4 拍：滚到 claude-sonnet-5-5 那一行，指过去并框住
    r.beat()
    row = pg.locator(".pt-row", has=pg.get_by_text("claude-sonnet-5-5", exact=True)).first
    y = row.bounding_box()["y"]
    r.scroll(y - 330, 1.0)
    pg.wait_for_timeout(200)
    r.move_to(row.locator(".pt-name"), 0.8, dx=150)
    r.highlight(row, pad=4)
    r.hold(0.3)

    # 第 5 拍：框收起，滚回表头，指向「实付 = 官方价 × 0.4」说明
    r.beat()
    r.clear_highlight()
    r.scroll(300 - pg.evaluate("window.scrollY"), 0.9)
    note = pg.get_by_text("实付 = 官方价 × 0.4", exact=True).first
    class TextBox:  # 只框文字本身，不框整行
        def bounding_box(self):
            return note.evaluate("e => { const g=document.createRange(); g.selectNodeContents(e); const b=g.getBoundingClientRect(); return {x:b.x,y:b.y,width:b.width,height:b.height} }")
    r.move(TextBox().bounding_box()["x"] + 40, TextBox().bounding_box()["y"] + 22, 0.7)
    r.highlight(TextBox(), pad=6)
