#!/usr/bin/python3
"""把 ptyrec 录下的真实终端输出，按时间轴在 xterm.js 里回放并出帧（jpg）。
plan 是 [(src_start, src_end, out_sec, badge)]：把真实时间 src_start~src_end 压缩/拉伸成画面 out_sec 秒，
out_sec=0 表示瞬间放完（用来跳过信任文件夹等无关画面）；badge 例如 "快进 ×8"，会显示在窗口右上角。
必须用 /usr/bin/python3 运行（只有它装了 playwright）。"""
import json, os, shutil
from playwright.sync_api import sync_playwright
HERE = os.path.dirname(os.path.abspath(__file__)); PAGE = "file://" + os.path.join(HERE, "pages", "xterm.html")
FPS = 30

def load(path):
    with open(path) as f:
        meta = json.loads(f.readline()); ev = [json.loads(l) for l in f if l.strip()]
    return meta, ev

class Player:
    def __init__(self, jsonl, frames_dir, title="终端", fs=26, start_n=0):
        self.meta, self.ev = load(jsonl); self.i = 0; self.n = start_n; self.dir = frames_dir; os.makedirs(frames_dir, exist_ok=True)
        self.title, self.fs = title, fs
    def __enter__(self):
        self.pw = sync_playwright().start(); self.b = self.pw.chromium.launch(channel="chrome", headless=True)
        self.pg = self.b.new_page(viewport={"width": 1920, "height": 1080}, device_scale_factor=1)
        self.pg.goto(f"{PAGE}?c={self.meta['cols']}&r={self.meta['rows']}&fs={self.fs}"); self.pg.wait_for_timeout(400)
        self.pg.evaluate("t=>setTitle(t)", self.title); return self
    def feed(self, upto):
        """把时间戳 <= upto 的输出全部写进终端。"""
        buf = []
        while self.i < len(self.ev) and self.ev[self.i][0] <= upto: buf.append(self.ev[self.i][1]); self.i += 1
        for b in buf: self.pg.evaluate("b=>W(b)", b)
    def snap(self):
        self.pg.screenshot(path=os.path.join(self.dir, f"f{self.n:05d}.jpg"), type="jpeg", quality=88); self.n += 1
    def run(self, plan):
        for s0, s1, out, badge in plan:
            if out <= 0: self.feed(s1); continue
            self.pg.evaluate("b=>setBadge(b)", badge or "")
            k = max(1, int(round(out * FPS)))
            for j in range(1, k + 1): self.feed(s0 + (s1 - s0) * j / k); self.snap()
        self.pg.evaluate("b=>setBadge(b)", "")
    def hold(self, sec):
        for _ in range(int(round(sec * FPS))): self.snap()
    def __exit__(self, *a): self.b.close(); self.pw.stop()

def encode(frames_dir, out, fps=FPS):
    import subprocess
    subprocess.check_call(["ffmpeg", "-y", "-loglevel", "error", "-framerate", str(fps), "-i", os.path.join(frames_dir, "f%05d.jpg"),
                           "-vf", "scale=1920:1080", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-r", "30", "-crf", "20", out])
