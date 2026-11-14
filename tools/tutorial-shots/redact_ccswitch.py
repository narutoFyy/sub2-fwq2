#!/usr/bin/env python3
"""把 CC Switch 的真实截图做脱敏（模糊供应商名称、地址、余额），用作教程里「启用供应商」那一步的插图。

用法：python3 tools/tutorial-shots/redact_ccswitch.py <原始截图.png>
原始截图含真实供应商和余额，不要放进仓库；只提交脱敏后的结果和 shots.json 里的标注框。
坐标按 990x773 的截图写死（CC Switch 4.x 的 Codex 页）；换了截图就要重调下面的 BLUR / MARKS。
"""
import base64
import json
import os
import sys

from playwright.sync_api import sync_playwright

SRC = os.path.abspath(sys.argv[1])
ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
OUT = os.path.join(ROOT, "frontend", "public", "tutorials", "shared", "ccswitch-enable.jpg")
META = os.path.join(ROOT, "frontend", "src", "features", "tutorials", "shots.json")
W, H = 990, 773
OUT_H = 748  # 截图最下面一条带着桌面背景的残片，裁掉

# (x, y, w, h)：要模糊的区域。第一行 OpenAI Official 是公共信息，保留。
ROWS = [180, 250, 390, 460, 530, 600, 670, 740]
BLUR = [(0, 19, 100, 18), (690, 70, 100, 22), (112, 698, 76, 18), (296, 328, 522, 44)]
BLUR += [(296, y + 8, 562, 46) for y in ROWS]
# 标注：1 左侧选工具，2 某个供应商右边的 ▶，3「使用中」
MARKS = [(1, 7, 126, 183, 28), (2, 855, 198, 26, 26), (3, 824, 340, 56, 22)]

B64 = base64.b64encode(open(SRC, "rb").read()).decode()
html = f"""<body style="margin:0;background:#1c1c1f"><div style="position:relative;width:{W}px;height:{H}px;overflow:hidden">
<img src="data:image/png;base64,{B64}" width="{W}" height="{H}" style="display:block">
{''.join(f'<div style="position:absolute;left:{x}px;top:{y}px;width:{w}px;height:{h}px;background:rgba(48,48,54,.78);backdrop-filter:blur(10px);border-radius:6px"></div>' for x, y, w, h in BLUR)}
</div></body>"""

with sync_playwright() as p:
    browser = p.chromium.launch(channel="chrome", headless=True)
    page = browser.new_page(viewport={"width": W, "height": H}, device_scale_factor=1)
    page.set_content(html)
    page.wait_for_timeout(400)
    os.makedirs(os.path.dirname(OUT), exist_ok=True)
    page.screenshot(path=OUT, type="jpeg", quality=88, clip={"x": 0, "y": 0, "width": W, "height": OUT_H})
    browser.close()

meta = json.load(open(META))
meta.setdefault("shared", {})["ccswitch-enable"] = {
    "src": "/tutorials/shared/ccswitch-enable.jpg",
    "w": W,
    "h": OUT_H,
    "marks": [{"n": n, "x": round(x / W * 100, 2), "y": round(y / OUT_H * 100, 2), "w": round(w / W * 100, 2), "h": round(h / OUT_H * 100, 2)} for n, x, y, w, h in MARKS],
}
json.dump(meta, open(META, "w"), ensure_ascii=False, indent=2)
print("wrote", OUT)
