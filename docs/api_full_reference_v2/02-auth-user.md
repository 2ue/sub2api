# Auth 与用户接口（当前代码版）

- 同步日期：**2026-03-18**
- 接口数量：**56**
- 文档策略：优先保证当前路由、鉴权、中间件与注册位置准确；字段级请求体请继续查对应 handler / dto。
- 本版已补入 `/api/v1/sora/**` 用户侧 Sora 客户端接口。
- `/api/v1/sora/**` 仅在后端实际注入 `SoraClient` handler 时注册。

## Sora 客户端

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| POST | `/api/v1/sora/generate` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.SoraClient.Generate` | `backend/internal/server/routes/sora_client.go:26` | 仅当后端注入 `SoraClient` handler 时注册 |
| GET | `/api/v1/sora/generations` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.SoraClient.ListGenerations` | `backend/internal/server/routes/sora_client.go:27` | 仅当后端注入 `SoraClient` handler 时注册 |
| GET | `/api/v1/sora/generations/:id` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.SoraClient.GetGeneration` | `backend/internal/server/routes/sora_client.go:28` | 仅当后端注入 `SoraClient` handler 时注册 |
| DELETE | `/api/v1/sora/generations/:id` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.SoraClient.DeleteGeneration` | `backend/internal/server/routes/sora_client.go:29` | 仅当后端注入 `SoraClient` handler 时注册 |
| POST | `/api/v1/sora/generations/:id/cancel` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.SoraClient.CancelGeneration` | `backend/internal/server/routes/sora_client.go:30` | 仅当后端注入 `SoraClient` handler 时注册 |
| POST | `/api/v1/sora/generations/:id/save` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.SoraClient.SaveToStorage` | `backend/internal/server/routes/sora_client.go:31` | 仅当后端注入 `SoraClient` handler 时注册 |
| GET | `/api/v1/sora/models` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.SoraClient.GetModels` | `backend/internal/server/routes/sora_client.go:33` | 仅当后端注入 `SoraClient` handler 时注册 |
| GET | `/api/v1/sora/quota` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.SoraClient.GetQuota` | `backend/internal/server/routes/sora_client.go:32` | 仅当后端注入 `SoraClient` handler 时注册 |
| GET | `/api/v1/sora/storage-status` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.SoraClient.GetStorageStatus` | `backend/internal/server/routes/sora_client.go:34` | 仅当后端注入 `SoraClient` handler 时注册 |

## 兑换

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| POST | `/api/v1/redeem` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Redeem.Redeem` | `backend/internal/server/routes/user.go:81` | - |
| GET | `/api/v1/redeem/history` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Redeem.GetHistory` | `backend/internal/server/routes/user.go:82` | - |

## 公告

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/announcements` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Announcement.List` | `backend/internal/server/routes/user.go:74` | - |
| POST | `/api/v1/announcements/:id/read` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Announcement.MarkRead` | `backend/internal/server/routes/user.go:75` | - |

## 公开设置

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/settings/public` | 公开接口（无需鉴权） | 否 | `h.Setting.GetPublicSettings` | `backend/internal/server/routes/auth.go:78` | - |

## 用户 API Key

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/keys` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.APIKey.List` | `backend/internal/server/routes/user.go:44` | - |
| POST | `/api/v1/keys` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.APIKey.Create` | `backend/internal/server/routes/user.go:46` | - |
| GET | `/api/v1/keys/:id` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.APIKey.GetByID` | `backend/internal/server/routes/user.go:45` | - |
| PUT | `/api/v1/keys/:id` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.APIKey.Update` | `backend/internal/server/routes/user.go:47` | - |
| DELETE | `/api/v1/keys/:id` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.APIKey.Delete` | `backend/internal/server/routes/user.go:48` | - |

## 用户分组

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/groups/available` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.APIKey.GetAvailableGroups` | `backend/internal/server/routes/user.go:54` | - |
| GET | `/api/v1/groups/rates` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.APIKey.GetUserGroupRates` | `backend/internal/server/routes/user.go:55` | - |

## 用户用量

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/usage` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Usage.List` | `backend/internal/server/routes/user.go:61` | - |
| GET | `/api/v1/usage/:id` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Usage.GetByID` | `backend/internal/server/routes/user.go:62` | - |
| POST | `/api/v1/usage/dashboard/api-keys-usage` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Usage.DashboardAPIKeysUsage` | `backend/internal/server/routes/user.go:68` | - |
| GET | `/api/v1/usage/dashboard/models` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Usage.DashboardModels` | `backend/internal/server/routes/user.go:67` | - |
| GET | `/api/v1/usage/dashboard/stats` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Usage.DashboardStats` | `backend/internal/server/routes/user.go:65` | - |
| GET | `/api/v1/usage/dashboard/trend` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Usage.DashboardTrend` | `backend/internal/server/routes/user.go:66` | - |
| GET | `/api/v1/usage/stats` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Usage.Stats` | `backend/internal/server/routes/user.go:63` | - |

## 用户订阅

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/subscriptions` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Subscription.List` | `backend/internal/server/routes/user.go:88` | - |
| GET | `/api/v1/subscriptions/active` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Subscription.GetActive` | `backend/internal/server/routes/user.go:89` | - |
| GET | `/api/v1/subscriptions/progress` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Subscription.GetProgress` | `backend/internal/server/routes/user.go:90` | - |
| GET | `/api/v1/subscriptions/summary` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Subscription.GetSummary` | `backend/internal/server/routes/user.go:91` | - |

## 用户资料

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| PUT | `/api/v1/user` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.User.UpdateProfile` | `backend/internal/server/routes/user.go:27` | - |
| PUT | `/api/v1/user/password` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.User.ChangePassword` | `backend/internal/server/routes/user.go:26` | - |
| GET | `/api/v1/user/profile` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.User.GetProfile` | `backend/internal/server/routes/user.go:25` | - |
| POST | `/api/v1/user/totp/disable` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Totp.Disable` | `backend/internal/server/routes/user.go:37` | - |
| POST | `/api/v1/user/totp/enable` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Totp.Enable` | `backend/internal/server/routes/user.go:36` | - |
| POST | `/api/v1/user/totp/send-code` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Totp.SendVerifyCode` | `backend/internal/server/routes/user.go:34` | - |
| POST | `/api/v1/user/totp/setup` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Totp.InitiateSetup` | `backend/internal/server/routes/user.go:35` | - |
| GET | `/api/v1/user/totp/status` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Totp.GetStatus` | `backend/internal/server/routes/user.go:32` | - |
| GET | `/api/v1/user/totp/verification-method` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Totp.GetVerificationMethod` | `backend/internal/server/routes/user.go:33` | - |

## 认证

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| POST | `/api/v1/auth/forgot-password` | 公开接口（`BackendModeAuthGuard`，带服务端限流） | 否 | `h.Auth.ForgotPassword` | `backend/internal/server/routes/auth.go:58` | - |
| POST | `/api/v1/auth/login` | 公开接口（`BackendModeAuthGuard`，带服务端限流） | 否 | `h.Auth.Login` | `backend/internal/server/routes/auth.go:34` | - |
| POST | `/api/v1/auth/login/2fa` | 公开接口（`BackendModeAuthGuard`，带服务端限流） | 否 | `h.Auth.Login2FA` | `backend/internal/server/routes/auth.go:37` | - |
| POST | `/api/v1/auth/logout` | 公开接口（`BackendModeAuthGuard`） | 否 | `h.Auth.Logout` | `backend/internal/server/routes/auth.go:48` | 允许未认证用户调用以撤销 Refresh Token |
| GET | `/api/v1/auth/me` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Auth.GetCurrentUser` | `backend/internal/server/routes/auth.go:86` | 还会经过 `BackendModeUserGuard` |
| GET | `/api/v1/auth/oauth/linuxdo/callback` | 公开接口（`BackendModeAuthGuard`） | 否 | `h.Auth.LinuxDoOAuthCallback` | `backend/internal/server/routes/auth.go:66` | - |
| POST | `/api/v1/auth/oauth/linuxdo/complete-registration` | 公开接口（`BackendModeAuthGuard`，带服务端限流） | 否 | `h.Auth.CompleteLinuxDoOAuthRegistration` | `backend/internal/server/routes/auth.go:67` | 新增 LinuxDo OAuth 补充注册完成接口 |
| GET | `/api/v1/auth/oauth/linuxdo/start` | 公开接口（`BackendModeAuthGuard`） | 否 | `h.Auth.LinuxDoOAuthStart` | `backend/internal/server/routes/auth.go:65` | - |
| POST | `/api/v1/auth/refresh` | 公开接口（`BackendModeAuthGuard`，带服务端限流） | 否 | `h.Auth.RefreshToken` | `backend/internal/server/routes/auth.go:44` | - |
| POST | `/api/v1/auth/register` | 公开接口（`BackendModeAuthGuard`，带服务端限流） | 否 | `h.Auth.Register` | `backend/internal/server/routes/auth.go:31` | - |
| POST | `/api/v1/auth/reset-password` | 公开接口（`BackendModeAuthGuard`，带服务端限流） | 否 | `h.Auth.ResetPassword` | `backend/internal/server/routes/auth.go:62` | - |
| POST | `/api/v1/auth/revoke-all-sessions` | `JWTAuthMiddleware`（用户登录态） | 否 | `h.Auth.RevokeAllSessions` | `backend/internal/server/routes/auth.go:88` | 还会经过 `BackendModeUserGuard` |
| POST | `/api/v1/auth/send-verify-code` | 公开接口（`BackendModeAuthGuard`，带服务端限流） | 否 | `h.Auth.SendVerifyCode` | `backend/internal/server/routes/auth.go:40` | - |
| POST | `/api/v1/auth/validate-invitation-code` | 公开接口（`BackendModeAuthGuard`，带服务端限流） | 否 | `h.Auth.ValidateInvitationCode` | `backend/internal/server/routes/auth.go:54` | - |
| POST | `/api/v1/auth/validate-promo-code` | 公开接口（`BackendModeAuthGuard`，带服务端限流） | 否 | `h.Auth.ValidatePromoCode` | `backend/internal/server/routes/auth.go:50` | - |
