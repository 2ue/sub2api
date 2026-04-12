# Admin-Key 支持范围（当前代码版）

- 同步日期：**2026-03-18**
- 支持 Admin-Key 的接口数量：**265**
- 本文件只保留结论摘要；详细接口归档见：`docs/api_full_reference_v2/08-admin-key-supported.md`

## 结论

- 支持：所有 `/api/v1/admin/**`
- 不支持：所有非 `/api/v1/admin/**` 路由

## 推荐调用头

```http
x-api-key: admin-<64hex>
Content-Type: application/json
```

幂等写接口建议额外带：

```http
Idempotency-Key: <your-unique-key>
```

## 唯一详细文档

- `docs/api_full_reference_v2/08-admin-key-supported.md`
