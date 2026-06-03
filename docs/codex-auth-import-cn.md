# Codex Auth（OpenAI OAuth）导入字段说明（Sub2API）

本文档说明：在 Sub2API 中“导入 Codex Auth”（即导入 **OpenAI OAuth** 账号凭证）时，可以设置哪些字段、字段含义与注意事项。

> 说明：本项目的 OpenAI OAuth 流程使用 **Codex CLI 官方 OAuth Client**（见 `backend/internal/pkg/openai/oauth.go` 中 `ClientID` 注释）。因此很多地方把这类凭证称为 “Codex Auth”。

---

## 1. 导入入口（你实际调用的接口）

### 1.1 直接创建账号（推荐用于“导入/批量导入”）

- **接口**：`POST /api/v1/admin/accounts`
- **用途**：把你已有的 OpenAI OAuth token（Codex Auth）直接写入 Sub2API 账号库。
- **配套脚本**：`tools/import_openai_oauth_accounts.py`（批量导入就是调用该接口）

### 1.2 走授权码创建账号（适合网页手动授权）

如果你没有现成 token，想通过浏览器授权拿到 code 再创建账号，可用：

- `POST /api/v1/admin/openai/generate-auth-url`
- `POST /api/v1/admin/openai/exchange-code`
- `POST /api/v1/admin/openai/create-from-oauth`（把“兑换 code + 创建账号”合并为一个接口）

本文重点是 **1.1 导入**。

---

## 2. `POST /api/v1/admin/accounts` 顶层可设置字段

> 导入 Codex Auth 时，一般固定 `platform="openai"`、`type="oauth"`，其余按需设置。

| 字段 | 类型 | 必填 | 说明 |
|---|---:|:---:|---|
| `name` | string | 是 | 账号名称（建议用邮箱/备注便于识别） |
| `notes` | string\|null | 否 | 备注（支持为空） |
| `platform` | string | 是 | 固定填 `openai` |
| `type` | string | 是 | 固定填 `oauth` |
| `credentials` | object | 是 | **OAuth 凭证对象**（见下节） |
| `extra` | object | 否 | 额外信息（常用：`email`） |
| `proxy_id` | number\|null | 否 | 代理 ID（用于上游请求/刷新 token） |
| `concurrency` | number | 否 | 账号最大并发；`<=0` 表示不限制 |
| `priority` | number | 否 | 调度优先级：**数值越小优先级越高** |
| `rate_multiplier` | number\|null | 否 | 账号计费倍率（必须 `>=0`） |
| `group_ids` | number[] | 否 | 绑定到哪些分组；为空时会自动绑定到 `openai-default`（不利于隔离，建议显式指定） |
| `expires_at` | number\|null | 否 | **账号到期时间**（Unix 秒）；注意它不是 token 的过期时间 |
| `auto_pause_on_expired` | boolean\|null | 否 | 账号过期后是否自动暂停（默认 `true`） |
| `confirm_mixed_channel_risk` | boolean\|null | 否 | 与混合渠道/分组绑定风险相关（一般导入 OpenAI OAuth 不用关心） |

---

## 3. `credentials`（Codex Auth / OpenAI OAuth）可设置字段

### 3.1 必填/强烈建议字段

| 字段 | 类型 | 建议 | 用途 |
|---|---:|:---:|---|
| `access_token` | string | 必填 | 上游请求的 `Authorization: Bearer ...` |
| `refresh_token` | string | 强烈建议必填 | 用于自动刷新 token；缺失会导致 token 到期后无法自愈 |
| `expires_at` | string\|number | 强烈建议 | token 过期时间，用于判断何时刷新 |

`expires_at` 支持的格式（系统会自动解析）：

- RFC3339 字符串：`"2025-01-01T00:00:00Z"`
- Unix 秒时间戳（字符串或数字）：`"1735689600"` / `1735689600`

### 3.2 OpenAI（ChatGPT 内部 API）相关字段

| 字段 | 类型 | 建议 | 用途 |
|---|---:|:---:|---|
| `chatgpt_account_id` | string | 建议填写 | 上游请求头 `chatgpt-account-id`（缺失可能导致上游报错或能力受限） |
| `chatgpt_user_id` | string | 可选 | 当前项目不用于转发，仅用于标识/排查 |
| `organization_id` | string | 可选 | 当前项目不用于转发（预留/兼容字段） |
| `id_token` | string | 可选 | 当前项目不用于转发（常用于从中解析/对账身份信息） |

### 3.3 可选覆盖/高级字段

| 字段 | 类型 | 说明 |
|---|---:|---|
| `user_agent` | string | 覆盖上游请求 `User-Agent` |
| `model_mapping` | object | 模型映射/白名单：`{"gpt-5.2-codex":"gpt-5.2-codex"}`；存在时仅允许请求映射表中的模型 |

---

## 4. 批量导入脚本的“输入 JSON 文件”字段（`tools/import_openai_oauth_accounts.py`）

脚本会从你的 token JSON 文件中读取并写入以下字段：

- `access_token`（必需）
- `refresh_token`（必需）
- `expires_at`（可选但强烈建议）
- `id_token` / `chatgpt_account_id` / `chatgpt_user_id` / `organization_id`（可选）
- `user_agent` / `model_mapping`（可选）
- `email`（可选，会写到 `extra.email`）

> 其他字段即使存在也会被脚本忽略（避免把无关数据写进数据库）。

### 4.1 最小输入示例（能用但不推荐长期）

```json
{
  "access_token": "eyJhbGciOi...",
  "refresh_token": "0.AAA..."
}
```

### 4.2 推荐输入示例（更稳定、可观测）

```json
{
  "email": "someone@example.com",
  "access_token": "eyJhbGciOi...",
  "refresh_token": "0.AAA...",
  "expires_at": "2026-02-01T08:00:00Z",
  "chatgpt_account_id": "act_...",
  "chatgpt_user_id": "user_...",
  "organization_id": "org_...",
  "user_agent": "codex_cli_rs/0.2.0",
  "model_mapping": {
    "gpt-5.2-codex": "gpt-5.2-codex"
  }
}
```

---

## 5. 安全与常见坑

- **不要把 token JSON 提交到 git**：文件包含高敏 `access_token/refresh_token/id_token`。
- **区分两个 `expires_at`**：
  - `credentials.expires_at`：token 过期时间（建议填写）
  - 顶层 `expires_at`：账号过期时间（可选，用于到期自动暂停）
- **建议显式绑定 `group_ids`**：为空会自动绑定到 `openai-default`，不利于隔离/灰度。
- **`chatgpt_account_id` 建议填写**：代码中会把它作为 `chatgpt-account-id` 请求头转发到上游。

---

## 6. （可选）脚本参数能设置哪些“账号级字段”

`tools/import_openai_oauth_accounts.py` 会把“账号级字段”通过 `POST /api/v1/admin/accounts` 一并写入：

| 脚本参数 | 写入字段 |
|---|---|
| `--name-prefix` | `name`（前缀 + email/chatgpt_user_id/#idx） |
| `--proxy-id` | `proxy_id` |
| `--account-concurrency` | `concurrency` |
| `--account-priority` | `priority` |
| `--account-rate-multiplier` | `rate_multiplier` |
| `--group-id` / `--group-name` | `group_ids`（建议用于隔离导入账号） |
| `--disable-scheduling` | 导入后额外调用 bulk-update，把 `schedulable=false`（更强隔离） |

> 管理员鉴权支持两种：`--admin-token`（JWT）或 `--admin-api-key`（x-api-key）。

---

## 7. （可选）直接调用 API 的示例（只展示字段结构）

```bash
curl -X POST "$BASE_URL/api/v1/admin/accounts" \
  -H "Authorization: Bearer $SUB2API_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "openai-oauth someone@example.com",
    "platform": "openai",
    "type": "oauth",
    "credentials": {
      "access_token": "REDACTED",
      "refresh_token": "REDACTED",
      "expires_at": "2026-02-01T08:00:00Z",
      "chatgpt_account_id": "act_REDACTED"
    },
    "extra": { "email": "someone@example.com" },
    "group_ids": [123],
    "concurrency": 1,
    "priority": 50
  }'
```
