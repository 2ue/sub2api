# Admin Ops 接口（当前代码版）

- 同步日期：**2026-03-18**
- 接口数量：**50**
- 文档策略：优先保证当前路由、鉴权、中间件与注册位置准确；字段级请求体请继续查对应 handler / dto。
- 本文件聚合 `/api/v1/admin/ops/**`。
- 已补入 `user-concurrency`、runtime logging、system-logs、snapshot-v2、openai-token-stats 等新增接口。

## Ops 运维

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/admin/ops/account-availability` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetAccountAvailability` | `backend/internal/server/routes/admin.go:103` | - |
| GET | `/api/v1/admin/ops/advanced-settings` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetAdvancedSettings` | `backend/internal/server/routes/admin.go:131` | - |
| PUT | `/api/v1/admin/ops/advanced-settings` | AdminAuthMiddleware | 是 | `h.Admin.Ops.UpdateAdvancedSettings` | `backend/internal/server/routes/admin.go:132` | - |
| GET | `/api/v1/admin/ops/alert-events` | AdminAuthMiddleware | 是 | `h.Admin.Ops.ListAlertEvents` | `backend/internal/server/routes/admin.go:111` | - |
| GET | `/api/v1/admin/ops/alert-events/:id` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetAlertEvent` | `backend/internal/server/routes/admin.go:112` | - |
| PUT | `/api/v1/admin/ops/alert-events/:id/status` | AdminAuthMiddleware | 是 | `h.Admin.Ops.UpdateAlertEventStatus` | `backend/internal/server/routes/admin.go:113` | - |
| GET | `/api/v1/admin/ops/alert-rules` | AdminAuthMiddleware | 是 | `h.Admin.Ops.ListAlertRules` | `backend/internal/server/routes/admin.go:107` | - |
| POST | `/api/v1/admin/ops/alert-rules` | AdminAuthMiddleware | 是 | `h.Admin.Ops.CreateAlertRule` | `backend/internal/server/routes/admin.go:108` | - |
| PUT | `/api/v1/admin/ops/alert-rules/:id` | AdminAuthMiddleware | 是 | `h.Admin.Ops.UpdateAlertRule` | `backend/internal/server/routes/admin.go:109` | - |
| DELETE | `/api/v1/admin/ops/alert-rules/:id` | AdminAuthMiddleware | 是 | `h.Admin.Ops.DeleteAlertRule` | `backend/internal/server/routes/admin.go:110` | - |
| POST | `/api/v1/admin/ops/alert-silences` | AdminAuthMiddleware | 是 | `h.Admin.Ops.CreateAlertSilence` | `backend/internal/server/routes/admin.go:114` | - |
| GET | `/api/v1/admin/ops/concurrency` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetConcurrencyStats` | `backend/internal/server/routes/admin.go:101` | - |
| GET | `/api/v1/admin/ops/dashboard/error-distribution` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetDashboardErrorDistribution` | `backend/internal/server/routes/admin.go:182` | - |
| GET | `/api/v1/admin/ops/dashboard/error-trend` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetDashboardErrorTrend` | `backend/internal/server/routes/admin.go:181` | - |
| GET | `/api/v1/admin/ops/dashboard/latency-histogram` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetDashboardLatencyHistogram` | `backend/internal/server/routes/admin.go:180` | - |
| GET | `/api/v1/admin/ops/dashboard/openai-token-stats` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetDashboardOpenAITokenStats` | `backend/internal/server/routes/admin.go:183` | - |
| GET | `/api/v1/admin/ops/dashboard/overview` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetDashboardOverview` | `backend/internal/server/routes/admin.go:178` | - |
| GET | `/api/v1/admin/ops/dashboard/snapshot-v2` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetDashboardSnapshotV2` | `backend/internal/server/routes/admin.go:177` | - |
| GET | `/api/v1/admin/ops/dashboard/throughput-trend` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetDashboardThroughputTrend` | `backend/internal/server/routes/admin.go:179` | - |
| GET | `/api/v1/admin/ops/email-notification/config` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetEmailNotificationConfig` | `backend/internal/server/routes/admin.go:117` | - |
| PUT | `/api/v1/admin/ops/email-notification/config` | AdminAuthMiddleware | 是 | `h.Admin.Ops.UpdateEmailNotificationConfig` | `backend/internal/server/routes/admin.go:118` | - |
| GET | `/api/v1/admin/ops/errors` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetErrorLogs` | `backend/internal/server/routes/admin.go:148` | - |
| GET | `/api/v1/admin/ops/errors/:id` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetErrorLogByID` | `backend/internal/server/routes/admin.go:149` | - |
| PUT | `/api/v1/admin/ops/errors/:id/resolve` | AdminAuthMiddleware | 是 | `h.Admin.Ops.UpdateErrorResolution` | `backend/internal/server/routes/admin.go:152` | - |
| GET | `/api/v1/admin/ops/errors/:id/retries` | AdminAuthMiddleware | 是 | `h.Admin.Ops.ListRetryAttempts` | `backend/internal/server/routes/admin.go:150` | - |
| POST | `/api/v1/admin/ops/errors/:id/retry` | AdminAuthMiddleware | 是 | `h.Admin.Ops.RetryErrorRequest` | `backend/internal/server/routes/admin.go:151` | - |
| GET | `/api/v1/admin/ops/realtime-traffic` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetRealtimeTrafficSummary` | `backend/internal/server/routes/admin.go:104` | - |
| GET | `/api/v1/admin/ops/request-errors` | AdminAuthMiddleware | 是 | `h.Admin.Ops.ListRequestErrors` | `backend/internal/server/routes/admin.go:155` | - |
| GET | `/api/v1/admin/ops/request-errors/:id` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetRequestError` | `backend/internal/server/routes/admin.go:156` | - |
| PUT | `/api/v1/admin/ops/request-errors/:id/resolve` | AdminAuthMiddleware | 是 | `h.Admin.Ops.ResolveRequestError` | `backend/internal/server/routes/admin.go:160` | - |
| POST | `/api/v1/admin/ops/request-errors/:id/retry-client` | AdminAuthMiddleware | 是 | `h.Admin.Ops.RetryRequestErrorClient` | `backend/internal/server/routes/admin.go:158` | - |
| GET | `/api/v1/admin/ops/request-errors/:id/upstream-errors` | AdminAuthMiddleware | 是 | `h.Admin.Ops.ListRequestErrorUpstreamErrors` | `backend/internal/server/routes/admin.go:157` | - |
| POST | `/api/v1/admin/ops/request-errors/:id/upstream-errors/:idx/retry` | AdminAuthMiddleware | 是 | `h.Admin.Ops.RetryRequestErrorUpstreamEvent` | `backend/internal/server/routes/admin.go:159` | - |
| GET | `/api/v1/admin/ops/requests` | AdminAuthMiddleware | 是 | `h.Admin.Ops.ListRequestDetails` | `backend/internal/server/routes/admin.go:169` | - |
| GET | `/api/v1/admin/ops/runtime/alert` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetAlertRuntimeSettings` | `backend/internal/server/routes/admin.go:123` | - |
| PUT | `/api/v1/admin/ops/runtime/alert` | AdminAuthMiddleware | 是 | `h.Admin.Ops.UpdateAlertRuntimeSettings` | `backend/internal/server/routes/admin.go:124` | - |
| GET | `/api/v1/admin/ops/runtime/logging` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetRuntimeLogConfig` | `backend/internal/server/routes/admin.go:125` | - |
| PUT | `/api/v1/admin/ops/runtime/logging` | AdminAuthMiddleware | 是 | `h.Admin.Ops.UpdateRuntimeLogConfig` | `backend/internal/server/routes/admin.go:126` | - |
| POST | `/api/v1/admin/ops/runtime/logging/reset` | AdminAuthMiddleware | 是 | `h.Admin.Ops.ResetRuntimeLogConfig` | `backend/internal/server/routes/admin.go:127` | 新增运行时日志配置重置接口 |
| GET | `/api/v1/admin/ops/settings/metric-thresholds` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetMetricThresholds` | `backend/internal/server/routes/admin.go:137` | - |
| PUT | `/api/v1/admin/ops/settings/metric-thresholds` | AdminAuthMiddleware | 是 | `h.Admin.Ops.UpdateMetricThresholds` | `backend/internal/server/routes/admin.go:138` | - |
| GET | `/api/v1/admin/ops/system-logs` | AdminAuthMiddleware | 是 | `h.Admin.Ops.ListSystemLogs` | `backend/internal/server/routes/admin.go:172` | - |
| POST | `/api/v1/admin/ops/system-logs/cleanup` | AdminAuthMiddleware | 是 | `h.Admin.Ops.CleanupSystemLogs` | `backend/internal/server/routes/admin.go:173` | - |
| GET | `/api/v1/admin/ops/system-logs/health` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetSystemLogIngestionHealth` | `backend/internal/server/routes/admin.go:174` | - |
| GET | `/api/v1/admin/ops/upstream-errors` | AdminAuthMiddleware | 是 | `h.Admin.Ops.ListUpstreamErrors` | `backend/internal/server/routes/admin.go:163` | - |
| GET | `/api/v1/admin/ops/upstream-errors/:id` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetUpstreamError` | `backend/internal/server/routes/admin.go:164` | - |
| PUT | `/api/v1/admin/ops/upstream-errors/:id/resolve` | AdminAuthMiddleware | 是 | `h.Admin.Ops.ResolveUpstreamError` | `backend/internal/server/routes/admin.go:166` | - |
| POST | `/api/v1/admin/ops/upstream-errors/:id/retry` | AdminAuthMiddleware | 是 | `h.Admin.Ops.RetryUpstreamError` | `backend/internal/server/routes/admin.go:165` | - |
| GET | `/api/v1/admin/ops/user-concurrency` | AdminAuthMiddleware | 是 | `h.Admin.Ops.GetUserConcurrencyStats` | `backend/internal/server/routes/admin.go:102` | 新增用户级并发统计接口 |
| GET | `/api/v1/admin/ops/ws/qps` | AdminAuthMiddleware | 是 | `h.Admin.Ops.QPSWSHandler` | `backend/internal/server/routes/admin.go:144` | - |
