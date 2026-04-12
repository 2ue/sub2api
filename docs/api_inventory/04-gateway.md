# Gateway 清单（当前代码版）

- 同步日期：**2026-03-18**
- 接口数量：**28（按注册点计）**
- 本文件只保留检索摘要；详细接口表见：`docs/api_full_reference_v2/03-gateway.md`

## 范围

- Claude / OpenAI 兼容：`/v1/messages`、`/v1/chat/completions`、`/v1/responses`
- OpenAI 别名：`/responses`、`/chat/completions`
- Gemini 原生：`/v1beta/**`
- Antigravity：`/antigravity/**`
- Sora：`/sora/v1/**`、`/sora/media/*filepath`、`/sora/media-signed/*filepath`

## 关键说明

- `/v1/messages` 会按分组平台自动切到 Claude 或 OpenAI Messages 兼容层。
- `/v1/messages/count_tokens` 在 OpenAI 分组下会直接返回 404。
- `/v1/responses` 与 `/responses` 同时存在 HTTP POST 和 WebSocket GET 入口。
- `/responses/*subpath` 会保留 `/compact` 等子路径后缀。
- `/sora/media/*filepath` 是否要求用户 API Key 取决于配置 `gateway.sora_media_require_api_key`。
- Gateway 接口使用的是用户 API Key，不支持 Admin-Key。

## 唯一详细文档

- `docs/api_full_reference_v2/03-gateway.md`
