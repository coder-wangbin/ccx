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

## [ ] 火山 coding plan 模型列表与功能 Bug (#204)

火山引擎（Volcano/Ark）的 coding plan 渠道一直有问题：模型列表不正确，存在 bug。需要排查火山 coding plan 渠道的模型映射、预设配置与上游 API 的对齐情况。

## [ ] 磁铁图标背景不透明 + 黄色光圈

最新版桌面 App 磁铁图标背景仍然不透明。商店要求的磁铁造型图标本身没问题，但有一个黄色光圈（ring）需要保留，而图标整体背景应改为透明，避免在深色/浅色系统托盘或任务栏上出现突兀的色块。需要修正 `desktop/design/icons/appicon-selected.svg` 及生成的 PNG/ICO/ICNS 资源。