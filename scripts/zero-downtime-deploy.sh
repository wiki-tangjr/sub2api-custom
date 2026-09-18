#!/bin/bash
# ============================================================================
# zero-downtime-deploy.sh -- 零中断发布（蓝绿 + 连接排空）
#
#   8080(旧, systemd) --切--> 8081(新, canary) --排空8080--> 8080(新) --> 停canary
#
#   * nginx 任何时刻都指向一个健康后端，在途请求（含 SSE 流式 AI 调用）不断开
#   * 全程 1s 间隔可用性采样，结束后打印「零中断」证明
#   * 任一环节失败 -> 自动把 nginx 指回仍健康的后端
#   * 新二进制起不来 -> 自动回滚旧二进制，站点保持可用
#
#   前置条件：先跑 scripts/build-and-stage.sh
#     - $STAGE/sub2api_new 存在且与线上二进制不同
#     - 8081 灰度实例已用新二进制运行且 /health = 200
#
#   用法:
#     bash scripts/zero-downtime-deploy.sh --dry-run   # 只体检 + 打印计划，不改任何东西
#     bash scripts/zero-downtime-deploy.sh             # 真正执行
# ============================================================================
set -uo pipefail

REPO="${REPO:-/root/.openclaw/workspace/sub2api-src}"
STAGE="${STAGE:-/root/deploy_stage}"
CONF="${CONF:-/www/server/panel/vhost/nginx/api.pixelqd.cn.conf}"
SITE="${SITE:-https://api.pixelqd.cn}"
LIVE_PORT="${LIVE_PORT:-8080}"
CANARY_PORT="${CANARY_PORT:-8081}"
CANARY_UNIT="${CANARY_UNIT:-sub2api-canary.service}"
LIVE_BIN="${LIVE_BIN:-/opt/sub2api/sub2api}"
NEW_BIN="${NEW_BIN:-$STAGE/sub2api_new}"
DRAIN_CAP="${DRAIN_CAP:-420}"
CANARY_DRAIN_CAP="${CANARY_DRAIN_CAP:-300}"

DRY_RUN=0
[ "${1:-}" = "--dry-run" ] && DRY_RUN=1

STAMP=$(date +%Y%m%d-%H%M%S)
NGX_BAK="$STAGE/nginx.bak.$STAMP"
OLD_BIN_BAK="$STAGE/sub2api.old.$STAMP"
AVAIL_LOG="$STAGE/availability.$STAMP.log"
AVAIL_PID=""
UPSTREAM_NAME=sub2api_backend

log(){ printf '[%s] %s\n' "$(date +%H:%M:%S)" "$*"; }
code(){ curl -s -o /dev/null -w '%{http_code}' --max-time 8 "$1" 2>/dev/null; }
site(){ curl -sk -o /dev/null -w '%{http_code}' --max-time 10 "$SITE$1" 2>/dev/null; }
conns(){ ss -Htn state established "( sport = :$1 )" 2>/dev/null | wc -l | tr -d ' '; }
bundle(){ curl -sk --max-time 10 "$SITE/" 2>/dev/null | grep -oE 'assets/index-[A-Za-z0-9_-]+\.js' | head -1; }
upstream_port(){ awk "/^upstream $UPSTREAM_NAME \{/,/^\}/" "$CONF" | grep -oE '127\.0\.0\.1:[0-9]+' | head -1 | cut -d: -f2; }
set_backend(){ sed -i -E "/^upstream $UPSTREAM_NAME \{/,/^\}/ s|server 127\.0\.0\.1:[0-9]+;|server 127.0.0.1:$1;|" "$CONF"; }
reload_ngx(){ nginx -t > "/tmp/ngx_t.$STAMP.log" 2>&1 || { log "nginx -t 失败"; tail -5 "/tmp/ngx_t.$STAMP.log"; return 1; }; nginx -s reload 2>/dev/null || return 1; return 0; }

point_nginx_at_healthy(){
  local p
  for p in "$LIVE_PORT" "$CANARY_PORT"; do
    if [ "$(code http://127.0.0.1:$p/health)" = "200" ]; then
      set_backend "$p"; reload_ngx; log "failover: nginx -> $p (健康)"; return 0
    fi
  done
  log "CRITICAL: $LIVE_PORT / $CANARY_PORT 均不健康"
  return 1
}
abort(){ log "!!! ABORT: $*"; log "--- 诊断 ---"; bash "$REPO/scripts/healthcheck.sh" "abort-diagnosis" 2>/dev/null || true; point_nginx_at_healthy; exit 1; }
wait_health(){ local p=$1 cap=${2:-120} i; for i in $(seq 1 "$cap"); do [ "$(code http://127.0.0.1:$p/health)" = "200" ] && { log "127.0.0.1:$p 健康 (${i}s)"; return 0; }; sleep 1; done; return 1; }
drain(){
  local p=$1 cap=$2 i n
  log "排空 127.0.0.1:$p (上限 ${cap}s)"
  for i in $(seq 1 "$cap"); do
    n=$(conns "$p")
    [ "$n" -eq 0 ] && { log "127.0.0.1:$p 已排空 (0 conns, ${i}s)"; return 0; }
    [ $((i % 20)) -eq 0 ] && log "  ...$p 仍有 $n 条连接, 已 ${i}s"
    sleep 1
  done
  log "WARN: $p 在 ${cap}s 后仍有 $(conns "$p") 条连接，继续执行"
  return 0
}
start_monitor(){
  ( while :; do
      printf '%s %s\n' "$(date +%s)" "$(curl -sk -o /dev/null -w '%{http_code}' --max-time 6 "$SITE/health" 2>/dev/null)" >> "$AVAIL_LOG"
      sleep 1
    done ) &
  AVAIL_PID=$!
  log "可用性采样已启动 (pid $AVAIL_PID) -> $AVAIL_LOG"
}
stop_monitor(){ [ -n "$AVAIL_PID" ] && kill "$AVAIL_PID" 2>/dev/null; wait "$AVAIL_PID" 2>/dev/null; AVAIL_PID=""; }
report_monitor(){
  local total bad mx
  total=$(wc -l < "$AVAIL_LOG" 2>/dev/null | tr -d ' ')
  bad=$(awk '$2!="200"' "$AVAIL_LOG" 2>/dev/null | wc -l | tr -d ' ')
  mx=$(awk '{ if ($2=="200") c=0; else { c++; if (c>m) m=c } } END{print m+0}' "$AVAIL_LOG" 2>/dev/null)
  printf '可用性采样: %s 次, 非200: %s 次, 最长连续不可用: %ss\n' "${total:-0}" "${bad:-0}" "${mx:-0}"
  if [ "${bad:-0}" -eq 0 ]; then echo "==> 全程零中断（无任何非200采样）"; else echo "==> 出现过非200采样，明细："; awk '$2!="200"' "$AVAIL_LOG" | head -20; fi
}

echo "================ 零中断发布 $STAMP ================"

# ---------------- Phase 0 ----------------
log "PHASE 0: 发布前体检"
FAILS=0
chk(){ if [ "$2" = "$3" ]; then log "  OK   $1 = $3"; else log "  FAIL $1 = $3 (期望 $2)"; FAILS=$((FAILS+1)); fi; }
chk "线上 $LIVE_PORT /health"  200 "$(code http://127.0.0.1:$LIVE_PORT/health)"
chk "灰度 $CANARY_PORT /health" 200 "$(code http://127.0.0.1:$CANARY_PORT/health)"
if [ -s "$NEW_BIN" ]; then log "  OK   待发布二进制存在 ($(stat -c%s "$NEW_BIN") bytes)"; else log "  FAIL 待发布二进制缺失: $NEW_BIN"; FAILS=$((FAILS+1)); fi
if [ -s "$NEW_BIN" ] && [ -s "$LIVE_BIN" ]; then
  OLD_MD5=$(md5sum "$LIVE_BIN" | cut -d' ' -f1); NEW_MD5=$(md5sum "$NEW_BIN" | cut -d' ' -f1)
  if [ "$OLD_MD5" != "$NEW_MD5" ]; then log "  OK   二进制不同 old=$OLD_MD5 new=$NEW_MD5"; else log "  FAIL 新旧二进制完全相同"; FAILS=$((FAILS+1)); fi
fi
nginx -t >/dev/null 2>&1 && log "  OK   nginx 配置语法" || { log "  FAIL nginx 配置语法"; FAILS=$((FAILS+1)); }
log "  当前 upstream -> $(upstream_port 2>/dev/null || echo '<未建立>')"
log "  在线连接  $LIVE_PORT=$(conns $LIVE_PORT)  $CANARY_PORT=$(conns $CANARY_PORT)"
BEFORE_BUNDLE=$(bundle); log "  当前边缘 bundle = ${BEFORE_BUNDLE:-<none>}"

if [ "$FAILS" -gt 0 ] && [ "$DRY_RUN" -eq 0 ]; then abort "发布前体检未通过 ($FAILS 项)"; fi

if [ "$DRY_RUN" -eq 1 ]; then
  echo
  log "DRY-RUN 结束：未对系统做任何修改。"
  [ "$FAILS" -gt 0 ] && log "注意：以上 $FAILS 项失败，正式发布会在此中止。"
  log "正式执行计划: 加 upstream -> nginx切$CANARY_PORT -> 排空$LIVE_PORT -> 换二进制重启 -> nginx切回$LIVE_PORT -> 排空$CANARY_PORT -> 灰度下线 -> 最终体检"
  exit 0
fi
log "PHASE 0 OK"

cp -a "$CONF" "$NGX_BAK" || abort "nginx 配置备份失败"
cp -a "$LIVE_BIN" "$OLD_BIN_BAK" || abort "二进制备份失败"
log "备份: $NGX_BAK | $OLD_BIN_BAK"
start_monitor

# ---------------- Phase 1 ----------------
log "PHASE 1: 建立 upstream（仍指向 $LIVE_PORT）"
if ! grep -q "^upstream $UPSTREAM_NAME" "$CONF"; then
  sed -i "1i upstream $UPSTREAM_NAME {\n    server 127.0.0.1:$LIVE_PORT;\n}\n" "$CONF"
  sed -i "s|proxy_pass http://127.0.0.1:$LIVE_PORT;|proxy_pass http://$UPSTREAM_NAME;|g" "$CONF"
fi
log "upstream=$(upstream_port) proxy_pass_lines=$(grep -c "proxy_pass http://$UPSTREAM_NAME;" "$CONF")"
[ "$(upstream_port)" = "$LIVE_PORT" ] || abort "phase1 upstream != $LIVE_PORT"
reload_ngx || abort "nginx reload 失败 (phase1)"
sleep 2
[ "$(site /health)" = "200" ] || abort "phase1 后站点不可用"
log "PHASE 1 OK (site=$(site /health), bundle=$(bundle))"

# ---------------- Phase 2 ----------------
log "PHASE 2: nginx -> 新后端 $CANARY_PORT"
set_backend "$CANARY_PORT"
[ "$(upstream_port)" = "$CANARY_PORT" ] || abort "phase2 upstream != $CANARY_PORT"
reload_ngx || abort "nginx reload 失败 (phase2)"
sleep 2
AFTER_BUNDLE=$(bundle); log "切换后边缘 bundle = ${AFTER_BUNDLE:-<none>}"
[ "$(site /health)" = "200" ] || abort "切到 $CANARY_PORT 后站点不可用"
log "PHASE 2 OK：全部用户流量已在新代码上，旧代码仍在 $LIVE_PORT 继续服务在途连接"

# ---------------- Phase 3 ----------------
log "PHASE 3: 排空 $LIVE_PORT -> 停止 -> 换新二进制 -> 启动"
drain "$LIVE_PORT" "$DRAIN_CAP"
systemctl stop sub2api.service || abort "停止 sub2api.service 失败"
log "旧 systemd 实例已停止 ($LIVE_PORT 释放)"
install -o sub2api -g sub2api -m 755 "$NEW_BIN" "$LIVE_BIN" || abort "安装新二进制失败"
[ "$(md5sum "$LIVE_BIN" | cut -d' ' -f1)" = "$NEW_MD5" ] || abort "安装后 md5 不一致"
systemctl start sub2api.service || abort "启动 sub2api.service 失败"
if ! wait_health "$LIVE_PORT" 120; then
  log "新实例不健康 -> 回滚旧二进制"
  journalctl -u sub2api.service -n 30 --no-pager 2>&1 | tail -32
  systemctl stop sub2api.service 2>/dev/null
  install -o sub2api -g sub2api -m 755 "$OLD_BIN_BAK" "$LIVE_BIN"
  systemctl start sub2api.service
  wait_health "$LIVE_PORT" 90
  point_nginx_at_healthy
  abort "新二进制启动失败，已回滚旧二进制（站点保持可用）"
fi
sleep 2
log "PHASE 3 OK：systemd $LIVE_PORT 现在运行新代码"

# ---------------- Phase 4 ----------------
log "PHASE 4: nginx -> $LIVE_PORT (新代码)"
set_backend "$LIVE_PORT"
[ "$(upstream_port)" = "$LIVE_PORT" ] || abort "phase4 upstream != $LIVE_PORT"
reload_ngx || abort "nginx reload 失败 (phase4)"
sleep 2
if [ "$(site /health)" != "200" ]; then
  log "站点在 $LIVE_PORT 上不健康 -> nginx 退回 $CANARY_PORT"
  set_backend "$CANARY_PORT"; reload_ngx
  abort "phase4 后站点不健康，已退回灰度"
fi
log "PHASE 4 OK (site=$(site /health), bundle=$(bundle))"

# ---------------- Phase 5 ----------------
log "PHASE 5: 排空 $CANARY_PORT -> 灰度下线"
drain "$CANARY_PORT" "$CANARY_DRAIN_CAP"
systemctl stop "$CANARY_UNIT" 2>/dev/null
sleep 2
ss -lntp 2>/dev/null | grep -q ":$CANARY_PORT" && log "WARN: $CANARY_PORT 仍在监听" || log "$CANARY_PORT 已释放"
systemctl reset-failed "$CANARY_UNIT" 2>/dev/null

# ---------------- Phase 6 ----------------
log "PHASE 6: 最终体检"
sleep 3
bash "$REPO/scripts/healthcheck.sh" "post-deploy" 2>/dev/null || true
echo; cd "$REPO" && bash scripts/customizations-verify.sh --live 2>&1 | tail -n 16
echo; log "最终边缘 bundle = $(bundle)"
echo; stop_monitor; report_monitor
echo
log "================ 发布完成 $STAMP ================"
echo "$STAMP" > "$STAGE/deploy.done"