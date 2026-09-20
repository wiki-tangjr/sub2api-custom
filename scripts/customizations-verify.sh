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

# 允许对灰度实例验证：CUSTOM_VERIFY_PORT=8081 ./scripts/customizations-verify.sh --live
VERIFY_PORT="${CUSTOM_VERIFY_PORT:-8080}"
CB="http://127.0.0.1:${VERIFY_PORT}"

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
not404(){ local c; c=$(curl -s -o /dev/null -w '%{http_code}' -X POST "${CB}$2" -H 'Content-Type: application/json' -d '{}' 2>/dev/null); if [ "$c" != "404" ] && [ -n "$c" ]; then ok "$1 (HTTP $c)"; else bad "$1 (HTTP $c = 路由丢了)"; fi; }

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
g "公开白名单含板块标题"   'SettingKeyContactSectionTitle,'           backend/internal/service/setting_public.go
g "公开白名单含板块描述"   'SettingKeyContactSectionDescription,'     backend/internal/service/setting_public.go
g "公开白名单含板块样式"   'SettingKeyContactSectionStyle,'           backend/internal/service/setting_public.go
g "公开白名单含 TG 文案"     'SettingKeyTelegramEntryLabel,'            backend/internal/service/setting_public.go
g "公开白名单含微信群文案"   'SettingKeyWeChatGroupEntryLabel,'         backend/internal/service/setting_public.go
g "公开白名单含微信客服文案" 'SettingKeyWeChatContactEntryLabel,'       backend/internal/service/setting_public.go
g "SSR 注入含板块标题"     'ContactSectionTitle:                 settings.ContactSectionTitle' backend/internal/service/setting_public.go
g "SSR 注入含板块样式"     'ContactSectionStyle:                 settings.ContactSectionStyle' backend/internal/service/setting_public.go
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

# ============ #15 充值/订阅公告 + 订阅套餐每排数量 ============
begin "#15 充值/订阅公告 + 订阅套餐每排数量"
g "设置键 充值公告"        PAYMENT_RECHARGE_NOTICE        backend/internal/service/payment_config_service.go
g "设置键 订阅公告"        PAYMENT_SUBSCRIPTION_NOTICE    backend/internal/service/payment_config_service.go
g "设置键 每排套餐数"      SUBSCRIPTION_PLANS_PER_ROW     backend/internal/service/payment_config_service.go
g "DTO 字段 充值公告"      RechargeNotice                 backend/internal/handler/dto/settings.go
g "DTO 字段 订阅公告"      SubscriptionNotice             backend/internal/handler/dto/settings.go
g "管理端写入校验"         recharge_notice                backend/internal/handler/admin/setting_handler_update.go
g "公开配置输出"           RechargeNotice                 backend/internal/service/payment_config_service.go
g "后台设置界面"           subscriptionNotice             frontend/src/views/admin/SettingsView.vue
g "充值页公告渲染"         renderedRechargeNotice         frontend/src/views/user/PaymentView.vue
g "订阅页公告渲染"         renderedSubscriptionNotice     frontend/src/views/user/PaymentView.vue
g "前台读取每排套餐数"     normalizePlansPerRow           frontend/src/views/user/PaymentView.vue
g "套餐页每排设置入口"     savePlansPerRow                frontend/src/views/admin/orders/AdminPaymentPlansView.vue
end

# ============ #16 充值档位 / 快捷金额 / 阶梯优惠 ============
begin "#16 充值档位 / 快捷金额 / 阶梯优惠"
g "设置键 快捷金额"        RECHARGE_QUICK_AMOUNTS         backend/internal/service/payment_config_service.go
g "设置键 阶梯优惠"        RECHARGE_DISCOUNT_TIERS        backend/internal/service/payment_config_service.go
f "金额计算服务"           backend/internal/service/payment_amounts.go
g "快捷金额解析"           parseRechargeQuickAmounts      backend/internal/service/payment_amounts.go
g "阶梯优惠解析"           parseRechargeDiscountTiers     backend/internal/service/payment_amounts.go
g "订单应用优惠"           resolveRechargeDiscount        backend/internal/service/payment_order.go
g "DTO 快捷金额"           RechargeQuickAmounts           backend/internal/handler/dto/settings.go
g "管理端写入校验"         recharge_quick_amounts         backend/internal/handler/admin/setting_handler_update.go
g "后台设置界面"           recharge_quick_amounts         frontend/src/views/admin/SettingsView.vue
g "前台快捷金额组件"       quickAmounts                   frontend/src/components/payment/AmountInput.vue
g "前台优惠提示"           rechargeDiscountTiers          frontend/src/views/user/PaymentView.vue
end

# ============ #17 代理商体系 + 邀请返利隐藏 ============
begin "#17 代理商体系 + 邀请返利隐藏"
f "代理商迁移"            backend/migrations/239_affiliate_agent_hierarchy.sql
g "迁移 agent_level"      agent_level                    backend/migrations/239_affiliate_agent_hierarchy.sql
g "迁移 上级代理"         agent_parent_user_id           backend/migrations/239_affiliate_agent_hierarchy.sql
g "迁移 完整邮箱"         show_full_email                backend/migrations/239_affiliate_agent_hierarchy.sql
g "迁移 隐藏自身返利"     hide_affiliate_for_self        backend/migrations/239_affiliate_agent_hierarchy.sql
g "仓储 设置代理商等级"   SetAgentLevel                  backend/internal/repository/affiliate_repo.go
g "仓储 下级代理列表"     ListSubAgents                  backend/internal/repository/affiliate_repo.go
g "隐藏判定含自身开关"    hide_affiliate_for_self        backend/internal/repository/affiliate_repo.go
g "服务 一级代理常量"     AffiliateAgentLevelFirst       backend/internal/service/affiliate_service.go
g "服务 二级代理常量"     AffiliateAgentLevelSecond      backend/internal/service/affiliate_service.go
g "服务 比例不得超过上级" AgentRateTooHigh               backend/internal/service/affiliate_service.go
g "服务 二级比例兜底夹取" AffiliateAgentLevelSecond      backend/internal/service/affiliate_service.go
g "服务 代理商列表"       GetAgentInvitees               backend/internal/service/affiliate_service.go
g "服务 设置二级代理"     SetSubAgent                    backend/internal/service/affiliate_service.go
g "服务 隐藏下级入口"     SetInviteeAffiliateHidden      backend/internal/service/affiliate_service.go
g "服务 完整邮箱策略"     showFullEmail                  backend/internal/service/affiliate_service.go
g "用户端处理器"          SetAffiliateSubAgent           backend/internal/handler/user_handler.go
g "用户端路由 代理商列表" "/aff/agents"                   backend/internal/server/routes/user.go
g "用户端路由 设置二级"   "/aff/agents/set"               backend/internal/server/routes/user.go
g "用户端路由 隐藏下级"   invitees/:user_id/hide         backend/internal/server/routes/user.go
g "管理端 代理商等级"     agent_level                    backend/internal/handler/admin/affiliate_handler.go
g "管理端 完整邮箱"       show_full_email                backend/internal/handler/admin/affiliate_handler.go
g "管理端 隐藏自身返利"   hide_affiliate_for_self        backend/internal/handler/admin/affiliate_handler.go
g "前台 一级代理卡片"     is_level_one_agent             frontend/src/views/user/AffiliateView.vue
g "前台 API 代理商"       getAffiliateAgents             frontend/src/api/user.ts
g "后台 代理商等级列"     agentLevel                     frontend/src/views/admin/SettingsView.vue
g "后台 完整邮箱选项"     showFullEmail                  frontend/src/views/admin/SettingsView.vue
g "后台 隐藏自身返利"     hideSelf                       frontend/src/views/admin/SettingsView.vue
end

# ============ #18 零中断更新工具链（蓝绿 + 连接排空）============
begin "#18 零中断更新工具链（蓝绿 + 连接排空）"
f "构建+灰度脚本"        scripts/build-and-stage.sh
f "零中断发布脚本"       scripts/zero-downtime-deploy.sh
f "只读健康检查脚本"     scripts/healthcheck.sh
g "连线排空"             drain                    scripts/zero-downtime-deploy.sh
g "灾备回指健康后端"     point_nginx_at_healthy   scripts/zero-downtime-deploy.sh
g "可用性采样"           start_monitor            scripts/zero-downtime-deploy.sh
g "可用性报告"           report_monitor           scripts/zero-downtime-deploy.sh
g "演练模式"             --dry-run                scripts/zero-downtime-deploy.sh
g "二进制差异闸门"       "新旧二进制完全相同"     scripts/zero-downtime-deploy.sh
g "embed 构建"           "go build -tags embed"   scripts/build-and-stage.sh
g "灰度日志隔离"         LOG_OUTPUT_FILE_PATH      scripts/build-and-stage.sh
g "灰度端口隔离"         SERVER_PORT              scripts/build-and-stage.sh
g "文档已记录工具链"     "零中断更新工具链"       CUSTOMIZATIONS.md
if bash -n scripts/build-and-stage.sh 2>/dev/null \
   && bash -n scripts/zero-downtime-deploy.sh 2>/dev/null \
   && bash -n scripts/healthcheck.sh 2>/dev/null; then
  ok "三个脚本 bash -n 语法检查通过"
else
  bad "脚本语法检查失败（bash -n）"
fi
CRF=0
for f in scripts/build-and-stage.sh scripts/zero-downtime-deploy.sh scripts/healthcheck.sh scripts/customizations-verify.sh; do
  if grep -q $'\r' "$f" 2>/dev/null; then bad "$f 含 CRLF（禁止用 sed 清 CR，会删掉所有字母 r）"; CRF=1; fi
done
[ "$CRF" -eq 0 ] && ok "工具链脚本均为 LF 换行（无 CRLF 隐患）"
end

# ============ #19 后台客服设置收敛为单一入口（旧字段折叠，2026-09-19）============
begin "#19 后台客服设置收敛（单一入口 + 旧字段折叠）"
# 条目列表必须是后台的第一个（也是唯一的主）客服配置入口
geq "唯一条目编辑器"     '<ContactEntriesEditor v-model="form.contact_entries" />' frontend/src/views/admin/SettingsView.vue 1
lmx "条目编辑器不重复"   '<ContactEntriesEditor v-model="form.contact_entries" />' frontend/src/views/admin/SettingsView.vue 1
g "唯一入口注释标记"     "魔改 #12 / #19"      frontend/src/views/admin/SettingsView.vue
# 旧字段必须仍然存在且每个只出现一次（被收进折叠面板，不允许被删除）
lmx "旧字段 contact_info 单处"        'v-model="form.contact_info"'           frontend/src/views/admin/SettingsView.vue 1
lmx "旧字段 telegram 单处"            'v-model="form.telegram_group_url"'     frontend/src/views/admin/SettingsView.vue 1
lmx "旧字段 微信群二维码 单处"        'v-model="form.wechat_group_qr_code"'   frontend/src/views/admin/SettingsView.vue 1
# 折叠面板本体
g "折叠状态 ref"         "const contactLegacyOpen = ref(false);" frontend/src/views/admin/SettingsView.vue
g "折叠面板标题"         "contactLegacy.title"        frontend/src/views/admin/SettingsView.vue
g "折叠面板说明"         "contactLegacy.notice"       frontend/src/views/admin/SettingsView.vue
g "折叠图标切换"         "contactLegacyOpen ? 'chevronUp' : 'chevronDown'" frontend/src/views/admin/SettingsView.vue
# 始终生效的板块标题 / 样式（被 AppHeader 与 ProfileView 消费，不能丢）
g "板块标题字段"         "form.contact_section_title"       frontend/src/views/admin/SettingsView.vue
g "板块样式字段"         "form.contact_section_style"       frontend/src/views/admin/SettingsView.vue
g "板块描述字段"         "form.contact_section_description" frontend/src/views/admin/SettingsView.vue
# 文案必须两种语言都有
g "中文折叠文案"         "contactLegacy:" frontend/src/i18n/locales/zh/admin/settings.ts
g "英文折叠文案"         "contactLegacy:" frontend/src/i18n/locales/en/admin/settings.ts
g "提交仍带 contact_info"  "contact_info: form.contact_info"            frontend/src/views/admin/SettingsView.vue
g "提交仍带 telegram"      "telegram_group_url: form.telegram_group_url"  frontend/src/views/admin/SettingsView.vue
g "提交仍带 二维码"        "wechat_group_qr_code: form.wechat_group_qr_code" frontend/src/views/admin/SettingsView.vue
g "文档已记录收敛"       "后台客服设置收敛" CUSTOMIZATIONS.md
end


# ============ #22 充值优惠区间化 + 后台控件化 + 公告美化（2026-09-19）============
begin "#22 充值优惠区间化 + 后台控件化 + 公告美化"
# 前后端共享同一套区间规则（前端只做展示预览，后端才是唯一真相）
f "前端档位工具"           frontend/src/utils/rechargeTiers.ts
g "前端 区间解析"          parseTierRangeText                 frontend/src/utils/rechargeTiers.ts
g "前端 下限兼容别名"      tierLowerBound                     frontend/src/utils/rechargeTiers.ts
g "前端 档位归一化"        normalizeRechargeDiscountTiers     frontend/src/utils/rechargeTiers.ts
g "前端 区间命中"          resolveRechargeDiscountPercent     frontend/src/utils/rechargeTiers.ts
g "前端 单测"              "describe('rechargeTiers'"          frontend/src/utils/__tests__/rechargeTiers.spec.ts
g "前端 金额输入改用共享规则" normalizeRechargeDiscountTiers  frontend/src/components/payment/AmountInput.vue
# 后端：区间解析 + 无条件档位判非法（防止 0:5 变成人人有优惠）
g "后端 区间解析"          parseRechargeTierRange             backend/internal/service/payment_amounts.go
g "后端 下限兼容别名"      tierLowerBound                     backend/internal/service/payment_amounts.go
g "后端 无条件档位判非法"  "maxVal <= 0"                      backend/internal/service/payment_amounts.go
g "后端 单测"              "func TestParseRechargeTierRange"  backend/internal/service/payment_recharge_tiers_test.go
# 后台：结构化控件替代裸文本框
g "后台 快捷金额编辑器"    RechargeQuickAmountsEditor         frontend/src/views/admin/SettingsView.vue
g "后台 优惠档位编辑器"    RechargeDiscountTiersEditor        frontend/src/views/admin/SettingsView.vue
g "后台 编辑器组件存在"    "defineProps<{"                    frontend/src/views/admin/settings/RechargeDiscountTiersEditor.vue
g "后台 每排套餐分段按钮"  PLAN_ROW_OPTIONS                   frontend/src/views/admin/SettingsView.vue
g "后台 提交前先归一化"    formatRechargeDiscountTiersText    frontend/src/views/admin/SettingsView.vue
# 公告视觉
g "公告紧凑样式"           announcement-markdown              frontend/src/styles/announcement-markdown.css
g "公告标题文案 zh"        "notice:"                          frontend/src/i18n/locales/zh/misc.ts
g "公告标题文案 en"        "notice:"                          frontend/src/i18n/locales/en/misc.ts
g "区间摘要文案 zh"        tierSummaryRange                   frontend/src/i18n/locales/zh/admin/settings.ts
g "区间摘要文案 en"        tierSummaryRange                   frontend/src/i18n/locales/en/admin/settings.ts
# 文档
g "文档已记录区间化"       "区间"                             CUSTOMIZATIONS.md
end

# ============ #23 客服条目分组 + 后台实时预览 + 悬停卡片修正（2026-09-20）============
begin "#23 客服条目分组 + 后台实时预览 + 悬停卡片修正"
# group 字段必须前后端镜像（少一边就会静默丢数据）
g "DTO group 字段"         'json:"group,omitempty"'  backend/internal/handler/dto/contact_entries.go
g "service 镜像结构"       'json:"group,omitempty"'  backend/internal/service/contact_entries.go
g "写入校验 group"         'item.Group = strings.TrimSpace(item.Group)'  backend/internal/handler/admin/setting_contact_entries.go
g "group 长度上限"         maxContactGroupLen                            backend/internal/handler/admin/setting_contact_entries.go
g "前端类型 group"         group?:                                    frontend/src/types/index.ts
# 前台渲染：分组 + 触屏兜底 + 悬停卡片定位
g "前台分组渲染"           groupedEntries                              frontend/src/components/common/ContactEntries.vue
g "分组不渲染空标题"       "if (!hasGroup) return [{ key: 'all', title: '', items }]" frontend/src/components/common/ContactEntries.vue
g "悬停卡片方向修正"       hoverCardClass                              frontend/src/components/common/ContactEntries.vue
g "顶栏下拉右对齐"         "if (props.variant === 'dropdown') return 'right-0 w-64'" frontend/src/components/common/ContactEntries.vue
g "触屏可悬停探测"         canHover                                    frontend/src/components/common/ContactEntries.vue
g "触屏点击回退弹窗"       "if (displayOf(item) === 'hover' && canHover.value) return" frontend/src/components/common/ContactEntries.vue
# 后台：实时预览 + 图标预览 + 控件提示
g "后台实时预览"           previewEntries                              frontend/src/views/admin/settings/ContactEntriesEditor.vue
g "后台预览区块文案"       previewTitle                                frontend/src/views/admin/settings/ContactEntriesEditor.vue
g "后台分组输入"           groupDatalistId                             frontend/src/views/admin/settings/ContactEntriesEditor.vue
g "后台控件提示"           displayHint                                 frontend/src/views/admin/settings/ContactEntriesEditor.vue
# 文案必须两种语言都有
g "中文 #23 文案"          displayHoverHint                            frontend/src/i18n/locales/zh/admin/settings.ts
g "英文 #23 文案"          displayHoverHint                            frontend/src/i18n/locales/en/admin/settings.ts
# 单测
f "前台单测"               frontend/src/components/common/__tests__/ContactEntries.spec.ts
g "单测覆盖分组"           "groups adjacent entries that share the same group name" frontend/src/components/common/__tests__/ContactEntries.spec.ts
g "单测覆盖触屏兜底"       "opens the modal when a hover entry is tapped on a device without hover support" frontend/src/components/common/__tests__/ContactEntries.spec.ts
# 文档
g "文档已记录 #23"         "客服条目分组"                              CUSTOMIZATIONS.md
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

  HC=$(curl -s -o /dev/null -w '%{http_code}' ${CB}/health 2>/dev/null)
  [ "$HC" = "200" ] && ok "/health 200" || bad "/health 返回 $HC"

  if curl -s ${CB}/api/v1/settings/public 2>/dev/null | grep -Fqs contact_entries; then
    ok "#12 公开设置返回 contact_entries"
  else
    bad "#12 公开设置缺少 contact_entries（前台读不到客服条目）"
  fi

  # #23: group 是纯可选字段，老配置的公开设置里不应出现 group（保证前台零变化）
  PUB=$(curl -s ${CB}/api/v1/settings/public 2>/dev/null)
  if printf '%s' "$PUB" | grep -Fqs contact_entries; then
    if printf '%s' "$PUB" | grep -Fqs '"group"'; then
      warn "#23 公开设置已含 group 字段（说明后台已启用分组，前台会显示分组标题）"
    else
      ok "#23 老配置无 group，前台与 #12 表现一致"
    fi
  fi

  CC=$(curl -s -o /dev/null -w '%{http_code}' -X OPTIONS ${CB}/v1/chat/completions -H 'Origin: https://x' -H 'Access-Control-Request-Method: POST' 2>/dev/null)
  [ "$CC" = "204" ] && ok "#9 CORS 预检 204" || bad "#9 CORS 预检返回 $CC（应为 204）"

  not404 "#4 /v1/videos"                       /v1/videos
  not404 "#4 /v1/jimeng/videos/generations"    /v1/jimeng/videos/generations
  not404 "#8 /seedance/.../tasks"              /seedance/v3/contents/generations/tasks
  not404 "#5 :predictLongRunning"              /v1beta/models/veo-3.0-generate-001:predictLongRunning
  end

  printf '%s============================================================%s\n' "$C_DIM" "$C_RST"
fi

if [ "$GBAD" -eq 0 ]; then
  printf '%s结论: 全部通过，%d 项检查（含构建 + 零中断发布工具链）与部署状态完好。%s\n' "$C_OK" "$MOD_N" "$C_RST"
  exit 0
else
  printf '%s结论: 发现 %d 项问题，见上方 XX 行。不要部署，先修复。%s\n' "$C_BAD" "$GBAD" "$C_RST"
  exit 1
fi
