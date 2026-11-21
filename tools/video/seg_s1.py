#!/usr/bin/python3
"""s1 开场：一张静态截图——真实 Claude Code 正在改代码、等你确认（来自 usage.jsonl 真实录制）。
先跑 rec_usage.py 生成 /Users/Shared/cc-demo/usage.jsonl。运行：/usr/bin/python3 tools/video/seg_s1.py"""
import base64, os, re, shutil, sys
HERE = os.path.dirname(os.path.abspath(__file__)); sys.path.insert(0, HERE)
from termplay import Player, load, encode
from ptyrec import ANSI
SRC = "/Users/Shared/cc-demo/usage.jsonl"; ROOT = os.path.dirname(os.path.dirname(HERE)); OUT = os.path.join(ROOT, "docs/marketing/video/seg/s1.mp4")
FR = "/tmp/seg_s1_frames"; shutil.rmtree(FR, ignore_errors=True)
meta, ev = load(SRC); buf = ""; ask = None
for t, b in ev:
    buf = (buf + ANSI.sub("", base64.b64decode(b).decode("utf-8", "ignore")))[-3000:]
    if re.search(r"Do\s*you\s*want", buf): ask = t; break
with Player(SRC, FR) as p:
    p.feed(ask + 2.5); p.hold(13.2)
encode(FR, OUT); print("ok", OUT)
