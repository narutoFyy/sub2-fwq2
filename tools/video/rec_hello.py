#!/usr/bin/python3
"""结尾真实录制：启动真实的 Claude Code，输入"你好"，等它回复。输出 /Users/Shared/cc-demo/hello.jsonl"""
import os, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from ptyrec import shell_rec, pretrust
ROOT = "/Users/Shared/cc-demo"; D = ROOT + "/my-project"; os.makedirs(D, exist_ok=True)
pretrust(D)
# 不加载用户级 CLAUDE.md 和记忆，避免回复里带出个人信息
r = shell_rec(D, env={"CLAUDE_CODE_DISABLE_CLAUDE_MDS": "1", "CLAUDE_CODE_DISABLE_AUTO_MEMORY": "1"})
try:
    r.wait_for(r"\$", 10); r.pump(1.0)
    r.type("claude", 7); r.pump(0.4); r.key("enter")
    r.wait_for(r"shortcuts", 40); r.pump(2.0)
    r.mark(); r.type("你好", 6); r.pump(0.8); r.key("enter")
    r.wait_quiet(4, 90); r.pump(1.5)
    print(r.screen_text()[-400:])
finally:
    r.save(ROOT + "/hello.jsonl"); r.close(); print("dur", r.now())
