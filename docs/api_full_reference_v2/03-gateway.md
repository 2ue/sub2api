# Gateway 接口（当前代码版）

- 同步日期：**2026-03-18**
- 接口数量：**28（按注册点计；唯一运行时路径为 27）**
- 文档策略：优先保证当前路由、鉴权、中间件与注册位置准确；字段级请求体请继续查对应 handler / dto。
- `/v1/messages` 会按分组平台自动在 Claude 兼容层与 OpenAI Messages 兼容层之间切换。
- `/v1/responses` / `/responses` 同时包含 HTTP POST 与 WebSocket GET 入口。
- 通过 Nginx 反代并服务 Codex CLI 时，仍需在 `http` 块开启 `underscores_in_headers on;`。

## Antigravity v1

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| POST | `/antigravity/v1/messages` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.Gateway.Messages` | `backend/internal/server/routes/gateway.go:115` | - |
| POST | `/antigravity/v1/messages/count_tokens` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.Gateway.CountTokens` | `backend/internal/server/routes/gateway.go:116` | - |
| GET | `/antigravity/v1/models` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.Gateway.AntigravityModels` | `backend/internal/server/routes/gateway.go:117` | - |
| GET | `/antigravity/v1/usage` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.Gateway.Usage` | `backend/internal/server/routes/gateway.go:118` | - |

## Antigravity v1beta

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/antigravity/v1beta/models` | `APIKeyAuthWithSubscriptionGoogle`（用户 API Key） | 否 | `h.Gateway.GeminiV1BetaListModels` | `backend/internal/server/routes/gateway.go:130` | - |
| POST | `/antigravity/v1beta/models/*modelAction` | `APIKeyAuthWithSubscriptionGoogle`（用户 API Key） | 否 | `h.Gateway.GeminiV1BetaModels` | `backend/internal/server/routes/gateway.go:132` | - |
| GET | `/antigravity/v1beta/models/:model` | `APIKeyAuthWithSubscriptionGoogle`（用户 API Key） | 否 | `h.Gateway.GeminiV1BetaGetModel` | `backend/internal/server/routes/gateway.go:131` | - |

## Antigravity 模型清单

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/antigravity/models` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.Gateway.AntigravityModels` | `backend/internal/server/routes/gateway.go:103` | - |

## Claude / OpenAI 兼容入口

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| POST | `/v1/chat/completions` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.OpenAIGateway.ChatCompletions` | `backend/internal/server/routes/gateway.go:77` | - |
| POST | `/v1/messages` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `inline route func` | `backend/internal/server/routes/gateway.go:49` | 按分组平台自动路由：OpenAI 组走 OpenAI Messages 兼容层，其余走 Claude 兼容层 |
| POST | `/v1/messages/count_tokens` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `inline route func` | `backend/internal/server/routes/gateway.go:57` | OpenAI 分组会直接返回 404，不支持 token counting |
| GET | `/v1/models` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.Gateway.Models` | `backend/internal/server/routes/gateway.go:70` | - |
| GET | `/v1/responses` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.OpenAIGateway.ResponsesWebSocket` | `backend/internal/server/routes/gateway.go:75` | GET 为 OpenAI Responses WebSocket upgrade 入口 |
| POST | `/v1/responses` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.OpenAIGateway.Responses` | `backend/internal/server/routes/gateway.go:73` | POST 为 OpenAI Responses HTTP 入口 |
| POST | `/v1/responses/*subpath` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.OpenAIGateway.Responses` | `backend/internal/server/routes/gateway.go:74` | 保留 `/compact` 等子路径后缀并转发 |
| GET | `/v1/usage` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.Gateway.Usage` | `backend/internal/server/routes/gateway.go:71` | - |

## Gemini 原生兼容

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/v1beta/models` | `APIKeyAuthWithSubscriptionGoogle`（用户 API Key） | 否 | `h.Gateway.GeminiV1BetaListModels` | `backend/internal/server/routes/gateway.go:89` | - |
| POST | `/v1beta/models/*modelAction` | `APIKeyAuthWithSubscriptionGoogle`（用户 API Key） | 否 | `h.Gateway.GeminiV1BetaModels` | `backend/internal/server/routes/gateway.go:92` | - |
| GET | `/v1beta/models/:model` | `APIKeyAuthWithSubscriptionGoogle`（用户 API Key） | 否 | `h.Gateway.GeminiV1BetaGetModel` | `backend/internal/server/routes/gateway.go:90` | - |

## OpenAI Chat Completions 别名

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| POST | `/chat/completions` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.OpenAIGateway.ChatCompletions` | `backend/internal/server/routes/gateway.go:100` | - |

## OpenAI Responses 别名

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/responses` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.OpenAIGateway.ResponsesWebSocket` | `backend/internal/server/routes/gateway.go:98` | GET 为 OpenAI Responses WebSocket upgrade 入口 |
| POST | `/responses` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.OpenAIGateway.Responses` | `backend/internal/server/routes/gateway.go:96` | POST 为 OpenAI Responses HTTP 入口 |
| POST | `/responses/*subpath` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.OpenAIGateway.Responses` | `backend/internal/server/routes/gateway.go:97` | 保留 `/compact` 等子路径后缀并转发 |

## Sora 媒体代理

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/sora/media/*filepath` | 公开接口或用户 API Key（取决于配置） | 否 | `h.SoraGateway.MediaProxy` | `backend/internal/server/routes/gateway.go:151-153` | 代码按 `gateway.sora_media_require_api_key` 分两种注册分支；运行时只会启用其中一种 |

## Sora 媒体签名下载

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/sora/media-signed/*filepath` | 公开接口（签名 URL） | 否 | `h.SoraGateway.MediaProxySigned` | `backend/internal/server/routes/gateway.go:156` | 签名下载地址，无需 API Key |

## Sora 网关

| Method | Path | 鉴权 | 支持 Admin-Key | Handler | 代码位置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| POST | `/sora/v1/chat/completions` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.SoraGateway.ChatCompletions` | `backend/internal/server/routes/gateway.go:145` | - |
| GET | `/sora/v1/models` | `APIKeyAuthMiddleware`（用户 API Key） | 否 | `h.Gateway.Models` | `backend/internal/server/routes/gateway.go:146` | - |
