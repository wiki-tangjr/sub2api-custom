#!/usr/bin/env bash
# ============================================================
# customizations-verify.sh — Sub2API 魔改完整性体检
#
# 用法:
#   ./scripts/customizations-verify.sh           # 只查源码/仓库（升级后跑）
#   ./scripts/customizations-verify.sh --live    # 源码 + 线上部署（部署后跑）
#
# 退出码: 0 = 全部保留   1 = 有魔改丢失
# 任何一条魔改丢失都会让本脚本以非 0 退出，可直接接进 git hook / CI。
# ============================================================
set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT" || exit 1

LIVE=0
[ "${1:-}" = "--live" ] && LIVE=1

C_OK=$'\033[32m'; C_BAD=$'\033[31m'; C_WARN=$'\033[33m'; C_DIM=$'\033[2m'; C_RST=$'\033[0m'
M_BAD=0; M_NAME=""; MOD_N=0; MOD_PASS=0; MOD_LOST=0; GBAD=0
LOST_LIST=()

begin(){ MOD_N=$((MOD_N+1)); M_NAME="$1"; M_BAD=0; printf '\n%s[%d] %s%s\n' "$C_DIM" "$MOD_N" "$1" "$C_RST"; }
ok(){ printf "   ${C_OK}OK${C_RST}  %s\n" "$1"; }
bad(){ M_BAD=$((M_BAD+1)); GBAD=$((GBAD+1)); printf "   ${C_BAD}XX${C_RST}  %s\n" "$1"; }
end(){ if [ "$M_BAD" -eq 0 ]; then MOD_PASS=$((MOD_PASS+1)); else MOD_LOST=$((MOD_LOST+1)); LOST_LIST+=("$M_NAME"); fi; }
warn(){ printf "   ${C_WARN}!!${C_RST}  %s\n" "$1"; }

# --- 断言工具 ---
f(){  [ -f "$2" ] && ok "$1" || bad "$1  [缺文件] $2"; }                      # 文件必须存在
g(){  if [ -f "$3" ] && grep -Fqs -- "$2" "$3"; then ok "$1"; else bad "$1  [标记 '$2' 不在 $3]"; fi; }   # 内容必须含标记
geq(){ local n; n=$(grep -Fo -- "$2" "$3" 2>/dev/null | wc -l); [ "$n" -ge "$4" ] && ok "$1" || bad "$1  [标记 '$2' 期望>=$4 实际 $n]"; }
lmx(){ local n; n=$(grep -Fo -- "$2" "$3" 2>/dev/null | wc -l); [ "$n" -le "$4" ] && ok "$1" || bad "$1  [标记 '$2' 期望<=$4 实际 $n]"; }
not404(){ local c; c=$(curl -s -o /dev/null -w '%{http_code}' -X POST "http://127.0.0.1:8080$2" -H 'Content-Type: application/json' -d '{}' 2>/dev/null); if [ "$c" != "404" ] && [ -n "$c" ]; then ok "$1 (HTTP $c)"; else bad "$1 (HTTP $c = 路由丢了)"; fi; }

printf '%sSub2API 魔改完整性体检%s  仓库: %s\n' "$C_DIM" "$C_RST" "$REPO_ROOT"

# ============ #1 分销/推广增强 ============
begin "#1 分销/推广增强"
f "返利周期迁移"        backend/migrations/145_affiliate_user_rebate_cycle_overrides.sql
f "隐藏分销迁移"        backend/migrations/146_affiliate_hide_for_invitees.sql
g "返利周期字段"        aff_rebate_duration_days      backend/internal/handler/admin/affiliate_handler.go
g "对受邀人隐藏"        hide_affiliate_for_invitees   backend/internal/handler/admin/affiliate_handler.go
g "前端隐藏标记"        affiliate_hidden              frontend/src/types/index.ts
end

# ============ #2 路由预取 ============
begin "#2 前端路由预取 / 性能优化"
f "预取 composable"     frontend/src/composables/useRoutePrefetch.ts
g "路由接入预取"        useRoutePrefetch              frontend/src/router/index.ts
g "fast-glass 样式"     fast-glass                    frontend/src/style.css
end

# ============ #3 图片延迟感知调度器 ============
begin "#3 OpenAI 图片延迟感知调度器"
f "调度器主体"          backend/internal/service/openai_image_scheduler.go
g "账号调度器接入"      ImageEndpoint                 backend/internal/service/openai_account_scheduler.go
g "网关调度接入"        ImageEndpoint                 backend/internal/service/openai_gateway_scheduling.go
end

# ============ #4 视频/即梦代理 ============
begin "#4 OpenAI 兼容视频 / 即梦代理"
f "视频 handler"        backend/internal/handler/openai_videos.go
g "即梦字段改写"        patchJimengVideoCreateBody    backend/internal/service/openai_videos.go
g "即梦模型判定"        isJimengVideoModel            backend/internal/service/openai_videos.go
g '/v1 group 前缀' 'gateway := r.Group("/v1")'   backend/internal/server/routes/gateway.go
g "videos 路由注册" 'gateway.Any("/videos"'    backend/internal/server/routes/gateway.go
g "jimeng 路由注册" '/jimeng/*subpath'       backend/internal/server/routes/gateway.go
end

# ============ #5 Gemini / Veo ============
begin "#5 Google Gemini / Veo 视频接口"
f "专用路径护栏"        backend/internal/service/gemini_upstream_path_guard.go
f "v1beta handler"      backend/internal/handler/gemini_v1beta_handler.go
g "必须调用专用护栏"    sanitizedGeminiUpstreamPath   backend/internal/service/gemini_messages_compat_service.go
g "Veo 长任务动作"      predictLongRunning            backend/internal/service/vertex_service_account.go
lmx "护栏不可被官方版顶替(调用)" 'sanitizedUpstreamPathSuffix('  backend/internal/service/gemini_messages_compat_service.go 0
end

# ============ #6 API Keys 工具栏 ============
begin "#6 API Keys 工具栏布局"
f "KeysView"            frontend/src/views/user/KeysView.vue
end

# ============ #7 移动端适配 ============
begin "#7 前端移动端适配"
f "useViewport"         frontend/src/composables/useViewport.ts
g "表格页接入断点"      useViewport                   frontend/src/components/layout/TablePageLayout.vue
end

# ============ #8 Seedance ============
begin "#8 Seedance 原生接口兼容"
f "Seedance handler"    backend/internal/handler/seedance_handler.go
g "模型映射表"          seedanceModelMap              backend/internal/handler/seedance_handler.go
g "CORS 放行 seedance"  '/seedance/'                   backend/internal/server/middleware/cors.go
g "embed 绕过 seedance" '/seedance/'                   backend/internal/web/embed_on.go
end

# ============ #9 CORS 预检 ============
begin "#9 OpenAI 端点 CORS 预检放行"
geq "无条件放行分支(>=2)" 'originAllowed = true'        backend/internal/server/middleware/cors.go 2
g   "放行 /v1/ 前缀"      '"/v1/"'                      backend/internal/server/middleware/cors.go
end

# ============ #10 ICP + 公安备案（合规）============
begin "#10 前端底部 ICP + 公安备案（合规，绝不能丢）"
g "登录页 ICP"    2026013786        frontend/src/components/layout/AuthLayout.vue
g "主界面 ICP"    2026013786        frontend/src/components/layout/AppLayout.vue
g "首页 ICP"      2026013786        frontend/src/views/HomeView.vue
g "登录页 公安号" 53011102001665    frontend/src/components/layout/AuthLayout.vue
g "主界面 公安号" 53011102001665    frontend/src/components/layout/AppLayout.vue
g "首页 公安号"   53011102001665    frontend/src/views/HomeView.vue
f "公安图标文件"  frontend/public/assets/image/gongan-beian.png
if [ -d backend/internal/web/dist ]; then
  nbeian=$(grep -Frl "53011102001665" backend/internal/web/dist/assets 2>/dev/null | wc -l)
  [ "$nbeian" -ge 1 ] && ok "内嵌前端含公安号 ($nbeian 个 asset)" || bad "内嵌前端无公安号！需重跑 npm run build + go build -tags embed"
fi
end

# ============ #11 自定义菜单打开方式 ============
begin "#11 自定义菜单打开方式 (iframe / new_tab)"
g "类型定义 open_mode"   open_mode        frontend/src/types/index.ts
g "后台选择控件"         open_mode        frontend/src/views/admin/SettingsView.vue
g "新标签页用原始URL"    externalUrl      frontend/src/views/user/CustomPageView.vue
g "open_mode 归一化"     'open_mode: item.open_mode === "new_tab" ? "new_tab" : "iframe"' frontend/src/views/admin/SettingsView.vue
g "侧栏新标签打开"       '_blank'         frontend/src/components/layout/AppSidebar.vue
g "DTO 字段"             open_mode        backend/internal/handler/dto/settings.go
end

# ============ #12 客服联系方式扩展（可配置条目 + 旧字段兼容）============
begin "#12 客服联系方式扩展（可配置条目列表 + 旧字段兼容）"
# --- 旧字段（v1）兼容，任何一条都不能丢 ---
g "Telegram 字段解析"    TelegramGroupURL     backend/internal/service/setting_parse.go
g "二维码字段解析"       WeChatGroupQRCode    backend/internal/service/setting_parse.go
g "公开设置暴露"         SettingKeyTelegramGroupURL backend/internal/service/setting_public.go
g "管理端回显(易漏)"     TelegramGroupURL     backend/internal/handler/admin/setting_handler.go
g "后台设置界面"         telegram_group_url   frontend/src/views/admin/SettingsView.vue
g "文案配置字段"         SettingKeyContactSectionTitle backend/internal/service/domain_constants.go
geq "设置键唯一写入"      'updates[SettingKeyTelegramGroupURL]'   backend/internal/service/setting_update.go 1
lmx "设置键不重复写入"    'updates[SettingKeyTelegramGroupURL]'   backend/internal/service/setting_update.go 1
# --- 新版（v2, 2026-09-18）：后台可增删改的 contact_entries 条目列表 ---
g "设置键 contact_entries" contact_entries        backend/internal/service/domain_constants.go
f "条目解析服务"           backend/internal/service/contact_entries.go
g "条目解析/归一化"        decodeContactEntriesJSON backend/internal/service/contact_entries.go
g "旧字段降级兼容"         resolveContactEntries backend/internal/service/contact_entries.go
g "公开只输出启用项"       resolvePublicContactEntries backend/internal/service/setting_public.go
g "管理端写入校验"         validateContactEntries backend/internal/handler/admin/setting_handler_update.go
g "条目 DTO"              ContactEntry          backend/internal/handler/dto/contact_entries.go
g "后台条目编辑器"         ContactEntriesEditor  frontend/src/views/admin/SettingsView.vue
f "编辑器组件"            frontend/src/views/admin/settings/ContactEntriesEditor.vue
f "共享展示组件"          frontend/src/components/common/ContactEntries.vue
f "条目内容渲染"          frontend/src/components/common/ContactEntryBody.vue
f "条目图标渲染"          frontend/src/components/common/ContactEntryIcon.vue
f "条目归一化工具"        frontend/src/utils/contactEntries.ts
g "顶栏接入"              ContactEntries        frontend/src/components/layout/AppHeader.vue
g "个人中心接入"          ContactEntries        frontend/src/views/user/ProfileView.vue
g "兑换页接入"            ContactEntries        frontend/src/views/user/RedeemView.vue
g "二维码扫码提示"        contactScanHint       frontend/src/i18n/locales/zh/common.ts
g "条目表单 API"          contact_entries        frontend/src/api/admin/settings.ts
g "条目类型定义"          ContactEntry          frontend/src/types/index.ts
end

# ============ #13 图片模型 driver 放行 ============
begin "#13 图片模型 driver 放行"
g "环境开关读取"    SUB2API_ALLOW_IMAGE_ONLY_RESPONSES_DRIVER backend/internal/service/openai_images.go
g "driver 放行函数" imageOnlyResponsesDriverAllowed             backend/internal/service/openai_images.go
g "Gemini 图片模型" isGeminiNativeImageModel                    backend/internal/service/openai_images.go
g "归一化改造"      imageOnlyResponsesDriverAllowed             backend/internal/service/openai_codex_transform.go
upstream_main=$(git rev-parse --verify -q origin/main >/dev/null 2>&1 && echo yes || echo no)
if [ -x scripts/install-deploy-assets.sh ]; then
  if ./scripts/install-deploy-assets.sh --check >/tmp/.deploy-check.txt 2>&1; then
    ok "部署资产一致（#13 生产开关 drop-in + 服务已生效）"
  else
    bad "部署资产缺失/不一致！跑 ./scripts/install-deploy-assets.sh 修复"
    sed 's/^/        /' /tmp/.deploy-check.txt
  fi
  rm -f /tmp/.deploy-check.txt
elif [ -f /etc/systemd/system/sub2api.service.d/50-image-only-driver.conf ]; then
  warn "drop-in 存在，但缺 scripts/install-deploy-assets.sh，无法校验一致性"
else
  bad "生产开关 drop-in 缺失！重装/迁移时必丢，跑 ./scripts/install-deploy-assets.sh"
fi
end

# ============ #14 构建工具链（魔改能否落地的前提）============
begin "#14 构建工具链（前端 build 脚本内部调用 pnpm）"
f "pnpm 构建白名单"     frontend/pnpm-workspace.yaml
g "esbuild 允许构建"    'esbuild: true'   frontend/pnpm-workspace.yaml
g "vue-demi 允许构建"   'vue-demi: true'  frontend/pnpm-workspace.yaml
if command -v pnpm >/dev/null 2>&1; then ok "pnpm 可用 ($(pnpm --version))"; else bad "缺少 pnpm：npm run build 会 exit 127（修复：corepack enable pnpm）"; fi
if command -v node >/dev/null 2>&1; then ok "node 可用 ($(node -v))";          else bad "缺少 node，无法构建前端"; fi
if command -v go   >/dev/null 2>&1; then ok "go 可用 ($(go version | awk '{print $3}'))"; else bad "缺少 go，无法构建后端"; fi
end

# ============ 源码体检小结 ============
printf '\n%s============================================================%s\n' "$C_DIM" "$C_RST"
if [ "$MOD_LOST" -eq 0 ]; then
  printf '%-28s %s%d/%d 全部保留%s\n' "源码体检" "$C_OK" "$MOD_PASS" "$MOD_N" "$C_RST"
else
  printf '%-28s %s%d/%d 保留，%d 条丢失%s\n' "源码体检" "$C_BAD" "$MOD_PASS" "$MOD_N" "$MOD_LOST" "$C_RST"
  for m in "${LOST_LIST[@]}"; do printf '   %s- 丢失: %s%s\n' "$C_BAD" "$m" "$C_RST"; done
fi

# ============ 线上部署体检 ============
if [ "$LIVE" -eq 1 ]; then
  begin "线上部署一致性（部署侧，git 管不到）"
  BIN=/opt/sub2api/sub2api
  if [ -f "$BIN" ]; then
    ok "二进制存在 ($(stat -c%s "$BIN") 字节, md5 $(md5sum "$BIN" | cut -c1-12)…)"
    ST=/tmp/.sub2api-verify-strings.txt
    strings -a "$BIN" > "$ST" 2>/dev/null
    geq "库内含 ICP 备案号"     2026013786      "$ST" 1
    geq "库内含公安备案号"      53011102001665  "$ST" 1
    geq "库内含 #13 驱动开关"   SUB2API_ALLOW_IMAGE_ONLY_RESPONSES_DRIVER "$ST" 1
    geq "库内含 #5 专用护栏"    sanitizedGeminiUpstreamPath "$ST" 1
    geq "库内含 #12 客服条目"   contact_entries "$ST" 1
    geq "库内含 #8 Seedance"    seedanceModelMap "$ST" 1
    rm -f "$ST"
  else
    bad "二进制不存在: $BIN"
  fi

  if systemctl is-active --quiet sub2api.service; then ok "服务 active"; else bad "服务未运行"; fi
  NR=$(systemctl show sub2api.service -p NRestarts --value 2>/dev/null)
  [ "${NR:-x}" = "0" ] && ok "NRestarts=0（未发生崩溃重启）" || bad "NRestarts=$NR（发生过崩溃重启）"

  HC=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8080/health 2>/dev/null)
  [ "$HC" = "200" ] && ok "/health 200" || bad "/health 返回 $HC"

  if curl -s http://127.0.0.1:8080/api/v1/settings/public 2>/dev/null | grep -Fqs contact_entries; then
    ok "#12 公开设置返回 contact_entries"
  else
    bad "#12 公开设置缺少 contact_entries（前台读不到客服条目）"
  fi

  CC=$(curl -s -o /dev/null -w '%{http_code}' -X OPTIONS http://127.0.0.1:8080/v1/chat/completions -H 'Origin: https://x' -H 'Access-Control-Request-Method: POST' 2>/dev/null)
  [ "$CC" = "204" ] && ok "#9 CORS 预检 204" || bad "#9 CORS 预检返回 $CC（应为 204）"

  not404 "#4 /v1/videos"                       /v1/videos
  not404 "#4 /v1/jimeng/videos/generations"    /v1/jimeng/videos/generations
  not404 "#8 /seedance/.../tasks"              /seedance/v3/contents/generations/tasks
  not404 "#5 :predictLongRunning"              /v1beta/models/veo-3.0-generate-001:predictLongRunning
  end

  printf '%s============================================================%s\n' "$C_DIM" "$C_RST"
fi

if [ "$GBAD" -eq 0 ]; then
  printf '%s结论: 全部通过，%d 项检查（含构建工具链）与部署状态完好。%s\n' "$C_OK" "$MOD_N" "$C_RST"
  exit 0
else
  printf '%s结论: 发现 %d 项问题，见上方 XX 行。不要部署，先修复。%s\n' "$C_BAD" "$GBAD" "$C_RST"
  exit 1
fi
