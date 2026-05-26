# 验证发布产物

CCX 的正式 Release 使用 [Sigstore](https://www.sigstore.dev/) keyless signing 对 checksums 清单签名。通过验证签名，可以确认发布产物确实由 `BenedictKing/ccx` 仓库的 GitHub Actions 工作流构建，而非被篡改或来自非官方渠道。

## 工作原理

1. CI 构建时，GitHub Actions 通过 OIDC 协议向 Sigstore 的 Fulcio CA 申请**短期签名证书**（有效期约 10 分钟），证书中绑定了 GitHub Actions 身份信息。
2. `cosign` 使用该证书对 `checksums.txt` 文件签名，生成 Sigstore bundle（`.sigstore.json`）。
3. 签名记录写入 Rekor 公开透明度日志，任何人都可审计。
4. 用户验证时，cosign 从 bundle 中恢复证书信息，校验签名是否来自预期的 GitHub Actions 身份。

整个过程**不需要任何密钥管理**，也不需要项目维护者存储或轮换密钥。

## 安装 cosign

macOS：

```bash
brew install cosign
```

Linux（二进制）：

```bash
# 参见 https://docs.sigstore.dev/cosign/installation/
COSIGN_VERSION="v3.8.0"
curl -LO "https://github.com/sigstore/cosign/releases/download/${COSIGN_VERSION}/cosign-linux-amd64"
chmod +x cosign-linux-amd64
sudo mv cosign-linux-amd64 /usr/local/bin/cosign
```

Windows：

```powershell
choco install cosign
```

## 验证步骤

### 1. 下载 checksums 和签名文件

从 [GitHub Releases](https://github.com/BenedictKing/ccx/releases) 下载：

- `checksums.txt` — 全平台 SHA256 清单
- `checksums.txt.sigstore.json` — 对应的 Sigstore 签名 bundle

如果只需要验证单个平台，也可使用 `checksums-{platform}.txt` 和 `checksums-{platform}.txt.sigstore.json`：

| 平台 | 清单文件 | 签名 bundle |
|------|---------|-------------|
| macOS | `checksums-macos.txt` | `checksums-macos.txt.sigstore.json` |
| Windows | `checksums-windows.txt` | `checksums-windows.txt.sigstore.json` |
| Linux | `checksums-linux.txt` | `checksums-linux.txt.sigstore.json` |

### 2. 验证 checksums 清单签名

```bash
VERSION=v2.7.12  # 替换为要验证的版本

cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity "https://github.com/BenedictKing/ccx/.github/workflows/release.yml@refs/tags/${VERSION}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  checksums.txt
```

验证成功时 cosign 输出 `Verified checksums.txt` 并返回 exit code 0。

如果不确定版本号，可使用正则匹配：

```bash
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp "^https://github\.com/BenedictKing/ccx/\.github/workflows/release\.yml@refs/tags/v.*$" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  checksums.txt
```

### 3. 验证具体文件的 SHA256

签名验证通过后，用 checksums 清单校验下载的具体文件：

Linux：

```bash
sha256sum -c checksums.txt --ignore-missing
```

macOS（部分版本 `shasum` 不支持 `--ignore-missing`，可手动比对）：

```bash
shasum -a 256 ccx-darwin-arm64
# 将输出与 checksums.txt 中对应行比较
```

## 验证失败的常见原因

| 现象 | 原因 |
|------|------|
| `error: verifying bundle: mismatched identities` | 签名不是由本项目 CI 产生，或使用了错误的版本号 |
| `error: matching certificate identity failed` | `--certificate-identity` 与实际签名身份不匹配，检查版本号是否正确 |
| `error: tlog entry is not trusted` | Rekor 透明度日志验证失败，可能是 bundle 文件损坏 |
| `error: no matching signatures` | `checksums.txt` 文件内容与签名时不一致，可能下载过程中损坏 |
| 签名验证通过但 SHA256 不匹配 | 下载的二进制文件本身损坏，重新下载即可 |

## 产物说明

| 文件 | 内容 |
|------|------|
| `checksums.txt` | 全平台所有产物的 SHA256 哈希列表（`sha256sum` 格式） |
| `checksums.txt.sigstore.json` | Sigstore bundle：包含签名、OIDC 证书、Rekor 透明度日志条目 |
| `checksums-{platform}.txt` | 单平台 SHA256 清单 |
| `checksums-{platform}.txt.sigstore.json` | 单平台 Sigstore bundle |
| `{artifact}.sha256` | 单个产物的 SHA256（仅含 hex hash，供 updater 自动校验） |
| `CCX-Desktop-*-windows-*-store.msix` | Windows Store/MSIX 提交与验证包，公开安装优先使用 Microsoft Store |

## 了解更多

- [Sigstore 官网](https://www.sigstore.dev/)
- [cosign 文档](https://docs.sigstore.dev/cosign/)
- [Rekor 透明度日志](https://docs.sigstore.dev/logging/overview/)
- [SLSA 供应链安全框架](https://slsa.dev/)
