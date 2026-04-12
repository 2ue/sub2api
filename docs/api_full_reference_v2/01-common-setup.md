# Common 与 Setup 接口（当前代码版）

- 同步日期：**2026-03-18**
- 接口数量：**7**
- 文档策略：优先保证当前路由、鉴权、中间件与注册位置准确；字段级请求体请继续查对应 handler / dto。
- 当前版直接以 `backend/internal/server/routes/common.go` 与 `backend/internal/setup/handler.go` 为准。
- `/setup/status` 在正常服务模式与 setup 服务模式下各有一条实现。

## 基础与安装

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| POST | `/api/event_logging/batch` | 公开接口（无需鉴权） | 否 | `inline route func` | `backend/internal/server/routes/common.go:17` | - |
| GET | `/health` | 公开接口（无需鉴权） | 否 | `inline route func` | `backend/internal/server/routes/common.go:12` | - |
| POST | `/setup/install` | `setupGuard`（仅 `NeedsSetup()==true` 可访问） | 否 | `install` | `backend/internal/setup/handler.go:34` | - |
| GET | `/setup/status` | 公开接口 / `setupGuard`（取决于运行模式） | 否 | `inline route func` | `backend/internal/server/routes/common.go:23` | 正常服务模式与 setup 服务模式各有一条实现 |
| GET | `/setup/status` | 公开接口 / `setupGuard`（取决于运行模式） | 否 | `getStatus` | `backend/internal/setup/handler.go:26` | 正常服务模式与 setup 服务模式各有一条实现 |
| POST | `/setup/test-db` | `setupGuard`（仅 `NeedsSetup()==true` 可访问） | 否 | `testDatabase` | `backend/internal/setup/handler.go:32` | - |
| POST | `/setup/test-redis` | `setupGuard`（仅 `NeedsSetup()==true` 可访问） | 否 | `testRedis` | `backend/internal/setup/handler.go:33` | - |
