from rec import Session
from vt import *

CARDS_DATA = [
    ("点导入没反应", "先确认 CC Switch 装好，并且打开过一次"),
    ("提示密钥无效", "多半是启用的不是刚导入的那一项"),
    ("还让你登录官方账号", "看 CC Switch 里是不是使用中，再把 Claude Code 关掉重开"),
]
CN = ["一", "二", "三"]

def C(r, fn, *a):
    r.page.evaluate("([f,a])=>C[f](...a)", [fn, list(a)])

with Session("s8", cursor=False) as r:
    r.page.goto(CARDS); r.page.wait_for_timeout(200)
    C(r, "title", "三个常见卡点", "big"); r.start()
    r.beat()                                   # 第 1 拍：标题
    for i, (q, a) in enumerate(CARDS_DATA):    # 第 2~4 拍：三张卡片依次出现
        r.beat()
        if i == 0: C(r, "shrink")
        r.hold(0.3)
        C(r, "card", q, a, CN[i])
