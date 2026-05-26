# 常见问题

## 启动失败

### 二进制文件未找到

**症状**：点击"启动服务"后显示"未找到 CCX 二进制"。

**解决**：

```bash
cd backend-go && make build
```

Desktop 会在以下路径查找二进制：

- `backend-go/ccx-go`（项目构建产物）
- 用户数据目录内的 `ccx-go`

### 端口冲突

**症状**：启动后健康检查超时，错误包含"connection refused"或"端口冲突"。

**解决**：

1. 检查端口占用：
   ```bash
   # macOS / Linux
   lsof -i :3000

   # Windows
   netstat -ano | findstr :3000
   ```
2. 停止占用进程，或修改 `.env` 中 `PORT` 为其他端口

### 健康检查超时

**症状**：启动进程已创建，但 15 秒内 `/health` 未返回 healthy。

**可能原因**：

- `.env` 配置有误
- 上游渠道配置错误导致启动报错
- 首次启动需较长初始化时间

**解决**：查看 Log Viewer 中的错误日志，按提示修正配置。

### 权限不足

**症状**：错误包含"permission denied"。

**解决**：

```bash
# macOS / Linux：赋予二进制执行权限
chmod +x backend-go/ccx-go

# Windows：以管理员身份运行 Desktop
```

## 自动更新问题

### macOS "无法验证开发者"

前往 **系统设置 → 隐私与安全性**，找到被阻止的应用，点击"仍要打开"。

### Linux AppImage 无法更新

仅 AppImage 格式支持应用内自动更新。如果使用 deb/rpm 安装：

```bash
# deb
sudo apt update && sudo apt upgrade ccx-desktop

# rpm
sudo dnf update ccx-desktop
```

### 更新下载失败

GitHub 安装包版会访问 GitHub Releases 检查和下载更新。检查网络连接；如果使用代理，确保 GitHub Releases 可访问。可手动从 [Releases 页面](https://github.com/BenedictKing/ccx/releases) 下载安装。

Microsoft Store 版由 Store 负责更新，不使用应用内 GitHub 更新器。如果 Store 更新失败，请在 Microsoft Store 的库页面重试或重新安装。

## 配置问题

### Agent 配置应用后无效

1. 确认网关已启动且端口正确
2. 检查 Agent 配置文件路径是否正确（Agent Config 页面会显示）
3. 重启对应的 Agent 工具（如 Claude Code、Codex）

### 渠道添加后请求失败

1. 检查 API Key 是否正确填写
2. 确认 Base URL 可访问
3. 查看 Log Viewer 中的具体错误信息
4. 使用 Web UI 的渠道管理页面查看渠道状态

## 其他

### 窗口位置/大小不恢复

CCX Desktop 会自动保存窗口状态到数据目录。如果状态文件损坏：

1. 关闭 Desktop
2. 删除数据目录中的 `window-state.json`
3. 重新打开

### 开机自启不生效

- macOS：检查 **系统设置 → 通用 → 登录项**
- Windows：检查 **任务管理器 → 启动** 选项卡
- Linux：检查桌面环境的自启动设置

### Web UI 无法访问

1. 确认网关已启动（Gateway Monitor 状态为绿色）
2. 检查 `ENABLE_WEB_UI` 是否为 `true`（Environment Params）
3. 尝试在浏览器直接访问 `http://localhost:3000`
