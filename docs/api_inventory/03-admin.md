# Admin 接口清单（当前代码版）

- 同步日期：**2026-03-18**
- Admin 接口总数：**265**
- 本文件只保留检索摘要；详细接口表见：`docs/api_full_reference_v2/04-admin-core.md` 到 `07-admin-misc.md`

## 分组统计

- Core：**68**
- Accounts：**71**
- Ops：**50**
- Misc：**76**

## 关键说明

- 所有 `/api/v1/admin/**` 都走 `AdminAuthMiddleware`。
- 所有 `/api/v1/admin/**` 都支持 `x-api-key: <admin-api-key>`。
- 新版已覆盖以下此前缺口较大的区域：
  - `dashboard/snapshot-v2`、`dashboard/groups`、`dashboard/users-ranking`、`dashboard/user-breakdown`
  - `accounts/check-mixed-channel`、`accounts/sync/crs/preview`、`accounts/:id/recover-state`
  - `ops/user-concurrency`、`ops/runtime/logging*`、`ops/system-logs*`
  - `settings/sora-s3*`
  - `data-management/*`
  - `backups/*`
  - `scheduled-test-plans/*`
  - `redeem-codes/create-and-redeem`

## 唯一详细文档

- Core：`docs/api_full_reference_v2/04-admin-core.md`
- Accounts：`docs/api_full_reference_v2/05-admin-accounts.md`
- Ops：`docs/api_full_reference_v2/06-admin-ops.md`
- Misc：`docs/api_full_reference_v2/07-admin-misc.md`
