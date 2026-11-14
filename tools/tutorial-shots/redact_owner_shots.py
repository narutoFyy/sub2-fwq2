#!/usr/bin/env python3
"""用站长自己的两张真实截图做教程插图，并遮住敏感信息：

  shop-code        小铺付款后的「订单详情」页：模糊兑换码（卡密）和订单号；
  ccswitch-import  CC Switch 的「确认导入供应商配置」弹窗：裁掉弹窗外的背景（背景里有其他供应商和余额），模糊 API 密钥。

用法：python3 tools/tutorial-shots/redact_owner_shots.py <小铺订单详情.png> <CC Switch 导入弹窗.png>
原图含真实兑换码、订单号和供应商列表，不要放进仓库；只提交处理后的 jpg 和 shots.json 里的标注框。
坐标按下面两张截图的尺寸（1183x763、582x638）写死，换了截图要重新量。
"""
import base64
import json
import os
import sys

from playwright.sync_api import sync_playwright

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
OUT_DIR = os.path.join(ROOT, "frontend", "public", "tutorials", "shared")
META = os.path.join(ROOT, "frontend", "src", "features", "tutorials", "shots.json")

# crop = (x, y, w, h)；blur / marks 用原图坐标，脚本会换算成相对裁剪区域的位置
SHOTS = {
    "shop-code": {
        "src": sys.argv[1],
        "crop": (25, 75, 1075, 440),
        "blur": [(112, 119, 135, 22), (58, 354, 272, 26)],
        "marks": [(1, 52, 346, 290, 36), (2, 1020, 322, 48, 30)],
        "fill": "rgba(206,212,224,.92)",
    },
    "ccswitch-import": {
        "src": sys.argv[2],
        "crop": (38, 22, 502, 598),  # 右边多裁几像素：弹窗外还露着背景里供应商余额的边角
        "blur": [(214, 260, 218, 26)],
        "marks": [(1, 207, 116, 72, 24), (2, 212, 150, 82, 26), (3, 212, 224, 168, 26), (4, 454, 573, 62, 36)],
        "fill": "rgba(70,70,76,.92)",
    },
}

meta = json.load(open(META))
with sync_playwright() as p:
    browser = p.chromium.launch(channel="chrome", headless=True)
    for shot_id, cfg in SHOTS.items():
        cx, cy, cw, ch = cfg["crop"]
        b64 = base64.b64encode(open(os.path.abspath(cfg["src"]), "rb").read()).decode()
        boxes = "".join(
            f'<div style="position:absolute;left:{x - cx}px;top:{y - cy}px;width:{w}px;height:{h}px;background:{cfg["fill"]};backdrop-filter:blur(9px);border-radius:5px"></div>'
            for x, y, w, h in cfg["blur"]
        )
        html = f'<body style="margin:0"><div id="c" style="position:relative;width:{cw}px;height:{ch}px;overflow:hidden"><img src="data:image/png;base64,{b64}" style="position:absolute;left:{-cx}px;top:{-cy}px">{boxes}</div></body>'
        page = browser.new_page(viewport={"width": cw, "height": ch}, device_scale_factor=1)
        page.set_content(html)
        page.wait_for_timeout(400)
        os.makedirs(OUT_DIR, exist_ok=True)
        page.locator("#c").screenshot(path=os.path.join(OUT_DIR, f"{shot_id}.jpg"), type="jpeg", quality=90)
        page.close()
        meta.setdefault("shared", {})[shot_id] = {
            "src": f"/tutorials/shared/{shot_id}.jpg",
            "w": cw,
            "h": ch,
            "marks": [
                {"n": n, "x": round((x - cx) / cw * 100, 2), "y": round((y - cy) / ch * 100, 2), "w": round(w / cw * 100, 2), "h": round(h / ch * 100, 2)}
                for n, x, y, w, h in cfg["marks"]
            ],
        }
        print("wrote", shot_id)
    browser.close()
json.dump(meta, open(META, "w"), ensure_ascii=False, indent=2)
