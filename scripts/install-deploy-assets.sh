#!/usr/bin/env bash
# ============================================================
# install-deploy-assets.sh — 安装「git 管不到」的部署资产
#
# 为什么需要它：魔改 #13 的生产开关放在 systemd drop-in 里，
# 不在仓库工作区内。重装系统 / 迁移服务器 / 重建服务时，
# 这部分配置会静默消失（表现为生图极慢、502），而 git 一无所知。
# 本脚本把这些资产纳入版本控制并可一键恢复。
#
# 用法:
#   ./scripts/install-deploy-assets.sh          # 安装（幂等）
#   ./scripts/install-deploy-assets.sh --check  # 只检查是否一致
# ============================================================
set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT" || exit 1

CHECK=0
[ "${1:-}" = "--check" ] && CHECK=1

SRC="deploy/50-image-only-driver.conf"
DST_DIR="/etc/systemd/system/sub2api.service.d"
DST="$DST_DIR/50-image-only-driver.conf"

C_OK=$'\033[32m'; C_BAD=$'\033[31m'; C_DIM=$'\033[2m'; C_RST=$'\033[0m'
RC=0

[ -f "$SRC" ] || { printf '%s[x] 仓库内缺少 %s%s\n' "$C_BAD" "$SRC" "$C_RST"; exit 1; }

# 只比对「生效指令」：忽略注释与空行，避免注释措辞差异造成误报
effective(){ grep -vE '^[[:space:]]*(#|$)' "$1" | sed 's/[[:space:]]*$//' | sort; }

if [ "$CHECK" = "1" ]; then
  if [ ! -f "$DST" ]; then
    printf '%s[!] 部署资产缺失：%s（跑 ./scripts/install-deploy-assets.sh 安装）%s\n' "$C_BAD" "$DST" "$C_RST"
    RC=1
  elif ! diff -q <(effective "$SRC") <(effective "$DST") >/dev/null; then
    printf '%s[!] 部署资产生效项不一致：%s%s\n' "$C_BAD" "$DST" "$C_RST"
    diff -u <(effective "$DST") <(effective "$SRC") | head -20
    RC=1
  else
    printf '%s[OK] 部署资产生效项一致：%s%s\n' "$C_OK" "$DST" "$C_RST"
  fi
  # 检查服务实际生效的环境变量
  if systemctl show sub2api.service -p Environment 2>/dev/null | grep -Fq "SUB2API_ALLOW_IMAGE_ONLY_RESPONSES_DRIVER=1"; then
    printf '%s[OK] 服务已生效 SUB2API_ALLOW_IMAGE_ONLY_RESPONSES_DRIVER=1%s\n' "$C_OK" "$C_RST"
  else
    printf '%s[!] 服务未生效该环境变量（可能未 daemon-reload / 未重启）%s\n' "$C_BAD" "$C_RST"
    RC=1
  fi
  exit $RC
fi

printf '%s==> 安装部署资产（魔改 #13 生产开关）%s\n' "$C_DIM" "$C_RST"
mkdir -p "$DST_DIR"
if [ -f "$DST" ] && diff -q <(effective "$SRC") <(effective "$DST") >/dev/null; then
  printf '    %s[=] 已是最新，无需改动%s\n' "$C_DIM" "$C_RST"
else
  [ -f "$DST" ] && cp -a "$DST" "$DST.bak-$(date +%Y%m%d-%H%M%S)"
  cp "$SRC" "$DST"
  chmod 644 "$DST"
  printf '    %s[+] 已写入 %s%s\n' "$C_OK" "$DST" "$C_RST"
  systemctl daemon-reload && printf '    %s[+] daemon-reload 完成%s\n' "$C_OK" "$C_RST"
fi

printf '\n提示：环境变量变更需要重启服务才生效：\n    systemctl restart sub2api.service\n'
printf '校验：./scripts/install-deploy-assets.sh --check\n'