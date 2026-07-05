# 渠道发现与配置推荐设计

## 背景

当前 CCX 已有两类相关能力：

- `capability-test`：按模型和协议实测上游是否能返回有效流式响应。
- `compat-diagnose`：对已保存渠道推荐部分兼容性开关和 BaseURL 修正。

用户的实际问题是：拿到一个上游 `baseURL` 和 `key` 后，不知道哪些真实模型可用、该配置成哪个协议、常见客户端模型名应该怎么重定向，以及目标渠道需要哪些适配性开关。现有功能需要用户先保存渠道并猜模型，反馈不够直接。

## 目标

新增“渠道发现”能力，把未知上游转换成可应用的 CCX 渠道配置建议：

1. 支持未保存渠道：后端通过请求体里的临时配置完成真实上游请求，不要求先创建渠道。
2. 发现真实模型：拉取上游 `/models`，并能在 `/models` 不可用时使用少量内置候选兜底。
3. 实测协议适配：对候选模型执行 `messages`、`responses`、`chat`、`gemini` 能力测试，输出可用协议和可用模型。
4. 推荐模型重定向：为常见客户端模型名生成 `modelMapping`，例如 Claude Code 的 `opus`、`sonnet`、`haiku`、`fable`，Codex 的 `gpt`、`mini`、`codex`。
5. 推荐适配开关：按推荐目标渠道运行兼容性诊断，输出可应用的开关建议和证据。
6. 不持久化副作用：发现流程不写 `.config/config.json`，不注册调度器渠道，不改变熔断状态；用户确认保存后才走现有渠道保存流程。

## 非目标

- 不做完整模型智能分类系统，只用可解释的启发式和实测结果。
- 不自动创建、保存或启用渠道。
- 不支持 `images` 和 `vectors` 的 capability-test 推断；本次仅覆盖 `messages`、`responses`、`chat`、`gemini` 文本协议。
- 不绕过现有鉴权、代理、TLS、custom headers、auth header 逻辑。

## 后端 API

新增管理端点：

```http
POST /api/channel-discovery
```

请求体使用临时渠道配置：

```json
{
  "channelKind": "responses",
  "serviceType": "openai",
  "baseUrls": ["https://example.com/v1"],
  "apiKey": "sk-...",
  "authHeader": "auto",
  "customHeaders": {},
  "proxyUrl": "",
  "insecureSkipVerify": false,
  "modelMapping": {},
  "reasoningMapping": {},
  "targetClients": ["codex", "claude-code"]
}
```

字段约束：

- `channelKind` 可选，空值表示让后端推荐；允许值为 `messages`、`responses`、`chat`、`gemini`。
- `serviceType` 必填；沿用现有 `UpstreamConfig.ServiceType` 语义。
- `baseUrls` 或 `baseUrl` 至少一个非空。
- `apiKey` 必填；只在本次请求内构造临时 `UpstreamConfig.APIKeys`。
- `targetClients` 可选；默认生成 Claude Code 和 Codex 的常见主模型别名，但不构成目标渠道强偏好。只有请求体显式传入 `targetClients` 时，才按目标客户端优先推荐渠道。

响应体示例：

```json
{
  "models": {
    "source": "models_endpoint",
    "url": "https://example.com/v1/models",
    "statusCode": 200,
    "items": ["actual-pro", "actual-main", "actual-mini"],
    "selected": {
      "strong": "actual-pro",
      "primary": "actual-main",
      "fast": "actual-mini"
    },
    "warnings": []
  },
  "protocols": [
    {
      "protocol": "responses",
      "success": true,
      "successModels": ["actual-main", "actual-mini"],
      "failedModels": [],
      "latencyMs": 812
    }
  ],
  "recommendation": {
    "channelKind": "responses",
    "serviceType": "openai",
    "baseUrls": ["https://example.com/v1"],
    "modelMapping": {
      "gpt": "actual-main",
      "mini": "actual-mini",
      "codex": "actual-main"
    },
    "reasoningMapping": {},
    "supportedModels": ["gpt*", "codex*", "mini*"],
    "compat": {
      "stripImageGenerationTool": true
    },
    "urlRecommendation": null
  },
  "evidence": [
    {
      "type": "models",
      "message": "/models returned 3 models"
    },
    {
      "type": "compat",
      "key": "stripImageGenerationTool",
      "message": "upstream rejected image_generation tool"
    }
  ]
}
```

## 后端结构

新增 `internal/handlers/channel_discovery.go` 作为 HTTP 入口，内部拆成小单元：

- `TransientChannelConfig`：请求体 DTO，负责校验和转成 `config.UpstreamConfig`。
- `DiscoverModels`：复用各渠道现有 `/models` URL 构造、鉴权、代理和响应解析逻辑，返回模型列表和错误证据。
- `SelectDiscoveryModels`：从模型列表中选最多少量候选，避免过量真实请求。
- `RunDiscoveryCapability`：复用现有 capability 请求构造和流式检测，支持直接传入临时 `UpstreamConfig`。
- `BuildMappingRecommendation`：根据实测成功模型和目标客户端生成 `modelMapping`、`reasoningMapping`、`supportedModels`。
- `RunDiscoveryCompat`：复用 `runCompatDiagnose`，但按推荐目标 `channelKind` 和临时渠道配置执行。

现有 `TestChannelCapability` 和 `DiagnoseChannelCompat` 保持 channelId 入口不变。公共探测逻辑只抽出必要函数，避免复制协议请求构造。

## 模型候选选择

模型选择必须可解释，优先级如下：

- `strong`：名称包含 `opus`、`pro`、`max`、`ultra`、`codex`，或内置 registry 显示更高上下文/输出能力。
- `primary`：名称包含 `sonnet`、`gpt`、`chat`、`main`，或列表中最像默认主模型的条目。
- `fast`：名称包含 `haiku`、`mini`、`flash`、`lite`、`fast`。

如果只能识别一个可用模型，则 `strong`、`primary`、`fast` 都指向同一真实模型，并在 evidence 中说明“未能区分模型层级，使用单模型覆盖常见别名”。

候选上限默认控制在 6 个以内：`strong`、`primary`、`fast` 加上最多 3 个按 registry 能力排序的补充模型。这样避免新建渠道时一次诊断消耗过高。

## 重定向推荐

按目标渠道生成映射：

- `messages` / Claude Code：
  - `opus` -> `strong`
  - `sonnet` -> `primary`
  - `haiku` -> `fast`
  - `fable` -> `strong`，保持现有配置迁移语义一致
- `responses` / Codex：
  - `gpt` -> `primary`
  - `mini` -> `fast`
  - `codex` -> `strong` 或 `primary`
- `chat` / OpenAI 兼容客户端：
  - `gpt` -> `primary`
  - `mini` -> `fast`
  - `codex` -> `strong` 或 `primary`
- `gemini`：
  - `gemini` -> `primary`
  - `pro` -> `strong`
  - `flash` -> `fast`

推荐只能使用实测成功的真实模型。未通过能力测试的模型不得进入 `modelMapping`。

## 适配开关诊断

发现流程必须按推荐目标渠道运行适配性诊断，而不是只按用户当前 UI Tab 诊断。需要覆盖现有 `compat-diagnose` 能力：

- `normalizeSystemRoleToTopLevel`
- `passbackReasoningContent`
- `passbackThinkingBlocks`
- `stripEmptyTextBlocks`
- `stripThoughtSignature`
- `stripImageGenerationTool`
- `normalizeMetadataUserId`
- `stripBillingHeader`
- BaseURL `#` 修正建议

诊断结果包含 `recommendations` 和 `evidence`。前端应用时只更新当前表单字段，不自动保存。

## 推荐目标渠道

推荐规则：

1. 如果 `targetClients` 明确包含 `codex`，优先选择成功的 `responses`，其次 `chat`。
2. 如果 `targetClients` 明确包含 `claude-code`，优先选择成功的 `messages`。
3. 如果用户没有指定目标客户端，按成功协议数量、首个成功延迟、模型覆盖层级综合排序，优先推荐 `responses` 或 `messages`，并保留其他可用协议作为备选。
4. 如果当前临时 `serviceType` 与推荐 `channelKind` 不同，仍保留 `serviceType`，因为它描述上游原生协议；`channelKind` 描述 CCX 对外入口配置位置。

## 前端交互

在新增/编辑渠道表单中加入“发现配置”按钮。按钮条件：

- 文本协议渠道可见，`images` 和 `vectors` 不显示。
- 至少有一个 Base URL 和一个 API key。

流程：

1. 用户填写 Base URL 和 API key。
2. 点击“发现配置”。
3. 弹出结果面板，展示模型列表、实测协议、推荐目标渠道、推荐重定向、适配开关和证据。
4. 用户点击“应用推荐”后，更新表单字段：
   - `serviceType`
   - `modelMapping`
   - `reasoningMapping`
   - `supportedModels`
   - 兼容性开关
   - BaseURL 修正
5. 用户仍然通过现有保存按钮持久化。

如果用户正在编辑已保存渠道，发现流程也走 transient request，使用表单当前草稿值，而不是旧的已保存 channelId 数据。

## 错误处理

- `/models` 失败但请求探测成功：返回 warning，继续使用内置候选和用户已有模型映射目标。
- 鉴权失败：停止流程，返回明确的 401/403 evidence，不尝试更多模型。
- 余额或额度不足：停止流程，返回上游错误摘要，避免重复消耗。
- 协议全部失败：返回模型发现证据、每个协议失败摘要和 BaseURL 修正建议；不生成自动应用推荐。
- 部分兼容诊断超时：保留模型/协议结果，compat evidence 标记 inconclusive。

所有错误摘要必须脱敏 API key、Authorization 和 multipart 内容。

## 测试计划

后端：

- transient 配置能构造 `UpstreamConfig`，且不写入 `ConfigManager`。
- `/models` 成功时能解析模型并选择 `strong`、`primary`、`fast`。
- 单模型上游能生成覆盖常见别名的映射。
- 未通过能力测试的模型不会进入推荐映射。
- 推荐目标渠道后会运行对应 compat 诊断并返回开关建议。
- `/models` 失败时能走候选兜底并保留 warning。
- API key 和 Authorization 不出现在 evidence。

前端：

- 新建渠道表单能用草稿 baseURL/key 调用发现接口。
- 编辑渠道表单使用当前草稿值，不依赖已保存 channelId。
- 应用推荐只更新表单，不自动保存。
- 结果面板能展示协议成功/失败、模型层级、映射和开关证据。

验证命令：

- `cd backend-go && go test ./internal/handlers ./internal/config`
- `cd frontend && bun run build`

## 设计约束

- KISS：新增一个发现入口，复用现有 capability 和 compat 探测，不复制协议转换。
- YAGNI：只覆盖文本协议和常见主模型别名，不引入用户不可解释的模型评分系统。
- DRY：模型请求、能力测试、适配诊断都抽公共函数复用。
- SOLID：HTTP handler 只做参数校验和响应组装；模型发现、候选选择、映射推荐、兼容诊断各自单一职责。
