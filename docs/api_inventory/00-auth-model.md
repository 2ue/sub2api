# 鉴权模型与 Admin-Key 判定（当前代码版）

- 同步日期：**2026-03-18**
- 本目录定位：轻量清单与检索入口。
- 唯一详细接口说明：`docs/api_full_reference_v2/`

## 代码依据

- `backend/internal/server/middleware/admin_auth.go`
- `backend/internal/server/middleware/jwt_auth.go`
- `backend/internal/server/middleware/api_key_auth.go`
- `backend/internal/server/middleware/api_key_auth_google.go`
- `backend/internal/server/routes/auth.go`
- `backend/internal/server/routes/user.go`
- `backend/internal/server/routes/admin.go`
- `backend/internal/server/routes/gateway.go`

## 结论

1. `admin-key` 仅用于 `AdminAuthMiddleware`
- 适用范围：`/api/v1/admin/**`
- Header：`x-api-key: <admin-api-key>`
- 也支持：`Authorization: Bearer <admin-jwt>`

2. 用户 JWT 仅用于用户登录态接口
- 中间件：`JWTAuthMiddleware`
- 典型范围：`/api/v1/user/**`、`/api/v1/keys/**`、`/api/v1/usage/**`
- Header：`Authorization: Bearer <jwt>`

3. Gateway 的 `x-api-key` 不是 admin-key
- 中间件：`APIKeyAuthMiddleware` 或 `APIKeyAuthWithSubscriptionGoogle`
- 典型范围：`/v1/**`、`/responses`、`/v1beta/**`、`/antigravity/**`
- 实际使用的是“用户 API Key”，不是系统设置里的 Admin API Key

4. 公开接口与 setup 接口不支持 admin-key
- 典型范围：`/health`、`/api/event_logging/batch`、`/setup/*`、`/api/v1/settings/public`

## 路径级矩阵

| 路径范围 | 主要鉴权 | 是否支持 Admin-Key |
| --- | --- | --- |
| `/api/v1/admin/**` | `AdminAuthMiddleware` | 是 |
| `/api/v1/auth/me`、`/api/v1/auth/revoke-all-sessions` | `JWTAuthMiddleware` | 否 |
| `/api/v1/user/**`、`/api/v1/keys/**`、`/api/v1/groups/**`、`/api/v1/usage/**`、`/api/v1/announcements/**`、`/api/v1/redeem/**`、`/api/v1/subscriptions/**`、`/api/v1/sora/**` | `JWTAuthMiddleware` | 否 |
| `/v1/**`、`/v1beta/**`、`/responses`、`/antigravity/**`、`/sora/v1/**` | 用户 API Key 中间件 | 否 |
| `/health`、`/api/event_logging/batch`、`/setup/*`、`/api/v1/settings/public` | 公开接口 / `setupGuard` | 否 |

## 详细文档入口

- 鉴权模型相关结论总览：`docs/api_full_reference_v2/08-admin-key-supported.md`
- 用户侧详细接口：`docs/api_full_reference_v2/02-auth-user.md`
- Gateway 详细接口：`docs/api_full_reference_v2/03-gateway.md`
