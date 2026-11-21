"""终端/卡片重放小工具：把保存下来的真实输出在 pages/terminal.html 里重放。重跑不会执行任何真实命令。"""
import json, math, os, re
from rec import FPS

HERE = os.path.dirname(os.path.abspath(__file__))
DATA = os.path.join(HERE, "pages", "data")
TERM = "file://" + os.path.join(HERE, "pages", "terminal.html")
CARDS = "file://" + os.path.join(HERE, "pages", "cards.html")

def rd(name):
    return open(os.path.join(DATA, name), encoding="utf-8").read()

def esc(s):
    return s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")

def md(s):
    return re.sub(r"`([^`]+)`", lambda m: "<code>" + m.group(1) + "</code>", esc(s))

def T(r, fn, *args):
    r.page.evaluate("([f,a])=>T[f](...a)", [fn, list(args)])

def play(r, ops, frac=0.85):
    """按权重把本拍剩余时间分给各操作（所以配音到位后重跑会自动适配时长）。ops:
    ("cmd", 文本, 提示符) 逐字打命令   ("box", 文本) 逐字打 Claude Code 输入框
    ("html", html, cls) / ("text", 文本, cls) 追加一行   ("hl",) 高亮   ("pause", 权重)   ("clear",)
    """
    start = r.n
    budget = max(3, int(r.left() * FPS * frac))
    def w(o):
        if o[0] in ("cmd", "box"): return 1 + len(o[1]) / 12
        if o[0] == "pause": return o[1]
        return 1
    unit = budget / sum(w(o) for o in ops)
    acc = 0.0
    for o in ops:
        acc += w(o) * unit
        target = start + int(round(acc))
        k = o[0]
        if k in ("cmd", "box"):
            T(r, "cmdStart", o[2] if len(o) > 2 else "$") if k == "cmd" else T(r, "boxStart")
            txt = o[1]; frames = max(1, target - r.n)
            step = max(1, math.ceil(len(txt) / frames))
            for i in range(0, len(txt), step):
                T(r, "cmdAdd", txt[i:i + step]); r.snap()
            T(r, "caretOff")
        elif k == "html": T(r, "html", o[1], o[2] if len(o) > 2 else ""); r.snap()
        elif k == "text": T(r, "text", o[1], o[2] if len(o) > 2 else ""); r.snap()
        elif k == "hl": T(r, "hl"); r.snap()
        elif k == "clear": T(r, "clear"); r.snap()
        while r.n < target: r.snap()

def stream_events():
    """从保存的 stream-json 里取出：工具调用、结果、最终文字。"""
    ev, final = [], ""
    results = {}
    msgs = [json.loads(l) for l in rd("s7_claude_stream.jsonl").splitlines() if l.strip()]
    for d in msgs:
        if d.get("type") == "user":
            for c in d["message"]["content"]:
                if c.get("type") == "tool_result": results[c["tool_use_id"]] = c["content"]
        if d.get("type") == "result": final = d.get("result", "")
    for d in msgs:
        if d.get("type") == "assistant":
            for c in d["message"]["content"]:
                if c["type"] == "tool_use": ev.append((c["name"], c["input"], results.get(c["id"], "")))
    return ev, final

def cc_ops():
    """真实那次运行拆成三块：读文件、说明+改动、再运行。返回 (prompt, read_ops, fix_ops, run_ops)。"""
    ev, final = stream_events()
    prompt = "运行 node average.js，输出的平均分不对，找出原因并修好，修完再运行一次确认。"
    read = next(e for e in ev if e[0] == "Read")
    edit = next(e for e in ev if e[0] == "Edit")
    runs = [e for e in ev if e[0] == "Bash"]
    last = runs[-1]
    nlines = len(rd("s7_average.js").splitlines())
    paras = [p for p in final.split("\n\n") if p.strip()]
    read_ops = [("html", '<span class="b ok">●</span> <b>Read</b>(average.js)', "tool"),
                ("html", "⎿  读取了 %d 行" % nlines, "sub")]
    fix_ops = [("html", "<b>●</b> " + md(paras[1]), "tool"),
               ("html", '<span class="b ok">●</span> <b>Update</b>(average.js)', "tool"),
               ("html", "- " + esc(edit[1]["old_string"].strip()), "del"),
               ("html", "+ " + esc(edit[1]["new_string"].strip()), "add")]
    out = str(last[2]).strip()
    out_html = esc(out).replace("90", '<span class="hl0">90</span>')
    run_ops = [("html", '<span class="b ok">●</span> <b>Bash</b>(' + esc(last[1]["command"]) + ")", "tool"),
               ("html", "⎿  " + out_html, "sub"),
               ("hl",),
               ("html", "<b>●</b> " + md(paras[0]), "tool")]
    return prompt, read_ops, fix_ops, run_ops
