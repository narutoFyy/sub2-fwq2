#!/usr/bin/python3
"""真实终端录制：在伪终端里启动真实命令（如 claude），把每一段原始输出连同时间戳存成 jsonl。
之后由 termplay.py 在 xterm.js 里按时间轴原样回放并出帧。这里没有任何"伪造输出"，输出全部来自真实进程。
用法见 rec_open.py / rec_usage.py。"""
import base64, fcntl, json, os, pty, random, re, select, signal, struct, sys, termios, time

ANSI = re.compile(r"\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[()][A-Za-z0-9]|\x1b[=>]")

class Rec:
    def __init__(self, cmd, cwd, cols=110, rows=28, env=None):
        self.events, self.buf, self.t0 = [], "", time.time()
        # 干净环境：不继承桌面端注入的 CLAUDE_* 变量，只保留 PATH/HOME 和连接用的 ANTHROPIC_*，看起来就是普通用户的终端
        e = {k: v for k, v in os.environ.items() if k in ("PATH", "HOME", "USER", "LOGNAME", "SHELL", "TMPDIR") or k.startswith("ANTHROPIC_")}
        e.update({"TERM": "xterm-256color", "COLORTERM": "truecolor", "LANG": "en_US.UTF-8", "LC_ALL": "en_US.UTF-8"}); e.update(env or {})
        pid, fd = pty.fork()
        if pid == 0:
            os.chdir(cwd); os.execvpe(cmd[0], cmd, e)
        self.pid, self.fd = pid, fd
        fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", rows, cols, 0, 0))
        self.cols, self.rows = cols, rows

    def now(self): return time.time() - self.t0
    def pump(self, secs):
        end = time.time() + secs
        while time.time() < end:
            r, _, _ = select.select([self.fd], [], [], 0.05)
            if r:
                try: d = os.read(self.fd, 65536)
                except OSError: return False
                if not d: return False
                self.events.append([round(self.now(), 3), base64.b64encode(d).decode()])
                self.buf += ANSI.sub("", d.decode("utf-8", "ignore")); self.buf = self.buf[-20000:]
        return True
    def mark(self): self.buf = ""
    def wait_for(self, pat, timeout=60, quiet=0.0):
        """等屏幕出现 pat（正则，匹配最近的无色文本）；quiet>0 时再等输出静默这么久。"""
        end = time.time() + timeout; rx = re.compile(pat)
        while time.time() < end:
            self.pump(0.2)
            if rx.search(self.buf):
                if quiet: self.wait_quiet(quiet)
                return True
        return False
    def wait_quiet(self, quiet=2.0, timeout=300):
        end = time.time() + timeout; last = len(self.events)
        lt = time.time()
        while time.time() < end:
            self.pump(0.2)
            if len(self.events) != last: last = len(self.events); lt = time.time()
            elif time.time() - lt >= quiet: return True
        return False
    def send(self, s): os.write(self.fd, s.encode())
    def type(self, s, cps=9.0):
        """像真人一样逐字敲（中文也逐字）。"""
        for ch in s:
            self.send(ch); self.pump(random.uniform(0.6, 1.4) / cps)
    def key(self, name):
        self.send({"enter": "\r", "esc": "\x1b", "up": "\x1b[A", "down": "\x1b[B", "tab": "\t", "ctrl_c": "\x03"}[name]); self.pump(0.05)
    def save(self, path):
        with open(path, "w") as f:
            f.write(json.dumps({"cols": self.cols, "rows": self.rows, "dur": round(self.now(), 3)}) + "\n")
            for ev in self.events: f.write(json.dumps(ev) + "\n")
    def close(self):
        for sig in (signal.SIGTERM, signal.SIGKILL):
            try: os.kill(self.pid, sig)
            except OSError: break
            for _ in range(20):
                try:
                    if os.waitpid(self.pid, os.WNOHANG)[0]: return
                except OSError: return
                time.sleep(0.1)
    def screen_text(self): return self.buf


def shell_rec(cwd, claude_flags="", cols=110, rows=28, env=None, path_first=""):
    """起一个提示符只有 "$ " 的 zsh；里面的 claude 就是真实的 claude（可预置启动参数，不会显示在画面里）。"""
    import tempfile
    z = tempfile.mkdtemp(prefix="zd_")
    open(os.path.join(z, ".zshrc"), "w").write(("export PATH=\"%s:$PATH\"\n" % path_first if path_first else "") + "precmd() { rehash; }\nPS1='$ '\nunsetopt PROMPT_SP\nsetopt NO_BEEP\nclaude() { command claude %s \"$@\"; }\n" % claude_flags)
    return Rec(["/bin/zsh", "-i"], cwd, cols, rows, env=dict({"ZDOTDIR": z}, **(env or {})))

def pretrust(cwd):
    """在不录制的情况下先过一遍"信任此文件夹"弹窗，正式录制时就不会出现。"""
    r = Rec(["claude"], cwd)
    try:
        if r.wait_for(r"Yes,\s*I\s*trust", 30): r.pump(1.0); r.key("down"); r.pump(0.4); r.key("enter"); r.wait_for(r"shortcuts", 30); r.pump(1.5)
    finally: r.close()
