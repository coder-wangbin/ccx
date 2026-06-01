# 常见问题

本页汇总 CCX Desktop 的常见问题与排障思路。

首次上手或需要完整操作说明时，请先参考 [CCX Desktop](./index)。

## 遇到问题先检查

### 关闭全局代理和 TUN

**这是最常见的排障第一步。**

全局代理、TUN 模式（如 Clash TUN、Surge 增强模式、v2rayN TUN 等）会劫持本机所有网络流量，导致 CCX 与上游之间的请求被错误转发或拦截。

**症状**：

- 请求超时或连接被重置
- 返回非预期的错误页面或 HTML 内容
- 上游渠道显示 unhealthy 但直接 curl 上游正常

**解决**：

1. 关闭系统代理或切换为规则模式（PAC / Rule），避免全局劫持。
2. 关闭 TUN 模式，或将 CCX 相关地址加入绕过列表。
3. 重新测试。

## 启动失败

### 二进制文件未找到

**症状**：点击“启动服务”后提示“未找到 CCX 二进制”。

**解决**：

1. 若本地开发，可先构建后端：

   ```bash
   cd backend-go && make build
   ```

2. 确认 Desktop 能找到 `ccx-go`。
3. 若修改了构建产物路径，重新打开 CCX Desktop 再试。

### 端口冲突

**症状**：启动后健康检查超时，错误包含 `connection refused` 或提示端口冲突。

**解决**：

1. 检查占用进程：

   ```bash
   # macOS / Linux
   lsof -i :3688

   # Windows
   netstat -ano | findstr :3688
   ```

2. 停止占用进程，或在 **Environment Params** 中修改 `PORT`，再重启服务。

### 健康检查超时

**症状**：进程已创建，但 `http://localhost:3688/health` 长时间未变为 healthy。

**可能原因**：

- `.env` 配置错误。
- 渠道配置异常。
- 首次启动初始化时间较长。

**解决**：在 **Gateway Monitor** 或 **Log Viewer** 中查看错误日志并逐步修正配置。

### 权限不足

**症状**：错误包含 `permission denied`。

**解决**：

```bash
# macOS / Linux
chmod +x backend-go/ccx-go

# Windows
以管理员身份运行 Desktop
```

## 密钥与访问

### Base URL 根路径与 `/v1` 的区别

- **Claude Code**：填写 `http://localhost:3688`。
- **Codex CLI / Codex App**：填写 `http://localhost:3688/v1`。
- **OpenCode**：填写 `http://localhost:3688/v1`。

这是常见错误来源，请严格按照客户端类型选择地址。

### 返回 401 Unauthorized

请按顺序检查：

1. 客户端中的 API Key 是否等于 CCX 的 `PROXY_ACCESS_KEY`。
2. Desktop 当前启动所用 `PROXY_ACCESS_KEY` 是否正确。
3. 是否存在旧的环境变量覆盖当前配置。
4. 是否误填了上游 API Key。

### 环境变量覆盖导致配置不生效

如果客户端 CLI 已经设置了以下环境变量，可能覆盖 Agent 配置：

- `ANTHROPIC_API_KEY`
- `ANTHROPIC_BASE_URL`
- `OPENAI_API_KEY`
- `OPENAI_BASE_URL`

可在终端中检查：

```bash
printenv ANTHROPIC_API_KEY
printenv ANTHROPIC_BASE_URL
printenv OPENAI_API_KEY
printenv OPENAI_BASE_URL
```

如果发现旧值，应清理或在对应终端会话中取消后再测试。

### Windows 下 localhost 无法访问

若客户端运行在 cmd、PowerShell、WSL 或 Docker 中，`localhost` 可能无法正确到达 CCX。

可改用 Windows 主机的局域网 IPv4 地址，例如：

```text
http://192.168.1.23:3688
```

并把客户端的 API Key 继续设为 `PROXY_ACCESS_KEY`。

## 配置问题

### Agent 配置应用后无效

1. 确认网关已启动且端口正确。
2. 确认目标配置文件路径与 Agent 工具一致。
3. 重启对应客户端，再发送测试消息。
4. 如仍无效，可删除旧环境变量后重试。

### 渠道添加后请求失败

1. 检查上游 API Key 是否正确。
2. 确认上游 Base URL 可访问。
3. 检查模型白名单或模型映射是否覆盖客户端请求的模型。
4. 在 **Log Viewer** 或 Web UI 中查看具体错误信息。

### 请求没有进入预期入口

不同客户端会请求不同路径：

- Claude Code：`/v1/messages`
- Codex CLI / Codex App：`/v1/responses`
- OpenCode：`/v1/chat/completions`

如果日志显示路径不符合预期，说明客户端未按目标方式接入 CCX，应重新检查 Agent 配置和客户端 Provider 设置。

### 快速验证各入口是否正常

当请求报错或不确定 CCX 通路是否正常时，用 curl 直接测试对应入口。

将下面的 `localhost:3688` 替换为你的实际地址和端口，`your-proxy-key` 替换为你的 `PROXY_ACCESS_KEY`。

**Messages（Claude Code）**

```bash
curl -sS -i "http://localhost:3688/v1/messages" \
  -H "x-api-key: your-proxy-key" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4-20250514",
    "max_tokens": 64,
    "messages": [{"role": "user", "content": "Reply with exactly: pong"}]
  }'
```

**Responses（Codex）**

```bash
curl -sS -i "http://localhost:3688/v1/responses" \
  -H "Authorization: Bearer your-proxy-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4",
    "input": "Reply with exactly: pong"
  }'
```

**Chat Completions（OpenCode）**

```bash
curl -sS -i "http://localhost:3688/v1/chat/completions" \
  -H "Authorization: Bearer your-proxy-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4",
    "messages": [{"role": "user", "content": "Reply with exactly: pong"}]
  }'
```

**判定标准**：

- 返回 `HTTP/1.1 200 OK` 且响应体有正常内容 → CCX 通路正常，问题在客户端配置。
- 返回 `401` → 检查 `PROXY_ACCESS_KEY` 是否匹配。
- 返回 `400` 且响应体提示 model not found → 检查渠道模型映射。
- 返回 `502` 或连接超时 → 上游不可达，检查渠道配置和网络。

## 自动更新问题

### macOS 提示“无法验证开发者”

前往 **系统设置 → 隐私与安全性**，找到被阻止的应用，点击“仍要打开”。

### Linux AppImage 无法更新

仅 AppImage 支持应用内自动更新。若通过 deb/rpm 安装：

```bash
# deb
sudo apt update && sudo apt upgrade ccx-desktop

# rpm
sudo dnf update ccx-desktop
```

### 更新下载失败

GitHub 安装包版会访问 GitHub Releases 检查并下载更新。

排查方式：

1. 检查网络连接。
2. 如使用代理，确认 GitHub Releases 可访问。
3. 必要时手动从 [Releases 页面](https://github.com/BenedictKing/ccx/releases) 下载安装。

Microsoft Store 版由 Store 负责更新。若 Store 更新失败，可在库页面重试或重新安装。

## 其他

### 窗口位置或大小不恢复

CCX Desktop 会保存窗口状态到数据目录。

如状态异常，可尝试：

1. 关闭 Desktop。
2. 删除数据目录中的 `window-state.json`。
3. 重新打开应用。

### 开机自启不生效

- macOS：检查 **系统设置 → 通用 → 登录项**。
- Windows：检查 **任务管理器 → 启动**。
- Linux：检查桌面环境的自启动设置。

### Web UI 无法访问

1. 确认网关已启动。
2. 确认 `ENABLE_WEB_UI` 为 `true`。
3. 尝试在浏览器中直接访问 `http://localhost:3688`。
