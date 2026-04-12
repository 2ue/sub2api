# Admin Misc 接口（当前代码版）

- 同步日期：**2026-03-18**
- 接口数量：**76**
- 文档策略：优先保证当前路由、鉴权、中间件与注册位置准确；字段级请求体请继续查对应 handler / dto。
- 本文件聚合公告、兑换码、优惠码、数据管理、备份、系统、用量、用户属性、定时测试、错误透传规则接口。

## 优惠码管理

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/admin/promo-codes` | AdminAuthMiddleware | 是 | `h.Admin.Promo.List` | `backend/internal/server/routes/admin.go:383` | - |
| POST | `/api/v1/admin/promo-codes` | AdminAuthMiddleware | 是 | `h.Admin.Promo.Create` | `backend/internal/server/routes/admin.go:385` | - |
| GET | `/api/v1/admin/promo-codes/:id` | AdminAuthMiddleware | 是 | `h.Admin.Promo.GetByID` | `backend/internal/server/routes/admin.go:384` | - |
| PUT | `/api/v1/admin/promo-codes/:id` | AdminAuthMiddleware | 是 | `h.Admin.Promo.Update` | `backend/internal/server/routes/admin.go:386` | - |
| DELETE | `/api/v1/admin/promo-codes/:id` | AdminAuthMiddleware | 是 | `h.Admin.Promo.Delete` | `backend/internal/server/routes/admin.go:387` | - |
| GET | `/api/v1/admin/promo-codes/:id/usages` | AdminAuthMiddleware | 是 | `h.Admin.Promo.GetUsages` | `backend/internal/server/routes/admin.go:388` | - |

## 兑换码管理

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/admin/redeem-codes` | AdminAuthMiddleware | 是 | `h.Admin.Redeem.List` | `backend/internal/server/routes/admin.go:368` | - |
| GET | `/api/v1/admin/redeem-codes/:id` | AdminAuthMiddleware | 是 | `h.Admin.Redeem.GetByID` | `backend/internal/server/routes/admin.go:371` | - |
| DELETE | `/api/v1/admin/redeem-codes/:id` | AdminAuthMiddleware | 是 | `h.Admin.Redeem.Delete` | `backend/internal/server/routes/admin.go:374` | - |
| POST | `/api/v1/admin/redeem-codes/:id/expire` | AdminAuthMiddleware | 是 | `h.Admin.Redeem.Expire` | `backend/internal/server/routes/admin.go:376` | - |
| POST | `/api/v1/admin/redeem-codes/batch-delete` | AdminAuthMiddleware | 是 | `h.Admin.Redeem.BatchDelete` | `backend/internal/server/routes/admin.go:375` | - |
| POST | `/api/v1/admin/redeem-codes/create-and-redeem` | AdminAuthMiddleware | 是 | `h.Admin.Redeem.CreateAndRedeem` | `backend/internal/server/routes/admin.go:372` | 支持固定兑换码创建并原子兑换到指定用户，要求 `Idempotency-Key`（启用强制幂等时） |
| GET | `/api/v1/admin/redeem-codes/export` | AdminAuthMiddleware | 是 | `h.Admin.Redeem.Export` | `backend/internal/server/routes/admin.go:370` | - |
| POST | `/api/v1/admin/redeem-codes/generate` | AdminAuthMiddleware | 是 | `h.Admin.Redeem.Generate` | `backend/internal/server/routes/admin.go:373` | - |
| GET | `/api/v1/admin/redeem-codes/stats` | AdminAuthMiddleware | 是 | `h.Admin.Redeem.GetStats` | `backend/internal/server/routes/admin.go:369` | - |

## 公告管理

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/admin/announcements` | AdminAuthMiddleware | 是 | `h.Admin.Announcement.List` | `backend/internal/server/routes/admin.go:294` | - |
| POST | `/api/v1/admin/announcements` | AdminAuthMiddleware | 是 | `h.Admin.Announcement.Create` | `backend/internal/server/routes/admin.go:295` | - |
| GET | `/api/v1/admin/announcements/:id` | AdminAuthMiddleware | 是 | `h.Admin.Announcement.GetByID` | `backend/internal/server/routes/admin.go:296` | - |
| PUT | `/api/v1/admin/announcements/:id` | AdminAuthMiddleware | 是 | `h.Admin.Announcement.Update` | `backend/internal/server/routes/admin.go:297` | - |
| DELETE | `/api/v1/admin/announcements/:id` | AdminAuthMiddleware | 是 | `h.Admin.Announcement.Delete` | `backend/internal/server/routes/admin.go:298` | - |
| GET | `/api/v1/admin/announcements/:id/read-status` | AdminAuthMiddleware | 是 | `h.Admin.Announcement.ListReadStatus` | `backend/internal/server/routes/admin.go:299` | - |

## 备份

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/admin/backups` | AdminAuthMiddleware | 是 | `h.Admin.Backup.ListBackups` | `backend/internal/server/routes/admin.go:461` | - |
| POST | `/api/v1/admin/backups` | AdminAuthMiddleware | 是 | `h.Admin.Backup.CreateBackup` | `backend/internal/server/routes/admin.go:460` | - |
| GET | `/api/v1/admin/backups/:id` | AdminAuthMiddleware | 是 | `h.Admin.Backup.GetBackup` | `backend/internal/server/routes/admin.go:462` | - |
| DELETE | `/api/v1/admin/backups/:id` | AdminAuthMiddleware | 是 | `h.Admin.Backup.DeleteBackup` | `backend/internal/server/routes/admin.go:463` | - |
| GET | `/api/v1/admin/backups/:id/download-url` | AdminAuthMiddleware | 是 | `h.Admin.Backup.GetDownloadURL` | `backend/internal/server/routes/admin.go:464` | - |
| POST | `/api/v1/admin/backups/:id/restore` | AdminAuthMiddleware | 是 | `h.Admin.Backup.RestoreBackup` | `backend/internal/server/routes/admin.go:467` | - |
| GET | `/api/v1/admin/backups/s3-config` | AdminAuthMiddleware | 是 | `h.Admin.Backup.GetS3Config` | `backend/internal/server/routes/admin.go:451` | - |
| PUT | `/api/v1/admin/backups/s3-config` | AdminAuthMiddleware | 是 | `h.Admin.Backup.UpdateS3Config` | `backend/internal/server/routes/admin.go:452` | - |
| POST | `/api/v1/admin/backups/s3-config/test` | AdminAuthMiddleware | 是 | `h.Admin.Backup.TestS3Connection` | `backend/internal/server/routes/admin.go:453` | - |
| GET | `/api/v1/admin/backups/schedule` | AdminAuthMiddleware | 是 | `h.Admin.Backup.GetSchedule` | `backend/internal/server/routes/admin.go:456` | - |
| PUT | `/api/v1/admin/backups/schedule` | AdminAuthMiddleware | 是 | `h.Admin.Backup.UpdateSchedule` | `backend/internal/server/routes/admin.go:457` | - |

## 定时测试

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| POST | `/api/v1/admin/scheduled-test-plans` | AdminAuthMiddleware | 是 | `h.Admin.ScheduledTest.Create` | `backend/internal/server/routes/admin.go:530` | - |
| PUT | `/api/v1/admin/scheduled-test-plans/:id` | AdminAuthMiddleware | 是 | `h.Admin.ScheduledTest.Update` | `backend/internal/server/routes/admin.go:531` | - |
| DELETE | `/api/v1/admin/scheduled-test-plans/:id` | AdminAuthMiddleware | 是 | `h.Admin.ScheduledTest.Delete` | `backend/internal/server/routes/admin.go:532` | - |
| GET | `/api/v1/admin/scheduled-test-plans/:id/results` | AdminAuthMiddleware | 是 | `h.Admin.ScheduledTest.ListResults` | `backend/internal/server/routes/admin.go:533` | - |

## 数据管理

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/admin/data-management/agent/health` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.GetAgentHealth` | `backend/internal/server/routes/admin.go:427` | - |
| GET | `/api/v1/admin/data-management/backups` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.ListBackupJobs` | `backend/internal/server/routes/admin.go:442` | - |
| POST | `/api/v1/admin/data-management/backups` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.CreateBackupJob` | `backend/internal/server/routes/admin.go:441` | - |
| GET | `/api/v1/admin/data-management/backups/:job_id` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.GetBackupJob` | `backend/internal/server/routes/admin.go:443` | - |
| GET | `/api/v1/admin/data-management/config` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.GetConfig` | `backend/internal/server/routes/admin.go:428` | - |
| PUT | `/api/v1/admin/data-management/config` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.UpdateConfig` | `backend/internal/server/routes/admin.go:429` | - |
| GET | `/api/v1/admin/data-management/s3/profiles` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.ListS3Profiles` | `backend/internal/server/routes/admin.go:436` | - |
| POST | `/api/v1/admin/data-management/s3/profiles` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.CreateS3Profile` | `backend/internal/server/routes/admin.go:437` | - |
| PUT | `/api/v1/admin/data-management/s3/profiles/:profile_id` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.UpdateS3Profile` | `backend/internal/server/routes/admin.go:438` | - |
| DELETE | `/api/v1/admin/data-management/s3/profiles/:profile_id` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.DeleteS3Profile` | `backend/internal/server/routes/admin.go:439` | - |
| POST | `/api/v1/admin/data-management/s3/profiles/:profile_id/activate` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.SetActiveS3Profile` | `backend/internal/server/routes/admin.go:440` | - |
| POST | `/api/v1/admin/data-management/s3/test` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.TestS3` | `backend/internal/server/routes/admin.go:435` | - |
| GET | `/api/v1/admin/data-management/sources/:source_type/profiles` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.ListSourceProfiles` | `backend/internal/server/routes/admin.go:430` | - |
| POST | `/api/v1/admin/data-management/sources/:source_type/profiles` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.CreateSourceProfile` | `backend/internal/server/routes/admin.go:431` | - |
| PUT | `/api/v1/admin/data-management/sources/:source_type/profiles/:profile_id` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.UpdateSourceProfile` | `backend/internal/server/routes/admin.go:432` | - |
| DELETE | `/api/v1/admin/data-management/sources/:source_type/profiles/:profile_id` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.DeleteSourceProfile` | `backend/internal/server/routes/admin.go:433` | - |
| POST | `/api/v1/admin/data-management/sources/:source_type/profiles/:profile_id/activate` | AdminAuthMiddleware | 是 | `h.Admin.DataManagement.SetActiveSourceProfile` | `backend/internal/server/routes/admin.go:434` | - |

## 用户属性

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/admin/user-attributes` | AdminAuthMiddleware | 是 | `h.Admin.UserAttribute.ListDefinitions` | `backend/internal/server/routes/admin.go:518` | - |
| POST | `/api/v1/admin/user-attributes` | AdminAuthMiddleware | 是 | `h.Admin.UserAttribute.CreateDefinition` | `backend/internal/server/routes/admin.go:519` | - |
| PUT | `/api/v1/admin/user-attributes/:id` | AdminAuthMiddleware | 是 | `h.Admin.UserAttribute.UpdateDefinition` | `backend/internal/server/routes/admin.go:522` | - |
| DELETE | `/api/v1/admin/user-attributes/:id` | AdminAuthMiddleware | 是 | `h.Admin.UserAttribute.DeleteDefinition` | `backend/internal/server/routes/admin.go:523` | - |
| POST | `/api/v1/admin/user-attributes/batch` | AdminAuthMiddleware | 是 | `h.Admin.UserAttribute.GetBatchUserAttributes` | `backend/internal/server/routes/admin.go:520` | - |
| PUT | `/api/v1/admin/user-attributes/reorder` | AdminAuthMiddleware | 是 | `h.Admin.UserAttribute.ReorderDefinitions` | `backend/internal/server/routes/admin.go:521` | - |

## 用量管理

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/admin/usage` | AdminAuthMiddleware | 是 | `h.Admin.Usage.List` | `backend/internal/server/routes/admin.go:505` | - |
| GET | `/api/v1/admin/usage/cleanup-tasks` | AdminAuthMiddleware | 是 | `h.Admin.Usage.ListCleanupTasks` | `backend/internal/server/routes/admin.go:509` | - |
| POST | `/api/v1/admin/usage/cleanup-tasks` | AdminAuthMiddleware | 是 | `h.Admin.Usage.CreateCleanupTask` | `backend/internal/server/routes/admin.go:510` | - |
| POST | `/api/v1/admin/usage/cleanup-tasks/:id/cancel` | AdminAuthMiddleware | 是 | `h.Admin.Usage.CancelCleanupTask` | `backend/internal/server/routes/admin.go:511` | - |
| GET | `/api/v1/admin/usage/search-api-keys` | AdminAuthMiddleware | 是 | `h.Admin.Usage.SearchAPIKeys` | `backend/internal/server/routes/admin.go:508` | - |
| GET | `/api/v1/admin/usage/search-users` | AdminAuthMiddleware | 是 | `h.Admin.Usage.SearchUsers` | `backend/internal/server/routes/admin.go:507` | - |
| GET | `/api/v1/admin/usage/stats` | AdminAuthMiddleware | 是 | `h.Admin.Usage.Stats` | `backend/internal/server/routes/admin.go:506` | - |

## 系统运维

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/admin/system/check-updates` | AdminAuthMiddleware | 是 | `h.Admin.System.CheckUpdates` | `backend/internal/server/routes/admin.go:475` | - |
| POST | `/api/v1/admin/system/restart` | AdminAuthMiddleware | 是 | `h.Admin.System.RestartService` | `backend/internal/server/routes/admin.go:478` | - |
| POST | `/api/v1/admin/system/rollback` | AdminAuthMiddleware | 是 | `h.Admin.System.Rollback` | `backend/internal/server/routes/admin.go:477` | - |
| POST | `/api/v1/admin/system/update` | AdminAuthMiddleware | 是 | `h.Admin.System.PerformUpdate` | `backend/internal/server/routes/admin.go:476` | - |
| GET | `/api/v1/admin/system/version` | AdminAuthMiddleware | 是 | `h.Admin.System.GetVersion` | `backend/internal/server/routes/admin.go:474` | - |

## 错误透传规则

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/v1/admin/error-passthrough-rules` | AdminAuthMiddleware | 是 | `h.Admin.ErrorPassthrough.List` | `backend/internal/server/routes/admin.go:542` | - |
| POST | `/api/v1/admin/error-passthrough-rules` | AdminAuthMiddleware | 是 | `h.Admin.ErrorPassthrough.Create` | `backend/internal/server/routes/admin.go:544` | - |
| GET | `/api/v1/admin/error-passthrough-rules/:id` | AdminAuthMiddleware | 是 | `h.Admin.ErrorPassthrough.GetByID` | `backend/internal/server/routes/admin.go:543` | - |
| PUT | `/api/v1/admin/error-passthrough-rules/:id` | AdminAuthMiddleware | 是 | `h.Admin.ErrorPassthrough.Update` | `backend/internal/server/routes/admin.go:545` | - |
| DELETE | `/api/v1/admin/error-passthrough-rules/:id` | AdminAuthMiddleware | 是 | `h.Admin.ErrorPassthrough.Delete` | `backend/internal/server/routes/admin.go:546` | - |
