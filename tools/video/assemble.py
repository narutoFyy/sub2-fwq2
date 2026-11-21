#!/usr/bin/python3
"""把各段无声画面 + 配音 + 字幕合成整片。用法：/usr/bin/python3 tools/video/assemble.py [输出文件名]
字幕按 script.json 的分拍对时（与录屏同一套时间），长句再按标点拆成不超过 22 字的小条。"""
import json, os, re, subprocess, sys, tempfile
from PIL import Image, ImageDraw, ImageFont
ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
VID = os.path.join(ROOT, "docs/marketing/video"); W, H = 1920, 1080
FONT = ImageFont.truetype("/System/Library/Fonts/Hiragino Sans GB.ttc", 44)
NO_SUBS = {"s8", "s9"}            # 这两段画面本身就是文字卡片
def dur(p): return float(subprocess.check_output(["ffprobe","-v","error","-show_entries","format=duration","-of","csv=p=0",p]).strip())
def split(text, n=22):
    parts = [s for s in re.split(r"(?<=[。！？；：，])", text) if s.strip()]; out, cur = [], ""
    for s in parts:
        if len(cur) + len(s) <= n: cur += s
        else:
            if cur: out.append(cur)
            cur = s
    if cur: out.append(cur)
    return out
def png(text, path):
    img = Image.new("RGBA", (W, H), (0, 0, 0, 0)); d = ImageDraw.Draw(img); t = text.strip().rstrip("，。：；")
    tw = d.textlength(t, font=FONT); x = (W - tw) / 2; y = H - 112
    d.rounded_rectangle([x - 26, y - 12, x + tw + 26, y + 62], radius=14, fill=(0, 0, 0, 185)); d.text((x, y), t, font=FONT, fill="white"); img.save(path)
DY = os.environ.get("VARIANT") == "dy"
segs = json.load(open(os.path.join(ROOT, "tools/video/script_dy.json" if DY else "tools/video/script.json"), encoding="utf-8")); tmp = tempfile.mkdtemp(); outs = []
for s in segs:
    sid = s["id"]; v = os.path.join(VID, "seg", sid + ".mp4"); a = os.path.join(VID, "audio", sid + ".wav")
    if DY:    # 抖音版：优先用 seg_dy / audio_dy 里的，没有就用原版
        v2 = os.path.join(VID, "seg_dy", sid + ".mp4"); a2 = os.path.join(VID, "audio_dy", sid + ".wav")
        v = v2 if os.path.exists(v2) else v; a = a2 if os.path.exists(a2) else a
    ad = dur(a); vd = dur(v); total = max(ad + 0.9, vd) if s.get('full_video') else ad + 0.9; chars = sum(len(b) for b in s["beats"]); t = 0.3; cues = []
    for b in s["beats"]:
        bd = ad * len(b) / chars; lines = split(b); lc = sum(len(l) for l in lines); tt = t
        for l in lines:
            ld = bd * len(l) / lc; cues.append((l, tt, tt + ld)); tt += ld
        t += bd
    if sid in NO_SUBS: cues = []
    cmd = ["ffmpeg", "-y", "-loglevel", "error", "-i", v, "-i", a]
    for i, (l, _, _) in enumerate(cues):
        p = os.path.join(tmp, f"{sid}_{i}.png"); png(l, p); cmd += ["-i", p]
    fc = f"[0:v]fps=30,scale={W}:{H},tpad=stop_mode=clone:stop_duration={max(0, total - vd) + 0.1:.2f}[v0];"
    for i, (_, st, en) in enumerate(cues): fc += f"[v{i}][{i+2}:v]overlay=enable='between(t,{st:.2f},{en:.2f})'[v{i+1}];"
    fc += "[1:a]aresample=48000,adelay=300|300,apad=whole_dur=%.2f[a]" % total
    o = os.path.join(tmp, sid + ".mp4")
    cmd += ["-filter_complex", fc, "-map", f"[v{len(cues)}]", "-map", "[a]", "-t", f"{total:.2f}", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-crf", "19", "-c:a", "aac", "-b:a", "160k", "-ar", "48000", o]
    subprocess.check_call(cmd); outs.append(o); print(f"{sid}: video {vd:.1f}s audio {ad:.1f}s -> {total:.1f}s, {len(cues)} subtitles")
lst = os.path.join(tmp, "list.txt"); open(lst, "w").write("".join(f"file '{o}'\n" for o in outs))
final = os.path.join(VID, sys.argv[1] if len(sys.argv) > 1 else "成片.mp4")
subprocess.check_call(["ffmpeg", "-y", "-loglevel", "error", "-f", "concat", "-safe", "0", "-i", lst, "-c", "copy", "-movflags", "+faststart", final])
print("final:", final, f"{dur(final):.1f}s", f"{os.path.getsize(final)/1e6:.1f} MB")
