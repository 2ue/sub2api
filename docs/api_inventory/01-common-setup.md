# Common 与 Setup 清单（当前代码版）

- 同步日期：**2026-03-18**
- 接口数量：**7**
- 本文件只保留检索摘要；详细接口表见：`docs/api_full_reference_v2/01-common-setup.md`

## 范围

- 健康检查：`/health`
- 事件日志：`/api/event_logging/batch`
- 安装流程：`/setup/status`、`/setup/test-db`、`/setup/test-redis`、`/setup/install`

## 关键说明

- `/setup/status` 在正常服务模式与 setup 服务模式下各有一条实现。
- `/setup/test-db`、`/setup/test-redis`、`/setup/install` 仅在 `NeedsSetup()==true` 时可访问。
- 本组全部接口都不支持 Admin-Key。

## 唯一详细文档

- `docs/api_full_reference_v2/01-common-setup.md`
