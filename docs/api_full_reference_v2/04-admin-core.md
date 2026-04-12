# Admin Core 接口（当前代码版）

- 同步日期：**2026-03-18**
- 接口数量：**68**
- 文档策略：优先保证当前路由、鉴权、中间件与注册位置准确；字段级请求体请继续查对应 handler / dto。
- 本文件聚合 Dashboard、用户、分组、订阅、系统设置与 Admin API Key 归属调整接口。

## Admin API Key 归属调整

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| PUT | `/api/v1/admin/api-keys/:id` | AdminAuthMiddleware | 是 | `h.Admin.APIKey.UpdateGroup` | `backend/internal/server/routes/admin.go:93` | - |

## Dashboard

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| POST | `/api/v1/admin/dashboard/aggregation/backfill` | AdminAuthMiddleware | 是 | `h.Admin.Dashboard.BackfillAggregation` | `backend/internal/server/routes/admin.go:202` | - |
| GET | `/api/v1/admin/dashboard/api-keys-trend` | AdminAuthMiddleware | 是 | `h.Admin.Dashboard.GetAPIKeyUsageTrend` | `backend/internal/server/routes/admin.go:196` | - |
| POST | `/api/v1/admin/dashboard/api-keys-usage` | AdminAuthMiddleware | 是 | `h.Admin.Dashboard.GetBatchAPIKeysUsage` | `backend/internal/server/routes/admin.go:200` | - |
| GET | `/api/v1/admin/dashboard/groups` | AdminAuthMiddleware | 是 | `h.Admin.Dashboard.GetGroupStats` | `backend/internal/server/routes/admin.go:195` | - |
| GET | `/api/v1/admin/dashboard/models` | AdminAuthMiddleware | 是 | `h.Admin.Dashboard.GetModelStats` | `backend/internal/server/routes/admin.go:194` | - |
| GET | `/api/v1/admin/dashboard/realtime` | AdminAuthMiddleware | 是 | `h.Admin.Dashboard.GetRealtimeMetrics` | `backend/internal/server/routes/admin.go:192` | - |
| GET | `/api/v1/admin/dashboard/snapshot-v2` | AdminAuthMiddleware | 是 | `h.Admin.Dashboard.GetSnapshotV2` | `backend/internal/server/routes/admin.go:190` | - |
| GET | `/api/v1/admin/dashboard/stats` | AdminAuthMiddleware | 是 | `h.Admin.Dashboard.GetStats` | `backend/internal/server/routes/admin.go:191` | - |
| GET | `/api/v1/admin/dashboard/trend` | AdminAuthMiddleware | 是 | `h.Admin.Dashboard.GetUsageTrend` | `backend/internal/server/routes/admin.go:193` | - |
| GET | `/api/v1/admin/dashboard/user-breakdown` | AdminAuthMiddleware | 是 | `h.Admin.Dashboard.GetUserBreakdown` | `backend/internal/server/routes/admin.go:201` | - |
| GET | `/api/v1/admin/dashboard/users-ranking` | AdminAuthMiddleware | 是 | `h.Admin.Dashboard.GetUserSpendingRanking` | `backend/internal/server/routes/admin.go:198` | - |
| GET | `/api/v1/admin/dashboard/users-trend` | AdminAuthMiddleware | 是 | `h.Admin.Dashboard.GetUserUsageTrend` | `backend/internal/server/routes/admin.go:197` | - |
| POST | `/api/v1/admin/dashboard/users-usage` | AdminAuthMiddleware | 是 | `h.Admin.Dashboard.GetBatchUsersUsage` | `backend/internal/server/routes/admin.go:199` | - |

## 分组管理

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/admin/groups` | AdminAuthMiddleware | 是 | `h.Admin.Group.List` | `backend/internal/server/routes/admin.go:228` | - |
| POST | `/api/v1/admin/groups` | AdminAuthMiddleware | 是 | `h.Admin.Group.Create` | `backend/internal/server/routes/admin.go:232` | - |
| GET | `/api/v1/admin/groups/:id` | AdminAuthMiddleware | 是 | `h.Admin.Group.GetByID` | `backend/internal/server/routes/admin.go:231` | - |
| PUT | `/api/v1/admin/groups/:id` | AdminAuthMiddleware | 是 | `h.Admin.Group.Update` | `backend/internal/server/routes/admin.go:233` | - |
| DELETE | `/api/v1/admin/groups/:id` | AdminAuthMiddleware | 是 | `h.Admin.Group.Delete` | `backend/internal/server/routes/admin.go:234` | - |
| GET | `/api/v1/admin/groups/:id/api-keys` | AdminAuthMiddleware | 是 | `h.Admin.Group.GetGroupAPIKeys` | `backend/internal/server/routes/admin.go:239` | - |
| GET | `/api/v1/admin/groups/:id/rate-multipliers` | AdminAuthMiddleware | 是 | `h.Admin.Group.GetGroupRateMultipliers` | `backend/internal/server/routes/admin.go:236` | - |
| PUT | `/api/v1/admin/groups/:id/rate-multipliers` | AdminAuthMiddleware | 是 | `h.Admin.Group.BatchSetGroupRateMultipliers` | `backend/internal/server/routes/admin.go:237` | - |
| DELETE | `/api/v1/admin/groups/:id/rate-multipliers` | AdminAuthMiddleware | 是 | `h.Admin.Group.ClearGroupRateMultipliers` | `backend/internal/server/routes/admin.go:238` | - |
| GET | `/api/v1/admin/groups/:id/stats` | AdminAuthMiddleware | 是 | `h.Admin.Group.GetStats` | `backend/internal/server/routes/admin.go:235` | - |
| GET | `/api/v1/admin/groups/:id/subscriptions` | AdminAuthMiddleware | 是 | `h.Admin.Subscription.ListByGroup` | `backend/internal/server/routes/admin.go:496` | - |
| GET | `/api/v1/admin/groups/all` | AdminAuthMiddleware | 是 | `h.Admin.Group.GetAll` | `backend/internal/server/routes/admin.go:229` | - |
| PUT | `/api/v1/admin/groups/sort-order` | AdminAuthMiddleware | 是 | `h.Admin.Group.UpdateSortOrder` | `backend/internal/server/routes/admin.go:230` | - |

## 用户管理

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/admin/users` | AdminAuthMiddleware | 是 | `h.Admin.User.List` | `backend/internal/server/routes/admin.go:209` | - |
| POST | `/api/v1/admin/users` | AdminAuthMiddleware | 是 | `h.Admin.User.Create` | `backend/internal/server/routes/admin.go:211` | - |
| GET | `/api/v1/admin/users/:id` | AdminAuthMiddleware | 是 | `h.Admin.User.GetByID` | `backend/internal/server/routes/admin.go:210` | - |
| PUT | `/api/v1/admin/users/:id` | AdminAuthMiddleware | 是 | `h.Admin.User.Update` | `backend/internal/server/routes/admin.go:212` | - |
| DELETE | `/api/v1/admin/users/:id` | AdminAuthMiddleware | 是 | `h.Admin.User.Delete` | `backend/internal/server/routes/admin.go:213` | - |
| GET | `/api/v1/admin/users/:id/api-keys` | AdminAuthMiddleware | 是 | `h.Admin.User.GetUserAPIKeys` | `backend/internal/server/routes/admin.go:215` | - |
| GET | `/api/v1/admin/users/:id/attributes` | AdminAuthMiddleware | 是 | `h.Admin.UserAttribute.GetUserAttributes` | `backend/internal/server/routes/admin.go:220` | - |
| PUT | `/api/v1/admin/users/:id/attributes` | AdminAuthMiddleware | 是 | `h.Admin.UserAttribute.UpdateUserAttributes` | `backend/internal/server/routes/admin.go:221` | - |
| POST | `/api/v1/admin/users/:id/balance` | AdminAuthMiddleware | 是 | `h.Admin.User.UpdateBalance` | `backend/internal/server/routes/admin.go:214` | 支持 set / add / subtract，建议始终发送 `Idempotency-Key` |
| GET | `/api/v1/admin/users/:id/balance-history` | AdminAuthMiddleware | 是 | `h.Admin.User.GetBalanceHistory` | `backend/internal/server/routes/admin.go:217` | - |
| GET | `/api/v1/admin/users/:id/subscriptions` | AdminAuthMiddleware | 是 | `h.Admin.Subscription.ListByUser` | `backend/internal/server/routes/admin.go:499` | - |
| GET | `/api/v1/admin/users/:id/usage` | AdminAuthMiddleware | 是 | `h.Admin.User.GetUserUsage` | `backend/internal/server/routes/admin.go:216` | - |

## 系统设置

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/admin/settings` | AdminAuthMiddleware | 是 | `h.Admin.Setting.GetSettings` | `backend/internal/server/routes/admin.go:395` | - |
| PUT | `/api/v1/admin/settings` | AdminAuthMiddleware | 是 | `h.Admin.Setting.UpdateSettings` | `backend/internal/server/routes/admin.go:396` | - |
| GET | `/api/v1/admin/settings/admin-api-key` | AdminAuthMiddleware | 是 | `h.Admin.Setting.GetAdminAPIKey` | `backend/internal/server/routes/admin.go:400` | - |
| DELETE | `/api/v1/admin/settings/admin-api-key` | AdminAuthMiddleware | 是 | `h.Admin.Setting.DeleteAdminAPIKey` | `backend/internal/server/routes/admin.go:402` | - |
| POST | `/api/v1/admin/settings/admin-api-key/regenerate` | AdminAuthMiddleware | 是 | `h.Admin.Setting.RegenerateAdminAPIKey` | `backend/internal/server/routes/admin.go:401` | - |
| GET | `/api/v1/admin/settings/beta-policy` | AdminAuthMiddleware | 是 | `h.Admin.Setting.GetBetaPolicySettings` | `backend/internal/server/routes/admin.go:410` | - |
| PUT | `/api/v1/admin/settings/beta-policy` | AdminAuthMiddleware | 是 | `h.Admin.Setting.UpdateBetaPolicySettings` | `backend/internal/server/routes/admin.go:411` | - |
| GET | `/api/v1/admin/settings/rectifier` | AdminAuthMiddleware | 是 | `h.Admin.Setting.GetRectifierSettings` | `backend/internal/server/routes/admin.go:407` | - |
| PUT | `/api/v1/admin/settings/rectifier` | AdminAuthMiddleware | 是 | `h.Admin.Setting.UpdateRectifierSettings` | `backend/internal/server/routes/admin.go:408` | - |
| POST | `/api/v1/admin/settings/send-test-email` | AdminAuthMiddleware | 是 | `h.Admin.Setting.SendTestEmail` | `backend/internal/server/routes/admin.go:398` | - |
| GET | `/api/v1/admin/settings/sora-s3` | AdminAuthMiddleware | 是 | `h.Admin.Setting.GetSoraS3Settings` | `backend/internal/server/routes/admin.go:413` | - |
| PUT | `/api/v1/admin/settings/sora-s3` | AdminAuthMiddleware | 是 | `h.Admin.Setting.UpdateSoraS3Settings` | `backend/internal/server/routes/admin.go:414` | - |
| GET | `/api/v1/admin/settings/sora-s3/profiles` | AdminAuthMiddleware | 是 | `h.Admin.Setting.ListSoraS3Profiles` | `backend/internal/server/routes/admin.go:416` | - |
| POST | `/api/v1/admin/settings/sora-s3/profiles` | AdminAuthMiddleware | 是 | `h.Admin.Setting.CreateSoraS3Profile` | `backend/internal/server/routes/admin.go:417` | - |
| PUT | `/api/v1/admin/settings/sora-s3/profiles/:profile_id` | AdminAuthMiddleware | 是 | `h.Admin.Setting.UpdateSoraS3Profile` | `backend/internal/server/routes/admin.go:418` | - |
| DELETE | `/api/v1/admin/settings/sora-s3/profiles/:profile_id` | AdminAuthMiddleware | 是 | `h.Admin.Setting.DeleteSoraS3Profile` | `backend/internal/server/routes/admin.go:419` | - |
| POST | `/api/v1/admin/settings/sora-s3/profiles/:profile_id/activate` | AdminAuthMiddleware | 是 | `h.Admin.Setting.SetActiveSoraS3Profile` | `backend/internal/server/routes/admin.go:420` | - |
| POST | `/api/v1/admin/settings/sora-s3/test` | AdminAuthMiddleware | 是 | `h.Admin.Setting.TestSoraS3Connection` | `backend/internal/server/routes/admin.go:415` | - |
| GET | `/api/v1/admin/settings/stream-timeout` | AdminAuthMiddleware | 是 | `h.Admin.Setting.GetStreamTimeoutSettings` | `backend/internal/server/routes/admin.go:404` | - |
| PUT | `/api/v1/admin/settings/stream-timeout` | AdminAuthMiddleware | 是 | `h.Admin.Setting.UpdateStreamTimeoutSettings` | `backend/internal/server/routes/admin.go:405` | - |
| POST | `/api/v1/admin/settings/test-smtp` | AdminAuthMiddleware | 是 | `h.Admin.Setting.TestSMTPConnection` | `backend/internal/server/routes/admin.go:397` | - |

## 订阅管理

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/admin/subscriptions` | AdminAuthMiddleware | 是 | `h.Admin.Subscription.List` | `backend/internal/server/routes/admin.go:485` | - |
| GET | `/api/v1/admin/subscriptions/:id` | AdminAuthMiddleware | 是 | `h.Admin.Subscription.GetByID` | `backend/internal/server/routes/admin.go:486` | - |
| DELETE | `/api/v1/admin/subscriptions/:id` | AdminAuthMiddleware | 是 | `h.Admin.Subscription.Revoke` | `backend/internal/server/routes/admin.go:492` | - |
| POST | `/api/v1/admin/subscriptions/:id/extend` | AdminAuthMiddleware | 是 | `h.Admin.Subscription.Extend` | `backend/internal/server/routes/admin.go:490` | - |
| GET | `/api/v1/admin/subscriptions/:id/progress` | AdminAuthMiddleware | 是 | `h.Admin.Subscription.GetProgress` | `backend/internal/server/routes/admin.go:487` | - |
| POST | `/api/v1/admin/subscriptions/:id/reset-quota` | AdminAuthMiddleware | 是 | `h.Admin.Subscription.ResetQuota` | `backend/internal/server/routes/admin.go:491` | - |
| POST | `/api/v1/admin/subscriptions/assign` | AdminAuthMiddleware | 是 | `h.Admin.Subscription.Assign` | `backend/internal/server/routes/admin.go:488` | - |
| POST | `/api/v1/admin/subscriptions/bulk-assign` | AdminAuthMiddleware | 是 | `h.Admin.Subscription.BulkAssign` | `backend/internal/server/routes/admin.go:489` | - |
