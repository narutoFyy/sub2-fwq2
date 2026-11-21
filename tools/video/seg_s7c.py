#!/usr/bin/python3
"""s7c 结尾（短版"用上了"）：真实回放 hello.jsonl——敲 claude、输入"你好"、真实回复。
先跑 rec_hello.py 生成 /Users/Shared/cc-demo/hello.jsonl。运行：/usr/bin/python3 tools/video/seg_s7c.py"""
import os, shutil, sys
HERE = os.path.dirname(os.path.abspath(__file__)); sys.path.insert(0, HERE)
from termplay import Player, encode
SRC = "/Users/Shared/cc-demo/hello.jsonl"; ROOT = os.path.dirname(os.path.dirname(HERE)); OUT = os.path.join(ROOT, "docs/marketing/video/seg/s7c.mp4")
FR = "/tmp/seg_s7c_frames"; shutil.rmtree(FR, ignore_errors=True)
# plan: (真实起, 真实止, 画面秒数, 角标)；out=0 表示瞬间放完。3.9~44.9 是等待启动的空白，跳过
with Player(SRC, FR) as p:
    p.run([(0, 1.2, 0.3, ""), (1.2, 3.9, 1.5, ""), (3.9, 44.9, 0, "")]); p.hold(0.5)
    p.run([(44.9, 46.3, 1.2, ""), (46.3, 50.4, 2.6, "")]); p.hold(1.4)
encode(FR, OUT); print("ok", OUT)
