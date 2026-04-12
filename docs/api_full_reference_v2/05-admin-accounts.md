# Admin 账号与 OAuth 接口（当前代码版）

- 同步日期：**2026-03-18**
- 接口数量：**71**
- 文档策略：优先保证当前路由、鉴权、中间件与注册位置准确；字段级请求体请继续查对应 handler / dto。
- 本文件聚合账号管理、各平台 OAuth、代理管理接口。
- 已补入账号批量操作、CRS 预览同步、recover-state、today-stats、clear-rate-limit、reset-quota 等新路由。

## Antigravity OAuth

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| POST | `/api/v1/admin/antigravity/oauth/auth-url` | AdminAuthMiddleware | 是 | `h.Admin.AntigravityOAuth.GenerateAuthURL` | `backend/internal/server/routes/admin.go:339` | - |
| POST | `/api/v1/admin/antigravity/oauth/exchange-code` | AdminAuthMiddleware | 是 | `h.Admin.AntigravityOAuth.ExchangeCode` | `backend/internal/server/routes/admin.go:340` | - |
| POST | `/api/v1/admin/antigravity/oauth/refresh-token` | AdminAuthMiddleware | 是 | `h.Admin.AntigravityOAuth.RefreshToken` | `backend/internal/server/routes/admin.go:341` | - |

## Gemini OAuth

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| POST | `/api/v1/admin/gemini/oauth/auth-url` | AdminAuthMiddleware | 是 | `h.Admin.GeminiOAuth.GenerateAuthURL` | `backend/internal/server/routes/admin.go:330` | - |
| GET | `/api/v1/admin/gemini/oauth/capabilities` | AdminAuthMiddleware | 是 | `h.Admin.GeminiOAuth.GetCapabilities` | `backend/internal/server/routes/admin.go:332` | - |
| POST | `/api/v1/admin/gemini/oauth/exchange-code` | AdminAuthMiddleware | 是 | `h.Admin.GeminiOAuth.ExchangeCode` | `backend/internal/server/routes/admin.go:331` | - |

## OpenAI OAuth

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| POST | `/api/v1/admin/openai/accounts/:id/refresh` | AdminAuthMiddleware | 是 | `h.Admin.OpenAIOAuth.RefreshAccountToken` | `backend/internal/server/routes/admin.go:309` | - |
| POST | `/api/v1/admin/openai/create-from-oauth` | AdminAuthMiddleware | 是 | `h.Admin.OpenAIOAuth.CreateAccountFromOAuth` | `backend/internal/server/routes/admin.go:310` | - |
| POST | `/api/v1/admin/openai/exchange-code` | AdminAuthMiddleware | 是 | `h.Admin.OpenAIOAuth.ExchangeCode` | `backend/internal/server/routes/admin.go:307` | - |
| POST | `/api/v1/admin/openai/generate-auth-url` | AdminAuthMiddleware | 是 | `h.Admin.OpenAIOAuth.GenerateAuthURL` | `backend/internal/server/routes/admin.go:306` | - |
| POST | `/api/v1/admin/openai/refresh-token` | AdminAuthMiddleware | 是 | `h.Admin.OpenAIOAuth.RefreshToken` | `backend/internal/server/routes/admin.go:308` | - |

## Sora OAuth

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| POST | `/api/v1/admin/sora/accounts/:id/refresh` | AdminAuthMiddleware | 是 | `h.Admin.OpenAIOAuth.RefreshAccountToken` | `backend/internal/server/routes/admin.go:322` | - |
| POST | `/api/v1/admin/sora/create-from-oauth` | AdminAuthMiddleware | 是 | `h.Admin.OpenAIOAuth.CreateAccountFromOAuth` | `backend/internal/server/routes/admin.go:323` | - |
| POST | `/api/v1/admin/sora/exchange-code` | AdminAuthMiddleware | 是 | `h.Admin.OpenAIOAuth.ExchangeCode` | `backend/internal/server/routes/admin.go:318` | - |
| POST | `/api/v1/admin/sora/generate-auth-url` | AdminAuthMiddleware | 是 | `h.Admin.OpenAIOAuth.GenerateAuthURL` | `backend/internal/server/routes/admin.go:317` | - |
| POST | `/api/v1/admin/sora/refresh-token` | AdminAuthMiddleware | 是 | `h.Admin.OpenAIOAuth.RefreshToken` | `backend/internal/server/routes/admin.go:319` | - |
| POST | `/api/v1/admin/sora/rt2at` | AdminAuthMiddleware | 是 | `h.Admin.OpenAIOAuth.RefreshToken` | `backend/internal/server/routes/admin.go:321` | - |
| POST | `/api/v1/admin/sora/st2at` | AdminAuthMiddleware | 是 | `h.Admin.OpenAIOAuth.ExchangeSoraSessionToken` | `backend/internal/server/routes/admin.go:320` | - |

## 代理管理

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/admin/proxies` | AdminAuthMiddleware | 是 | `h.Admin.Proxy.List` | `backend/internal/server/routes/admin.go:348` | - |
| POST | `/api/v1/admin/proxies` | AdminAuthMiddleware | 是 | `h.Admin.Proxy.Create` | `backend/internal/server/routes/admin.go:353` | - |
| GET | `/api/v1/admin/proxies/:id` | AdminAuthMiddleware | 是 | `h.Admin.Proxy.GetByID` | `backend/internal/server/routes/admin.go:352` | - |
| PUT | `/api/v1/admin/proxies/:id` | AdminAuthMiddleware | 是 | `h.Admin.Proxy.Update` | `backend/internal/server/routes/admin.go:354` | - |
| DELETE | `/api/v1/admin/proxies/:id` | AdminAuthMiddleware | 是 | `h.Admin.Proxy.Delete` | `backend/internal/server/routes/admin.go:355` | - |
| GET | `/api/v1/admin/proxies/:id/accounts` | AdminAuthMiddleware | 是 | `h.Admin.Proxy.GetProxyAccounts` | `backend/internal/server/routes/admin.go:359` | - |
| POST | `/api/v1/admin/proxies/:id/quality-check` | AdminAuthMiddleware | 是 | `h.Admin.Proxy.CheckQuality` | `backend/internal/server/routes/admin.go:357` | - |
| GET | `/api/v1/admin/proxies/:id/stats` | AdminAuthMiddleware | 是 | `h.Admin.Proxy.GetStats` | `backend/internal/server/routes/admin.go:358` | - |
| POST | `/api/v1/admin/proxies/:id/test` | AdminAuthMiddleware | 是 | `h.Admin.Proxy.Test` | `backend/internal/server/routes/admin.go:356` | - |
| GET | `/api/v1/admin/proxies/all` | AdminAuthMiddleware | 是 | `h.Admin.Proxy.GetAll` | `backend/internal/server/routes/admin.go:349` | - |
| POST | `/api/v1/admin/proxies/batch` | AdminAuthMiddleware | 是 | `h.Admin.Proxy.BatchCreate` | `backend/internal/server/routes/admin.go:361` | - |
| POST | `/api/v1/admin/proxies/batch-delete` | AdminAuthMiddleware | 是 | `h.Admin.Proxy.BatchDelete` | `backend/internal/server/routes/admin.go:360` | - |
| GET | `/api/v1/admin/proxies/data` | AdminAuthMiddleware | 是 | `h.Admin.Proxy.ExportData` | `backend/internal/server/routes/admin.go:350` | - |
| POST | `/api/v1/admin/proxies/data` | AdminAuthMiddleware | 是 | `h.Admin.Proxy.ImportData` | `backend/internal/server/routes/admin.go:351` | - |

## 账号管理

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/admin/accounts` | AdminAuthMiddleware | 是 | `h.Admin.Account.List` | `backend/internal/server/routes/admin.go:246` | - |
| POST | `/api/v1/admin/accounts` | AdminAuthMiddleware | 是 | `h.Admin.Account.Create` | `backend/internal/server/routes/admin.go:248` | - |
| GET | `/api/v1/admin/accounts/:id` | AdminAuthMiddleware | 是 | `h.Admin.Account.GetByID` | `backend/internal/server/routes/admin.go:247` | - |
| PUT | `/api/v1/admin/accounts/:id` | AdminAuthMiddleware | 是 | `h.Admin.Account.Update` | `backend/internal/server/routes/admin.go:252` | - |
| DELETE | `/api/v1/admin/accounts/:id` | AdminAuthMiddleware | 是 | `h.Admin.Account.Delete` | `backend/internal/server/routes/admin.go:253` | - |
| POST | `/api/v1/admin/accounts/:id/clear-error` | AdminAuthMiddleware | 是 | `h.Admin.Account.ClearError` | `backend/internal/server/routes/admin.go:259` | - |
| POST | `/api/v1/admin/accounts/:id/clear-rate-limit` | AdminAuthMiddleware | 是 | `h.Admin.Account.ClearRateLimit` | `backend/internal/server/routes/admin.go:263` | - |
| GET | `/api/v1/admin/accounts/:id/models` | AdminAuthMiddleware | 是 | `h.Admin.Account.GetAvailableModels` | `backend/internal/server/routes/admin.go:268` | - |
| POST | `/api/v1/admin/accounts/:id/recover-state` | AdminAuthMiddleware | 是 | `h.Admin.Account.RecoverState` | `backend/internal/server/routes/admin.go:255` | - |
| POST | `/api/v1/admin/accounts/:id/refresh` | AdminAuthMiddleware | 是 | `h.Admin.Account.Refresh` | `backend/internal/server/routes/admin.go:256` | - |
| POST | `/api/v1/admin/accounts/:id/refresh-tier` | AdminAuthMiddleware | 是 | `h.Admin.Account.RefreshTier` | `backend/internal/server/routes/admin.go:257` | - |
| POST | `/api/v1/admin/accounts/:id/reset-quota` | AdminAuthMiddleware | 是 | `h.Admin.Account.ResetQuota` | `backend/internal/server/routes/admin.go:264` | - |
| POST | `/api/v1/admin/accounts/:id/schedulable` | AdminAuthMiddleware | 是 | `h.Admin.Account.SetSchedulable` | `backend/internal/server/routes/admin.go:267` | - |
| GET | `/api/v1/admin/accounts/:id/scheduled-test-plans` | AdminAuthMiddleware | 是 | `h.Admin.ScheduledTest.ListByAccount` | `backend/internal/server/routes/admin.go:536` | - |
| GET | `/api/v1/admin/accounts/:id/stats` | AdminAuthMiddleware | 是 | `h.Admin.Account.GetStats` | `backend/internal/server/routes/admin.go:258` | - |
| GET | `/api/v1/admin/accounts/:id/temp-unschedulable` | AdminAuthMiddleware | 是 | `h.Admin.Account.GetTempUnschedulable` | `backend/internal/server/routes/admin.go:265` | - |
| DELETE | `/api/v1/admin/accounts/:id/temp-unschedulable` | AdminAuthMiddleware | 是 | `h.Admin.Account.ClearTempUnschedulable` | `backend/internal/server/routes/admin.go:266` | - |
| POST | `/api/v1/admin/accounts/:id/test` | AdminAuthMiddleware | 是 | `h.Admin.Account.Test` | `backend/internal/server/routes/admin.go:254` | - |
| GET | `/api/v1/admin/accounts/:id/today-stats` | AdminAuthMiddleware | 是 | `h.Admin.Account.GetTodayStats` | `backend/internal/server/routes/admin.go:261` | - |
| GET | `/api/v1/admin/accounts/:id/usage` | AdminAuthMiddleware | 是 | `h.Admin.Account.GetUsage` | `backend/internal/server/routes/admin.go:260` | - |
| GET | `/api/v1/admin/accounts/antigravity/default-model-mapping` | AdminAuthMiddleware | 是 | `h.Admin.Account.GetAntigravityDefaultModelMapping` | `backend/internal/server/routes/admin.go:279` | - |
| POST | `/api/v1/admin/accounts/batch` | AdminAuthMiddleware | 是 | `h.Admin.Account.BatchCreate` | `backend/internal/server/routes/admin.go:269` | - |
| POST | `/api/v1/admin/accounts/batch-clear-error` | AdminAuthMiddleware | 是 | `h.Admin.Account.BatchClearError` | `backend/internal/server/routes/admin.go:275` | - |
| POST | `/api/v1/admin/accounts/batch-refresh` | AdminAuthMiddleware | 是 | `h.Admin.Account.BatchRefresh` | `backend/internal/server/routes/admin.go:276` | - |
| POST | `/api/v1/admin/accounts/batch-refresh-tier` | AdminAuthMiddleware | 是 | `h.Admin.Account.BatchRefreshTier` | `backend/internal/server/routes/admin.go:273` | - |
| POST | `/api/v1/admin/accounts/batch-update-credentials` | AdminAuthMiddleware | 是 | `h.Admin.Account.BatchUpdateCredentials` | `backend/internal/server/routes/admin.go:272` | - |
| POST | `/api/v1/admin/accounts/bulk-update` | AdminAuthMiddleware | 是 | `h.Admin.Account.BulkUpdate` | `backend/internal/server/routes/admin.go:274` | - |
| POST | `/api/v1/admin/accounts/check-mixed-channel` | AdminAuthMiddleware | 是 | `h.Admin.Account.CheckMixedChannel` | `backend/internal/server/routes/admin.go:249` | 新增账号混用渠道风险检测接口 |
| POST | `/api/v1/admin/accounts/cookie-auth` | AdminAuthMiddleware | 是 | `h.Admin.OAuth.CookieAuth` | `backend/internal/server/routes/admin.go:286` | - |
| GET | `/api/v1/admin/accounts/data` | AdminAuthMiddleware | 是 | `h.Admin.Account.ExportData` | `backend/internal/server/routes/admin.go:270` | - |
| POST | `/api/v1/admin/accounts/data` | AdminAuthMiddleware | 是 | `h.Admin.Account.ImportData` | `backend/internal/server/routes/admin.go:271` | - |
| POST | `/api/v1/admin/accounts/exchange-code` | AdminAuthMiddleware | 是 | `h.Admin.OAuth.ExchangeCode` | `backend/internal/server/routes/admin.go:284` | - |
| POST | `/api/v1/admin/accounts/exchange-setup-token-code` | AdminAuthMiddleware | 是 | `h.Admin.OAuth.ExchangeSetupTokenCode` | `backend/internal/server/routes/admin.go:285` | - |
| POST | `/api/v1/admin/accounts/generate-auth-url` | AdminAuthMiddleware | 是 | `h.Admin.OAuth.GenerateAuthURL` | `backend/internal/server/routes/admin.go:282` | - |
| POST | `/api/v1/admin/accounts/generate-setup-token-url` | AdminAuthMiddleware | 是 | `h.Admin.OAuth.GenerateSetupTokenURL` | `backend/internal/server/routes/admin.go:283` | - |
| POST | `/api/v1/admin/accounts/setup-token-cookie-auth` | AdminAuthMiddleware | 是 | `h.Admin.OAuth.SetupTokenCookieAuth` | `backend/internal/server/routes/admin.go:287` | - |
| POST | `/api/v1/admin/accounts/sync/crs` | AdminAuthMiddleware | 是 | `h.Admin.Account.SyncFromCRS` | `backend/internal/server/routes/admin.go:250` | - |
| POST | `/api/v1/admin/accounts/sync/crs/preview` | AdminAuthMiddleware | 是 | `h.Admin.Account.PreviewFromCRS` | `backend/internal/server/routes/admin.go:251` | 新增 CRS 同步预览接口 |
| POST | `/api/v1/admin/accounts/today-stats/batch` | AdminAuthMiddleware | 是 | `h.Admin.Account.GetBatchTodayStats` | `backend/internal/server/routes/admin.go:262` | - |
