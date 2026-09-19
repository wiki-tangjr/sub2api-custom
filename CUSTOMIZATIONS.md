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

### 10. 前端底部 ICP 与公安备案标识（合规要求，2026-07-19；公安备案 2026-08-12；移动端自适应优化 2026-09-08）
- **永久保留要求**：这是正式魔改功能。每次升级/合并上游后必须逐页验证，不得因前端冲突、布局调整或重新构建而丢失任何备案文字、链接或公安图标。
- **内容**：按规定在网站所有页面底部同时显示 ICP 与公安备案信息：
  - ICP：**滇ICP备2026013786号-1**（带序号，必须保留 `-1`），链接 `https://beian.miit.gov.cn`（新标签页打开，`rel="noopener noreferrer"`）。
  - 公安备案：**滇公网安备53011102001665号**，链接 `https://beian.mps.gov.cn/#/query/webSearch?code=53011102001665`（必须保留查询参数，按平台代码使用 `target="_blank"`、`rel="noreferrer"`），并显示公安备案图标 `frontend/public/assets/image/gongan-beian.png`（20×20 PNG，取自全国互联网安全管理服务平台公开资源 `/img/ghs.png`）。
- **三处覆盖全部页面**：登录/未登录页（`AuthLayout.vue`，版权行下方）+ 登录后主界面（`AppLayout.vue`，底部 `<footer>`）+ 公开首页（`HomeView.vue`，底部 `<footer>`）。
- **移动端自适应规范**：
  - 容器统一采用 `flex flex-wrap items-center justify-center gap-2`（或 `gap-x-4 gap-y-2`），PC 端自然横向居中排列，移动端自适应折行居中；
  - 链接采用自适应内容宽度（严禁加 `w-full` 或 `items-stretch`，避免移动端被强行拉伸为占满整屏宽度的臃肿条状）；
  - 控制台页脚底部留有安全间距 `pb-6 sm:py-5`，防手机浏览器底部手势条遮挡。
- **标记**：上述三个前端文件中搜索 `2026013786`、`53011102001665`、`beian.miit.gov.cn`、`beian.mps.gov.cn` 应各存在；公安图标文件必须存在。构建后 `grep -R 53011102001665 frontend/dist` 应命中，`frontend/dist/assets/image/gongan-beian.png` 应存在。
- **关键文件**：`frontend/src/components/layout/AuthLayout.vue`、`frontend/src/components/layout/AppLayout.vue`、`frontend/src/views/HomeView.vue`、`frontend/public/assets/image/gongan-beian.png`。
- **验证**：前端构建及 `-tags embed` 构建后，登录页、主界面与公开首页底部均可见可点击的 ICP 和公安备案号；公安备案链接应打开备案详情查询页；移动端自适应居中排列且无全屏撑满畸变。

### 11. 自定义菜单打开方式（iframe / 新标签页，2026-09-10）
- **永久保留要求**：这是正式本地魔改。每次升级/合并上游后，后台“系统设置 → 自定义菜单页面”必须仍可为每个菜单项选择打开方式。
- **内容**：自定义菜单项使用 `open_mode` 字段，支持 `iframe` 与 `new_tab`。管理员界面必须提供选择控件；新建项及历史项缺失、非法值均归一为 `iframe`，保证旧配置继续以内嵌方式打开。
- **安全约束**：`new_tab` 只能使用原始菜单 URL 打开，绝不能调用 `buildEmbeddedUrl()` 或向外链附加用户 ID、token、主题、语言等 iframe 专用查询参数。侧栏外链必须保留 `target="_blank" rel="noopener noreferrer"`。
- **标记**：搜索 `admin.settings.customMenu.openMode`、`open_mode: "iframe"`、`item.open_mode === "new_tab" ? "new_tab" : "iframe"`；`CustomPageView.vue` 中 `new_tab` 的链接和 `window.open` 必须使用 `externalUrl`。
- **关键文件**：`frontend/src/views/admin/SettingsView.vue`、`frontend/src/components/layout/AppSidebar.vue`、`frontend/src/views/user/CustomPageView.vue`、`frontend/src/types/index.ts`、`frontend/src/i18n/locales/{zh,en}/admin/settings.ts`。
- **验证**：前端 typecheck/build 通过；新增菜单默认选择 iframe；旧配置打开后台后显示 iframe；选 `new_tab` 保存后，侧栏以新标签页打开原始 URL；直接访问 `/custom/{id}` 时，回退链接和自动打开 URL 均不含 `token=`。

### 12. 客服联系方式扩展（可配置条目列表 contact_entries，2026-09-12 v1 / 2026-09-18 v2）
- **永久保留要求**：这是正式本地魔改。每次升级/合并上游后，后台“系统设置 → 站点设置 → 客服板块”必须继续保留**可自由增删改的客服条目列表**（图标 / 名称 / 类型 / 展示方式 / 打开方式），以及保存、公开读取、用户端展示的完整链路。丢失任何一环都会让 `customizations-verify.sh` 变红。
- **v2 核心数据结构**：设置键 `contact_entries`（JSON 数组字符串，字段名必须与 `dto.ContactEntry` / `types/index.ts` 完全一致）。每条字段：
  - `id`：条目标识（缺省自动生成；≤32 字符，仅允许 `a-z A-Z 0-9 - _`）；
  - `enabled`：是否启用；公开接口只下发启用项，禁用项不外泄；
  - `label`：展示名称（必填，≤50 字符）；
  - `icon_type`：`emoji` 或 `image`；
  - `icon`：`emoji` 时是表情符号；`image` 时是 `data:image/*` 或 `http(s)` 图片地址（≤500KB）；
  - `type`：`link`（链接） / `qrcode`（二维码） / `text`（纯文本）；
  - `url`：`link` 必填，绝对 `http(s)` 地址，≤2048；
  - `qr_code`：`qrcode` 必填，`data:image/*` 或 `http(s)`（≤2MB）；
  - `value`：`text` 必填，纯文本内容（≤200）；
  - `description`：补充说明（≤200 字符）；
  - `display`：`modal`（点击弹窗） / `hover`（鼠标悬停浮层） / `inline`（内联直接展开）；
  - `open_target`：`new_tab`（新标签页） / `current_tab`（当前标签页）；
  - `sort_order`：后端按数组顺序回填，前端排序用。
- **内容与安全约束**：最多 20 条；`label` ≤50；`url` ≤2048 且必须是 `http(s)` 绝对地址；`value`/`description` ≤200；图标 ≤500KB；二维码 ≤2MB；用户端再次用 `sanitizeUrl` 校验，二维码只允许 `data:image/*` 或 `http(s)`。
- **旧字段降级兼容（v1，任何一条都不能丢）**：`contact_info`（微信文本）、`telegram_group_url`（Telegram 群组 HTTPS 链接）、`wechat_group_qr_code`（微信群二维码图片），以及文案/样式字段 `contact_section_title`、`contact_section_description`、`telegram_entry_label`、`wechat_group_entry_label`、`wechat_contact_entry_label`、`contact_section_style`（`card` 卡片式 / `list` 列表式）。当 `contact_entries` 为空时，`resolveContactEntries()` 必须自动从上述旧字段合成条目，保证老站点升级后入口不消失；一旦后台保存过 `contact_entries`，则以新列表为准。
- **公开读取白名单阱阱（务必同时修改两处，2026-09-19 修补）**：`contact_section_title`、`contact_section_description`、`contact_section_style`、`telegram_entry_label`、`wechat_group_entry_label`、`wechat_contact_entry_label` 这 6 个键**必须同时**出现在
  `setting_public.go` 的两个地方：① `GetPublicSettings()` 的 `keys := []string{...}` 白名单；② `GetPublicSettingsForInjection()` 的 `PublicSettingsInjectionPayload` 组装块。
  **只改一处不会报错**，但后果是：后台能保存、DB 有值，前台却永远拿到空字符串，只能靠 i18n 兜底（`zh/common.ts` 的 `contactSectionDefaultTitle` = 「联系客服」），
  导致管理员改了标题/描述/样式**完全不生效**。
  回归测试：`backend/internal/service/setting_service_public_test.go` 的 `TestSettingService_GetPublicSettings_ExposesContactSectionFields`（同时断言公开设置与 SSR 注入双路径）；
  体检脚本已在 #12 段加入 8 条守卫，合并上游后若被删会直接变红。
- **用户端展示**：顶栏用户菜单、个人资料页、兑换页均使用共享组件 `ContactEntries.vue` 渲染；支持 `card` / `list` / `dropdown` 三种版式；`modal` 打开弹窗（含二维码扫码提示 `contactScanHint`），`hover` 悬停浮层，`inline` 直接内联；`link` 按 `open_mode` 决定新标签页或当前页跳转。
- **标记**：`contact_entries`、`ContactEntry`、`ContactEntries`、`ContactEntryBody`、`ContactEntryIcon`、`ContactEntriesEditor`、`resolveContactEntries`、`resolvePublicContactEntries`、`validateContactEntries`、`decodeContactEntriesJSON`、`contactScanHint` 应存在；`setting_update.go` 中 `updates[SettingKeyTelegramGroupURL]` 只能写入一次。
- **关键文件**：`backend/internal/service/domain_constants.go`、`setting_parse.go`、`setting_update.go`、`setting_public.go`、`contact_entries.go`、`backend/internal/handler/admin/setting_handler.go`（管理端回显组装，勿漏字段）、`setting_handler_update.go`、`setting_handler_audit.go`、`setting_contact_entries.go`、`backend/internal/handler/dto/settings.go`、`dto/contact_entries.go`、`frontend/src/views/admin/SettingsView.vue`、`frontend/src/views/admin/settings/ContactEntriesEditor.vue`、`frontend/src/components/common/ContactEntries.vue`、`ContactEntryBody.vue`、`ContactEntryIcon.vue`、`frontend/src/utils/contactEntries.ts`、`frontend/src/components/layout/AppHeader.vue`、`frontend/src/views/user/ProfileView.vue`、`frontend/src/views/user/RedeemView.vue`、`frontend/src/types/index.ts`、`frontend/src/api/admin/settings.ts`。
- **验证**：`./scripts/customizations-verify.sh` 覆盖源码标记；`--live` 额外检查二进制内含 `contact_entries`、`/api/v1/settings/public` 返回 `contact_entries`；前端 `npm run typecheck && npm run build`、后端 `go build -tags embed` 通过后替换二进制并重启。

### 13. 图片模型 driver 放行（image-only responses driver + Gemini 原生图片模型，2026-09-16）
- **永久保留要求**：这是正式本地魔改。每次升级/合并上游后必须保留，否则 `/v1/responses` 用 `gpt-image-*` 作 driver 会被改写成文本模型导致上游 503 `model_not_found` 并烧掉整个 failover 预算；`/v1/images/generations` 也会拒绝 `imagen-*` / `nano-banana*` / `gemini-*-image*` 等模型。
- **内容**：
  - `imageOnlyResponsesDriverAllowed()`：读环境变量 `SUB2API_ALLOW_IMAGE_ONLY_RESPONSES_DRIVER`（`1/true/yes/on` 为开）。开启时 `validateOpenAIResponsesImageModel()` 不再拒绝图片模型作 driver，`normalizeOpenAIResponsesImageOnlyModel()` 保持 driver == tool.model == 客户端请求的图片模型。
  - `isGeminiNativeImageModel()`：识别 `imagen-*`、`nano-banana*`、`gemini-*-image*`，`validateOpenAIImagesModel()` 直接放行。该判定**故意不并入** `isOpenAIImageGenerationModel()`，因为后者还驱动 Codex `/responses` 的 image-only 归一化，那里 Gemini 图片模型作为普通直通 driver 是合法的。
- **生产环境开关**：systemd drop-in `/etc/systemd/system/sub2api.service.d/50-image-only-driver.conf` 设置 `SUB2API_ALLOW_IMAGE_ONLY_RESPONSES_DRIVER=1`。默认（不设该变量）行为与官方一致，不影响存量用户。
- **标记**：搜索 `SUB2API_ALLOW_IMAGE_ONLY_RESPONSES_DRIVER`、`imageOnlyResponsesDriverAllowed`、`isGeminiNativeImageModel` 应存在（各 2 处以上）。
- **关键文件**：`backend/internal/service/openai_images.go`（+34 行）、`backend/internal/service/openai_codex_transform.go`（+8 行）。
- **验证**：`go build ./...` 通过；开环境变量后 `/v1/responses` 带 `gpt-image-1` driver 不再返回模型不合法错误；`/v1/images/generations` 传 `nano-banana-pro`、`imagen-4.0-generate-preview-06-06`、`gemini-2.5-flash-image` 均不被 400 拒绝。
- **来源**：2026-09-16 合并 0.2.5 时随补丁一起提交（原为工作区未提交改动，快照 `/tmp/wip-image-only-driver.patch`）。

### 14. 系统更新守护脚本（2026-09-17 新增）

> 这一条不是业务功能，而是**保证上面所有魔改不丢的机制**。将来新增魔改时，必须同步往验证脚本里加检查项（数量以脚本实际检查的条目为准，不要在文档里写死数字）。

- **完整性验证脚本**：`scripts/customizations-verify.sh`
  - 对上面每一条魔改逐条做「文件存在 + 固定字符串(FIXED-STRING) 标记 grep + 出现次数」校验，全部通过才 `exit 0`，未通过 `exit 1`（可直接当 CI 闸门用）。
  - `--live` 额外校验生产环境：二进制存在及 md5、二进制内字符串（ICP/公安备案、#13 开关、#5 guard、#8 seedance）、服务 active、`NRestarts=0`、`/health` 200、CORS OPTIONS 204、关键路由非 404（401 即通过）。
  - 用法：`bash scripts/customizations-verify.sh`（代码层）、`bash scripts/customizations-verify.sh --live`（生产层）。
- **部署资产安装脚本**：`scripts/install-deploy-assets.sh`
  - 把 `deploy/50-image-only-driver.conf`（#13 的 systemd drop-in）安装到 `/etc/systemd/system/sub2api.service.d/`，幂等可重复执行。
  - `--check` 只比对**生效指令**（忽略注释措辞差异），并额外断言服务进程里真的读到了该环境变量。
  - 换机器 / 重装 / 迁移后**必须先跑这个脚本**，否则 #13 静默失效（表现：图片生成 502 或极慢）。
- **更新脚本已内置三道闸门**：`scripts/update-from-upstream.sh`
  1. 合并前基线检查：如果**合并前**就是红的，直接拒绝开始 —— 保证之后变红一定是这次合并造成的。
  2. 合并后硬闸门：验证脚本不过直接 `die`。
  3. 部署前 `--live` 检查：不过直接 `die`。
  4. 部署后提示再跑一次 `--live` 复验。
- **提交时守卫（第三层防线，2026-09-17 新增）**：`scripts/git-hooks/pre-commit` + `pre-merge-commit`。安装在 `core.hooksPath=scripts/git-hooks`（由 `update-from-upstream.sh` 每次自愈）。目的：手工 `git merge --continue` 或手工解决冲突后提交时，也能拦住被官方冲掉的魔改。行为：合并提交一律检查；普通提交只在动到 `backend/`、`frontend/`、`scripts/`、`CUSTOMIZATIONS.md` 时检查（约 0.3s），纯文档提交直接放行；失败时给出逐条清单。临时绕过：`git commit --no-verify`。
- **验证脚本已经过负向测试**（证明不是「永远绿」）：分别人为制造了 #5 guard 被覆盖、#10 备案号被删、#12 管理端回显字段丢失、#5 guard 文件被删、#9 CORS 只剩 1 处放行、drop-in 缺失/自愈/被篡改成 `=0` 等场景，全部被正确捕获；恢复后回到全绿。
- **当前状态**：`bash scripts/customizations-verify.sh --live` → **全部保留**，退出码 0。

---

### 15. 充值/订阅公告 + 订阅套餐每排数量 —— 2026-09-18

- **需求**：管理员能在后台为「充值页」和「订阅页」各设置一段公告（开发票说明 / 退款说明 / 联系客服等）；并把「订阅套餐一排展示几个」做成可配置项，放在「创建套餐」按钮旁边。
- **实现**：
  - 新设置键：`PAYMENT_RECHARGE_NOTICE`、`PAYMENT_SUBSCRIPTION_NOTICE`、`SUBSCRIPTION_PLANS_PER_ROW`（1–6 夹取，默认 3）。
  - 后台：`SettingsView.vue` 支付设置区新增两个公告文本框 + 每排套餐数输入；`AdminPaymentPlansView.vue` 在创建套餐按钮旁新增「每排数量」控件，保存后立即生效。
  - 前台：`PaymentView.vue` 在充值区 / 订阅区顶部渲染公告（Markdown 渲染，空值时不渲染任何东西）。
- **零影响保障**：三个设置键默认空 / 默认 3（与上游现有的「一排 3 个」完全一致），未配置时前台 DOM 与改动前**没有任何差别**。

### 16. 充值档位 / 快捷金额 / 阶梯优惠 —— 2026-09-18

- **需求**：管理员能控制充值的最小 / 最大金额、有几个快捷充值按钮；并支持「充得越多送得越多」的阶梯优惠，快捷按钮直接显示「到账多少 / 优惠多少 / 实际支付多少」。
- **实现**：
  - 新设置键：`RECHARGE_QUICK_AMOUNTS`（逗号分隔，如 `10,50,100,500`；**留空 = 沿用前端原有硬编码默认**）、`RECHARGE_DISCOUNT_TIERS`（区间语法，如 `100-500:2,500:3` 表示 100~500 元优惠 2%、满 500 元优惠 3%；`仅写下限` 等同于无上限；**留空 = 关闭优惠**）。详见第 22 节。
  - 新增 `backend/internal/service/payment_amounts.go`（解析 + 金额/优惠计算）与 `payment_order.go` 的优惠应用逻辑；旧签名保留并转发 `discount=0`，保证既有调用方行为不变。
  - 前台 `AmountInput.vue` / `PaymentView.vue` 按配置渲染快捷按钮与优惠文案。
- **零影响保障**：两个键默认留空 → 快捷金额回退到原硬编码、优惠关闭 → 充值金额与到账数额与改动前完全一致。

### 17. 代理商体系 + 邀请返利入口隐藏 —— 2026-09-18

- **需求**：
  1. 管理员可对单个账号设置「其邀请来的客户是否看得到邀请返利入口」（整体隐藏，看不到邀请码自然无法发展下级）。
  2. 管理员可把某账号设为**一级代理**；一级代理登录后在邀请返利页可查看自己邀请用户的注册 / 返利明细（邮箱默认打码），并可自主把某个下级设为**二级代理**、设置其专属返利比例（**不得超过一级代理自己的比例**），也可开关下级的邀请返利入口。
  3. 管理员可单独控制某账号在邀请返利页**是否显示完整（不打码）邮箱**。
- **实现**：
  - 迁移 `239_affiliate_agent_hierarchy.sql`（幂等 `ADD COLUMN IF NOT EXISTS`）：`agent_level`(0 普通 / 1 一级 / 2 二级)、`agent_parent_user_id`、`show_full_email`、`hide_affiliate_for_self` + 索引。
  - 后端：仓储新增 `SetAgentLevel` / `SetShowFullEmail` / `SetHideAffiliateForSelf` / `ListSubAgents`；`IsAffiliateHiddenByInviter` 改为「自身开关 OR 邀请人开关」；服务层新增 `GetAgentInvitees` / `SetSubAgent` / `SetInviteeAffiliateHidden`，比例校验 `ErrAffiliateAgentRateTooHigh`，并在返利入账时对二级代理比例**运行时兜底夹到父级比例**。
  - 用户端路由：`GET /user/aff/agents`、`POST /user/aff/agents/set`、`PUT /user/aff/invitees/:user_id/hide`。
  - 后台：专属用户配置弹窗新增「代理商等级 / 上级代理ID / 完整邮箱 / 隐藏自身返利」，列表新增徽标列。
  - 前台：`AffiliateView.vue` 一级代理可见「我的下级代理」卡片（邮箱、用户名、注册时间、返佣、当前比例、设为二级代理 / 改比例 / 取消 / 开关对方返利）。
- **零影响保障**：`agent_level` 默认 0、`show_full_email` 默认 false、`hide_affiliate_for_self` 默认 false → 所有账号默认行为与改动前一致（邮箱照样打码、入口照样可见）。

---

### 19. 零中断更新工具链（蓝绿 + 连接排空）—— 2026-09-18

> 同样不是业务功能，而是**「每次更新系统都要保留魔改、且不能让网站掉线」**的执行机制。
> 背景：上游原先的 `deploy_final.sh` 走 `systemctl restart`，而 `main.go` 里 `Shutdown` 超时只有 5 秒 →
> 重启会直接切断在途请求，**正在进行中的流式 AI 调用（SSE）会被掐断**。用户明确要求「更新期间禁止网站不可访问、禁止影响正在调用 AI 的客户」，故废弃单次 restart。

- **两个脚本，职责分离**：
  1. `scripts/build-and-stage.sh` —— 只构建 + 起灰度，**绝不碰线上 8080、绝不碰 nginx**
     - 前置先查线上 `/health`，不健康直接拒绝开始
     - 构建前端（`pnpm run build`）→ 后端（**必须 `go build -tags embed`**，漏了 embed 就会丢全部前端魔改）
     - 跑代码层体检；通过后在 **8081** 起灰度实例（`systemd-run --unit=sub2api-canary`，用 `--uid=sub2api`）
     - 再跑 `CUSTOM_VERIFY_PORT=8081 customizations-verify.sh --live` 灰度体检
     - 任何一步失败立刻退出并停掉灰度，**线上全程无感知**
     - ⚠️ 灰度实例必须隔离日志：`LOG_OUTPUT_FILE_PATH=/tmp/sub2api_canary.log`、`SUB2API_DEBUG_GATEWAY_BODY=/tmp/gateway_debug_canary.log`，否则会和线上双写同一个 300MB+ 的 `gateway_debug.log`
  2. `scripts/zero-downtime-deploy.sh` —— 蓝绿切换 + 连接排空，**全程 nginx 始终指向健康后端**
     - Phase 1：给 nginx 加 `upstream sub2api_backend`（仍指向 8080），reload
     - Phase 2：`upstream` 切到 **8081（新代码）**，reload —— 新用户流量从此走新代码
     - Phase 3：轮询 `ss` 等 **8080 的连接数归零**（在途 SSE 自然结束）→ 才 `systemctl stop` → 换二进制 → `start`
     - Phase 4：`upstream` 切回 **8080（已是新代码）**，reload
     - Phase 5：排空 8081 → 灰度下线（`systemctl stop` + `reset-failed`）
     - Phase 6：健康检查 + `customizations-verify.sh --live` + 可用性报告
     - 灾备：任一环节失败 → `point_nginx_at_healthy()` 自动把 nginx 指回仍健康的端口；新二进制起不来 → 自动装回旧二进制。备份落在 `/root/deploy_stage/`
- **零中断证明（可复现）**：脚本内部以 **1 秒**间隔采样 `https://api.pixelqd.cn/health`，结束时打印
  `可用性采样: N 次, 非200: 0 次, 最长连续不可用: 0s`。2026-09-18 实际发布：**106/106 采样全 200，非 200 为 0**；
  同时 8080 排空耗时 39s、8081 排空 116s —— 这段时间正是被保护的**在途长连接**。
- **`--dry-run`**：只做发布前体检并打印计划，**不写任何文件**（实测 nginx 配置 md5 前后一致），可用于上线前预演。
- **发布标准流程**：
  ```bash
  cd /root/.openclaw/workspace/sub2api-src
  bash scripts/build-and-stage.sh            # 构建 + 灰度体检（线上无感知）
  bash scripts/zero-downtime-deploy.sh --dry-run   # 可选：预演
  bash scripts/zero-downtime-deploy.sh       # 零中断发布
  bash scripts/customizations-verify.sh --live     # 发布后复验
  ```
- **⚠️ CRITICAL 教训：不要用 `sed -i 's/\r$//'` 清理 CRLF**。GNU sed 会把 `\r` 当成字符 `r`，从而把脚本里**所有字母 r 删掉**
  （曾把 `start_monitor` 变成 `start_monito`、`report_monitor` 变成 `report_monito`，导致调用报 `command not found`，切换虽成功但可用性报告没打印）。
  正确做法：本地用 PowerShell `[System.IO.File]::WriteAllText($p, ($s -replace "`r`n","`n"), (New-Object System.Text.UTF8Encoding($false)))` 写 LF-only 再上传，或上传后用 `bash -n` + `grep -c $'\r'` 双重校验。

### 20. 构建工具链（pnpm）—— 2026-09-17

> 上游 `frontend/package.json` 的 `build` 脚本是 `pnpm run check:i18n && vue-tsc -b && vite build`，**内部直接调用 pnpm**。

- **现象**：本机环境里 pnpm 曾经消失（node 为 `/www/server/nodejs/v24.18.0`，并非标准安装），导致 `npm run build` **exit 127**（`sh: pnpm: command not found`）。这会让 `update-from-upstream.sh` 在验证阶段直接 `die`，没法完成发布。
- **修复**：`corepack enable pnpm`（已执行，现为 pnpm 12.3.4）。备用：`npm i -g pnpm`。
- **`frontend/pnpm-workspace.yaml`（已入库）**：pnpm 12 需要它才会跑 esbuild / vue-demi 的构建脚本：

  ```yaml
  allowBuilds:
    esbuild: true
    vue-demi: true
  ```

  上游仓库在该路径下**没有文件**，因此永远不会与官方产生合并冲突。
- **预防**：`update-from-upstream.sh` 新增「0.4 工具链预检」，缺 node/go/pnpm 时直接拒绝开始合并并给出修复命令；`customizations-verify.sh` 新增第 14 项同步校验。
- **备注**：因口径不一致，本机上 `node_modules/.bin/*` 可能残缺；建议用 `pnpm install --frozen-lockfile` 重建依赖。

---

### 21. 后台客服设置收敛为单一入口（2026-09-19）

- **背景**：`设置 → 通用设置` 里同时存在三套客服/群组配置（上游原生旧单字段、我方 #12 的「客服板块展示配置」、以及 #12 的「客服联系方式条目」），管理员看着重复又混乱。
- **结论**：全站**本来就只有这一处**能配置客服，不存在第二个管理页；乱的原因是**同一页里三套字段并存**。
- **收敛方案（只动后台界面，不动后端、不动前台展示）**：
  1. 「客服联系方式条目」（`contact_entries`）提升为**唯一主入口**，放在最前；
  2. 「客服板块展示配置」保留**标题 / 描述 / 展示样式**（这三项在 `AppHeader.vue`、`ProfileView.vue` 里**始终生效**，不是旧字段，不能删）；
  3. 旧单字段（`contact_info` / `telegram_group_url` / `wechat_group_qr_code`）与旧入口文案（`telegram_entry_label` / `wechat_group_entry_label` / `wechat_contact_entry_label`）一起收进**默认收起的「兼容设置：旧版单字段」折叠面板**，并明确提示「条目列表一旦有内容，旧字段就不再展示给用户」。
- **为什么不能直接删旧字段**：`resolveContactEntries()` 的降级链依赖它们（`contact_entries` 为空时由旧字段合成条目），且 `customizations-verify.sh` 的 #12 段把旧字段列为**必须保留**的兼容项；线上当前用户端展示的那条「添加微信客服」正是由空 `contact_entries` + `contact_info` 合成而来。
- **零影响保障**：纯前端后台模板 + i18n 文案改动；`form.*` 绑定字段名全部不变，提交 payload 与回填逻辑不变，后端设置读写与前台渲染完全不变 → 用户端展示零变化。
- **标记**：`魔改 #12 / #19`、`contactLegacyOpen`、`contactLegacy.title`、`contactLegacy.notice`；`form.contact_info` / `form.telegram_group_url` / `form.wechat_group_qr_code` 各**只允许出现一次**。
- **关键文件**：`frontend/src/views/admin/SettingsView.vue`、`frontend/src/i18n/locales/zh/admin/settings.ts`、`frontend/src/i18n/locales/en/admin/settings.ts`。
- **对应体检段**：`scripts/customizations-verify.sh` 的 **#19**。

---

## ⚠️ 两个 git 看不见的盲区（最容易静默丢功能）

官方更新合并完、`git status` 全绿，**不代表魔改没丢**。下面两处 git 不会报任何错：

### 盲区 1：`backend/internal/web/dist` 不被 git 跟踪

- `.gitignore:102` 把嵌入前端的目录 negate 掉了 —— 仓库里该目录 **0 个 tracked 文件**，改了也不会出现在 `git status`。
- 后果：前端没重新构建，或后端编译**漏了 `-tags embed`** → 二进制里还是旧前端 → 备案号 / 自定义菜单 / 客服入口全部「消失」，而 git 一切正常。
- 对策：每次部署前必须 `cd frontend && npm run typecheck && npm run build`，再 `cd backend && go build -tags embed`；用 `customizations-verify.sh --live` 检查二进制内字符串是否还在。

### 盲区 2：#13 的 systemd drop-in 不在仓库里（现已版本化）

- `/etc/systemd/system/sub2api.service.d/50-image-only-driver.conf` 是**机器上的系统文件**，不在 git 里 —— 它跟着机器走，不跟着仓库走。
- 后果：换机器 / 重装 / 恢复镜像后环境变量丢失 → #13 静默失效（图片生成 502 或极慢），而仓库代码却完全正常。
- 对策：该文件已入库为 `deploy/50-image-only-driver.conf`；重装或迁移后跑 `bash scripts/install-deploy-assets.sh`，用 `--check` 确认。

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

部署（**禁止再直接 `systemctl restart`**，会掐断在途 AI 流式调用）：

```bash
cd /root/.openclaw/workspace/sub2api-src
bash scripts/build-and-stage.sh            # 构建 + 8081 灰度体检，线上无感知
bash scripts/zero-downtime-deploy.sh --dry-run   # 可选：只体检并打印计划，不改任何东西
bash scripts/zero-downtime-deploy.sh       # 蓝绿切换 + 连接排空，零中断
bash scripts/customizations-verify.sh --live     # 发布后复验
```

详见第 19 节「零中断更新工具链」。旧脚本 `deploy_final.sh` 已废弃，仅保留作参考。

---

### 22. 充值优惠区间化 + 后台控件化 + 公告美化 —— 2026-09-19

> 本轮是对第 15 / 16 节的**体验与安全加固**，不新增业务语义，重点在：
> 优惠从「满 X」升级为**区间**、把裸文本框换成结构化控件、公告视觉收敛、并堵死前台自定义金额绕过优惠的路径。

- **需求**：
  1. 后台支付设置区原先是裸 `textarea` + 裸 `input`，排版乱；要按站点既有设计语言（card / input-label / badge / 图标）重排，控件该用按钮的用按钮、该分段的用分段按钮。
  2. 优惠档位要从「满 X 送 Y%」改为**【多少钱 ~ 多少钱】→ 优惠多少**的区间；并且**必须防止用户在前台自定义输入金额绕过优惠判定**。
  3. 充值页 / 订阅页公告板块要好看、与站点风格一致。
- **实现**：
  - **后端（唯一真相）**：`payment_amounts.go` 新增 `parseRechargeTierRange`，支持 `100`（=无上限）、`100-500`、`100~500`、`100-`、`-500`，兼容全角 `～—–`；`min<=0 && max<=0`（如 `0:5`）**直接判非法**，避免优惠变成无条件生效。`parseRechargeDiscountTiers` 按下限升序、同下限时无上限排最后；`resolveRechargeDiscountPercent` 改为区间匹配，多区间同时命中取「下限最高、其次上限最窄」。
  - **防绕过**：优惠判定在 `payment_order.go` 里始终用**订单真实金额**（`limitAmount`）计算，前台输入框只负责预览，改前端参数无法影响实际优惠。
  - **前端共享规则**：新增 `frontend/src/utils/rechargeTiers.ts`（归一化 / 区间解析 / 区间命中 / 序列化），与后端逐条同规则；`AmountInput.vue` / `PaymentView.vue` 统一改用它，保证「前台看到的优惠 = 后端算出来的优惠」。
  - **旧配置零影响**：仅当 `Max > 0` 时才序列化成区间；历史「满 X」数据原样保留、行为不变（已由单测覆盖）。
  - **后台控件化**：新增 `RechargeQuickAmountsEditor.vue`（chip 式金额 + 一键常用金额 + 24 上限）与 `RechargeDiscountTiersEditor.vue`（结构化档位行：起始金额 / 结束金额 / 优惠% + 行内实时校验 + 中文摘要句 + 高级原始文本回退）；「每排套餐数」改为 1–6 分段按钮。提交前先经共享 parser 归一化，脏值不可能进后端。
  - **公告美化**：充值页 / 订阅页公告改为「渐变顶条 + 圆角图标徽章 + 标题 + 分隔线 + 紧凑 Markdown」；新增 `.announcement-markdown` 紧凑排版，**不影响**原有全局公告弹窗 / 铃铛的 `.markdown-body`。
- **零影响保障**：`RECHARGE_QUICK_AMOUNTS` / `RECHARGE_DISCOUNT_TIERS` / 两个公告键默认留空 → 快捷金额回退原硬编码、优惠关闭、公告不渲染（`v-if` 仍在）→ **前台零视觉变化**。所有取值由管理员自己在后台设置。
- **验证**：后端定向单测 `ok ... 0.223s`；前端 `rechargeTiers` 单测 11/11 通过；i18n 完整性 3/3；`vue-tsc` typecheck 通过。

---

_本文件随魔改更新持续维护。新增魔改时，在上面加一节并提交。_
