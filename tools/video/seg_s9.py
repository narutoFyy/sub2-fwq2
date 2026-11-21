from rec import Session
from vt import *
from rec import DY

STEPS = ["注册账号" if DY else "注册时填体验码", "创建密钥", "装两个软件", "一键导入", "点启用"]

def C(r, fn, *a):
    r.page.evaluate("([f,a])=>C[f](...a)", [fn, list(a)])

with Session("s9", cursor=False) as r:
    r.page.goto(CARDS); r.page.wait_for_timeout(200)
    C(r, "steps", STEPS); r.start()
    r.beat()                                   # 第 1 拍：依次点亮五步
    gap = max(2, int(r.left() * 10 * 0.8 / len(STEPS)))
    for i in range(len(STEPS)):
        C(r, "light", i); r.hold(gap / 10)
    r.beat()                                   # 第 2 拍：站名 + 体验码
    C(r, "hero", *(("注册入口", "见置顶评论") if DY else ("石头中转站", "体验码 STONE-BILI")))
