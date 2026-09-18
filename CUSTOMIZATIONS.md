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
- **当前状态**：`bash scripts/customizations-verify.sh --live` → **13/13 全部保留**，退出码 0。

---

### 15. 构建工具链（pnpm）—— 2026-09-17

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

部署：备份 `/opt/sub2api/sub2api` → 替换 → `systemctl restart sub2api.service` → 查 `/health`。

---

_本文件随魔改更新持续维护。新增魔改时，在上面加一节并提交。_
