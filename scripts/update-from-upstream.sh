#!/usr/bin/env bash
#
# update-from-upstream.sh — 一键从官方合并更新，保留本地二开（魔改）
#
# 用法:
#   ./scripts/update-from-upstream.sh              # 交互式：合并官方 -> 停在冲突/测试处
#   ./scripts/update-from-upstream.sh --no-verify  # 跳过测试/构建（不推荐）
#
# 设计原则:
#   - 永远用 merge，不 reset/rebase，保留你分支里的魔改提交
#   - 合并前自动打保护分支 + 快照，随时可回退
#   - 已开启 git rerere：同类冲突第二次自动复用上次解决
#   - 冲突时脚本停下，人工按 CUSTOMIZATIONS.md 解决，再继续
#
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

TS="$(date +%Y%m%d-%H%M%S)"
WORK_BRANCH="$(git branch --show-current)"
UPSTREAM_REMOTE="origin"
UPSTREAM_BRANCH="main"
BACKUP_REMOTE="custom"
VERIFY=1
[ "${1:-}" = "--no-verify" ] && VERIFY=0

log(){ printf '\n\033[1;36m==> %s\033[0m\n' "$*"; }
warn(){ printf '\n\033[1;33m[!] %s\033[0m\n' "$*"; }
die(){ printf '\n\033[1;31m[x] %s\033[0m\n' "$*" >&2; exit 1; }

# 0. 前置检查
[ -f CUSTOMIZATIONS.md ] || warn "缺少 CUSTOMIZATIONS.md，合并时无魔改核对清单！"
if ! git diff --quiet || ! git diff --cached --quiet; then
  die "工作区有未提交改动，请先 commit 或 stash 再更新。"
fi

# 0.4 工具链预检：前端 build 依赖 pnpm（官方 build 脚本内部调用 pnpm）。
#     缺 pnpm 时 npm run build 会 exit 127，且报错很难懂。
if [ "$VERIFY" = "1" ]; then
  log "工具链预检"
  command -v node >/dev/null 2>&1 || die "缺少 node，无法构建前端。"
  command -v go   >/dev/null 2>&1 || die "缺少 go，无法构建后端。"
  if ! command -v pnpm >/dev/null 2>&1; then
    warn "缺少 pnpm（前端 build 脚本内部调用 pnpm，缺失会 exit 127）。"
    echo "    修复：corepack enable pnpm   # 或 npm i -g pnpm"
    die "工具链不完整，拒绝开始合并。"
  fi
  log "node $(node -v) / pnpm $(pnpm --version) / $(go version | awk '{print $3}')"
fi

# 0.5 合并前基线体检：确认「动手前」全部魔改是完好的
#     这样合并后一旦变红，就能确定是这次合并弄坏的，定位零成本。
if [ -x scripts/customizations-verify.sh ]; then
  log "合并前基线体检（确认起点是干净的）"
  if ! ./scripts/customizations-verify.sh >/tmp/custom-baseline-$TS.log 2>&1; then
    warn "合并前基线就是红的！请先修好这些再更新（日志：/tmp/custom-baseline-$TS.log）"
    tail -20 /tmp/custom-baseline-$TS.log
    die "基线不干净，拒绝开始合并。"
  fi
  log "基线全绿，魔改与构建工具链完好。开始合并。"
fi

# 1. 开启 rerere（复用冲突解决记忆）
# 确保版本库内的 git hook 生效（core.hooksPath 存在 .git/config 里，不随仓库走，
# 新克隆/新机器上会丢，所以每次更新时自愈一次）。
if [ -d scripts/git-hooks ]; then
  git config core.hooksPath scripts/git-hooks
  log "已启用提交守卫 core.hooksPath=scripts/git-hooks"
fi
git config rerere.enabled true
git config rerere.autoupdate true

# 2. 保护：备份分支 + 源码快照
log "创建保护分支 protect-before-update-$TS"
git branch "protect-before-update-$TS" "$WORK_BRANCH"
mkdir -p backups
log "创建源码快照 backups/sub2api-src-before-update-$TS.tar.gz"
tar --exclude='./.git' --exclude='./frontend/node_modules' --exclude='./backups' \
    -czf "backups/sub2api-src-before-update-$TS.tar.gz" . || warn "快照打包有告警（可忽略）"

# 3. 拉官方
log "拉取官方 $UPSTREAM_REMOTE/$UPSTREAM_BRANCH"
git fetch "$UPSTREAM_REMOTE" --tags
AHEAD="$(git rev-list --count "$UPSTREAM_REMOTE/$UPSTREAM_BRANCH".."$WORK_BRANCH" 2>/dev/null || echo '?')"
BEHIND="$(git rev-list --count "$WORK_BRANCH".."$UPSTREAM_REMOTE/$UPSTREAM_BRANCH" 2>/dev/null || echo '?')"
log "你领先官方 $AHEAD 个提交（魔改）；官方领先你 $BEHIND 个提交（待合并）"
if [ "$BEHIND" = "0" ]; then
  log "已是最新，无官方更新需要合并。"
  exit 0
fi

# 4. 合并
log "合并官方更新到 $WORK_BRANCH ..."
if git merge --no-edit "$UPSTREAM_REMOTE/$UPSTREAM_BRANCH"; then
  log "合并成功，无冲突。"
else
  warn "出现合并冲突。请按以下步骤处理："
  echo "   1) 查看冲突文件：git status | grep both"
  echo "   2) 对照 CUSTOMIZATIONS.md，保留你的魔改功能（不要被官方覆盖）"
  echo "   3) 编辑解决后：git add <文件>"
  echo "   4) 全部解决后：git commit（完成合并）"
  echo "   5) 再手动跑一次本脚本的验证段，或： cd frontend && npm run typecheck && npm run build"
  echo
  echo "   已开启 rerere：下次同样冲突会自动复用你这次的解决方式。"
  die "合并暂停在冲突处（这是正常的，解决后继续）。回退可用：git merge --abort 或切回 protect-before-update-$TS"
fi

# 4.5 魔改完整性闸门（最重要：魔改丢了一条都不许继续）
if [ -x scripts/customizations-verify.sh ]; then
  log "魔改完整性体检（硬闸门，逐条核对 CUSTOMIZATIONS.md）"
  if ! ./scripts/customizations-verify.sh; then
    warn "魔改体检未通过！上面标记 XX 的功能已被官方改动覆盖。"
    echo "   1) 对照 CUSTOMIZATIONS.md 逐条恢复被覆盖的代码"
    echo "   2) 重新跑 scripts/customizations-verify.sh 直到全绿"
    echo "   3) 修复后重新 commit，再继续部署"
    echo "   回退：git merge --abort 或切回 protect-before-update-$TS 分支"
    die "魔改体检失败，已阻止继续（保护你的功能不被静默丢掉）。"
  fi
else
  warn "缺少 scripts/customizations-verify.sh，无法自动核对魔改，请手工对照 CUSTOMIZATIONS.md。"
fi

# 5. 验证
if [ "$VERIFY" = "1" ]; then
  log "验证：前端 typecheck + build"
  ( cd frontend && npm run typecheck && npm run build ) || die "前端验证失败，请检查。"
  log "验证：后端关键包测试"
  ( cd backend && go test ./internal/service ./internal/handler ./internal/server/... ) || warn "后端测试有失败，请人工确认是否与魔改相关。"
  log "验证：带 embed 的后端构建"
  ( cd backend && go build -tags embed -o /tmp/sub2api-update-$TS ./cmd/server ) && log "构建产物：/tmp/sub2api-update-$TS"
else
  warn "已跳过验证（--no-verify）。"
fi

# 6. 推送备份
log "推送到私有备份远程 $BACKUP_REMOTE/$WORK_BRANCH"
git push "$BACKUP_REMOTE" "$WORK_BRANCH" || warn "推送备份失败（可稍后手动 git push $BACKUP_REMOTE $WORK_BRANCH）"

log "部署前最后确认（含线上状态与二进制内容）："
./scripts/customizations-verify.sh --live || die "部署前体检未通过，请勿部署。"

log "完成。部署步骤（零中断，禁止直接 systemctl restart —— 会掐断在途 AI 流式调用）："
echo "   bash scripts/build-and-stage.sh                 # 构建 + 8081 灰度体检（线上无感知）"
echo "   bash scripts/zero-downtime-deploy.sh --dry-run  # 可选：只体检并打印计划，不改任何东西"
echo "   bash scripts/zero-downtime-deploy.sh            # 蓝绿切换 + 连接排空，零中断"
echo "   部署后务必再跑一次： ./scripts/customizations-verify.sh --live"
echo "   详见 CUSTOMIZATIONS.md 第 19 节「零中断更新工具链」。"
