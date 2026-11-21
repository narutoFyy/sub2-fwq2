"""录屏小框架：用 Playwright 连续截图，按口播分拍（beat）控制每段画面时长。
只用系统 Chrome（channel=chrome），不需要 playwright install。必须用 /usr/bin/python3 运行。

用法见 rec_plaza.py 旁的各段脚本：
    from rec import Session
    with Session("s2") as r:            # 读 script.json 里 id=s2 的分拍
        pg = r.page
        pg.goto(...)                    # 第一拍开始前的准备不计入画面，准备好后调用 r.start()
        r.start()
        r.beat()                        # 进入第 1 拍
        r.move_to(locator); r.click(locator)
        r.beat()                        # 补帧到第 1 拍结束，进入第 2 拍
        ...
    # 退出时补帧到最后一拍结束，并把帧合成 docs/marketing/video/seg/s2.mp4（无声）
每拍时长 = 该段配音时长 × 该拍字数占比；配音文件 docs/marketing/video/audio/<id>.wav 不存在时按每秒 6 字估算。
一拍内动作超时会占用后面拍的时间，所以每拍的动作要短于该拍时长（r.left() 可查剩余秒数）。
"""
import json, os, subprocess, shutil, time
from playwright.sync_api import sync_playwright

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
VID = os.path.join(ROOT, "docs/marketing/video")
FPS = 30
DY = os.environ.get("VARIANT") == "dy"        # 抖音版：抹掉站名/网址/体验码/价格，输出到 seg_dy
SEGD = "seg_dy" if DY else "seg"
CSS_W, CSS_H, SCALE = 1280, 720, 1.5          # 截图为 1920x1080
CURSOR = """
(() => { const mk = () => { if (document.getElementById('__cur')) return;
  const c = document.createElement('div'); c.id='__cur';
  c.style.cssText='position:fixed;z-index:2147483647;left:0;top:0;width:22px;height:22px;pointer-events:none;transform:translate(640px,360px)';
  c.innerHTML='<svg width="22" height="22" viewBox="0 0 24 24"><path d="M4 2l15 9-6.5 1.6L16 20l-3 1.4-3.4-7.4L5 18z" fill="#fff" stroke="#111" stroke-width="1.4"/></svg>';
  document.documentElement.appendChild(c);
  addEventListener('mousemove', e => { c.style.transform = `translate(${e.clientX}px,${e.clientY}px)`; }, true); };
  if (document.documentElement) mk(); addEventListener('DOMContentLoaded', mk); })();
"""

# 录制时把站内自带、但短视频平台容易判违规的字样处理掉（只影响录屏画面，不改网站）
SCRUB_DY = """
(() => { const R = [['石头中转站','AI 平台'],['yf-mail.com','••••••••'],['Claude 特惠','Claude']]; const HIDE = ['兑换','限时活动','购买套餐'];
  const fix = () => { if (!document.body) return;
    const w = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT); let n;
    while ((n = w.nextNode())) { let t = n.nodeValue; if (!t) continue;
      for (const [a,b] of R) if (t.includes(a)) { t = t.split(a).join(b); n.nodeValue = t; }
      if (HIDE.includes(t.trim())) { const e = n.parentElement && (n.parentElement.closest('a,button,li') || n.parentElement); if (e) e.style.display = 'none'; } } };
  const go = () => { fix(); new MutationObserver(fix).observe(document.documentElement, {childList:true, subtree:true, characterData:true}); };
  if (document.body) go(); else addEventListener('DOMContentLoaded', go); })();
"""

SCRUB = """
(() => { const fix = () => { if (!document.body) return;
  const w = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT); let n;
  while ((n = w.nextNode())) { const t = n.nodeValue; if (!t) continue;
    if (t.includes('越便宜')) n.nodeValue = t.replace('越便宜', '单价越低');
    if (t.includes('玄学')) { const e = n.parentElement && (n.parentElement.closest('button,a,li') || n.parentElement); if (e) e.style.display = 'none'; }
    if (/加群|QQ ?群/.test(t)) { let e = n.parentElement; if (e && e.innerText.length < 60) e.style.visibility = 'hidden'; } } };
  const go = () => { fix(); new MutationObserver(fix).observe(document.documentElement, {childList:true, subtree:true, characterData:true}); };
  if (document.body) go(); else addEventListener('DOMContentLoaded', go); })();
"""

def wav_dur(path):
    try:
        return float(subprocess.check_output(["ffprobe","-v","error","-show_entries","format=duration","-of","csv=p=0",path]).strip())
    except Exception:
        return None

class Session:
    def __init__(self, seg_id, locale="zh", init_script="", cursor=True):
        self.id = seg_id
        spec = next(s for s in json.load(open(os.path.join(os.path.dirname(__file__), "script_dy.json" if DY else "script.json"), encoding="utf-8")) if s["id"] == seg_id)
        self.beats = spec["beats"]
        total = wav_dur(os.path.join(VID, "audio_dy" if DY and os.path.exists(os.path.join(VID, "audio_dy", seg_id + ".wav")) else "audio", seg_id + ".wav")) or sum(len(b) for b in self.beats) / 6.0
        chars = sum(len(b) for b in self.beats)
        # 每拍结束时应累计到的帧数（配音前有 0.3 秒静音，结尾多留 0.6 秒）
        acc, self.ends = 0.3, []
        for b in self.beats:
            acc += total * len(b) / chars; self.ends.append(int(round(acc * FPS)))
        self.ends[-1] += int(0.6 * FPS)
        os.makedirs(os.path.join(VID, SEGD), exist_ok=True); self.frames_dir = os.path.join(VID, SEGD, "_frames_" + seg_id)
        shutil.rmtree(self.frames_dir, ignore_errors=True); os.makedirs(self.frames_dir)
        self.n = 0; self.cur = -1; self.pos = [CSS_W / 2, CSS_H / 2]; self.started = False
        self._init = f"try{{localStorage.setItem('sub2api_locale','{locale}')}}catch(e){{}};" + (CURSOR if cursor else "") + SCRUB + (SCRUB_DY if DY else "") + init_script

    def __enter__(self):
        self._pw = sync_playwright().start()
        self.browser = self._pw.chromium.launch(channel="chrome", headless=True)
        self.ctx = self.browser.new_context(viewport={"width": CSS_W, "height": CSS_H}, device_scale_factor=SCALE, locale="zh-CN", color_scheme="dark")
        self.ctx.add_init_script(self._init)
        self.page = self.ctx.new_page()
        return self

    # ---- 帧 ----
    def start(self):
        self.started = True; self.page.mouse.move(*self.pos)
    def snap(self):
        if not self.started: return
        self.page.screenshot(path=os.path.join(self.frames_dir, f"f{self.n:05d}.jpg"), type="jpeg", quality=90); self.n += 1
    def hold(self, sec):
        for _ in range(max(1, int(round(sec * FPS)))): self.snap()
    def left(self):
        """当前拍还剩多少秒。"""
        return max(0.0, (self.ends[self.cur] - self.n) / FPS) if self.cur >= 0 else 0.0
    def beat(self):
        """补帧到当前拍结束，进入下一拍。返回新一拍的秒数。"""
        if self.cur >= 0:
            while self.n < self.ends[self.cur]: self.snap()
        self.cur += 1
        return (self.ends[self.cur] - self.n) / FPS

    # ---- 动作（都会出帧）----
    def move(self, x, y, sec=0.8):
        steps = max(2, int(sec * FPS)); x0, y0 = self.pos
        for i in range(1, steps + 1):
            k = i / steps; k = k * k * (3 - 2 * k)
            self.page.mouse.move(x0 + (x - x0) * k, y0 + (y - y0) * k); self.snap()
        self.pos = [x, y]
    def move_to(self, locator, sec=0.8, dx=0, dy=0):
        locator.scroll_into_view_if_needed(); b = locator.bounding_box()
        self.move(b["x"] + b["width"] / 2 + dx, b["y"] + b["height"] / 2 + dy, sec)
    def click(self, locator, sec=0.8):
        self.move_to(locator, sec); self.hold(0.2); locator.click(); self.hold(0.3)
    def type(self, locator, text, sec=1.2):
        self.click(locator, 0.6); per = max(1, int(sec * FPS / max(1, len(text))))
        for ch in text:
            self.page.keyboard.type(ch)
            for _ in range(per): self.snap()
    def scroll(self, dy, sec=1.0):
        steps = max(2, int(sec * FPS))
        for _ in range(steps):
            self.page.mouse.wheel(0, dy / steps); self.snap()
    def highlight(self, locator, pad=6):
        """给元素加一圈亮框（下一次调用 clear_highlight 前一直在）。"""
        b = locator.bounding_box()
        self.page.evaluate("""([x,y,w,h]) => { let d=document.getElementById('__hl'); if(!d){d=document.createElement('div');d.id='__hl';document.documentElement.appendChild(d)}
          d.style.cssText=`position:fixed;z-index:2147483646;pointer-events:none;border:3px solid #ff5a36;border-radius:10px;box-shadow:0 0 0 4px rgba(255,90,54,.25);left:${x}px;top:${y}px;width:${w}px;height:${h}px`; }""",
          [b["x"] - pad, b["y"] - pad, b["width"] + 2 * pad, b["height"] + 2 * pad])
    def clear_highlight(self):
        self.page.evaluate("document.getElementById('__hl')?.remove()")

    def __exit__(self, et, ev, tb):
        try:
            if et is None:
                while self.cur < len(self.beats) - 1: self.beat()
                while self.n < self.ends[-1]: self.snap()
        finally:
            self.ctx.close(); self.browser.close(); self._pw.stop()
        if et is None:
            out = os.path.join(VID, SEGD, self.id + ".mp4")
            subprocess.check_call(["ffmpeg","-y","-loglevel","error","-framerate",str(FPS),"-i",os.path.join(self.frames_dir,"f%05d.jpg"),
                                   "-vf","scale=1920:1080","-c:v","libx264","-pix_fmt","yuv420p","-r","30","-crf","20",out])
            shutil.rmtree(self.frames_dir, ignore_errors=True)
            print(f"[{self.id}] {self.n} frames -> {out} ({self.n / FPS:.1f}s)")
        return False
