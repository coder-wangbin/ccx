# TODO

## 更新规范

### 提交问题

在本文档末尾添加新条目，格式：

```markdown
## [ ] 简短标题

问题描述，包含复现条件和预期行为。
```

如有对应 GitHub Issue，在标题中标注，如 `## [ ] 标题 (#issue号)`。

### 解决更新

问题修复后，将 `[ ]` 改为 `[x]`，并在描述下方追加：

```markdown
**关键提交：**
- `commit_hash` commit message
```

如涉及多文件变更，可补充 `**关键变更：**` 列出受影响文件。

---

## [x] 优化 200 超时重试机制

当前代理层在上游返回 HTTP 200 但实际超时（如流式响应中断、空响应体等场景）时的重试行为需要优化，确保能够正确识别"伪成功"响应并触发跨渠道故障转移。

**关键提交：**
- `017597c7` feat(stream): 新增流式 200 伪成功检测与 failover 机制
- `e15e9fc4` feat(chat): Chat handler 流式两阶段 preflight 检测
- `30bc1535` feat(responses): Responses handler 流式两阶段 preflight 检测
- `4fca03a6` fix(stream): 捕捉已提交流式响应断流

## [x] 渠道级配置超时参数

支持为每个渠道独立配置超时参数（连接超时、读取超时、总超时），替代当前的全局统一超时设置，以适应不同上游服务商的响应速度差异。

**关键提交：**
- `c7091345` feat(channels): 支持渠道级请求超时配置
- `29549dea` fix(stream): 移除流式超时 off 档并为渠道新增超时覆盖开关
- `1c14aea2` feat(web): Web 前端熔断器配置新增流式超时滑块
- `b1e1bb0a` feat(desktop): Desktop 前端熔断器配置新增流式超时滑块

## [x] Windows 桌面应用图标透明度

Windows 系统下桌面应用图标周边没有透明背景，需要修正图标资源以确保图标在 Windows 任务栏和桌面显示时具有正确的透明边缘。

**关键提交：**
- `cf1d867b` fix(desktop): 修复 Windows 图标透明度并补全 Codex 预设模板

## [x] 桌面端渠道中心成功提示清理

桌面端渠道中心中，一个渠道添加成功后切换到另一个渠道时，之前的"添加成功"提示没有清除。需要在渠道切换或表单重置时同步清理成功提示状态，避免误导用户。

**修复方式：** 在 `ChannelTab.vue` 的 `watch(selectedProvider)` 中同步清除 `result` 和 `localError`。

## [x] GPT 类型上游模型测试覆盖 codex-auto-review

GPT 类型的上游模型测试用例应包含 `codex-auto-review`，确保 Codex 自动评审相关模型能力在 GPT 类渠道中被覆盖。

**关键变更：**
- `capability_probe_models.go`: chat/responses 探测模型列表新增 `codex-auto-review`
- `frontend/src/App.vue`: 同步前端占位模型列表
- `capability_probe_models_test.go`: 新增 `TestGetCapabilityProbeModels_ContainsCodexAutoReview` 和 `TestGetCapabilityProbeModels_CodexAutoReviewRedirect`
- `capability_test_redirect_test.go`: 新增 `TestRunRedirectVerification_CodexAutoReview`（GPT 类渠道重定向集成测试）和 `TestRunRedirectVerification_CodexAutoReviewDedup`（去重验证）

## [x] 桌面端同步 stripImageGenerationTool 开关

Web 端与后端已支持渠道级「去除 image_generation 工具」开关（Responses/Chat 透传路径剥离 image_generation 工具，规避无图片生成权限上游的 permission_error 误拉黑）。桌面端尚未同步，需补：`desktop/internal/channelpreset/preset.go` 已加字段，仍需 `desktop/frontend/src/services/admin-api.ts` 类型、`desktop/frontend/src/utils/channel-payload.ts`、`desktop/frontend/src/components/console/ChannelEditDialog.vue` 表单/UI/预设。

**关键提交：**
- `2a46906c` feat(channel): 新增 stripImageGenerationTool 渠道开关（Web 端 + 后端）
- 本次提交完成桌面端同步

## [x] 疑似 bug 修复 (#162 #187 #188)

- **#188** 桌面端密钥恢复接口路径与后端不匹配，改为 Body 传参
- **#162** 桌面端 `buildEnv` 强制 `ENV=production` 覆盖用户配置，改为尊重 `.env` 设置
- **#187** Responses→Claude Messages 转换中连续 `tool_use`/`tool_result` 未合并，导致 Bedrock 上游报 `TOOL_USE_RESULT_MISMATCH`；新增缓冲合并逻辑

**关键提交：**
- `48c17606` fix: 修复三个疑似 bug (#162 #187 #188)

## [x] 上游主动限速与动态调速 (#190)

支持渠道级主动限速（每分钟请求数 RPM 令牌桶 + 最大并发信号量），在请求发往上游前主动限流，规避免费/低额度上游（如 MiMo）触发 429。可选「自动学习上游限流头」开关：解析上游 `Retry-After` / `anthropic-ratelimit-*` / `x-ratelimit-*` 响应头，命中限流时对该渠道动态冷却（cooldown），到期自动恢复；调度器选择渠道时跳过 cooldown 中的渠道。限速作用域为渠道级（同渠道多 Key 共享令牌桶）。桌面端一键添加 MiMo 渠道内置保守默认 RPM=80（官方 RPM=100 的 80%）。

**关键变更：**
- `backend-go/internal/ratelimit/`: 新增限速器包（`limiter.go` 令牌桶+并发信号量+cooldown、`manager.go` 渠道级管理、`hints.go` 限流头解析）+ 单测
- `backend-go/internal/config/config.go`: `UpstreamConfig` / `UpstreamUpdate` 新增 `rateLimitRpm` / `rateLimitBurst` / `rateLimitMaxConcurrent` / `rateLimitAutoFromHeaders`，五类渠道 Add/Update 函数同步赋值与校验，`Clone` 深拷贝
- `backend-go/main.go`: 初始化 `ratelimit.Manager` + 配置热重载回调，传入 scheduler
- `backend-go/internal/handlers/common/upstream_failover.go`: 发请求前 `Acquire`、resp 后 `ApplyUpstreamHints`、并发信号量释放
- `backend-go/internal/scheduler/channel_scheduler.go`: `SelectChannel` 跳过 cooldown 渠道
- `frontend/`、`desktop/frontend/`: 渠道编辑表单 + 字段贯穿 + 三语 i18n
- `desktop/internal/channelpreset/preset.go`: MiMo 预设默认 `rateLimitRpm=80`
- `docs/guide/environment.md`: 渠道级限速字段说明

## [x] 用户体验提升

如果有渠道频繁缓存写很高，说明大概率设置有问题，应该在渠道列表给他一个badge提醒用户注意这个问题
还有runapi现在的messages协议很偏向原生协议，那么应该从桌面端渠道中心添加的时候关闭userid兼容。还有就是cch头关闭的问题，之前有这个问题的渠道比较多，所以放在了全局开关，但是现在有问题的很少了，那么应该放在messages协议渠道级开关，默认是关闭的。

**关键变更：**
- `frontend/src/components/ChannelOrchestration.vue`: 基于 15 分钟窗口新增缓存写偏高 warning chip（比例+门槛规则）
- `desktop/internal/channelpreset/preset.go`: RunAPI messages preset 显式下发 `normalizeMetadataUserId=false`
- `backend-go/internal/config/` + `backend-go/internal/handlers/common/upstream_failover.go`: 新增 messages 渠道级 `stripBillingHeader` 配置，并把 CCH 移除逻辑从全局入口下沉到按渠道转发阶段
- `frontend/src/components/AddChannelModal.vue` / `desktop/frontend/src/components/console/ChannelEditDialog.vue`: Web/桌面表单新增 messages 渠道级 CCH 开关，Web 顶栏移除全局 CCH 按钮

**验证：**
- `cd desktop && go test ./internal/channelpreset/...` ✅
- `cd backend-go && go test ./internal/config ./internal/handlers/common ./internal/handlers/messages ./internal/handlers` ✅
- `cd backend-go && make copy-frontend && make test` ✅
- `cd frontend && bun run type-check && bun run test && bun run build` ✅
- `cd desktop/frontend && bun run type-check && bun run test && bun run build` ✅（构建通过，Rolldown 对 `@vueuse/core` 的 `/* #__PURE__ */` 给出上游依赖 warning，不影响产物生成）

---

> **上游版本变更**

## [ ] Claude Code v2.1.169 上游协议/工具变更评估

发现协议/工具/用法变更：stream。请评估对 ccx Messages 渠道的影响。

## [ ] Codex rust-v0.138.0 上游协议/工具变更评估

发现协议/工具/用法变更：code mode, code-mode, compact, effort, environment, goal extension, image generation, multi-agent, permission, plugin, Plugin, reasoning, Reasoning, remote control, remote-control, sandbox, Sandbox, session, skill。请评估对 ccx Responses 渠道的影响。

## [ ] Claude Code v2.1.170 上游协议/工具变更评估

有新版本 v2.1.170 可用（本地: 2.1.168）。非紧急，可在方便时升级。主要内容：新增 Claude Fable 5 模型支持；修复 VS Code 集成终端启动时会话未保存问题。

## [ ] Codex rust-v0.139.0 上游协议/工具变更评估

发现协议/工具/用法变更：code mode, Code mode, compact, environment, multi-agent, permission, plugin, Plugin, remote-control, sandbox, Sandbox, session, skill, web search。请评估对 ccx Responses 渠道的影响。主要变更：Code mode 支持独立 web search 调用；工具与连接器输入 schema 保留 oneOf/allOf 并优化大 schema compact 结构；codex doctor 新增编辑器与 pager 环境信息；Plugin marketplace 列表增强。

