#!/usr/bin/python3
"""结尾"怎么用"真实录制：进项目文件夹 -> 输入 claude -> 用中文说需求 -> 它改文件前征求同意 -> 回车同意 -> 退出后运行验证。输出 /Users/Shared/cc-demo/usage.jsonl"""
import os, shutil, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from ptyrec import shell_rec, pretrust
ROOT = "/Users/Shared/cc-demo"; D = ROOT + "/my-project"; shutil.rmtree(D, ignore_errors=True); os.makedirs(D)
shutil.copy(os.path.join(os.path.dirname(os.path.abspath(__file__)), "pages/data/s7_average.js"), D + "/average.js")
pretrust(D)
r = shell_rec(ROOT, "--permission-mode default")
try:
    r.wait_for(r"\$", 10); r.pump(1.0)
    r.type("cd my-project", 8); r.pump(0.3); r.key("enter"); r.pump(0.8)
    r.type("claude", 7); r.pump(0.4); r.key("enter")
    r.wait_for(r"shortcuts", 40); r.pump(2.0)
    r.mark(); r.type("平均分不对，应该是 90，帮我找出原因并修好", 9); r.pump(0.8); r.key("enter")
    # 等它征求同意（改文件前一定会问）
    ok = r.wait_for(r"Do\s*you\s*want|Yes", 120)
    print("asked:", ok); r.pump(2.5)
    r.key("enter")                         # 选默认的 Yes
    r.wait_quiet(6, 180); r.pump(1.5)
    print(r.screen_text()[-700:])
    r.type("/exit", 8); r.pump(0.4); r.key("enter"); r.pump(1.5)
    r.type("node average.js", 9); r.pump(0.3); r.key("enter"); r.pump(1.5)
finally:
    r.save(ROOT + "/usage.jsonl"); r.close(); print("dur", r.now())
