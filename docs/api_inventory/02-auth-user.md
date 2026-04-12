# Auth 与用户侧清单（当前代码版）

- 同步日期：**2026-03-18**
- 接口数量：**56**
- 本文件只保留检索摘要；详细接口表见：`docs/api_full_reference_v2/02-auth-user.md`

## 范围

- 公开认证：`/api/v1/auth/register`、`/login`、`/refresh`、`/forgot-password`、`/oauth/linuxdo/*`
- 公开设置：`/api/v1/settings/public`
- 登录态用户接口：`/api/v1/user/**`、`/api/v1/keys/**`、`/api/v1/groups/**`、`/api/v1/usage/**`
- 公告、兑换、订阅：`/api/v1/announcements/**`、`/api/v1/redeem/**`、`/api/v1/subscriptions/**`
- 用户侧 Sora：`/api/v1/sora/**`

## 关键说明

- 公开认证接口当前都经过 `BackendModeAuthGuard`，其中高风险入口还带服务端限流。
- 只有 `/api/v1/auth/me` 和 `/api/v1/auth/revoke-all-sessions` 需要用户 JWT。
- `/api/v1/sora/**` 仅在后端注入 `SoraClient` handler 时注册。
- 本组接口不支持 Admin-Key。

## 唯一详细文档

- `docs/api_full_reference_v2/02-auth-user.md`
