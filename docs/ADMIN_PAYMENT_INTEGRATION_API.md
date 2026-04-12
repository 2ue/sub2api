# ADMIN_PAYMENT_INTEGRATION_API

> 单文件中英双语文档 / Single-file bilingual documentation (Chinese + English)
>
> 当前代码版，按 **2026-03-18** 仓库实现同步。

---

## 中文

### 目标
本文档用于对接外部支付系统（如 `sub2apipay`）与 Sub2API 当前版 Admin API，覆盖：
- 支付成功后为指定用户充值或发放订阅
- 用户查询
- 管理员人工余额修正
- 购买页与自定义 iframe 页面 URL Query 透传
- 幂等与重试语义

### 基础地址
- 生产：`https://<your-domain>`
- 自建开发环境：按你的部署地址为准，例如 `http://<your-host>:8084`

### 认证
推荐使用 Admin API Key：
- `x-api-key: admin-<64hex>`
- `Content-Type: application/json`

建议所有支付写接口都带：
- `Idempotency-Key: <unique-key>`

说明：
- 管理员 JWT 也可访问 `/api/v1/admin/**`。
- 对接支付回调时，建议服务间只使用 Admin API Key，不使用用户 JWT。
- 当前代码支持通过配置 `idempotency.observe_only` 控制是否强制要求幂等头。即使观察期可能暂时放行无幂等头请求，也**不应依赖该行为**。

### 统一响应格式
当前 Admin API 默认使用统一信封：

成功：
```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

失败：
```json
{
  "code": 400,
  "message": "idempotency key is required",
  "reason": "IDEMPOTENCY_KEY_REQUIRED",
  "metadata": {
    "retry_after": "5"
  }
}
```

说明：
- `reason` 与 `metadata` 仅在对应错误有结构化信息时返回。
- 并发中的幂等冲突或 retry backoff 可能返回 `Retry-After` 响应头。
- 幂等重放成功时会返回 `X-Idempotency-Replayed: true`。

### 1) 一步完成“创建并兑换”
`POST /api/v1/admin/redeem-codes/create-and-redeem`

用途：原子完成“创建固定兑换码 + 立即兑换到指定用户”。

请求头：
- `x-api-key`
- `Idempotency-Key`

请求体：
```json
{
  "code": "s2p_cm1234567890",
  "type": "balance",
  "value": 100.0,
  "user_id": 123,
  "notes": "sub2apipay order: cm1234567890"
}
```

字段说明：
- `code`: 必填，固定兑换码，长度 `3-128`
- `type`: 可选，支持 `balance` / `concurrency` / `subscription` / `invitation`
  - 不传时默认 `balance`，这是当前代码保留的向后兼容行为
- `value`: 必填，`> 0`
- `user_id`: 必填，目标用户 ID，`> 0`
- `notes`: 可选
- `group_id`: `subscription` 类型必填
- `validity_days`: `subscription` 类型必填，且必须 `> 0`

`subscription` 示例：
```json
{
  "code": "s2p_sub_cm1234567890",
  "type": "subscription",
  "value": 1,
  "user_id": 123,
  "group_id": 9,
  "validity_days": 30,
  "notes": "sub2apipay order: sub-cm1234567890"
}
```

成功响应示例：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "redeem_code": {
      "id": 123,
      "code": "s2p_cm1234567890",
      "type": "balance",
      "value": 100,
      "status": "used",
      "used_by": 123
    }
  }
}
```

当前幂等语义：
- 相同 `Idempotency-Key` + 相同 payload：返回同一结果，且响应头带 `X-Idempotency-Replayed: true`
- 相同 `Idempotency-Key` + 不同 payload：`409`，`reason=IDEMPOTENCY_KEY_CONFLICT`
- 若第一次执行已创建 code 但尚未完成兑换，后续重放会继续尝试兑换
- 若 code 已被同一 `user_id` 使用，再次调用会按幂等成功返回 `200`
- 若 code 已被其他用户使用，返回 `409`，错误码为 `REDEEM_CODE_CONFLICT`
- 若部署启用了强制幂等，缺少 `Idempotency-Key` 时会返回 `400`，`reason=IDEMPOTENCY_KEY_REQUIRED`

curl 示例：
```bash
curl -X POST "${BASE}/api/v1/admin/redeem-codes/create-and-redeem" \
  -H "x-api-key: ${KEY}" \
  -H "Idempotency-Key: pay-cm1234567890-success" \
  -H "Content-Type: application/json" \
  -d '{
    "code":"s2p_cm1234567890",
    "value":100.00,
    "user_id":123,
    "notes":"sub2apipay order: cm1234567890"
  }'
```

### 2) 查询用户（可选前置校验）
`GET /api/v1/admin/users/:id`

用途：在支付回调入账前，确认目标用户存在。

curl 示例：
```bash
curl -s "${BASE}/api/v1/admin/users/123" \
  -H "x-api-key: ${KEY}"
```

成功响应形态：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 123,
    "email": "user@example.com"
  }
}
```

### 3) 人工余额调整
`POST /api/v1/admin/users/:id/balance`

用途：人工补偿 / 扣减 / 覆盖余额，支持：
- `set`
- `add`
- `subtract`

请求头：
- `x-api-key`
- `Idempotency-Key`

请求体示例（扣减）：
```json
{
  "balance": 100.0,
  "operation": "subtract",
  "notes": "manual correction"
}
```

字段说明：
- `balance`: 必填，`> 0`
- `operation`: 必填，`set | add | subtract`
- `notes`: 可选

成功响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 123,
    "balance": 900
  }
}
```

当前幂等语义：
- 相同 key + 相同 payload：`200`，并带 `X-Idempotency-Replayed: true`
- 相同 key + 不同 payload：`409`
- 若部署启用了强制幂等，缺少 `Idempotency-Key`：`400`

curl：
```bash
curl -X POST "${BASE}/api/v1/admin/users/123/balance" \
  -H "x-api-key: ${KEY}" \
  -H "Idempotency-Key: balance-subtract-cm1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "balance":100.00,
    "operation":"subtract",
    "notes":"manual correction"
  }'
```

### 4) 推荐的支付侧重试策略
推荐将“支付成功”和“入账成功”拆成两个状态。

建议：
1. 支付平台回调验签通过后，先把订单标记为 `paid`
2. 再调用 `create-and-redeem`
3. 若接口失败，但支付已成功，允许后台任务或人工重试
4. 重试时保持业务唯一 `code` 不变
5. 如果是完整重放同一次请求，优先继续使用同一个 `Idempotency-Key`
6. 如果是你在业务层明确发起“新的修复动作”，再换新的 `Idempotency-Key`

### 5) 购买页 / 自定义 iframe 页面 URL Query 透传
当前前端不只给 `purchase_subscription_url` 追加参数，也会给用户侧自定义 iframe 页面追加同一套参数。

当前实际透传参数：
- `user_id`
- `token`
- `theme`（`light` / `dark`）
- `lang`（例如 `zh` / `en`）
- `ui_mode`（固定 `embedded`）
- `src_host`（当前站点 origin）
- `src_url`（当前页面完整 URL）

购买页示例：
```text
https://pay.example.com/pay?user_id=123&token=<jwt>&theme=light&lang=zh&ui_mode=embedded&src_host=https%3A%2F%2Fpanel.example.com&src_url=https%3A%2F%2Fpanel.example.com%2Fpurchase-subscription
```

说明：
- 透传逻辑由前端 `buildEmbeddedUrl()` 统一实现。
- `purchase_subscription_enabled=false` 时，购买页入口会显示未开启状态，不会嵌入目标地址。
- 目标站点如果设置了 `X-Frame-Options` 或 `CSP frame-ancestors`，iframe 可能被浏览器拦截；用户可使用“新窗口打开”。

### 6) 当前实现的关键点
- `create-and-redeem` 现已是正式 admin 路由，不需要手工组合“先创建再兑换”
- `type` 在 `create-and-redeem` 中现在是可选，默认 `balance`
- 购买页 / 自定义 iframe 现在多传 `src_host` 与 `src_url`
- 幂等接口的真实返回是统一信封，不是裸 JSON
- `users/:id/balance` 现在也明确走幂等写路径
- 幂等是否强制要求请求头，取决于部署时的 `idempotency.observe_only` 配置

### 7) `doc_url` 配置建议
- 查看链接：`https://github.com/Wei-Shaw/sub2api/blob/main/docs/ADMIN_PAYMENT_INTEGRATION_API.md`
- 下载链接：`https://raw.githubusercontent.com/Wei-Shaw/sub2api/main/docs/ADMIN_PAYMENT_INTEGRATION_API.md`

---

## English

### Purpose
This document describes the current Sub2API Admin API surface for payment integrations such as `sub2apipay`, including:
- Recharge or subscription assignment after payment success
- User lookup
- Manual balance correction
- Purchase/custom iframe page query forwarding
- Idempotency and retry semantics

### Base URL
- Production: `https://<your-domain>`
- Self-hosted dev/staging: use your own deployment address, for example `http://<your-host>:8084`

### Authentication
Recommended headers:
- `x-api-key: admin-<64hex>`
- `Content-Type: application/json`
- `Idempotency-Key` for payment write operations

Notes:
- Admin JWT can also access `/api/v1/admin/**`.
- For server-to-server payment callbacks, prefer Admin API Key instead of user JWT.
- Whether missing `Idempotency-Key` is rejected depends on the deployed `idempotency.observe_only` setting. Do not rely on observe-only behavior.

### Response envelope
Current Admin APIs use the standard response envelope.

Success:
```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

Error:
```json
{
  "code": 400,
  "message": "idempotency key is required",
  "reason": "IDEMPOTENCY_KEY_REQUIRED",
  "metadata": {
    "retry_after": "5"
  }
}
```

### 1) Create and redeem in one step
`POST /api/v1/admin/redeem-codes/create-and-redeem`

Use case: atomically create a fixed redeem code and redeem it to a target user.

Headers:
- `x-api-key`
- `Idempotency-Key`

Request body:
```json
{
  "code": "s2p_cm1234567890",
  "type": "balance",
  "value": 100.0,
  "user_id": 123,
  "notes": "sub2apipay order: cm1234567890"
}
```

Field rules:
- `code`: required, length `3-128`
- `type`: optional, one of `balance | concurrency | subscription | invitation`
  - if omitted, the current code defaults it to `balance`
- `value`: required, must be `> 0`
- `user_id`: required, must be `> 0`
- `group_id`: required for `subscription`
- `validity_days`: required and `> 0` for `subscription`

Idempotency behavior:
- Same key + same payload: replayed success, with `X-Idempotency-Replayed: true`
- Same key + different payload: `409`, `reason=IDEMPOTENCY_KEY_CONFLICT`
- If the code already exists and was already redeemed by the same `user_id`, the endpoint still returns `200`
- If the code was used by another user, it returns `409` with `REDEEM_CODE_CONFLICT`
- If strict idempotency enforcement is enabled, missing `Idempotency-Key` returns `400`

### 2) Query user
`GET /api/v1/admin/users/:id`

Use case: optional pre-check before crediting a paid order.

### 3) Manual balance adjustment
`POST /api/v1/admin/users/:id/balance`

Supported operations:
- `set`
- `add`
- `subtract`

Headers:
- `x-api-key`
- `Idempotency-Key`

Request body example:
```json
{
  "balance": 100.0,
  "operation": "subtract",
  "notes": "manual correction"
}
```

The endpoint returns the updated user object inside the standard envelope.

### 4) Recommended retry strategy
Recommended flow:
1. Mark the order as `paid` after signature verification succeeds
2. Call `create-and-redeem`
3. If payment succeeded but crediting failed, allow background or manual retry
4. Keep the same business `code`
5. Reuse the same `Idempotency-Key` when replaying the same operation
6. Use a new `Idempotency-Key` only when you intentionally start a new repair action

### 5) Purchase/custom iframe query forwarding
The frontend now appends the same parameter set both for `purchase_subscription_url` and user-facing custom iframe pages.

Current forwarded query params:
- `user_id`
- `token`
- `theme`
- `lang`
- `ui_mode=embedded`
- `src_host`
- `src_url`

### 6) Key points of the current implementation
- `create-and-redeem` is now a first-class admin route
- `type` is now optional and defaults to `balance`
- iframe URLs now also receive `src_host` and `src_url`
- idempotent admin endpoints return the standard response envelope
- whether missing idempotency headers are rejected depends on deployed config

### 7) Recommended `doc_url`
- View URL: `https://github.com/Wei-Shaw/sub2api/blob/main/docs/ADMIN_PAYMENT_INTEGRATION_API.md`
- Download URL: `https://raw.githubusercontent.com/Wei-Shaw/sub2api/main/docs/ADMIN_PAYMENT_INTEGRATION_API.md`
