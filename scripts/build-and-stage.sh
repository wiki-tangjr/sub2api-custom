#!/bin/bash
# ============================================================================
# build-and-stage.sh -- 构建 前端(embed) + 后端，并启动 8081 灰度实例
#
#   * 只做「构建 + 起灰度」，绝不碰线上 8080，也绝不碰 nginx
#   * 任何一步失败立即退出，线上服务完全不受影响
#   * 成功后：$STAGE/sub2api_new 为待发布二进制，8081 已用它在跑
#
#   下一步: bash scripts/zero-downtime-deploy.sh
#   用法:   bash scripts/build-and-stage.sh
# ============================================================================
set -uo pipefail

REPO="${REPO:-/root/.openclaw/workspace/sub2api-src}"
STAGE="${STAGE:-/root/deploy_stage}"
LIVE_PORT="${LIVE_PORT:-8080}"
CANARY_PORT="${CANARY_PORT:-8081}"
CANARY_UNIT="${CANARY_UNIT:-sub2api-canary.service}"
LIVE_BIN="${LIVE_BIN:-/opt/sub2api/sub2api}"
CANARY_BIN="${CANARY_BIN:-/opt/sub2api/sub2api.canary}"
CANARY_LOG="${CANARY_LOG:-/tmp/sub2api_canary.log}"
CANARY_GWLOG="${CANARY_GWLOG:-/tmp/gateway_debug_canary.log}"
CANARY_ENV="${CANARY_ENV:-$STAGE/canary.env}"

log(){ printf '[%s] %s\n' "$(date +%H:%M:%S)" "$*"; }
die(){ log "FAIL: $*"; exit 1; }
code(){ curl -s -o /dev/null -w '%{http_code}' --max-time 8 "$1" 2>/dev/null; }

echo "================ BUILD & STAGE $(date +%Y%m%d-%H%M%S) ================"
cd "$REPO" || die "repo not found: $REPO"

if [ "$(code http://127.0.0.1:$LIVE_PORT/health)" != "200" ]; then
  die "线上 $LIVE_PORT 不健康，拒绝开始构建（先修线上）"
fi
log "线上 $LIVE_PORT 健康，开始构建"

# ---------- 1. 前端 ----------
log "STEP 1: 前端构建"
command -v pnpm >/dev/null 2>&1 || die "缺少 pnpm（corepack enable pnpm）"
rm -rf "$STAGE"; mkdir -p "$STAGE"
cd "$REPO/frontend" || die "no frontend dir"
rm -rf "$REPO/backend/internal/web/dist" 2>/dev/null
pnpm run build > "$STAGE/fe_build.log" 2>&1 || { tail -30 "$STAGE/fe_build.log"; die "前端构建失败"; }
log "前端 OK，dist 文件数: $(find "$REPO/backend/internal/web/dist" -type f | wc -l)"

# ---------- 2. 后端（必须 -tags embed） ----------
log "STEP 2: 后端构建 (-tags embed)"
cd "$REPO/backend" || die "no backend dir"
go build -tags embed -o "$STAGE/sub2api_new" ./cmd/server > "$STAGE/be_build.log" 2>&1 \
  || { tail -30 "$STAGE/be_build.log"; die "后端构建失败"; }
[ -s "$STAGE/sub2api_new" ] || die "产物为空"
chmod 755 "$STAGE/sub2api_new"
NEW_MD5=$(md5sum "$STAGE/sub2api_new" | cut -d' ' -f1)
OLD_MD5=$(md5sum "$LIVE_BIN" | cut -d' ' -f1)
log "后端 OK size=$(stat -c%s "$STAGE/sub2api_new") md5=$NEW_MD5"
[ "$NEW_MD5" != "$OLD_MD5" ] || die "新二进制与线上完全相同，无需发布"

# ---------- 3. 代码层体检 ----------
log "STEP 3: 代码层体检"
bash "$REPO/scripts/customizations-verify.sh" > "$STAGE/verify_code.log" 2>&1 \
  || { tail -25 "$STAGE/verify_code.log"; die "代码层体检未通过，拒绝进入灰度"; }
tail -3 "$STAGE/verify_code.log"

# ---------- 4. 启动 8081 灰度 ----------
log "STEP 4: 启动 $CANARY_PORT 灰度实例"
ss -lntp 2>/dev/null | grep -q ":$CANARY_PORT" && die "$CANARY_PORT 已被占用"
systemctl stop "$CANARY_UNIT" 2>/dev/null
systemctl reset-failed "$CANARY_UNIT" 2>/dev/null
MP=$(systemctl show sub2api -p MainPID --value)
[ -n "$MP" ] && [ -r "/proc/$MP/environ" ] || die "拿不到线上进程环境变量"
tr '\0' '\n' < "/proc/$MP/environ" \
  | grep -E '^(GIN_MODE|TOTP_ENCRYPTION_KEY|SUB2API_|TZ=)' \
  | grep -vE '^SERVER_PORT=' > "$CANARY_ENV"
echo "SERVER_PORT=$CANARY_PORT" >> "$CANARY_ENV"
echo "LOG_OUTPUT_FILE_PATH=$CANARY_LOG" >> "$CANARY_ENV"
echo "SUB2API_DEBUG_GATEWAY_BODY=$CANARY_GWLOG" >> "$CANARY_ENV"
chmod 600 "$CANARY_ENV"
install -o sub2api -g sub2api -m 755 "$STAGE/sub2api_new" "$CANARY_BIN" || die "staging 二进制失败"
systemd-run --unit="${CANARY_UNIT%.service}" \
  --uid=sub2api --gid=sub2api --working-directory=/opt/sub2api \
  -p EnvironmentFile="$CANARY_ENV" -p Restart=no \
  "$CANARY_BIN" > /tmp/canary_start.log 2>&1 || { tail -5 /tmp/canary_start.log; die "启动灰度失败"; }

OK=0
for i in $(seq 1 45); do
  sleep 2
  [ "$(code http://127.0.0.1:$CANARY_PORT/health)" = "200" ] && { OK=1; log "灰度健康 (${i}x2s)"; break; }
  systemctl is-active --quiet "$CANARY_UNIT" || { log "灰度进程退出 (${i}x2s)"; break; }
done
if [ "$OK" != "1" ]; then
  journalctl -u "$CANARY_UNIT" -n 40 --no-pager 2>&1 | tail -45
  tail -20 "$CANARY_LOG" 2>&1
  systemctl stop "$CANARY_UNIT" 2>/dev/null
  die "灰度未能健康启动（线上 $LIVE_PORT 未受影响）"
fi

# ---------- 5. 灰度功能体检 ----------
log "STEP 5: 灰度功能体检（$CANARY_PORT）"
CUSTOM_VERIFY_PORT=$CANARY_PORT bash "$REPO/scripts/customizations-verify.sh" --live > "$STAGE/verify_canary.log" 2>&1 \
  || { tail -30 "$STAGE/verify_canary.log"; systemctl stop "$CANARY_UNIT" 2>/dev/null; die "灰度体检未通过，已停掉灰度（线上未受影响）"; }
tail -3 "$STAGE/verify_canary.log"

echo
log "BUILD & STAGE OK"
log "  待发布二进制: $STAGE/sub2api_new (md5=$NEW_MD5)"
log "  灰度实例:     $CANARY_UNIT on :$CANARY_PORT"
log "  下一步:       bash scripts/zero-downtime-deploy.sh"
echo "STAGE_READY"