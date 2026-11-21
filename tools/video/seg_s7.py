from rec import Session
from vt import *

prompt, read_ops, fix_ops, run_ops = cc_ops()
with Session("s7", cursor=False) as r:
    r.page.goto(TERM); r.page.wait_for_timeout(300); r.start()
    # 第 1 拍：文件内容 + node average.js -> 67.5
    r.beat()
    code = [("cmd", "cat average.js", "$"), ("html", "<span style='display:block'>" + esc(rd("s7_average.js").rstrip("\n")) + "</span>", "")]
    play(r, code + [("cmd", "node average.js", "$"), ("text", rd("s7_run1.txt").strip(), "")])
    # 第 2 拍：claude + 输入的话
    r.beat()
    play(r, [("clear",), ("cmd", "claude", "$"), ("pause", 0.5), ("box", prompt)], frac=0.8)
    # 第 3 拍：读文件、原因、改动
    r.beat()
    play(r, read_ops + fix_ops, frac=0.8)
    # 第 4 拍：再运行，输出 90 高亮
    r.beat()
    play(r, run_ops, frac=0.7)
