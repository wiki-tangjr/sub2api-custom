#!/bin/bash
# ============================================================================
# healthcheck.sh -- Sub2API 只读健康检查（绝不重启任何东西）
#   用法: bash scripts/healthcheck.sh [标签]
# ============================================================================
set -uo pipefail
LABEL="${1:-healthcheck}"
PORT="${LIVE_PORT:-8080}"
BASE="http://127.0.0.1:$PORT"

echo "==================== $LABEL ===================="
echo "active:        $(systemctl is-active sub2api)"
echo "health:        $(curl -s -o /dev/null -w '%{http_code}' --max-time 10 "$BASE/health")"
echo "public_sets:   $(curl -s -o /dev/null -w '%{http_code}' --max-time 10 "$BASE/api/v1/settings/public")"
echo "chat_auth:     $(curl -s -o /dev/null -w '%{http_code}' --max-time 10 -X POST "$BASE/v1/chat/completions" -H 'Content-Type: application/json' -d '{}')"
echo "login_page:    $(curl -s -o /dev/null -w '%{http_code}' --max-time 10 "$BASE/")"
systemctl show sub2api -p NRestarts -p ActiveEnterTimestamp -p MainPID
echo "listening:     $(ss -ltnp 2>/dev/null | grep -c ":$PORT")"
echo "================================================"