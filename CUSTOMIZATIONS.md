# CUSTOMIZATIONS.md — 本地二开（魔改）清单

> **这是本仓库的权威魔改记录。不依赖任何人的记忆。**
> 每次合并官方更新后，逐条核对本文件，确认下列功能全部保留并通过验证。
>
> - 你的工作分支：`image-latency-aware-scheduler-20260603`
> - 官方远程：`origin` → https://github.com/Wei-Shaw/sub2api.git
> - 你的私有备份远程：`custom` → https://github.com/wiki-tangjr/sub2api-custom.git
> - 更新流程：见 `scripts/update-from-upstream.sh`（一键更新脚本）

---

## 更新时的黄金法则

1. **永远用合并（merge），不要重来。** 你的魔改是分支里的真实提交，官方更新用 `git merge origin/main` 合并进来。
2. **冲突时，保留下列功能，不要被官方版本覆盖。** 已开启 `git rerere`，同类冲突第二次会自动复用你上次的解决方式。
3. **合并后必须跑验证**（见每条末尾 + 脚本内置的测试/构建）。
4. **迁移文件编号冲突**：不要强行重编号。迁移执行器按“文件名全文校验和”识别，两个同号 `NNN_*.sql` 可共存。历史上曾把本地迁移从 140/141 顺延为 142/143，仅为避免与官方新增迁移撞名，不是必须。
5. **必须用 `go build -tags embed` 构建**，否则前端不会嵌入二进制 → 整站 404。

---

## 魔改功能清单（合并后逐条核对）

### 1. 分销/推广（affiliate）增强
- **内容**：返利周期覆盖、对被邀请人隐藏分销信息（`aff_rebate_duration_days`、`hide_affiliate_for_invitees`、`affiliate_hidden`）。
- **关键文件**：`backend/internal/handler/admin/affiliate_handler.go`、`backend/internal/repository/affiliate_repo.go`、`backend/internal/handler/dto/settings.go`、相关前端分销页。
- **标记**：搜索 `aff_rebate_duration_days`、`hide_affiliate_for_invitees`、`affiliate_hidden` 应存在。
- **来源提交**：`399b5d85`。

### 2. 前端路由预取 / 性能优化（fast-glass / route prefetch）
- **内容**：路由预取 composable、首页与玻璃拟态性能优化。
- **标记**：搜索 `useRoutePrefetch`、`fast-glass` 应存在。
- **关键文件**：`frontend/src/composables/useRoutePrefetch.ts`、`frontend/src/views/HomeView.vue`。
- **来源提交**：`dfb44101`。

### 3. OpenAI 图片延迟感知调度器（image latency-aware scheduler）
- **内容**：图片端点的账号延迟感知调度、冷却、image endpoint 处理。
- **标记**：搜索 `openai_image_scheduler`、`ImageEndpoint`、文件 `backend/internal/service/openai_image_scheduler.go` 应存在。
- **关键文件**：`backend/internal/service/openai_image_scheduler.go`、`openai_account_scheduler.go`、`openai_gateway_service.go`、`backend/internal/handler/openai_images.go`。
- **来源提交**：`dfb44101`、`149ca269`（合并后修端点）。

### 4. OpenAI 兼容视频 / 即梦（Jimeng）代理转发
- **内容**：通用视频/即梦端点代理。**即梦创建任务走上游 `POST /v1/videos`**（字段 `model`/`prompt`/字符串 `seconds`/`aspect_ratio`），旧路径 `/v1/videos/generations`、`/v1/jimeng/videos/generations` 兼容转发，并自动把 `duration`/`second` 转字符串 `seconds`、删除上游不支持字段（`width`/`height`/`size`/`model_name`）。
- **标记**：搜索 `OpenAIVideos`、`video-ds-2.0-fast`、`isJimengVideoModel`、`patchJimengVideoCreateBody`、`openAIVideoUpstreamEndpoint` 应存在。
- **关键文件**：`backend/internal/service/openai_videos.go`、`backend/internal/handler/openai_videos.go`、`backend/internal/server/routes/gateway.go`（含精确 `/v1/videos`、`/videos` 路由）。
- **上游文档**：https://zz1cc.cc.cd/docs
- **来源提交**：`63f378ae`（新增）、`11c04137`（对齐上游根路径 + 字段改写）、`6eb9e3fb`（测试）。

### 5. Google Gemini / Veo 视频生成完整接口
- **内容**：Gemini native `predictLongRunning`、operations（含 `:cancel`/`:wait`/delete）、`/v1beta/files/*` 下载/媒体转发；Vertex/service-account 路径允许 `predictLongRunning`。
- **标记**：搜索 `predictLongRunning`、`v1beta/files`、文件 `backend/internal/handler/gemini_v1beta_handler.go`、`vertex_service_account.go` 应存在。
- **关键文件**：`backend/internal/handler/gemini_v1beta_handler.go`、`backend/internal/service/gemini_messages_compat_service.go`、`vertex_service_account.go`、`backend/internal/server/routes/gateway.go`。
- **来源提交**：`84bf0038`、`060d3993`。

### 6. API Keys 工具栏布局修复
- **内容**：用户端 KeysView 工具栏布局修复。
- **关键文件**：`frontend/src/views/user/KeysView.vue`。
- **来源提交**：`dc4d7191`。

### 7. 前端移动端适配修复（2026-07-04）
- **内容**：统一视口断点（`useViewport`，md=768px）、固定宽度浮层加 `min(92vw,…)` 保护、AffiliateView 表格移动端卡片化。
- **标记**：文件 `frontend/src/composables/useViewport.ts` 应存在；`TablePageLayout.vue` 使用 `useViewport()`。
- **关键文件**：`frontend/src/composables/useViewport.ts`、`components/layout/TablePageLayout.vue`、`components/common/DateRangePicker.vue`、`components/common/SubscriptionProgressMini.vue`、`views/user/KeysView.vue`、`views/user/AffiliateView.vue`。

---

## 合并后验证清单（照做即可）

```bash
# 后端测试（定向 + 全量）
cd backend && go test -tags unit ./internal/service ./internal/handler/...
cd backend && go test ./internal/service ./internal/handler ./internal/repository ./internal/server/...
# 前端
cd frontend && npm run typecheck && npm run build
# 嵌入前端的后端构建（务必带 -tags embed）
cd backend && go build -tags embed -o /tmp/sub2api-new ./cmd/server
# 无 key smoke（应 401 而非 404，证明路由在）
#   POST /v1/videos, /v1/jimeng/videos/generations, /v1beta/models/{m}:predictLongRunning
```

部署：备份 `/opt/sub2api/sub2api` → 替换 → `systemctl restart sub2api.service` → 查 `/health`。

---

_本文件随魔改更新持续维护。新增魔改时，在上面加一节并提交。_
