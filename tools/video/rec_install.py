#!/usr/bin/python3
"""第 5 段"装软件"真实录制：真实 node -v、真实 npm 安装、真实 claude --version。
本机的 npm 12 默认拦截安装脚本（npm 10 不拦），用环境变量放行，命令本身不变。安装装到一个临时目录（NPM_CONFIG_PREFIX），不动本机全局的 claude；命令和输出都是真的。输出 /Users/Shared/cc-demo/install.jsonl"""
import os, shutil, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from ptyrec import shell_rec
ROOT = "/Users/Shared/cc-demo"; PFX = ROOT + "/npm-prefix"; WORK = ROOT + "/install-work"
shutil.rmtree(PFX, ignore_errors=True); os.makedirs(PFX); os.makedirs(WORK, exist_ok=True)
INSTALL = "npm install -g @anthropic-ai/claude-code --registry=https://registry.npmmirror.com"
r = shell_rec(WORK, cols=92, rows=20, path_first=PFX + "/bin", env={"NPM_CONFIG_PREFIX": PFX, "NPM_CONFIG_ALLOW_SCRIPTS": "@anthropic-ai/claude-code", "PATH": PFX + "/bin:" + os.environ["PATH"]})
try:
    r.wait_for(r"\$", 10); r.pump(0.6)
    r.type("node -v", 10); r.pump(0.4); r.key("enter"); r.wait_for(r"v\d+\.\d+", 10); r.pump(1.2)
    r.type(INSTALL, 90); r.pump(0.5); r.key("enter")
    print("installed:", r.wait_for(r"added \d+ package|ERR", 180)); r.pump(1.5)
    r.type("claude --version", 14); r.pump(0.4); r.key("enter"); r.wait_for(r"Claude Code", 15); r.pump(1.5)
    print(r.screen_text()[-600:])
finally:
    r.save(ROOT + "/install.jsonl"); r.close(); print("dur", r.now())
