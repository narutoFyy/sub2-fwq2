#!/usr/bin/python3
"""开场真实录制：在空文件夹里启动真实的 Claude Code，让它做一个小游戏。输出 /Users/Shared/cc-demo/open.jsonl（含真实时间戳）。"""
import os, shutil, sys, time
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from ptyrec import shell_rec, pretrust
D = "/Users/Shared/cc-demo/game"; shutil.rmtree(D, ignore_errors=True); os.makedirs(D)
PROMPT = "做一个霓虹风格的飞机大战网页游戏，单个 HTML 文件。鼠标控制飞机自动射击，敌人越来越多，要有粒子爆炸特效、分数和最高分记录"
pretrust(D)
r = shell_rec(D)
try:
    r.wait_for(r"\$", 10); r.pump(1.0)
    r.type("claude", 7); r.pump(0.4); r.key("enter")
    r.wait_for(r"shortcuts|auto mode", 40); r.pump(2.0)
    r.mark(); r.type(PROMPT, 14); r.pump(0.8); r.key("enter")
    t_send = r.now(); print("sent at", t_send)
    r.wait_quiet(10, 600)
    r.pump(2)
    print(r.screen_text()[-900:])
finally:
    r.save("/Users/Shared/cc-demo/open.jsonl"); r.close(); print("dur", r.now())
