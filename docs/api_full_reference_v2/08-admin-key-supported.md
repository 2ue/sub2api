# 支持 Admin-Key 的接口（当前代码版）

- 同步日期：**2026-03-18**
- 接口数量：**265**
- 结论：所有 `/api/v1/admin/**` 路由都走 `AdminAuthMiddleware`，因此都支持 `x-api-key: <admin-api-key>`。
- 也支持：`Authorization: Bearer <admin-jwt>`。

## 范围结论

- 支持 Admin-Key：
  - `/api/v1/admin/**`
- 不支持 Admin-Key：
  - `/v1/**`、`/responses`、`/chat/completions` 等网关路由
  - `/api/v1/auth/**`、`/api/v1/user/**`、`/api/v1/keys/**` 等用户侧路由
  - `/sora/media-signed/*filepath` 等公开路由

## 分册映射

- `04-admin-core.md`：68 条
- `05-admin-accounts.md`：71 条
- `06-admin-ops.md`：50 条
- `07-admin-misc.md`：76 条

## 推荐调用头

```http
x-api-key: admin-<64hex>
Content-Type: application/json
```

涉及幂等写接口时，建议额外传：

```http
Idempotency-Key: <your-unique-key>
```

## 特别说明

- 支付系统对接优先使用 Admin API Key，不要混用用户 JWT。
- 如果需要逐条查看某个 Admin 路由是否支持 Admin-Key，请直接查看 04-07 四个分册；本文件不再重复维护 265 条完整列表。
