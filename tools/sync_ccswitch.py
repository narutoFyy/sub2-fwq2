#!/usr/bin/env python3
"""把 CC Switch 的最新安装包同步到自托管下载目录（新手引导的「下载 CC Switch」用）。

用法：
    python3 tools/sync_ccswitch.py --dir /opt/sub2api-downloads            # 同步最新版
    python3 tools/sync_ccswitch.py --dir ./downloads --tag v4.0.31        # 指定版本
    python3 tools/sync_ccswitch.py --dir ./downloads --dry-run            # 只看会下载什么

做的事：
  1. 通过 GitHub API 读取 farion1231/cc-switch 的发布信息；
  2. 只下载引导页用到的几个安装包（Windows x64/arm64 msi、macOS dmg、Linux x86_64 deb）；
  3. 每个文件按 GitHub 给出的 sha256 校验，不一致就中止，不覆盖已有文件；
  4. 写出 <dir>/cc-switch/manifest.json（版本、文件名、大小、sha256），前端据此判断「已自托管」；
  5. 清理 <dir>/cc-switch 里不属于本次版本的旧安装包。

服务端需设置环境变量 DOWNLOADS_DIR=<dir> 并重启，/downloads/cc-switch/* 才会对外提供。
只用标准库，无需安装依赖。
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import sys
import tempfile
import urllib.request

REPO = "farion1231/cc-switch"
API = f"https://api.github.com/repos/{REPO}/releases"
# 引导页用到的安装包后缀；文件名形如 CC-Switch-v4.0.31-Windows.msi
WANTED_SUFFIXES = ("-Windows.msi", "-Windows-arm64.msi", "-macOS.dmg", "-Linux-x86_64.deb")


def fetch_json(url: str) -> dict:
    request = urllib.request.Request(url, headers={"Accept": "application/vnd.github+json", "User-Agent": "sync-ccswitch"})
    with urllib.request.urlopen(request, timeout=30) as response:
        return json.load(response)


def sha256_of(path: str) -> str:
    digest = hashlib.sha256()
    with open(path, "rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def download(url: str, dest: str) -> None:
    request = urllib.request.Request(url, headers={"User-Agent": "sync-ccswitch"})
    with urllib.request.urlopen(request, timeout=60) as response, open(dest, "wb") as out:
        while True:
            chunk = response.read(1024 * 1024)
            if not chunk:
                break
            out.write(chunk)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--dir", required=True, help="自托管下载根目录（即服务端的 DOWNLOADS_DIR）")
    parser.add_argument("--tag", help="指定版本标签，默认最新")
    parser.add_argument("--dry-run", action="store_true", help="只列出将要下载的文件")
    args = parser.parse_args()

    release = fetch_json(f"{API}/tags/{args.tag}" if args.tag else f"{API}/latest")
    tag = release["tag_name"]
    assets = [a for a in release["assets"] if a["name"].endswith(WANTED_SUFFIXES)]
    if not assets:
        print(f"版本 {tag} 里没有找到需要的安装包（后缀 {WANTED_SUFFIXES}）", file=sys.stderr)
        return 1

    target = os.path.join(args.dir, "cc-switch")
    print(f"CC Switch {tag}：{len(assets)} 个文件 → {target}")
    for asset in assets:
        print(f"  {asset['name']}  {asset['size'] / 1e6:.1f} MB")
    if args.dry_run:
        return 0

    os.makedirs(target, exist_ok=True)
    files = []
    for asset in assets:
        name = asset["name"]
        expected = (asset.get("digest") or "").removeprefix("sha256:")
        if not expected:
            print(f"{name}: GitHub 没有提供 sha256，无法校验，中止", file=sys.stderr)
            return 1
        final = os.path.join(target, name)
        if os.path.exists(final) and sha256_of(final) == expected:
            print(f"  已是最新：{name}")
        else:
            fd, tmp = tempfile.mkstemp(dir=target, prefix=".download-")
            os.close(fd)
            try:
                print(f"  下载：{name}")
                download(asset["browser_download_url"], tmp)
                actual = sha256_of(tmp)
                if actual != expected:
                    print(f"{name}: sha256 不一致（期望 {expected}，实际 {actual}），中止", file=sys.stderr)
                    return 1
                os.chmod(tmp, 0o644)
                os.replace(tmp, final)
            finally:
                if os.path.exists(tmp):
                    os.remove(tmp)
        files.append({"name": name, "size": asset["size"], "sha256": expected})

    manifest = {"version": tag, "files": sorted(files, key=lambda f: f["name"])}
    manifest_path = os.path.join(target, "manifest.json")
    fd, tmp_manifest = tempfile.mkstemp(dir=target, prefix=".manifest-")
    with os.fdopen(fd, "w") as handle:
        json.dump(manifest, handle, indent=2, ensure_ascii=False)
        handle.write("\n")
    os.chmod(tmp_manifest, 0o644)
    os.replace(tmp_manifest, manifest_path)

    keep = {f["name"] for f in files} | {"manifest.json"}
    for entry in os.listdir(target):
        if entry not in keep and not entry.startswith("."):
            os.remove(os.path.join(target, entry))
            print(f"  清理旧文件：{entry}")
    print(f"完成：{manifest_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
