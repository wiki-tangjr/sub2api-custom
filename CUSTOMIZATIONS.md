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
- **关键文件**：`backend/internal/handler/gemini_v1beta_handler.go`、`backend/internal/service/gemini_messages_compat_service.go`、`gemini_upstream_path_guard.go`、`vertex_service_account.go`、`backend/internal/server/routes/gateway.go`。
- **⚠️ 路径护栏（0.1.170 起，务必保留）**：官方新增 `upstream_path_guard.go`，其 `sanitizedUpstreamPathSuffix` 用闭集白名单 `[A-Za-z0-9_.-]`，会把 Veo/Gemini 原生路径的**冒号动作**（`:download`/`:cancel`/`:wait`/`:predictLongRunning`）与**查询串**（`?alt=media`）判为 `invalid path`，导致魔改功能失效（表现为 `TestGeminiForwardAIStudio*` 测试失败）。因此本地新增 **`backend/internal/service/gemini_upstream_path_guard.go`**（`sanitizedGeminiUpstreamPath`）：query 与 path 分离校验，每个片段最多允许一个 `:`，冒号两侧仍走官方片段校验；`..`、空片段、纯点片段、控制字符、超长/超量片段一律仍拒绝。`forwardAIStudioRaw` 必须调用 **Gemini 专用**护栏，不能被官方版本覆盖回去。
- **标记**：文件 `backend/internal/service/gemini_upstream_path_guard.go` 应存在；`gemini_messages_compat_service.go` 中应调用 `sanitizedGeminiUpstreamPath`（而非 `sanitizedUpstreamPathSuffix`）。
- **来源提交**：`84bf0038`、`060d3993`、`f73e49f36`（0.1.170 护栏适配 + 测试）。

### 6. API Keys 工具栏布局修复
- **内容**：用户端 KeysView 工具栏布局修复。
- **关键文件**：`frontend/src/views/user/KeysView.vue`。
- **来源提交**：`dc4d7191`。

### 7. 前端移动端适配修复（2026-07-04）
- **内容**：统一视口断点（`useViewport`，md=768px）、固定宽度浮层加 `min(92vw,…)` 保护、AffiliateView 表格移动端卡片化。
- **标记**：文件 `frontend/src/composables/useViewport.ts` 应存在；`TablePageLayout.vue` 使用 `useViewport()`。
- **关键文件**：`frontend/src/composables/useViewport.ts`、`components/layout/TablePageLayout.vue`、`components/common/DateRangePicker.vue`、`components/common/SubscriptionProgressMini.vue`、`views/user/KeysView.vue`、`views/user/AffiliateView.vue`。


### 8. Seedance 原生视频接口兼容（桥豆麻衣酱客户端，2026-07-07）
- **内容**：让写死火山方舟 Seedance 原生协议的客户端（“桥豆麻衣酱”等）能通过本站生成即梦视频。原生 `POST /seedance/v3/contents/generations/tasks` 与 `GET .../tasks/{id}` 翻译成上游 OpenAI 兼容 `/v1/videos` 复用现有转发。解析顶层 `duration`/`ratio` 及 text 内 `--duration/--ratio/--resolution` 命令行式参数；`adaptive/auto` 等非法比例省略交上游默认；模型名映射 `seedance-2.0-fast-sdols→as-sd2.0-fast`、`seedance-2.0-sdols→as-sd2.0`。
- **标记**：文件 `backend/internal/handler/seedance_handler.go` 应存在；搜索 `SeedanceCreate`、`SeedanceQuery`、`seedanceModelMap`、`/seedance/v3/contents/generations/tasks` 应存在；`cors.go` 中 `/seedance/` 无条件放行；`embed_on.go` 中 `/seedance/` 绕过嵌入前端。
- **关键文件**：`backend/internal/handler/seedance_handler.go`、`backend/internal/server/routes/gateway.go`、`backend/internal/server/middleware/cors.go`、`backend/internal/web/embed_on.go`。
- **上游注意**：zz1cc.cc.cd 的 `/v1/videos` 分辨率掺水，1080p/4k 实际恒定 720p（上游问题，非本魔改）。
- **来源提交**：`3ced1eb5`（新增）。

### 9. OpenAI 兼容端点 CORS 预检放行（浏览器/WebView 客户端，2026-07-08）
- **内容**：让浏览器/WebView 类客户端（桥豆麻衣酱等）能直连本站 OpenAI 兼容端点。之前只给 `/seedance/` 放行了跨域预检，`/v1/*`（chat/completions、responses、models、images/generations 等）的 `OPTIONS` 预检被 403 拦截，浏览器报 `Failed to fetch`，真正的 POST 无法发出。此魔改在 `cors.go` 中对 `/v1/`、`/videos`、`/jimeng/` 前缀的请求**无条件放行跨域预检**（返回 204 + CORS 头）。
- **仅影响预检**：不改变任何鉴权/计费/转发逻辑，POST 仍照常校验 API key（无效 key 仍 401）并计费。
- **标记**：`cors.go` 中除 `/seedance/` 外，还应存在对 `/v1/`（以及 `/videos`、`/jimeng/`）前缀的 `originAllowed = true` 无条件放行分支。
- **关键文件**：`backend/internal/server/middleware/cors.go`。
- **验证**：`curl -X OPTIONS http://127.0.0.1:8080/v1/chat/completions -H 'Origin: https://x' -H 'Access-Control-Request-Method: POST'` 应返回 204 + `Access-Control-Allow-Origin`。

### 10. 前端底部 ICP 备案号标识（合规要求，2026-07-19）
- **内容**：按工信部规定，在网站所有页面底部显示 ICP 备案号并链接到工信部备案系统。备案号：**滇ICP备2026013786号-1**（带序号，必须保留 `-1`），链接指向 `https://beian.miit.gov.cn`（新标签页打开，`rel="noopener noreferrer"`）。
- **三处覆盖全部页面**：登录/未登录页（`AuthLayout.vue`，版权行下方）+ 登录后主界面（`AppLayout.vue`，底部 `<footer>`）+ 公开首页（`HomeView.vue`，底部 `<footer>`）。
- **待办**：公安备案审核中；通过后需把公安备案号（含图标 + 链接 `https://www.beian.gov.cn`）同样补到这三处 footer。
- **标记**：上述三个前端文件中搜索 `2026013786`、`beian.miit.gov.cn` 应各存在；构建后 `strings sub2api | grep -c 2026013786` 应 ≥ 3。
- **关键文件**：`frontend/src/components/layout/AuthLayout.vue`、`frontend/src/components/layout/AppLayout.vue`、`frontend/src/views/HomeView.vue`。
- **验证**：构建 `-tags embed` 后，登录页、主界面与公开首页底部均可见可点击的备案号链接。

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
