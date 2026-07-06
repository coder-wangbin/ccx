# CCX 渠道自动托管 (Channel Autopilot) 设计方案

## 1. 设计目标

> 用户添加渠道只需 baseURL + apiKey，系统自动完成协议发现、模型映射、能力画像、健康诊断、智能调度。
> 高级用户可覆盖任何自动决策，但默认不需要碰。

### 1.1 核心用户故事

| # | 作为… | 我希望… | 这样我就能… |
|---|-------|---------|------------|
| U1 | 有 30-40 个中转站的用户 | 添加渠道时只填 URL + Key | 不用手动配 modelMapping / supportedModels / 兼容开关 |
| U2 | 想用 Opus 监工 + 白嫖子代理的用户 | 主代理自动走高智商稳定渠道，子代理自动走便宜/快速渠道 | 不用手动给每个渠道调优先级 |
| U3 | 渠道会死/会限流的用户 | 一眼看到哪些渠道死了、限流了、配置错了 | 快速清理或修复，不用逐个 ping |
| U4 | 有临时薅羊毛渠道的用户 | 临时池优先消耗，用完自动切常规池 | 不浪费免费额度 |

### 1.2 设计原则

- **SOLID**：Analyzer、Profiler、ModelResolver、SmartRouter 职责单一，接口隔离
- **KISS**：先用可解释规则，不做复杂 AI 打分
- **DRY**：复用现有 MetricsManager、能力测试、模型注册表、CandidateFilter
- **YAGNI**：Phase 1 只做自动画像 + 健康诊断，Phase 2 做智能调度，Phase 3 做自愈

---

## 2. 现有基础（可直接复用）

### 2.1 调度器筛选链

`backend-go/internal/scheduler/select.go` 中 `SelectChannelWithOptions` 已有完整筛选链：

```text
Active+Model过滤 → RoutePrefix过滤 → 上下文过滤 → CandidateFilter回调
→ Channel Pinning → Manual Override → Promotion优先 → Trace亲和
→ Priority遍历(含健康检查/熔断/限速/视觉保护) → Soft-skip回退 → Degraded兜底
```

**关键扩展点**：`SelectionOptions.CandidateFilter` 是注入式回调，autopilot 可在此注入标签/画像过滤逻辑。

### 2.2 指标系统

`backend-go/internal/metrics/` 提供：

| 能力 | 接口 | 说明 |
|------|------|------|
| 健康判断 | `IsChannelHealthyMultiURL` | 多 URL 聚合 |
| 失败率 | `CalculateChannelFailureRateMultiURL` | 滑动窗口 |
| 聚合指标 | `GetChannelAggregatedMetrics` | 15m 成功率/请求数/缓存率 |
| 熔断状态 | `GetChannelCircuitStateMultiURL` | Closed/Open/HalfOpen |
| 失败分类 | `FailureClass` | retryable/overloaded/non_retryable/quota |
| 请求日志 | `ChannelLog` | 含 AgentRole、模型、延迟、流健康 |

### 2.3 能力测试与模型发现

`backend-go/internal/handlers/channel_discovery.go` 已实现：

- 协议自动探测：对 messages/responses/chat/gemini 四协议并发探测
- 模型自动发现：拉 `/v1/models`，失败时用内置候选模型回退
- 模型映射推荐：根据探测结果自动生成 modelMapping
- 能力探测：工具调用、视觉、thinking passback 测试

`capability_test_runner.go` 提供完整的多模型轮询测试框架。

### 2.4 模型注册表

`backend-go/internal/config/generated_model_registry.go` + `model_registry.go`：

- `ResolveUpstreamCapability`：四层解析（channel → global → builtin → default）
- `ResolveAgentModelProfile`：下游代理模型上下文窗口
- 覆盖 Claude/GPT/Gemini/DeepSeek/Kimi/GLM 等主流模型
- 每个模型有 ContextWindowTokens、MaxOutputTokens、Capabilities、Pricing

### 2.5 角色识别

`backend-go/internal/utils/headers.go` 的 `ExtractAgentContext`：

- Codex 子代理：`client_metadata.x-openai-subagent` 精确识别
- Claude Code 子代理：`X-Claude-Code-Agent-Id` header 精确识别
- 启发式识别：消息数 + 工具调用模式
- 已用于 trace 亲和隔离（`:subagent` 后缀）

## 3. 数据模型

### 3.1 渠道画像 (ChannelProfile)

每个渠道自动生成一份多维画像，存储在 SQLite 中，运行时缓存：

```go
// backend-go/internal/autopilot/profile.go

type ChannelProfile struct {
    ChannelID   int        `json:"channelId"`
    ChannelKind string     `json:"channelKind"` // messages/chat/responses/...
    UpdatedAt   time.Time  `json:"updatedAt"`

    // ── 自动推导维度 ──
    HealthState    HealthState    `json:"healthState"`    // healthy/degraded/limited/misconfigured/dead/unknown
    HealthConfidence float64      `json:"healthConfidence"` // 0.0-1.0
    QualityTier    QualityTier    `json:"qualityTier"`    // low/normal/high/premium
    StabilityTier  StabilityTier  `json:"stabilityTier"`  // unstable/normal/stable
    SpeedTier      SpeedTier      `json:"speedTier"`      // slow/normal/fast
    CostTier       CostTier       `json:"costTier"`       // free/cheap/normal/expensive

    // ── 能力标签 ──
    SupportsVision     bool `json:"supportsVision"`
    SupportsToolCalls  bool `json:"supportsToolCalls"`
    SupportsReasoning  bool `json:"supportsReasoning"`
    SupportsLongCtx    bool `json:"supportsLongCtx"` // >200K

    // ── 运行时指标快照 ──
    SuccessRate15m   float64  `json:"successRate15m"`
    P95LatencyMs     int64    `json:"p95LatencyMs"`
    ConsecutiveFail  int      `json:"consecutiveFail"`
    LastSuccessAt    *time.Time `json:"lastSuccessAt,omitempty"`
    LastFailureAt    *time.Time `json:"lastFailureAt,omitempty"`

    // ── 诊断 ──
    HealthEvidence   []string        `json:"healthEvidence"`
    SuggestedAction  SuggestedAction `json:"suggestedAction"`
    AvailableModels  int             `json:"availableModels"`

    // ── 元数据 ──
    Source           string  `json:"source"`    // auto_probe | manual_override | runtime_update
    Confidence       float64 `json:"confidence"` // 画像整体置信度
}

type HealthState    string // "healthy" | "degraded" | "limited" | "misconfigured" | "dead" | "unknown"
type QualityTier    string // "low" | "normal" | "high" | "premium"
type StabilityTier  string // "unstable" | "normal" | "stable"
type SpeedTier      string // "slow" | "normal" | "fast"
type CostTier       string // "free" | "cheap" | "normal" | "expensive"
type SuggestedAction string // "none" | "replace_keys" | "fix_config" | "wait_recovery" | "delete" | "observe"
```

### 3.2 模型画像 (ModelProfile)

每个"渠道 + 模型"组合的画像：

```go
// backend-go/internal/autopilot/model_profile.go

type ModelProfile struct {
    ChannelID   int    `json:"channelId"`
    ChannelKind string `json:"channelKind"`
    ModelID     string `json:"modelId"`     // 渠道内实际模型名
    UpdatedAt   time.Time `json:"updatedAt"`

    // ── 能力 ──
    QualityTier   QualityTier `json:"qualityTier"`
    SpeedTier     SpeedTier   `json:"speedTier"`
    ContextTokens int         `json:"contextTokens"`
    SupportsVision    bool    `json:"supportsVision"`
    SupportsToolCalls bool    `json:"supportsToolCalls"`
    SupportsReasoning bool    `json:"supportsReasoning"`

    // ── 探测结果 ──
    ProbeSuccess    bool      `json:"probeSuccess"`
    LastProbeAt     time.Time `json:"lastProbeAt"`
    ProbeLatencyMs  int64     `json:"probeLatencyMs"`
    ProbeConfidence float64   `json:"probeConfidence"`

    // ── 来源 ──
    Source string `json:"source"` // builtin_registry | auto_probe | capability_test | manual
}
```

### 3.3 请求画像 (RequestProfile)

每次请求在进入调度器前生成，不持久化：

```go
// backend-go/internal/autopilot/request_profile.go

type RequestProfile struct {
    // ── 来自请求 ──
    Model       string // 请求的目标模型
    AgentRole   string // "main" | "subagent"
    AgentType   string // "codex_subagent" | "claude_code_subagent"
    HasImage    bool   // 是否包含图片
    EstTokens   int    // 估算输入 token 数

    // ── 来自模型注册表 ──
    QualityNeed   QualityTier   // 该模型对应的质量需求
    ContextNeed   int           // 最小上下文窗口
    VisionNeed    bool          // 是否需要识图
    ToolUseNeed   bool          // 是否需要工具调用
    ReasoningNeed bool          // 是否需要推理

    // ── 任务分类 ──
    TaskClass TaskClass // supervisor | worker | lightweight | vision | long_context
}

type TaskClass string
const (
    TaskClassSupervisor   TaskClass = "supervisor"    // 主代理/监工
    TaskClassWorker       TaskClass = "worker"         // 子代理/干活
    TaskClassLightweight  TaskClass = "lightweight"    // 轻任务（摘要/标题）
    TaskClassVision       TaskClass = "vision"         // 识图任务
    TaskClassLongContext  TaskClass = "long_context"   // 长上下文任务
)
```

### 3.4 存储方案

| 数据 | 存储 | TTL |
|------|------|-----|
| ChannelProfile | SQLite `channel_profiles` 表 + 内存缓存 | 持久化，运行时 5min 刷新 |
| ModelProfile | SQLite `model_profiles` 表 + 内存缓存 | 持久化，运行时 10min 刷新 |
| RequestProfile | 内存 | 请求级，不持久化 |
| 健康证据 | SQLite `health_evidence` 表 | 7 天滚动清理 |
| 画像变更日志 | SQLite `profile_changelog` 表 | 30 天滚动清理 |

```sql
CREATE TABLE channel_profiles (
    channel_id   INTEGER NOT NULL,
    channel_kind TEXT    NOT NULL,
    profile_json TEXT    NOT NULL,
    updated_at   TEXT    NOT NULL,
    PRIMARY KEY (channel_id, channel_kind)
);

CREATE TABLE model_profiles (
    channel_id   INTEGER NOT NULL,
    channel_kind TEXT    NOT NULL,
    model_id     TEXT    NOT NULL,
    profile_json TEXT    NOT NULL,
    updated_at   TEXT    NOT NULL,
    PRIMARY KEY (channel_id, channel_kind, model_id)
);

CREATE TABLE health_evidence (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id  INTEGER NOT NULL,
    kind        TEXT    NOT NULL,
    evidence    TEXT    NOT NULL,
    severity    TEXT    NOT NULL,
    created_at  TEXT    NOT NULL
);
CREATE INDEX idx_health_evidence_channel ON health_evidence(channel_id, created_at);
```

## 4. 核心组件设计

### 4.1 组件总览

```text
┌─────────────────────────────────────────────────────────┐
│                    Channel Autopilot                     │
│                                                         │
│  ┌───────────┐  ┌───────────┐  ┌─────────────────────┐ │
│  │ Discovery  │  │ Profiler  │  │ HealthAnalyzer      │ │
│  │ (协议发现  │→│ (画像生成  │→│ (健康诊断/分层)      │ │
│  │  模型发现) │  │  能力推导) │  │                     │ │
│  └───────────┘  └───────────┘  └─────────────────────┘ │
│        │              │               │                 │
│        └──────────────┴───────────────┘                 │
│                       ▼                                 │
│              ┌─────────────────┐                        │
│              │  ProfileStore   │ ← SQLite + 内存缓存    │
│              └─────────────────┘                        │
│                       ▼                                 │
│              ┌─────────────────┐                        │
│              │  SmartRouter    │ ← 注入 CandidateFilter │
│              │ (任务分类→标签   │                        │
│              │  匹配→排序)     │                        │
│              └─────────────────┘                        │
│                       ▼                                 │
│              ┌─────────────────┐                        │
│              │ Scheduler       │ (现有，不修改核心链路)  │
│              │ SelectChannel   │                        │
│              └─────────────────┘                        │
└─────────────────────────────────────────────────────────┘
```

### 4.2 Discovery — 协议与模型发现

**职责**：渠道添加后自动探测协议、发现模型、推荐映射。

**触发时机**：
1. 渠道首次添加（`autoManaged: true` 时）
2. 手动触发「重新发现」
3. 定时刷新（每天一次）

**流程**：

```text
添加渠道(baseURL + key)
  │
  ├─ 1. 协议探测：并发探测 messages/responses/chat/gemini
  │     └─ 复用 capability_test_runner.executeModelTest
  │
  ├─ 2. 模型发现：GET /v1/models (或各协议等价端点)
  │     └─ 复用 channel_discovery.discoverTransientModels
  │     └─ 失败时用内置候选模型列表回退
  │
  ├─ 3. 模型选择：从发现的模型中选 Strong/Primary/Fast 三档
  │     └─ 复用 channel_discovery.selectDiscoveryModels
  │
  ├─ 4. 能力探测：对选中模型做 tool_call/vision/thinking 测试
  │     └─ 复用 channel_discovery.runDiscoveryToolCallProbe
  │     └─ 复用 channel_discovery.runDiscoveryVisionProbe
  │
  ├─ 5. 映射推荐：根据协议类型生成 modelMapping
  │     └─ 复用 channel_discovery.buildDiscoveryMappingRecommendation
  │
  └─ 6. 生成 ModelProfile：写入 model_profiles 表
```

**与现有 Channel Discovery 的关系**：

现有 `POST /channel-discovery` 是一个"预览"接口，返回推荐但不自动应用。Autopilot 复用其核心逻辑，但：
- 自动写入 `modelMapping`、`supportedModels`、兼容开关
- 自动生成 `ModelProfile` 记录
- 对 `autoManaged` 渠道静默执行，对非 auto 渠道提供「建议应用」按钮

### 4.3 Profiler — 画像生成器

**职责**：综合模型注册表、探测结果、运行时指标，生成 ChannelProfile 和 ModelProfile。

**推导规则**：

#### QualityTier 推导

```text
优先级 1：模型注册表 BuiltinUpstreamModelCapabilities 中的模型族
  claude-opus-* / gpt-5.5 / gpt-5.4     → premium
  claude-sonnet-* / gpt-5.3-codex        → high
  claude-haiku-* / gpt-5.4-mini          → normal
  其他                                    → low

优先级 2：渠道级 LowQuality 标记
  lowQuality: true → 最高 normal

优先级 3：capability-test 探测质量
  探测响应长度 > 100 tokens 且无截断 → 保持注册表推导
  探测失败或空响应                   → 降一档
```

#### StabilityTier 推导

```text
基于最近 1 小时指标：
  成功率 >= 95% 且 429 率 < 5%    → stable
  成功率 >= 80% 且 429 率 < 20%   → normal
  其他                            → unstable

额外信号（任一命中则降级）：
  连续失败 >= 5 次                → 最高 normal
  熔断器 open                     → unstable
  最近成功 > 6 小时前             → unstable
```

#### SpeedTier 推导

```text
基于最近 100 次请求的 p95 首 token 延迟：
  < 500ms   → fast
  < 2000ms  → normal
  >= 2000ms → slow

冷启动：无足够数据时用 capability-test 的 ProbeLatencyMs
  < 800ms   → fast
  < 3000ms  → normal
  >= 3000ms → slow
```

#### CostTier 推导

```text
优先级 1：模型注册表中的 Pricing 字段
  Input/Output 都是 0            → free
  Input < $1/M 且 Output < $5/M  → cheap
  Input < $10/M 且 Output < $30/M → normal
  其他                           → expensive

优先级 2：用户手动标记（costHint 字段）

优先级 3：运行时行为启发
  频繁 429 且无 Retry-After      → 可能是免费/低配额，标记 cheap
  频繁 402/余额不足              → 有成本，标记 normal
```

#### 能力标签推导

**⚠️ 原则：只做硬失败检测，不判定软质量问题**

识图/工具/reasoning 的"虚标"（渠道声称支持但实际返回垃圾）可靠判定很难。策略是：

- **硬失败**（可自动检测）：调用报错、格式错误、HTTP 错误码、解析失败
- **软质量问题**（留给人工）：答非所问、内容质量低、thinking 输出无意义

```text
SupportsVision：
  ── 硬条件（自动判定）──
  1. 注册表 Capabilities["vision"] == true
  2. 且 NoVision != true
  3. 且 (NoVisionModels 不含该模型 || VisionFallbackModel 已设置)
  ── 可选验证（L3 探测）──
  4. vision probe 返回 HTTP 200 且响应可解析（不要求内容质量）
  5. 如果 probe 返回 400/415/unsupported → 明确标记 SupportsVision=false

SupportsToolCalls：
  ── 硬条件（自动判定）──
  1. 注册表 Capabilities["toolCalls"] == true
  ── 可选验证（L3 探测）──
  2. tool_call probe 返回 HTTP 200 且响应含合法 tool_use block
  3. 如果 probe 返回 400/tool_not_found → 明确标记 SupportsToolCalls=false

SupportsReasoning：
  ── 硬条件（自动判定）──
  1. 注册表 ThinkingMode 非空
  ── 可选验证（L3 探测）──
  2. reasoning probe 返回 HTTP 200 且响应含 thinking/reasoning block
  3. 如果 probe 返回 400/thinking_not_supported → 明确标记 SupportsReasoning=false

SupportsLongCtx：
  1. ContextWindowTokens >= 200_000（来自注册表，无需探测）
  2. 或注册表 Supports1M == true
```

**虚标处理**：如果 L1 被动信号显示某渠道的 vision/tool/reasoning 请求**成功但用户标记为"结果差"**，系统不自动关闭标签，而是在 UI 上显示「⚠️ 用户反馈能力可能不准确」，允许人工 override。这避免了系统在"质量差"和"不支持"之间误判。

### 4.4 HealthAnalyzer — 健康诊断器

**职责**：持续分析渠道健康，生成 HealthState 和证据。

**⚠️ 核心原则：被动优先 (Passive-First)**

30-40 渠道 × 多模型的主动探测有 quota 成本，且白嫖渠道本身就抖。诊断信号分三层，成本递增：

| 层级 | 信号来源 | 成本 | 频率 | 适用场景 |
|------|---------|------|------|---------|
| L1 被动信号 | 真实请求的 MetricsManager | 零 | 实时/每次请求 | **默认层**，所有健康判定的主要依据 |
| L2 轻量探测 | 单模型 ping（最小 prompt） | 极低 | cooldown 复测 | L1 无数据（新渠道/长时间无请求） |
| L3 深度探测 | capability-test（多模型多协议） | 中 | 手动/每天 | 新渠道首次画像、用户主动触发、misconfigured 修复后 |

**分析周期**：
- L1 被动：每次请求后增量更新（复用 MetricsManager 已有的 RecordSuccess/RecordFailure）
- L1 聚合：每 5 分钟做一次滑动窗口聚合，更新 ChannelProfile
- L2 探测：仅在以下条件触发：
  - 渠道状态为 `unknown` 且添加超过 10 分钟
  - `limited`/`dead` 的 cooldown 到期
  - L1 数据不足（最近 1 小时请求数 < 3）
- L3 深度：仅在以下条件触发：
  - 用户手动点击「重新探测」
  - 新渠道 `autoManaged` 首次添加
  - `misconfigured` 状态修复后

**被动信号指标**（全部来自 MetricsManager 现有数据，无需额外请求）：

```text
成功/失败率     → MetricsManager.CalculateChannelFailureRateMultiURL
429 率         → FailureClass=overloaded 计数 / 总请求数
断流率         → ChannelLog.Status="streaming" 但无 "completed" 的比率
空响应率       → ChannelLog 中 DurationMs > 0 但 usage.InputTokens=0 的比率
p95 延迟       → ChannelLog.DurationMs 的 p95 分位
连续失败       → MetricsManager 滑动窗口 consecutiveFail
最后成功时间   → MetricsManager.GetChannelAggregatedMetrics.LastSuccessAt
熔断器状态     → MetricsManager.GetChannelCircuitStateMultiURL
Key 健康       → DisabledAPIKeys 数量 vs 总 Key 数量
```

**诊断逻辑**（见第 6 章详细设计）。

### 4.5 SmartRouter — 智能路由注入

**职责**：根据请求画像 + 渠道画像，通过 CandidateFilter 注入调度链。

**集成方式**：不修改 `SelectChannelWithOptions` 核心链路，而是：

```go
// backend-go/internal/autopilot/smart_router.go

type SmartRouter struct {
    profileStore  *ProfileStore
    modelResolver *ModelResolver
    config        *SmartRoutingConfig
}

// BuildCandidateFilter 为每次请求构建 CandidateFilter
func (r *SmartRouter) BuildCandidateFilter(profile *RequestProfile) scheduler.CandidateFilterFunc {
    return func(channels []scheduler.ChannelInfo, upstreamFor func(scheduler.ChannelInfo) *config.UpstreamConfig, candidateAvailable func(scheduler.ChannelInfo, *config.UpstreamConfig) bool) ([]scheduler.ChannelInfo, error) {
        // 1. 获取每个渠道的画像
        // 2. 根据 TaskClass 应用过滤/排序策略
        // 3. 返回重排后的候选列表
    }
}
```

**注入点**：在各 handler 调用 `HandleMultiChannelFailover` 前，设置 `SelectionOptions.CandidateFilter`。

**与现有 CandidateFilter 的兼容**：SmartRouter 生成的 filter 在外层，handler 自有的 filter（如 vision 保护）在内层。SmartRouter 负责"粗筛"（标签匹配、健康过滤），handler filter 负责"细选"（视觉保护、上下文限制）。

## 5. 智能调度策略

### 5.1 任务分类 (TaskClassifier)

请求进入时自动分类，决定调度策略：

```go
// backend-go/internal/autopilot/task_classifier.go

func ClassifyRequest(profile *RequestProfile) TaskClass {
    // 1. 识图任务优先判定
    if profile.HasImage && profile.VisionNeed {
        return TaskClassVision
    }

    // 2. 长上下文任务
    if profile.ContextNeed > 200_000 {
        return TaskClassLongContext
    }

    // 3. 主代理/监工
    if profile.AgentRole == "main" || profile.AgentRole == "" {
        // 主代理默认走 Supervisor 策略
        return TaskClassSupervisor
    }

    // 4. 子代理
    if profile.AgentRole == "subagent" {
        // 子代理默认走 Worker 策略
        return TaskClassWorker
    }

    return TaskClassWorker // 兜底
}
```

**轻任务识别**（可选，Phase 2+）：
- 模型名包含 `haiku` / `mini` / `flash`
- 上下文 < 10K 且无图片
- 请求类型为 `count_tokens` / 简单分类

### 5.2 调度策略矩阵

每个 TaskClass 对应一组优先级规则：

#### Supervisor（主代理/监工）

```text
优先级 1：qualityTier=high|premium + stabilityTier=stable + 长上下文
优先级 2：qualityTier=high + stabilityTier=normal
优先级 3：qualityTier=normal + stabilityTier=stable
降级    ：qualityTier=high + stabilityTier=degraded（仅当无稳定高智商渠道时）
禁止    ：stabilityTier=unstable, costTier=free, qualityTier=low
```

#### Worker（子代理）

```text
优先级 1：costTier=free|cheap + qualityTier=normal|high（临时池/白嫖池）
优先级 2：costTier=cheap + speedTier=fast
优先级 3：speedTier=fast + qualityTier=low|normal
优先级 4：qualityTier=normal + stabilityTier=stable（常规池）
默认跳过：costTier=expensive, qualityTier=premium
```

#### Lightweight（轻任务）

```text
优先级 1：speedTier=fast + costTier=free|cheap
优先级 2：costTier=free
优先级 3：speedTier=fast
禁止    ：qualityTier=premium, 视觉池, 长上下文池
```

#### Vision（识图任务）

```text
硬过滤  ：SupportsVision=true 且 SupportsToolCalls=true（如需要）
优先级 1：qualityTier=high|premium + vision
优先级 2：qualityTier=normal + vision
降级    ：当所有 vision 渠道不可用时，尝试 visionFallbackModel
禁止    ：SupportsVision=false 的渠道
```

#### LongContext（长上下文）

```text
硬过滤  ：ContextWindowTokens >= 请求需要的最小窗口
优先级 1：qualityTier=high|premium + longContext + stable
优先级 2：qualityTier=normal + longContext
禁止    ：ContextWindowTokens < 需求 或 SupportsLongCtx=false
```

### 5.3 CandidateFilter 实现

```go
func (r *SmartRouter) filterByTaskStrategy(
    channels []scheduler.ChannelInfo,
    profiles map[int]*ChannelProfile,
    strategy taskStrategy,
) []scheduler.ChannelInfo {

    // 1. 硬过滤：排除不满足硬约束的渠道
    filtered := hardFilter(channels, profiles, strategy)

    // 2. 标签评分：每个渠道按策略规则打分
    scored := scoreChannels(filtered, profiles, strategy)

    // 3. 按分数降序排列
    sort.Slice(scored, func(i, j int) bool {
        return scored[i].Score > scored[j].Score
    })

    // 4. 返回重排后的 ChannelInfo 列表
    return scored.ToChannelInfoList()
}
```

**评分公式**：

```text
Score = w_quality * qualityScore
      + w_stability * stabilityScore
      + w_speed * speedScore
      + w_cost * costScore
      + w_tier_match * tierMatchBonus
      - penalty

其中：
  qualityScore:   low=1, normal=2, high=3, premium=4
  stabilityScore: unstable=0, normal=1, stable=2
  speedScore:     slow=0, normal=1, fast=2
  costScore:      expensive=0, normal=1, cheap=2, free=3

  tierMatchBonus: 渠道画像标签匹配策略优先标签时 +10
  penalty:        healthState=degraded 时 -5, limited 时 -20

  权重根据 TaskClass 不同：
  Supervisor: w_quality=3, w_stability=2, w_speed=1, w_cost=0
  Worker:     w_quality=1, w_stability=1, w_speed=2, w_cost=3
  Lightweight:w_quality=0, w_stability=1, w_speed=3, w_cost=3
  Vision:     w_quality=2, w_stability=2, w_speed=1, w_cost=1
  LongContext: w_quality=2, w_stability=2, w_speed=1, w_cost=0
```

### 5.4 模型自动映射 (ModelResolver)

当请求的模型在某个渠道的 `supportedModels` 中不存在时，自动寻找最佳映射。

**⚠️ 核心约束：能力下界 (Capability Floor)**

模型映射最大的风险是**语义降级**：用户以为在用 opus 级能力，实际被路由到白嫖模型，输出质量下降但无信号。因此映射必须满足能力下界约束：

```go
// backend-go/internal/autopilot/model_resolver.go

type CapabilityFloor struct {
    MinContextTokens   int    // 请求模型的 AgentModelProfile.ContextWindowTokens
    NeedsReasoning     bool   // 请求模型的 ThinkingMode 非空
    NeedsVision        bool   // 请求包含图片
    NeedsToolCalls     bool   // 请求包含工具定义
    MinQualityTier     QualityTier // 请求模型对应的质量档
}

type ModelResolver struct {
    profileStore *ProfileStore
}

// ResolveModel 将请求模型映射到渠道实际支持的模型
// 返回 (mappedModel, resolved, reason)
// resolved=false 表示该渠道无满足下界约束的模型，应跳过此渠道
func (r *ModelResolver) ResolveModel(
    requestModel string,
    channelID int,
    channelKind string,
    floor CapabilityFloor,
) (string, bool, string) {
    // 1. 查现有 modelMapping（精确匹配 → 模糊匹配）
    //    复用 config.RedirectModel
    //    如果有显式映射，信任用户配置，不做下界检查
    //    （用户手动设的映射视为已知正确）

    // 2. autoManaged 渠道：查 ModelProfile 表
    candidates := r.profileStore.GetModelProfiles(channelID, channelKind)

    // 3. 硬过滤：只保留满足能力下界的模型
    eligible := filterByCapabilityFloor(candidates, floor)
    if len(eligible) == 0 {
        return "", false, "no model meets capability floor"
    }

    // 4. 在满足下界的模型中选最佳匹配
    //    匹配优先级：
    //    a. 同模型族（opus→opus, sonnet→sonnet）—— 最高优先
    //    b. 同质量档（premium→premium）
    //    c. 上下文窗口最接近（不超也不差太多）
    //    d. 探测延迟最低
    best := rankBySimilarity(eligible, requestModel, floor)

    return best.ModelID, true, fmt.Sprintf("mapped %s→%s (family:%s, quality:%s)",
        requestModel, best.ModelID, best.Family, best.QualityTier)
}

// filterByCapabilityFloor 只保留满足所有下界约束的模型
func filterByCapabilityFloor(profiles []ModelProfile, floor CapabilityFloor) []ModelProfile {
    var eligible []ModelProfile
    for _, p := range profiles {
        if !p.ProbeSuccess {
            continue // 未验证通过的模型不参与自动映射
        }
        if p.ContextTokens < floor.MinContextTokens {
            continue
        }
        if floor.NeedsReasoning && !p.SupportsReasoning {
            continue
        }
        if floor.NeedsVision && !p.SupportsVision {
            continue
        }
        if floor.NeedsToolCalls && !p.SupportsToolCalls {
            continue
        }
        if qualityTierRank(p.QualityTier) < qualityTierRank(floor.MinQualityTier) {
            continue
        }
        eligible = append(eligible, p)
    }
    return eligible
}
```

**映射示例**：

| 请求模型 | 能力下界 | 渠道实际模型 | 映射依据 | 是否通过 |
|----------|---------|-------------|---------|---------|
| `claude-opus-4-8` | context:1M, reasoning, quality:premium | `claude-opus-4-7` | 同 opus 族，满足全部下界 | ✓ |
| `claude-opus-4-8` | context:1M, reasoning, quality:premium | `claude-haiku-4-5` | haiku 不满足 quality:premium | ✗ 跳过渠道 |
| `gpt-5.5` | quality:premium, reasoning | `gpt-5.4` | 同 premium 档，满足下界 | ✓ |
| `claude-sonnet-5` | quality:high | `claude-sonnet-4-6` | 同 sonnet 族，满足下界 | ✓ |
| 请求含图片 | vision:true | 某渠道无 vision 模型 | 不满足 vision 下界 | ✗ 跳过渠道 |

**映射结果回显**：

这是调试的关键。映射发生时，必须在响应中标注真实使用的模型：

```text
方案 A（推荐）：在 response header 中回显
  X-CCX-Mapped-Model: claude-opus-4-7
  X-CCX-Original-Model: claude-opus-4-8
  X-CCX-Mapping-Source: auto_resolved

方案 B：在 Claude Messages 响应 body 的 model 字段用真实模型
  {"model": "claude-opus-4-7", ...}  // 而非请求的 claude-opus-4-8

方案 C：两者都做（最利于调试）
```

**安全边界**：
- 仅 `autoManaged: true` 的渠道触发自动映射
- 显式 `modelMapping` 始终优先，不经过下界检查（信任用户配置）
- 映射结果持久化到 `modelMapping`，用户可在 UI 查看、修改、删除
- 映射日志记录每次决策（requestModel → mappedModel → floor → reason），写入 ChannelLog
- **禁止链式映射**：A→B 后不再 B→C，避免不可预测的降级链

## 6. 健康诊断系统

### 6.1 HealthState 状态机

```text
                    ┌──────────┐
          添加渠道 →│ unknown  │
                    └────┬─────┘
                         │ L2 探测成功 或 首次真实请求成功
                         ▼
                    ┌──────────┐
         ┌────────→│ healthy  │←────────┐
         │         └──┬───┬───┘         │
         │ L1被动信号  │   │ L1被动信号  │ L2 探测成功
         │ 成功率↓    │   │ 连续失败≥3  │ 或 真实请求成功
         │ 429增多    │   │             │
         │         ┌──▼┐  │        ┌────┴─────┐
         │         │deg│  │        │ limited  │
         │         │rad│  │        │(429/quota)│
         │         └─┬─┘  │        └────┬─────┘
         │           │    │             │
         │  L1连续≥10│    │ L1连续≥5   │ cooldown 到期
         │  或成功率  │    │ 且全部key   │ L2 探测失败
         │  <50%     │    │ 认证失败    │
         │         ┌─▼──┐ │        ┌────▼─────┐
         │         │dead│◄┘        │ dead     │
         │         └─┬──┘          └──────────┘
         │           │
         │  L2恢复   │  L1/L3 检测到配置错误
         │  探测成功  │
         │           │         ┌──────────────┐
         └───────────┘         │ misconfigured│← 用户修复后 L3 重测
                               └──────────────┘
```

**关键：所有正常状态转换基于 L1 被动信号，不需要额外请求。**

### 6.2 诊断规则

#### Dead（高置信度死亡）

```text
── 硬死（confidence >= 0.95，全部来自 L1 被动信号）──
  - 全部 Key 返回 401/403（最近 1 小时内的真实请求，FailureClass=non_retryable）
  - DNS/TLS 连接失败（ChannelLog 中 error 含 "dial tcp"/"tls"/"certificate"）
  - 连续失败 >= 15 次（MetricsManager 滑动窗口）

── 软死（confidence >= 0.80，L1 被动信号）──
  - 最近 24 小时无成功请求，且有失败记录
  - 熔断器 open 且 lastSuccessAt > 6 小时前
  - 成功率 < 10%（最近 1 小时，且请求样本 >= 5）

── 确认（仅在 L1 不足时触发 L2）──
  - L1 数据不足（请求数 < 5）但 L2 探测连续失败 >= 3 次
```

#### Degraded（可用但质量差）

```text
── 全部来自 L1 被动信号 ──
  - 成功率 50%-80%（最近 1 小时，请求样本 >= 10）
  - p95 延迟 > 5000ms（最近 1 小时）
  - 断流率 > 20%（ChannelLog 中 streaming→非completed 的比率，最近 30 分钟）
  - 空响应率 > 10%（usage 全零但无报错，最近 30 分钟）
```

#### Limited（限流中）

```text
── 全部来自 L1 被动信号 ──
  - FailureClass=overloaded 占比 > 30%（最近 15 分钟）
  - Retry-After header 出现在最近 5 分钟内的 ChannelLog
  - FailureClass=quota（402/insufficient_balance/insufficient_quota）
  - 熔断器 open 但 lastSuccessAt < 6 小时前（区别于 dead）
```

#### Misconfigured（配置疑似错误）

```text
── L1 被动信号 ──
  - 全部请求返回 404（modelMapping 指向不存在的模型）
  - 501/505（协议不支持）
  - capability-test 中仅部分协议成功，但 serviceType 配的是失败协议

── L3 深度探测确认（可选，用户手动触发）──
  - chat 协议成功但 serviceType 配为 claude
  - authHeader 类型与响应不匹配
```

#### Unknown（证据不足）

```text
  - 新添加的渠道，无历史数据
  - 最近 24 小时内请求数 < 5 且未运行 L2 探测
  - L3 capability-test 未运行或已过期（> 7 天）
```

### 6.3 死亡类型细分

```go
type DeathType string
const (
    DeathTypeHard       DeathType = "hard"        // DNS/TLS/认证（L1 被动即可判定）
    DeathTypeSoft       DeathType = "soft"        // 429/quota/临时错误（L1 被动即可判定）
    DeathTypeModel      DeathType = "model"       // 模型不可用（L1 或 L3）
    DeathTypeQuality    DeathType = "quality"     // 空响应/断流（L1 被动即可判定）
    DeathTypeUnknown    DeathType = "unknown"     // 无法分类
)
```

**注意：Quality 死亡只检测"硬失败"**（空响应、断流、格式错误），不检测"答非所问"等软质量问题。软质量问题留给人工标签 override。

### 6.4 健康诊断对调度的影响

```text
HealthState      │ 调度行为                    │ UI 表现
─────────────────┼─────────────────────────────┼──────────────
healthy          │ 正常参与调度                │ 绿色
degraded         │ 降权，只在同池不足时使用     │ 黄色
limited          │ cooldown 内跳过，到期 L2 复测│ 橙色
misconfigured    │ 不参与自动调度，提示修复     │ 紫色
dead             │ 默认移出调度，建议清理       │ 红色
unknown          │ 低风险请求小流量试探         │ 灰色
```

**自动恢复机制**：
- `limited` 渠道：cooldown 到期后触发 L2 轻量探测（单模型、最小 prompt），成功则回到 `healthy`
- `dead` 软死渠道：每 30 分钟检查一次 L1 被动信号，如果有真实请求成功则回到 `healthy`；无真实请求时每 2 小时触发 L2 探测，连续 3 次成功则恢复
- `dead` 硬死渠道：每 6 小时触发 L2 探测，连续 3 次成功则回到 `unknown`（不是直接 `healthy`，需真实请求验证）
- `misconfigured` 渠道：用户修复配置后手动触发 L3 深度探测

**L2 探测成本控制**：
- 每次 L2 探测只用 1 个模型、1 个 prompt、max_tokens=50
- 所有渠道的 L2 探测串行执行，间隔 >= 5 秒
- 每天 L2 探测总次数上限：`渠道数 × 12`（每 2 小时最多一次）
- 白嫖渠道的 L2 探测频率自动降低：如果连续 3 次 L2 探测失败，间隔翻倍（2h→4h→8h→24h）

### 6.5 白嫖池快速衰减机制

白嫖/临时池渠道的可用性是移动靶，不能靠滑动窗口慢慢反应。需要独立的衰减机制：

```go
// backend-go/internal/autopilot/fast_decay.go

// FastDecayScore 实时衰减评分，用于白嫖/临时池渠道
type FastDecayScore struct {
    ChannelID       int
    BaseScore       float64   // 基于 ChannelProfile 的基础分
    DecayFactor     float64   // 衰减系数 0.0-1.0
    LastUpdate      time.Time
    ConsecutiveFail int
}

// OnSuccess 请求成功时
func (s *FastDecayScore) OnSuccess() {
    s.DecayFactor = math.Min(1.0, s.DecayFactor+0.15) // 快速回升 +15%
    s.ConsecutiveFail = 0
}

// OnFailure 请求失败时
func (s *FastDecayScore) OnFailure() {
    s.ConsecutiveFail++
    // 指数衰减：连续失败越多，衰减越快
    // 1次失败: ×0.85, 2次: ×0.72, 3次: ×0.61, 5次: ×0.44, 10次: ×0.20
    s.DecayFactor *= math.Pow(0.85, float64(s.ConsecutiveFail))
}

// OnStreamBreak 断流时（比普通失败更严重）
func (s *FastDecayScore) OnStreamBreak() {
    s.ConsecutiveFail++
    s.DecayFactor *= math.Pow(0.70, float64(s.ConsecutiveFail)) // 更激进衰减
}

// EffectiveScore = BaseScore × DecayFactor
func (s *FastDecayScore) EffectiveScore() float64 {
    return s.BaseScore * s.DecayFactor
}
```

**触发条件**：`costTier=free|cheap` 或 `poolTag=temp` 的渠道自动启用 FastDecay。

**调度效果**：
- 一个白嫖渠道连续断流 3 次，EffectiveScore 从 1.0 降到 0.61，自动让位给下一个渠道
- 连续失败 10 次，降到 0.20，几乎不会被选中
- 成功一次立即回升 15%，避免"一朝被蛇咬"的永久惩罚
- 这比滑动窗口快得多：滑动窗口需要窗口滚动才反映变化，FastDecay 是请求级即时反应

## 7. API 设计

### 7.1 新增 API 端点

#### 渠道画像

```text
GET  /api/{kind}/channels/profiles          → 获取所有渠道画像
GET  /api/{kind}/channels/{id}/profile      → 获取单个渠道画像
POST /api/{kind}/channels/{id}/profile/refresh → 手动刷新画像
```

#### 模型画像

```text
GET  /api/{kind}/channels/{id}/model-profiles → 获取渠道下所有模型画像
```

#### 健康中心

```text
GET  /api/health-center/overview            → 全局健康概览（跨所有 kind）
GET  /api/health-center/channels            → 渠道健康列表（支持过滤/排序）
POST /api/health-center/batch-action        → 批量操作（复测/暂停/删除）
POST /api/health-center/channels/{id}/probe → 手动深度探测
```

#### 自动托管

```text
POST /api/{kind}/channels/auto-add          → 自动添加渠道（仅需 URL+Key）
POST /api/{kind}/channels/{id}/auto-discover → 重新触发自动发现
GET  /api/{kind}/channels/{id}/auto-status   → 自动托管状态
```

#### 智能路由（诊断用）

```text
POST /api/smart-routing/diagnose            → 智能路由诊断（dry-run）
GET  /api/smart-routing/config              → 获取自动路由配置
PUT  /api/smart-routing/config              → 更新自动路由配置
```

### 7.2 与现有 API 的关系

| 现有端点 | 变更 |
|---------|------|
| `POST /channel-discovery` | 不变，autopilot 内部复用其逻辑 |
| `POST /{kind}/channels` | 新增 `autoManaged` 字段，为 true 时自动触发 Discovery |
| `POST /{kind}/channels/{id}/capability-test` | 不变，autopilot 结果写入 ModelProfile |
| `POST /{kind}/channels/scheduler/diagnose` | 增加智能路由 trace 输出 |
| `GET /{kind}/channels/dashboard` | 增加 `healthState`、`qualityTier` 字段 |

### 7.3 WebSocket 推送

新增 `ws://api/autopilot/events` 通道，推送：

```json
{
  "type": "profile_updated",
  "channelId": 5,
  "channelKind": "messages",
  "healthState": "dead",
  "suggestedAction": "delete",
  "evidence": ["5/5 keys returned 401"]
}
```

事件类型：`profile_updated` / `health_changed` / `discovery_completed` / `auto_mapping_applied`

---

## 8. 前端设计

### 8.1 健康中心视图

新增 `HealthCenter.vue` 页面，作为渠道管理的高级视图。

#### 布局

```text
┌──────────────────────────────────────────────────────────┐
│ 渠道健康中心                    [批量复测] [批量暂停] [筛选] │
├──────────────────────────────────────────────────────────┤
│ ┌─ 统计卡片 ────────────────────────────────────────────┐│
│ │ 🟢 12 健康  🟡 3 降级  🟠 5 限流  🔴 4 死亡  ⚪ 2 新  ││
│ └───────────────────────────────────────────────────────┘│
│                                                          │
│ ┌─ 分组标签 ────────────────────────────────────────────┐│
│ │ [建议清理(4)] [需要修复(3)] [限流恢复中(5)]           ││
│ │ [质量较差(2)] [观察池(2)] [可用渠道(12)]              ││
│ └───────────────────────────────────────────────────────┘│
│                                                          │
│ ┌─ 渠道表格 ────────────────────────────────────────────┐│
│ │ 状态 │ 渠道 │ 协议 │ 模型数 │ 最后成功 │ 成功率 │p95 │ │
│ │      │      │      │        │          │        │延迟│ │
│ │ 🔴   │ xxx  │ chat │ 3/5    │ 72h前    │ 2%     │ -  │ │
│ │ 🟡   │ yyy  │ msgs │ 7/7    │ 5m前     │ 78%    │ 3s │ │
│ └───────────────────────────────────────────────────────┘│
│                                                          │
│ ┌─ 渠道详情侧栏（点击展开）─────────────────────────────┐│
│ │ 健康状态：🔴 Dead (confidence: 96%)                    ││
│ │ 死亡类型：硬死 - 全部 Key 认证失败                     ││
│ │ 证据：                                                 ││
│ │   • 5/5 keys returned 401                              ││
│ │   • capability-test failed 3 consecutive times         ││
│ │   • no successful request in 72h                       ││
│ │ 建议操作：[替换 Key] [删除渠道] [标记观察]             ││
│ │ 画像：quality=high, stability=unstable, speed=-        ││
│ │ 可用模型：gpt-5.4(✓), gpt-5.5(✗), ...                ││
│ └───────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────┘
```

#### 标签系统

每个渠道显示标签 chip：

| 标签 | 颜色 | 条件 |
|------|------|------|
| 高智商稳定 | 蓝 | qualityTier=high\|premium + stabilityTier=stable |
| 白嫖池 | 绿 | costTier=free |
| 临时池 | 橙 | 画像来源=auto_probe 且 confidence < 0.7 |
| 仅子代理 | 灰 | qualityTier=low + costTier=free\|cheap |
| 可识图 | 紫 | supportsVision=true |
| 长上下文 | 青 | supportsLongCtx=true |
| 全部 Key 失效 | 红 | evidence 含 "all keys failed" |
| 限流中 | 黄 | healthState=limited |
| 疑似配置错 | 紫 | healthState=misconfigured |

### 8.2 渠道卡片增强

在现有 `ChannelOrchestration.vue` 的每行中增加：

1. **健康状态 badge**：替换现有简单状态，使用 HealthState 六态 badge
2. **质量/稳定性/速度/成本标签**：在渠道名下方显示小 chip
3. **自动托管图标**：`autoManaged` 渠道显示机器人图标，hover 提示「自动托管中」
4. **一键操作**：死渠道显示「清理」快捷按钮

### 8.3 添加渠道流程简化

当用户选择「快速添加」模式：

```text
┌─ 快速添加渠道 ──────────────────────────┐
│                                          │
│ 名称：[________________] (可选，自动生成) │
│ 地址：[https://xxx/v1________________]   │
│ Key ：[sk-__________________________]    │
│                                          │
│ [x] 自动托管（推荐）                     │
│     系统将自动探测协议、发现模型、        │
│     生成映射、持续监控健康               │
│                                          │
│          [添加并探测]                    │
└──────────────────────────────────────────┘
```

点击「添加并探测」后：
1. 创建渠道（status=unknown）
2. 自动触发 Discovery + 能力测试
3. 显示进度条和实时探测日志
4. 完成后自动写入 modelMapping / supportedModels / 兼容开关
5. 生成初始 ChannelProfile

---

## 9. 配置设计

### 9.1 全局智能路由配置

在 `config.json` 新增顶层字段：

```json
{
  "smartRouting": {
    "enabled": true,
    "mode": "auto",

    "defaultAutoManaged": true,
    "autoDiscoveryOnAdd": true,

    "subagentUseCheapPool": true,
    "unknownChannelPolicy": "observe",
    "premiumFallbackForSubagent": false,
    "protectVisionChannels": true,
    "protectLongContextChannels": true,

    "healthCheck": {
      "enabled": true,
      "passiveSignalsOnly": false,
      "l2ProbeEnabled": true,
      "l2ProbeIntervalMinutes": 120,
      "l2ProbeMaxPerDay": 12,
      "deadProbeIntervalHours": 6,
      "deadConfidenceThreshold": 0.80,
      "autoExcludeDead": true
    },

    "fastDecay": {
      "enabled": true,
      "applyToCostTiers": ["free", "cheap"],
      "applyToPoolTags": ["temp"],
      "recoveryRate": 0.15,
      "decayBase": 0.85,
      "streamBreakDecayBase": 0.70
    },

    "modelMapping": {
      "autoResolve": true,
      "capabilityFloorEnabled": true,
      "echoMappedModel": true,
      "forbidChainMapping": true
    },

    "taskStrategies": {
      "supervisor": {
        "preferQuality": ["high", "premium"],
        "requireStability": ["stable", "normal"],
        "excludeTags": ["unstable", "free"]
      },
      "worker": {
        "preferCost": ["free", "cheap"],
        "preferSpeed": ["fast"],
        "excludeQuality": ["premium"]
      }
    }
  }
}
```

### 9.2 渠道级配置

现有 `UpstreamConfig` 新增字段：

```go
type UpstreamConfig struct {
    // ... 现有字段 ...

    // ── 自动托管 ──
    AutoManaged       bool   `json:"autoManaged,omitempty"`       // 启用自动托管
    AutoManagedAt     *time.Time `json:"autoManagedAt,omitempty"` // 开始托管时间
    CostHint          string `json:"costHint,omitempty"`          // 用户成本提示：free/cheap/normal/expensive
    QualityHint       string `json:"qualityHint,omitempty"`       // 用户质量提示（override 自动推导）
    PoolTag           string `json:"poolTag,omitempty"`           // 池标签：temp/regular/premium
    RoutingPriority   string `json:"routingPriority,omitempty"`   // 路由优先级 hint
}
```

**用户覆盖规则**：
- 用户手动设置的字段（QualityHint/CostHint/PoolTag）优先级高于自动推导
- 但自动推导的运行时指标（健康状态/熔断）始终生效，不受 override 影响
- 用户可通过 `autoManaged: false` 退出自动托管，回到手动模式

---

## 10. 分阶段落地计划

### Phase 1：自动画像 + 健康诊断（MVP）

**目标**：用户能看到渠道健康状态，系统自动推导画像。白嫖池有即时衰减保护。

**范围**：
- [ ] ChannelProfile / ModelProfile 数据模型 + SQLite 存储
- [ ] Profiler 画像推导（基于现有 MetricsManager + 模型注册表，L1 被动信号为主）
- [ ] HealthAnalyzer 健康诊断（被动优先：L1 为主，L2 仅在数据不足时触发）
- [ ] FastDecay 快速衰减（白嫖/临时池渠道的请求级即时评分）
- [ ] 健康中心 API（`/api/health-center/*`）
- [ ] 前端健康中心视图
- [ ] 渠道卡片健康 badge 增强
- [ ] 标签系统（白嫖池/临时池/高智商稳定等）

**不做的事**：
- 不修改调度器核心链路
- 不自动写入 modelMapping
- 不做模型自动映射
- 不做 L3 深度探测自动触发

**预估工期**：2-3 周

### Phase 2：自动发现 + 智能调度

**目标**：添加渠道时自动配置，调度时自动选择。

**范围**：
- [ ] `autoManaged` 字段 + 快速添加流程
- [ ] Discovery 自动触发（复用现有 channel_discovery 逻辑）
- [ ] 自动写入 modelMapping / supportedModels / 兼容开关
- [ ] SmartRouter + CandidateFilter 注入
- [ ] TaskClassifier + 五种任务策略
- [ ] 智能路由诊断 API（dry-run）
- [ ] 前端自动托管指示器 + 快速添加 UI

**预估工期**：3-4 周

### Phase 3：动态画像 + 自愈

**目标**：画像实时更新，渠道自动恢复。

**范围**：
- [ ] 运行时指标驱动画像实时更新
- [ ] 模型自动映射（ModelResolver）
- [ ] 自动恢复探测（limited/dead → healthy）
- [ ] 晋升/降级机制（连续成功→升级，连续失败→降级）
- [ ] WebSocket 推送画像变更事件
- [ ] 前端画像变更历史/时间线

**预估工期**：2-3 周

### Phase 4：高级特性

**范围**：
- [ ] 多维度标签系统扩展（用户自定义标签）
- [ ] A/B 测试：同一请求走不同渠道对比
- [ ] 成本优化：自动选择最便宜的满足条件的渠道
- [ ] 批量渠道管理（导入/导出/模板）
- [ ] 渠道推荐：根据使用模式推荐新渠道

---

## 11. 关键代码锚点

### 11.1 需要扩展的文件

| 文件 | 行号 | 扩展内容 |
|------|------|---------|
| `scheduler/select.go` | `SelectChannelWithOptions` | 在 CandidateFilter 回调点注入 SmartRouter |
| `config/config.go` | `UpstreamConfig` | 新增 AutoManaged/CostHint/QualityHint/PoolTag 字段 |
| `handlers/common/multi_channel_failover.go` | `HandleMultiChannelFailover*` | 构建 RequestProfile，传入 SmartRouter filter |
| `metrics/channel_metrics.go` | `MetricsManager` | 新增画像相关查询方法 |
| `handlers/channel_discovery.go` | 全文 | Profiler 复用其探测逻辑 |
| `handlers/capability_test_runner.go` | `runCapabilityTestJob` | 测试结果写入 ModelProfile |

### 11.2 需要新增的文件

```text
backend-go/internal/autopilot/
├── profile.go              # ChannelProfile / ModelProfile 类型
├── request_profile.go      # RequestProfile + TaskClassifier
├── profile_store.go        # SQLite 持久化 + 内存缓存
├── profiler.go             # 画像推导逻辑（L1 被动信号为主）
├── health_analyzer.go      # 健康诊断逻辑（被动优先 + L2 探测补充）
├── fast_decay.go           # 白嫖/临时池快速衰减评分
├── smart_router.go         # SmartRouter + CandidateFilter 构建
├── model_resolver.go       # 模型自动映射 + CapabilityFloor 约束
└── handlers.go             # API handlers

backend-go/internal/autopilot/
├── autopilot_test.go       # 画像推导测试
├── health_analyzer_test.go # 健康诊断测试
├── fast_decay_test.go      # 快速衰减测试
└── smart_router_test.go    # 路由策略测试

frontend/src/components/
├── HealthCenter.vue        # 健康中心主视图
├── HealthCenterStats.vue   # 统计卡片
├── HealthChannelTable.vue  # 渠道健康表格
├── HealthChannelDetail.vue # 渠道详情侧栏
├── QuickAddChannel.vue     # 快速添加渠道
└── ChannelHealthBadge.vue  # 健康状态 badge（增强现有）
```

### 11.3 与现有代码的接口契约

| 接口 | 方向 | 说明 |
|------|------|------|
| `CandidateFilterFunc` | autopilot → scheduler | SmartRouter 实现此接口注入调度链 |
| `MetricsManager.GetChannelAggregatedMetrics` | autopilot ← metrics | 画像推导读取运行时指标 |
| `config.ResolveUpstreamCapability` | autopilot ← config | 画像推导读取模型能力 |
| `config.ResolveAgentModelProfile` | autopilot ← config | RequestProfile 推导质量需求 |
| `channelDiscovery.*` | autopilot ← handlers | 复用探测逻辑（需抽取为可复用函数） |
| `PersistenceStore` | autopilot → metrics | 新表复用同一 SQLite 连接 |

---

## 12. 风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| ~~自动模型映射语义降级~~ | ~~用户以为用 opus 实际用 haiku~~ | **已缓解**：CapabilityFloor 硬约束 + 不满足则跳过渠道而非降级映射 + response header 回显真实模型 |
| ~~健康诊断烧 quota~~ | ~~30-40 渠道主动探测成本高~~ | **已缓解**：L1 被动优先（零成本），L2 仅在数据不足时触发，每天总次数上限 `渠道数×12` |
| ~~白嫖池状态抖动~~ | ~~渠道反复断流导致调度震荡~~ | **已缓解**：FastDecay 请求级即时衰减 + 成功快速回升 + 断流比普通失败衰减更快 |
| ~~能力虚标误判~~ | ~~系统误关 vision/tool 标签~~ | **已缓解**：只做硬失败检测（HTTP 错误/解析失败），软质量问题留给人工 override |
| SmartRouter 增加调度延迟 | 请求耗时增加 | 画像缓存在内存，CandidateFilter 只做内存操作（< 1ms） |
| 与现有 Priority/Promotion 系统冲突 | 调度行为不一致 | SmartRouter 的 CandidateFilter 在 Priority 排序之后执行，只做"重排候选"而非"改变优先级" |
| Phase 1 无智能调度时画像价值不明显 | 用户感知弱 | 健康中心 + 标签系统本身就有独立价值；dry-run 诊断接口提前展示"如果启用自动调度会选谁" |
| 自动 modelMapping 覆盖用户手动配置 | 用户设置被意外覆盖 | 显式 modelMapping 始终优先，不经过下界检查；autoManaged=false 时完全不触发自动映射 |
| 被动信号在低流量渠道不足 | 新渠道/冷渠道无法诊断 | 低流量时自动降级为 L2 探测，探测频率随请求量动态调整 |
