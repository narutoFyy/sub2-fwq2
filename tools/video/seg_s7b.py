"""s7b：本地「使用记录」，几条 claude-sonnet-5-5 调用记录（种子数据），鼠标指到第一条。
测试用户和数据开头建、结尾删。运行：/usr/bin/python3 tools/video/seg_s7b.py"""
import os, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from rec import Session
import _local as L

uid = L.setup(balance=0.93)
try:
    kid = L.seed_key(uid)
    L.seed_usage(uid, kid, [(2, 1860, 642, 5200), (4, 3420, 1180, 9100), (7, 920, 310, 2800), (11, 5600, 1930, 14300), (16, 2480, 870, 6700), (23, 1330, 450, 3900)])
    with Session("s7b") as r:
        pg = r.page
        L.install_rename(r.ctx)
        L.login(pg); L.dismiss_guide(pg, uid)
        pg.goto(L.BASE + "/dashboard"); pg.wait_for_timeout(1500)
        r.start()

        r.beat()
        r.hold(0.3)
        L.click_fast(r, pg.locator('aside a[href="/usage"]'), 0.6)
        pg.wait_for_selector("table tbody tr", timeout=15000); pg.wait_for_timeout(1200)
        # 页面上有好几张图表小表格，使用记录是最后一张
        pg.evaluate("""() => { const t=[...document.querySelectorAll('table')].pop(); t.setAttribute('data-shot','records');
            t.querySelector('tbody tr').setAttribute('data-shot','record-row'); }""")
        row = pg.locator('[data-shot="record-row"]')
        row.scroll_into_view_if_needed()
        y = row.bounding_box()["y"]
        if y > 420: r.scroll(y - 360, 0.6)
        pg.wait_for_timeout(300)
        r.move_to(row, 0.7, dx=-120)
        class Clamp:   # 表格比窗口宽，框只画到窗口内
            def bounding_box(self):
                b = row.bounding_box(); b["width"] = min(b["width"], 1272 - b["x"] - 6); return b
        r.highlight(Clamp(), pad=3)
finally:
    L.cleanup()
