# CCX Desktop

CCX Desktop 是一个桌面后台应用，提供本地 CCX 网关管理、Agent 配置、渠道模板和日志监控。

## 下载

Windows 用户推荐优先使用 Microsoft Store 版本（上架后提供），由 Store 负责签名与自动更新。开发者或无法使用 Store 的环境，可继续从 [GitHub Releases](https://github.com/BenedictKing/ccx/releases) 下载对应平台安装包：

| 平台 | 文件名格式 |
|------|-----------|
| macOS (Apple Silicon) | `CCX-Desktop-{version}-darwin-arm64.dmg` |
| macOS (Intel) | `CCX-Desktop-{version}-darwin-amd64.dmg` |
| Windows (GitHub) | `CCX-Desktop-{version}-windows-{arch}-setup.exe` |
| Windows (Store/MSIX) | `CCX-Desktop-{version}-windows-{arch}-store.msix` |
| Linux | `CCX-Desktop-{version}-linux-amd64.AppImage` |

`*-store.msix` 主要用于 Microsoft Store 提交与验证。公开安装优先使用 Microsoft Store；直接 sideload `.msix` 仍可能需要受信任签名环境。

Release 页面同时附带 `.sha256` 校验文件，可验证下载完整性：

```bash
shasum -a 256 -c CCX-Desktop-*.sha256
```

## 安装

### macOS

1. 双击 `.dmg` 文件
2. 将 `CCX Desktop` 拖入 `Applications` 文件夹
3. 首次打开时，macOS 可能提示"无法验证开发者"，前往 **系统设置 → 隐私与安全性** 点击"仍要打开"

### Windows

推荐渠道：

1. Microsoft Store 版本：从 Store 安装，签名与更新由 Microsoft Store 处理。
2. GitHub 版本：双击 `-setup.exe` 安装程序并按提示完成安装。

如果 GitHub 版本触发 SmartScreen 警告，点击"更多信息 → 仍要运行"。如果不希望处理签名/SmartScreen 提示，优先使用 Store 版本。

### Linux (AppImage)

```bash
chmod +x CCX-Desktop-*.AppImage
./CCX-Desktop-*.AppImage
```

AppImage 支持应用内自动更新。如果通过 deb/rpm 安装，需通过系统包管理器更新。

## 首次使用

### 1. 启动网关

打开 CCX Desktop 后，在 **Gateway Monitor** 页面点击 **启动服务**。

首次启动前需确保后端已构建。如果提示"二进制文件未找到"：

```bash
cd backend-go && make build
```

### 2. 配置渠道

进入 **Channel Center**，选择预设渠道（如 DeepSeek、MiMo）一键添加，或使用通用 OpenAI-compatible 模板自定义配置。

添加渠道后需配置 API Key。

### 3. 配置 Agent

进入 **Agent Config**，选择 Claude Code 或 Codex 的目标 Provider：

- **CCX**：通过本地网关代理（推荐，支持多渠道故障转移）
- **DeepSeek / MiMo**：直连上游

点击 **应用配置** 写入 Agent 的配置文件。

### 4. 验证

启动服务后，**Gateway Monitor** 应显示：

- 状态指示灯变绿
- Metrics 网关显示端口和运行时长
- Log Viewer 无错误输出

## 自动更新

GitHub 安装包版内置自动更新：

- 启动 5 秒后自动检查一次
- 之后每 30 分钟检查一次
- 也可在侧边栏底部点击版本号手动检查

Microsoft Store 版不走 GitHub Releases 自动更新，侧边栏和托盘会提示由 Store 更新。

GitHub 版更新流程：

1. 发现新版本 → 弹出更新对话框
2. 下载安装包（带进度条）
3. SHA256 校验
4. macOS：打开 DMG 手动替换
5. Windows：自动启动安装程序
6. Linux (AppImage)：自动替换并重启

## 系统托盘

CCX Desktop 关闭窗口后会最小化到系统托盘，托盘菜单提供：

- 查看运行状态（端口、PID）
- 启动 / 停止 / 重启服务
- 打开 Web UI
- 复制 Web UI 地址和 PROXY_ACCESS_KEY
- 开机自启开关
- 检查更新（Store 版本显示由 Microsoft Store 更新）

## 环境配置

进入 **Environment Params** 编辑 `.env` 文件，常用配置：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PORT` | 3000 | 网关端口 |
| `PROXY_ACCESS_KEY` | - | 代理访问密钥 |
| `ADMIN_ACCESS_KEY` | - | 管理密钥 |
| `LOG_LEVEL` | info | 日志级别 |

修改后需重启服务生效。
