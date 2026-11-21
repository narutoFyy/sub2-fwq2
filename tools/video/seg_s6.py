"""s6：本地密钥页指到「导入到 CCS」并框住 + 两张现成截图（本地 HTML 展示，缓慢放大）。
运行：/usr/bin/python3 tools/video/seg_s6.py"""
import os, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from rec import Session, DY
import _local as L

HERE = os.path.dirname(os.path.abspath(__file__))
SHARED = os.path.join(L.ROOT, "frontend/public/tutorials/shared")
def shot_url(name):
    src = f"{HERE}/pages/data/ccswitch-import-dy.jpg" if (DY and name == "ccswitch-import.jpg") else f"{SHARED}/{name}"   # 抖音版：用抹掉站名/网址的导入确认图
    return f"file://{HERE}/pages/shot.html?i={name}#{src}"

def zoom(r, s0, s1, p0, p1, sec):
    """逐帧放大：缩放 s0→s1，聚焦点 p0→p1（图上比例坐标）。"""
    n = max(2, int(sec * 10))
    for i in range(1, n + 1):
        k = i / n; k = k * k * (3 - 2 * k)
        r.page.evaluate("([s,x,y]) => setZoom(s,x,y)", [s0 + (s1 - s0) * k, p0[0] + (p1[0] - p0[0]) * k, p0[1] + (p1[1] - p0[1]) * k]); r.snap()

def open_shot(r, name):
    r.page.goto(shot_url(name)); r.page.wait_for_function("document.getElementById('im').complete && document.getElementById('im').naturalWidth>0")
    r.page.evaluate("setZoom(1,.5,.5)"); r.page.mouse.move(*r.pos)

def img_point(r, fx, fy):
    b = r.page.locator("#im").bounding_box()   # 当前（含缩放）图的位置
    return b["x"] + b["width"] * fx, b["y"] + b["height"] * fy

uid = L.setup(balance=1.0)
try:
    L.seed_key(uid)
    with Session("s6") as r:
        pg = r.page
        L.install_rename(r.ctx)
        L.login(pg); L.dismiss_guide(pg, uid)
        pg.goto(L.BASE + "/keys"); pg.wait_for_selector('[data-tour="key-import-ccs"]'); pg.wait_for_timeout(1200)
        r.start()

        # 第 1 拍：密钥页，鼠标移到那一行的「导入到 CCS」并框住（不点击）
        r.beat()
        r.hold(0.3)
        btn = pg.locator('[data-tour="key-import-ccs"]').first
        r.move_to(btn, 0.9)
        r.highlight(btn, pad=8)

        # 第 2 拍：导入确认图，指向「导入」按钮
        r.beat()
        r.clear_highlight()
        open_shot(r, "ccswitch-import.jpg")
        r.hold(0.3)
        x, y = img_point(r, 0.88, 0.95)
        zoom_n = 0
        r.move(x, y, 0.9)
        zoom(r, 1.0, 1.05, (.5, .5), (.5, .5), max(0.3, r.left() - 0.2))

        # 第 3 拍：CC Switch 图，鼠标指到左边的 Claude Code
        r.beat()
        open_shot(r, "ccswitch-enable.jpg")
        r.hold(0.2)
        x, y = img_point(r, 0.08, 0.112)
        r.move(x, y, 0.9)
        zoom(r, 1.0, 1.05, (.5, .5), (.5, .5), max(0.3, r.left() - 0.2))

        # 第 4 拍：放大到「▶ / 使用中」那一块，鼠标指过去
        r.beat()
        zoom(r, 1.05, 2.4, (.5, .5), (.76, .47), min(1.4, max(0.6, r.left() - 1.2)))
        x, y = img_point(r, 0.865, 0.47)
        r.move(x, y, 0.7)
finally:
    L.cleanup()
