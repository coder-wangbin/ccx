# Microsoft Store Submission API 参考

本参考记录 CCX Desktop Store 更新技能使用的官方 API 要点。

## 官方文档

- Create and manage submissions using Microsoft Store services: https://learn.microsoft.com/en-us/windows/uwp/monetize/create-and-manage-submissions-using-windows-store-services
- Manage app submissions: https://learn.microsoft.com/en-us/windows/uwp/monetize/manage-app-submissions
- Update an app submission: https://learn.microsoft.com/en-us/windows/uwp/monetize/update-an-app-submission
- Commit an app submission: https://learn.microsoft.com/en-us/windows/uwp/monetize/commit-an-app-submission
- Get status for an app submission: https://learn.microsoft.com/en-us/windows/uwp/monetize/get-status-for-an-app-submission

## 凭据与 applicationId 获取位置

调用 Microsoft Store submission API 需要 `tenant_id`、`client_id`、`client_secret`、`applicationId` 四个值，全部在 Partner Center（`partner.microsoft.com`）后台获取，只需配置一次。

### tenant_id / client_id / client_secret

路径：**Account settings → Users** 页面，关联一个 Azure AD 应用。

1. 先把组织 Partner Center 账号关联到组织 Azure AD 目录。参考 https://learn.microsoft.com/en-us/windows/apps/publish/partner-center/associate-azure-ad-with-partner-center
2. 在 Users 页面 add Azure AD application，必须授予 **Manager** 角色（调用 submission API 必需）。
3. 点进该应用，复制 **Tenant ID** 和 **Client ID**。
4. 点 **Add new key**，复制 **Key**（即 client secret）。离开页面后无法再查看，必须当场保存。参考 https://learn.microsoft.com/en-us/windows/apps/publish/partner-center/manage-azure-ad-applications-in-partner-center#manage-keys

对应脚本环境变量：`MS_STORE_TENANT_ID`、`MS_STORE_CLIENT_ID`、`MS_STORE_CLIENT_SECRET`。

### applicationId

路径：Partner Center 进入目标 app → **Product identity**（App identity）页面，那里的 **Store ID** 就是 `applicationId`，格式形如 `9NBLGGH4R315`。参考 https://learn.microsoft.com/en-us/windows/apps/publish/view-app-identity-details

对应脚本环境变量：`MS_STORE_APPLICATION_ID`。

### 前置条件（不满足 API 会失败）

1. app 必须已存在：API 不能创建 app，必须先在 Partner Center 预留名称创建。
2. 至少一次人工提交：该 app 必须先在 Partner Center 手动完成过一次提交，包括 age ratings 问卷；之后才能用 API 创建后续提交。
3. CCX Desktop 已上架 Store，上述条件通常已满足。

### 备选工具

微软官方开源的 StoreBroker PowerShell 模块封装了这套 API：https://github.com/Microsoft/StoreBroker 。功能与本 skill 脚本等价，可作为替代方案。

## OAuth

Token endpoint:

```http
POST https://login.microsoftonline.com/<tenant_id>/oauth2/token
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials
&client_id=<client_id>
&client_secret=<client_secret>
&resource=https://manage.devcenter.microsoft.com
```

Store submission API 使用 Azure AD / Entra client credentials。access token 通常 60 分钟有效。

## App submission endpoint

Base URL:

```text
https://manage.devcenter.microsoft.com/v1.0/my/applications/{applicationId}
```

常用端点：

| 操作 | 方法与路径 |
| --- | --- |
| 创建 submission | `POST /submissions` |
| 获取 submission | `GET /submissions/{submissionId}` |
| 更新 submission | `PUT /submissions/{submissionId}` |
| 提交 submission | `POST /submissions/{submissionId}/commit` |
| 查询状态 | `GET /submissions/{submissionId}/status` |
| 删除 submission | `DELETE /submissions/{submissionId}` |

## 包上传模型

创建 submission 的响应包含：

- `id`: submission id
- `fileUploadUrl`: Azure Blob SAS URI，用于上传包含 app packages/listing assets 的 ZIP archive
- `applicationPackages`: 当前包列表
- `status`: 通常为 `PendingCommit`

更新包时：

1. 创建 ZIP archive，内部包含两个 MSIX 文件。
2. 上传 ZIP 到 `fileUploadUrl`。
3. 在 submission JSON 中设置 `applicationPackages`。
4. 调用 update submission。
5. 调用 commit submission。

官方文档说明 `fileUploadUrl` 是上传 packages 的 SAS URI；添加新 packages/listing images/trailers 时上传包含这些文件的 ZIP archive。

## applicationPackages 字段

`update an app submission` 文档说明：更新 submission 时，每个 package object 只需要以下字段，其他字段由 Partner Center 填充：

```json
{
  "fileName": "CCX-Desktop-v2.9.28-windows-amd64-store.msix",
  "fileStatus": "PendingUpload",
  "minimumDirectXVersion": "None",
  "minimumSystemRam": "None"
}
```

对 Windows 10/11 MSIX，`minimumDirectXVersion` 与 `minimumSystemRam` 字段仍需存在，但值会被忽略。

## Store listing releaseNotes

`update an app submission` 的 request body 包含 `listings` 字段。每个语言 listing 的 `baseListing.releaseNotes` 对应 Store 页面“此版本的新功能/更新内容”。Microsoft Store listing release notes 通常限制为 1000 字符。

本技能默认从 GitHub Release body 读取更新内容，转换规则：

- 去掉 Markdown 标题标记、粗体、链接 URL、反引号和 HTML tag。
- 去掉 `Full Changelog` 及其之后内容。
- 默认超过 1000 字符时失败，避免静默丢内容；只有显式 `--truncate-release-notes` 才截断。
- 将同一份 release notes 写入 submission 中已有的所有 `listings.*.baseListing.releaseNotes`。

## 状态轮询

提交后调用：

```text
GET /submissions/{submissionId}/status
```

成功路径通常会从 `CommitStarted` 进入 `PreProcessing`，随后可继续在 Partner Center 或 API 中观察认证进度。失败时关注：

- `status`
- `statusDetails.errors`
- HTTP response body 中的 error code/message

## 风险

官方文档强调：如果使用 API 创建 app/package flight/add-on submission，后续修改同一 submission 时应继续使用 API，不要在 Partner Center UI 中修改同一 submission。混用可能导致该 submission 无法继续由 API 修改或提交，严重时需要删除并重新创建。
