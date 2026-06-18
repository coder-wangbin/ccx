# 发布指南

本文档说明 CCX 的标准发布流程。

## 版本来源

项目使用根目录 `VERSION` 作为唯一版本源。

- 根版本文件：`VERSION`
- 构建注入位置：`backend-go/Makefile`
- 运行时版本变量：`backend-go/version.go`

后端构建时会读取根目录 `VERSION`，并通过 `-ldflags` 注入版本、构建时间和 Git 提交信息。

## 版本规范

项目遵循语义化版本（Semantic Versioning）：`MAJOR.MINOR.PATCH`。

- `MAJOR`：不兼容的 API 变更
- `MINOR`：向下兼容的功能新增
- `PATCH`：向下兼容的问题修复

## 发布流程

### 推荐方式：优先使用项目内技能

本仓库发布时，优先使用 `.claude/skills/` 下的两个技能：

1. **`version-bump`**（`.claude/skills/version-bump/`）
   - 用于升级根目录 `VERSION`
   - 自动更新或生成根目录 `CHANGELOG.md`
   - 可按约定创建 commit、tag，并推送到远程

2. **`github-release`**（`.claude/skills/github-release/`）
   - 用于根据 `CHANGELOG.md` 生成发布公告
   - 更新或发布 GitHub Release / Draft Release

推荐顺序：
1. 先使用 `version-bump` 完成版本号、changelog、commit/tag/push
2. 再使用 `github-release` 生成并发布 Release 公告

下面的手工步骤作为兜底流程：

### 步骤 1：准备工作

1. 确保本地 `main` 已同步最新代码。
2. 确认计划内功能和修复已经合并。
3. 发布前执行基础验证：

```bash
make build
cd "backend-go" && make test
cd "frontend" && bun run build
```

## 步骤 2：更新 `CHANGELOG.md`

根目录 `CHANGELOG.md` 是唯一持续维护的发布历史。

1. 在文件顶部新增版本标题，格式如下：

```md
## [vX.Y.Z] - YYYY-MM-DD
```

2. 沿用当前 changelog 的分类：

- `### Added`
- `### Changed`
- `### Fixed`
- `### Removed`
- `### Other`

3. 如需整理变更，可查看上一个 tag 之后的提交：

```bash
git log vX.Y.(Z-1)...HEAD --oneline
```

## 步骤 3：更新版本号

编辑根目录 `VERSION` 文件，将内容更新为新版本号：

```text
vX.Y.Z
```

不要更新 `frontend/package.json` 的 `version` 作为发布版本来源；前端包版本不是项目发布的权威版本号。

## 步骤 4：提交发布准备

将发布相关文件加入暂存并提交：

```bash
git add CHANGELOG.md VERSION
git commit -m "chore(release): prepare for vX.Y.Z"
```

## 步骤 5：创建并推送标签

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin main
git push origin vX.Y.Z
```

推送 tag 后，GitHub Actions 会自动触发多平台构建与 Docker 构建。

## 步骤 6：创建 GitHub Release

1. 进入 GitHub 的 Releases 页面。
2. 选择刚推送的 tag。
3. 将 `CHANGELOG.md` 中对应版本的内容整理到发布说明。
4. 发布 Release。

## 发布检查清单

- [ ] `VERSION` 已更新
- [ ] `CHANGELOG.md` 已补齐
- [ ] `make build` 通过
- [ ] `cd "backend-go" && make test` 通过
- [ ] `cd "frontend" && bun run build` 通过
- [ ] 已创建并推送 `vX.Y.Z` tag

## Release Signing

Releases use two signing layers:

- [Sigstore](https://www.sigstore.dev/) / cosign keyless signing signs the checksums manifests so users can verify Release asset integrity and CI origin.
- [SignPath Foundation](https://signpath.org/) signs the Windows GitHub installer chain with Authenticode. CI signs `ccx-windows-*.exe`, the desktop `ccx-desktop.exe`, then creates and signs `CCX-Desktop-*-windows-*-setup.exe`.

After pushing the tag, CI automatically:

1. Builds artifacts on all three platforms (macOS / Windows / Linux); Windows GitHub installer artifacts are signed by SignPath first
2. Generates per-platform checksums and signs them with Sigstore
3. The `finalize` job merges all per-platform checksums into `checksums.txt` and signs the consolidated manifest
4. All signature files are published alongside the release

After publishing, verify these signature assets exist in the Release:

- `checksums.txt` — cross-platform consolidated SHA256 manifest
- `checksums.txt.sigstore.json` — Sigstore bundle for the consolidated manifest
- `checksums-macos.txt(.sigstore.json)` — macOS platform
- `checksums-windows.txt(.sigstore.json)` — Windows platform
- `checksums-linux.txt(.sigstore.json)` — Linux platform

Existing `.sha256` sidecar files are preserved; desktop/backend updater behavior is unchanged.

The Windows SignPath integration requires these GitHub settings:

| Type | Name | Notes |
|------|------|-------|
| Secret | `SIGNPATH_API_TOKEN` | SignPath API token with submitter permissions for the target project/policy |
| Variable | `SIGNPATH_ARTIFACT_CONFIGURATION_SLUG` | Optional; empty uses the project's default artifact configuration |

The workflow currently uses SignPath organization `4ffeb1d3-df31-4323-a3e9-15fecdcbaad2`, project `ccx`, and signing policy `test-signing`.

Keep the SignPath artifact configuration scoped to single Windows executable/installer signing. If it is generated from sample artifacts, review it so third-party components are not signed with the project certificate.

See [Verifying Release Artifacts](./verification.md) for user verification instructions.
