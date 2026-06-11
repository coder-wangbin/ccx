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

## [x] OpenRouter 免费路由工具调用失败

使用 OpenRouter 的免费路由（free routing）时，工具调用（tool call）会报失败。需要排查 OpenRouter 免费层对 tool_use 请求的处理差异，确认是否为上游限制或协议转换问题，并给出相应修复或降级提示。

**排查结论：上游平台限制，非 CCX Bug。**

OpenRouter `:free` 变体模型被路由到受限 provider 池，这些 provider 频繁不支持原生 API 级别的 tool calling / tool_choice。请求携带 `tools` 参数时，路由器按 tool-use 支持能力过滤 endpoint，免费池中往往无符合条件的 endpoint，返回 404：`No endpoints found that support the provided tool_choice value.`。CCX 协议转换链路（Claude ↔ OpenAI ↔ Responses ↔ Gemini）完整且测试覆盖充分，不存在转换 Bug。

参考：goose#3054（确认是 OpenRouter 限制而非 Bug，已关闭）、claude-task-master#696（free endpoint 不暴露 tool_use 能力）。

**可选改进（未实施）：**
1. 用户教育：渠道预设描述标注免费模型 tool_use 限制
2. 错误识别与降级提示：识别 OpenRouter tool_use 404 错误，返回友好中文提示
3. Prompt-based 降级回退（不建议：实现复杂且可靠性差）

## [x] 火山 coding plan 模型列表与功能 Bug (#204)

火山引擎（Volcano/Ark）的 coding plan 渠道一直有问题：模型列表不正确，存在 bug。需要排查火山 coding plan 渠道的模型映射、预设配置与上游 API 的对齐情况。

**关键提交：**
- `5f470512` fix(preset): 火山方舟 Coding Plan 渠道补充模型映射与特性配置 (#204)
- `8abacd11` fix(preset): 千帆 Coding Plan 渠道补充模型映射与特性配置

**关键变更：**
- `desktop/internal/channelpreset/preset.go`
- `desktop/internal/channelpreset/preset_test.go`

## [x] 磁铁图标背景不透明 + 黄色光圈

黄色光圈是图标边缘半透明像素在 Windows 磁贴渲染时产生的 Bug，而非设计元素。修正 SVG 源图后背景已透明、光圈已消除。

**关键变更：**
- 移除 SVG 不透明背景矩形 (`terminal-bg`) 和阴影滤镜 (`terminal-shadow`)，消除边缘黄色光圈渲染 Bug
- 终端面板渐变改为完全不透明（移除 `stop-opacity`），避免半透明渗色
- 移除 `terminal-glow` 模糊滤镜，防止边缘半透明像素产生光圈
- 从修改后 SVG 重新生成 `appicon.png` / `appicon-windows.png`（透明背景、四角 alpha=0、无黄色像素）
- 补齐 Windows Store/磁贴完整静态资产：`Square30x30Logo` 到 `Square310x310Logo`、`StoreLogo`、`Wide310x150Logo`、`SplashScreen`
- MSIX `package.ps1` 改为优先复制静态磁贴资产，回退到动态生成；`sourceIcon` 改用 `appicon-windows.png`
- 重新生成 `.icns` 和 `.ico`

## [ ] 桌面端管理面板显示用量信息 (#199)

在桌面端管理面板上直接显示使用情况（如 WebUI 中的用量图表），无需再打开网页版面板。

## [x] Codex remote compaction v2 在 DeepSeek think 响应下失败 (#179)

当 Codex 对话触发远程 compaction v2 时，CCX 将 `<think>...</think>` 内容无条件拆分为独立的 reasoning output item，导致 compaction 响应产生两个 output items（reasoning + message），而 Codex compaction v2 解析器期望恰好一个 compaction output item，报错退出。

修复方向：检测 compaction 响应时跳过 think tag 拆分，将 reasoning 合并回 message output item；或增加渠道/模型级 toggle 控制 think 内容是否拆分为独立 reasoning item。

相关 Issue：#97（reasoning_tokens 缺失）、#83/#82（MiniMax think tag 拆分问题）。

**已修复**（2026-06-11）：
- 非流式 compact：`normalizeCompactOutput` 过滤 reasoning items，仅保留 message item
- 原生 compact：`normalizeCompactResponseBody` 规范化响应体
- 流式 compact：新增 `shouldSkipCompactStreamEvent` 过滤所有 reasoning 相关 SSE 事件
- 测试覆盖：新增 `TestShouldSkipCompactStreamEvent` 验证事件过滤逻辑