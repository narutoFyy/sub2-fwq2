#!/usr/bin/python3
"""录开场里那个游戏的游玩画面：真实打开 Claude Code 写出的 index.html。
为了画面流畅，用"手动步进"的方式推进游戏时间（每帧 1/30 秒），鼠标由脚本里的"替身玩家"控制
（它只读取游戏里敌人/子弹的位置来躲避和瞄准，游戏代码本身没有改动，录制副本只多了一行读取状态的代码）。
输出 /Users/Shared/cc-demo/gameplay/f%05d.jpg（30fps），meta.json 记录点击开始的帧号。"""
import json, math, os, shutil, sys
from playwright.sync_api import sync_playwright
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__))); from rec import CURSOR
SRC = "/Users/Shared/cc-demo/game/index.html"; BOT = "/Users/Shared/cc-demo/game_bot/index.html"
OUT = "/Users/Shared/cc-demo/gameplay"; shutil.rmtree(OUT, ignore_errors=True); os.makedirs(OUT); os.makedirs(os.path.dirname(BOT), exist_ok=True)
s = open(SRC, encoding="utf-8").read(); i = s.rindex("})();")
open(BOT, "w", encoding="utf-8").write(s[:i] + "window.__dbg=()=>({state,player,enemies,eBullets,powerups,W,H});\n" + s[i:])
STEP = """(() => { let now = 0, q = []; window.requestAnimationFrame = cb => { q.push(cb); return q.length; };
  performance.now = () => now; const dn = Date.now; window.__step = ms => { now += ms; const c = q; q = []; c.forEach(f => f(now)); }; })();"""
PLAY_SEC = 45; FPS = 30
with sync_playwright() as p:
    b = p.chromium.launch(channel="chrome", headless=True); ctx = b.new_context(viewport={"width": 960, "height": 540}, device_scale_factor=2)
    ctx.add_init_script(STEP + CURSOR); pg = ctx.new_page(); pg.goto("file://" + BOT); pg.wait_for_timeout(500)
    n = [0]
    def shot(adv=True):
        if adv: pg.evaluate(f"__step({1000 / FPS})")
        pg.screenshot(path=os.path.join(OUT, f"f{n[0]:05d}.jpg"), type="jpeg", quality=88); n[0] += 1
    pg.mouse.move(750, 150)
    for _ in range(25): shot()                              # 菜单停一下
    bb = pg.locator('#menu button').first.bounding_box(); bx, by = bb['x'] + bb['width'] / 2, bb['y'] + bb['height'] / 2
    for i in range(1, 31):                                   # 鼠标滑到"开始游戏"
        k = i / 30; k = k * k * (3 - 2 * k); pg.mouse.move(750 + (bx - 750) * k, 150 + (by - 150) * k); shot()
    for _ in range(8): shot()
    pg.mouse.click(bx, by); pg.evaluate("document.getElementById('__cur').style.display='none'"); start_n = n[0]
    def bot():
        d = pg.evaluate("(()=>{const g=__dbg();if(g.state!=='play')return null;const m=a=>a.map(o=>({x:o.x,y:o.y,r:o.r||0,vx:o.vx||0,vy:o.vy||0}));return {p:{x:g.player.x,y:g.player.y},W:g.W,H:g.H,e:m(g.enemies),b:m(g.eBullets),u:m(g.powerups)}})()")
        if not d: return None
        px, py, W, H = d["p"]["x"], d["p"]["y"], d["W"], d["H"]; tgt = None
        for u in sorted(d["u"], key=lambda u: -u["y"]):
            if u["y"] > H * 0.3: tgt = u["x"]; break
        if tgt is None:
            ens = [e for e in d["e"] if e["y"] > 0]
            tgt = max(ens, key=lambda e: e["y"])["x"] if ens else px
        best, bs = px, 1e18
        for cx in range(40, int(W) - 40, 20):
            danger = 0
            for o in d["b"] + d["e"]:
                for f in (0.0, 0.25, 0.5):
                    ox, oy = o["x"] + o["vx"] * f, o["y"] + o["vy"] * f; dx, dy = ox - cx, oy - py
                    danger += 1000 if (abs(dy) < 90 + o["r"] and abs(dx) < 50 + o["r"]) else 3000 / (dx * dx + dy * dy + 1)
            sc = danger + abs(cx - tgt) * 0.02 + abs(cx - px) * 0.01
            if sc < bs: bs, best = sc, cx
        return best, H * 0.82
    for _ in range(PLAY_SEC * FPS):
        r = bot()
        if r is None: break
        pg.mouse.move(*r); shot()
    for _ in range(15): shot()
    b.close()
json.dump({"start_n": start_n, "total": n[0], "fps": FPS}, open(os.path.join(OUT, "meta.json"), "w"))
print("frames", n[0], "start_n", start_n)
