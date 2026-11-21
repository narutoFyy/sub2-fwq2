#!/usr/bin/python3
"""把无声录屏 + 配音 + 口播文字合成带字幕的 mp4。
用法：/usr/bin/python3 tools/video/compose.py <video.mp4> <voice.wav> <narration.txt> <out.mp4>
字幕按句切分，时长按字数比例分配到配音时长上；画面比配音短时定格最后一帧。"""
import re, subprocess, sys, tempfile, os
from PIL import Image, ImageDraw, ImageFont
video, wav, txt, out = sys.argv[1:5]
W, H = 1920, 1080
FONT = "/System/Library/Fonts/Hiragino Sans GB.ttc"
def dur(p):
    return float(subprocess.check_output(["ffprobe","-v","error","-show_entries","format=duration","-of","csv=p=0",p]).strip())
text = open(txt, encoding="utf-8").read().strip()
parts = [s for s in re.split(r"(?<=[。！？；：，])", text) if s.strip()]
lines, cur = [], ""
for s in parts:                       # 合并成每条不超过约 24 字
    if len(cur) + len(s) <= 24: cur += s
    else:
        if cur: lines.append(cur)
        cur = s
if cur: lines.append(cur)
a = dur(wav); total = sum(len(l) for l in lines); tmp = tempfile.mkdtemp(); t = 0.0; cues = []
font = ImageFont.truetype(FONT, 46)
for i, l in enumerate(lines):
    d = a * len(l) / total; show = l.rstrip("，。：；")
    img = Image.new("RGBA", (W, H), (0, 0, 0, 0)); dr = ImageDraw.Draw(img)
    tw = dr.textlength(show, font=font); x = (W - tw) / 2; y = H - 130
    dr.rounded_rectangle([x - 28, y - 14, x + tw + 28, y + 66], radius=16, fill=(0, 0, 0, 170))
    dr.text((x, y), show, font=font, fill=(255, 255, 255, 255))
    p = os.path.join(tmp, f"s{i}.png"); img.save(p); cues.append((p, t, t + d)); t += d
v = dur(video); pad = max(0.0, a + 0.6 - v)
cmd = ["ffmpeg", "-y", "-loglevel", "error", "-i", video, "-i", wav]
for p, _, _ in cues: cmd += ["-i", p]
fc = f"[0:v]scale={W}:{H},tpad=stop_mode=clone:stop_duration={pad:.2f}[v0];"
for i, (_, s, e) in enumerate(cues):
    fc += f"[v{i}][{i+2}:v]overlay=enable='between(t,{s:.2f},{e:.2f})'[v{i+1}];"
fc += "[1:a]adelay=300|300,apad=pad_dur=0.3[a]"
cmd += ["-filter_complex", fc, "-map", f"[v{len(cues)}]", "-map", "[a]", "-c:v", "libx264", "-pix_fmt", "yuv420p",
        "-crf", "20", "-c:a", "aac", "-b:a", "128k", "-t", f"{max(v, a + 0.6):.2f}", out]
subprocess.check_call(cmd); print("ok", out, f"{max(v, a + 0.6):.1f}s", len(cues), "cues")
