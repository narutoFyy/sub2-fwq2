#!/usr/bin/python3
"""s5 装软件：前三拍是真实终端（rec_install.py 录的真实 node -v / npm 安装 / claude --version，在 xterm.js 里回放），
第四拍是 CC Switch 说明卡。先跑 rec_install.py。运行：/usr/bin/python3 tools/video/seg_s5.py"""
import os, shutil, sys
HERE = os.path.dirname(os.path.abspath(__file__)); sys.path.insert(0, HERE)
from termplay import Player
from rec import Session, FPS
from vt import CARDS

SRC = "/Users/Shared/cc-demo/install.jsonl"; FR = "/tmp/seg_s5_term"; shutil.rmtree(FR, ignore_errors=True)
r = Session("s5", cursor=False)
n3 = r.ends[2]                                     # 前三拍（终端部分）总帧数
with Player(SRC, FR, fs=31) as p:
    def upto(sec):                                  # 补静止帧到画面时刻 sec
        while p.n < int(round(sec * FPS)) and p.n < n3: p.snap()
    # 第 1 拍：先停在空提示符，再敲 node -v
    p.feed(0.8); upto(3.6)
    p.run([(0.87, 2.2, 1.4, ""), (2.2, 3.6, 0.2, "")]); upto(6.68)
    # 第 2 拍：敲安装命令（真实敲了约 5 秒，画面压到 1.6 秒）→ 回车 → 真实输出
    p.run([(3.6, 8.61, 1.6, ""), (8.61, 11.0, 1.2, "")]); upto(9.46)
    # 第 3 拍：claude --version
    p.run([(11.0, 14.36, 1.5, ""), (14.36, 18.3, 0.5, "")]); upto(n3 / FPS)
    p.snap() if p.n < n3 else None
with r:
    r.page.goto(CARDS); r.page.wait_for_timeout(200)
    for f in sorted(os.listdir(FR)): shutil.copy(os.path.join(FR, f), os.path.join(r.frames_dir, f))
    r.n = len(os.listdir(r.frames_dir)); r.cur = 2; r.start()
    r.beat()   # 第 4 拍：CC Switch 卡片
    r.page.evaluate("C.cc('CC Switch','把密钥配进 Claude Code 的桌面小工具','在新手引导里点下载'); document.querySelector('.d').style.visibility='hidden'; document.querySelector('.d2').style.visibility='hidden'")
    n = max(2, int(r.left() * FPS * 0.25))
    r.hold(n / FPS)
    r.page.evaluate("document.querySelector('.d').style.visibility='visible'"); r.hold(n / FPS)
    r.page.evaluate("document.querySelector('.d2').style.visibility='visible'")
