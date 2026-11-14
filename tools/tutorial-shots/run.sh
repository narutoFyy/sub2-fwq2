#!/usr/bin/env bash
# 重新生成图文教程截图。前提：本地后端 8085 和前端 3000 在运行。
#   bash tools/tutorial-shots/run.sh            # 站内页面（中/英）
set -euo pipefail
cd "$(dirname "$0")/../.."
export PGPASSWORD="$(grep -A8 '^database:' backend/config.yaml | grep password | awk '{print $2}' | tr -d '"')"
export SHOTS_PASSWORD='ShotsDemo#2026'
hash="$(htpasswd -bnBC 10 '' "$SHOTS_PASSWORD" | tr -d ':\n' | sed 's/^\$2y/$2a/')"
psql -h 127.0.0.1 -U sub2api -d sub2api -v ON_ERROR_STOP=1 -v hash="$hash" -q -f tools/tutorial-shots/seed.sql
trap 'psql -h 127.0.0.1 -U sub2api -d sub2api -v ON_ERROR_STOP=1 -q -f tools/tutorial-shots/cleanup.sql' EXIT
python3 -I tools/tutorial-shots/capture.py "$@"
