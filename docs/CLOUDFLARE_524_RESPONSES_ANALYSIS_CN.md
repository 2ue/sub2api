# Cloudflare 524 与 OpenAI Responses 长请求分析

更新时间：2026-04-30

本文记录当前项目在 Cloudflare 代理后访问 OpenAI `/responses` 长请求时，遇到 `524` 的原因、项目内已有能力、配置可缓解范围、无法仅靠配置解决的场景，以及推荐方案。

## 结论摘要

当前系统只能通过配置缓解一部分 Cloudflare `524`，不能完全靠项目配置绕过 Cloudflare 的约 120 秒 HTTP 代理读取限制。

可配置缓解的前提是：客户端请求为流式请求，即 `stream: true`；请求走项目普通 OpenAI gateway SSE 路径；后端已经拿到上游响应头并开始向 Cloudflare 写出 `text/event-stream`；中间反向代理没有缓冲或压缩 SSE。

不能靠现有配置解决的主要场景：

- 非流式长请求。后端需要等完整上游结果后才返回，Cloudflare 可能先超时。
- 上游在返回响应头前阻塞超过 Cloudflare 限制。此时后端还没有开始给 Cloudflare 写任何响应，项目内 SSE keepalive 还没有机会生效。
- OpenAI 自动透传路径。当前透传流式处理会缓存首个 client-output 之前的事件，但没有使用 `gateway.stream_keepalive_interval` 的下游 keepalive ticker。
- 客户端仍然走 HTTP `/responses` 时，`gateway.openai_ws` 主要影响上游/WS ingress，不会自动让 HTTP 请求绕过 Cloudflare 的 HTTP read timeout。

最稳妥的部署方案是：给 API 单独准备一个 DNS-only 灰云子域名，例如 `api.example.com`，让长请求直连源站；或者改成长请求异步任务/轮询；如果必须继续走 Cloudflare 橙云，则需要确保流式输出能在 120 秒内持续向 Cloudflare flush 字节。

## Cloudflare 524 的含义

Cloudflare `524` 表示 Cloudflare 已经成功连接到源站，但源站没有在 Cloudflare 的代理读取超时时间内返回可读响应。Cloudflare 文档说明默认 Proxy Read Timeout 通常为约 120 秒，Enterprise 用户可调整到更长时间。

官方参考：

- Cloudflare 524 文档：https://developers.cloudflare.com/support/troubleshooting/http-status-codes/cloudflare-5xx-errors/error-524/
- Cloudflare connection limits：https://developers.cloudflare.com/fundamentals/reference/connection-limits/

关键点：后端自己的超时配置如果设置为 600 秒，只表示后端愿意等上游 600 秒；这不会让 Cloudflare 愿意等源站 600 秒。只要 Cloudflare 在自己的 read timeout 内没有读到源站响应字节，就仍然可能返回 `524`。

## 当前项目已有的相关配置

项目已有以下配置项：

```yaml
gateway:
  response_header_timeout: 600
  stream_data_interval_timeout: 180
  stream_keepalive_interval: 10
  openai_ws:
    enabled: true
    responses_websockets_v2: true
    force_http: false
```

配置字段定义：

- `gateway.response_header_timeout`：等待上游响应头的超时时间。代码位置：`backend/internal/config/config.go:582`。
- `gateway.stream_data_interval_timeout`：流式上游数据间隔超时。代码位置：`backend/internal/config/config.go:635`。
- `gateway.stream_keepalive_interval`：下游流式 keepalive 间隔。代码位置：`backend/internal/config/config.go:637`。
- `gateway.openai_ws`：OpenAI Responses WebSocket 相关配置。代码位置：`backend/internal/config/config.go:606`。

默认值：

- `gateway.response_header_timeout = 600`，代码位置：`backend/internal/config/config.go:1620`。
- `gateway.openai_ws.enabled = true`，代码位置：`backend/internal/config/config.go:1630`。
- `gateway.openai_ws.responses_websockets_v2 = true`，代码位置：`backend/internal/config/config.go:1642`。
- `gateway.stream_data_interval_timeout = 180`，代码位置：`backend/internal/config/config.go:1690`。
- `gateway.stream_keepalive_interval = 10`，代码位置：`backend/internal/config/config.go:1691`。

配置校验范围：

- `gateway.stream_data_interval_timeout` 必须为 `0`，或 `30-300` 秒。代码位置：`backend/internal/config/config.go:2269`。
- `gateway.stream_keepalive_interval` 必须为 `0`，或 `5-30` 秒。代码位置：`backend/internal/config/config.go:2276`。

环境变量支持：

项目使用 Viper 的 `AutomaticEnv()`，并将 `.` 替换为 `_`。代码位置：`backend/internal/config/config.go:1219` 和 `backend/internal/config/config.go:1220`。

因此可以使用：

```env
GATEWAY_RESPONSE_HEADER_TIMEOUT=600
GATEWAY_STREAM_DATA_INTERVAL_TIMEOUT=180
GATEWAY_STREAM_KEEPALIVE_INTERVAL=10
GATEWAY_OPENAI_WS_ENABLED=true
GATEWAY_OPENAI_WS_RESPONSES_WEBSOCKETS_V2=true
GATEWAY_OPENAI_WS_FORCE_HTTP=false
```

示例配置文件里也有这些字段：

- `deploy/config.example.yaml:151`：`response_header_timeout`
- `deploy/config.example.yaml:235`：`openai_ws`
- `deploy/config.example.yaml:339`：`stream_data_interval_timeout`
- `deploy/config.example.yaml:342`：`stream_keepalive_interval`

## 普通 OpenAI `/responses` 流式路径的能力

普通 OpenAI gateway 流式路径具备下游 SSE keepalive 能力。

相关流程：

1. 后端收到 HTTP `/responses` 请求后，构造上游请求并调用上游。
2. 上游返回响应头后，如果请求为 `stream: true`，进入 `handleStreamingResponse`。
3. `handleStreamingResponse` 设置 SSE 相关响应头。
4. 如果一段时间没有真实下游数据写出，则按 `gateway.stream_keepalive_interval` 写出 SSE comment `:\n\n` 并 flush。

关键代码位置：

- 普通流式入口调用：`backend/internal/service/openai_gateway_service.go:2747`
- 普通流式处理函数：`backend/internal/service/openai_gateway_service.go:4153`
- SSE 响应头：`backend/internal/service/openai_gateway_service.go:4159` 到 `backend/internal/service/openai_gateway_service.go:4162`
- 读取 `StreamKeepaliveInterval`：`backend/internal/service/openai_gateway_service.go:4209` 到 `backend/internal/service/openai_gateway_service.go:4212`
- keepalive 写出 `:\n\n`：`backend/internal/service/openai_gateway_service.go:4499` 到 `backend/internal/service/openai_gateway_service.go:4515`

这条路径对 Cloudflare 524 的缓解逻辑是：只要 HTTP 响应已经开始，后端就可以每 10 秒写出一次 SSE comment，让 Cloudflare 不认为源站长期无响应。

注意：这个 keepalive 是下游 keepalive，也就是服务端写给客户端/Cloudflare 的保活。它不是上游 OpenAI 连接保活。

## `response_header_timeout` 不能解决 Cloudflare 524 的原因

`gateway.response_header_timeout` 被用于 HTTP transport 的 `ResponseHeaderTimeout`：

- 配置读取：`backend/internal/repository/http_upstream.go:731`
- transport 字段设置：`backend/internal/repository/http_upstream.go:763` 到 `backend/internal/repository/http_upstream.go:768`
- 注释说明该字段只等待响应头，不影响流式传输：`backend/internal/repository/http_upstream.go:761`

这意味着它控制的是本服务等待上游响应头的时间，不是 Cloudflare 等待本服务响应的时间。

如果上游 OpenAI/ChatGPT 在返回响应头前卡住 180 秒，后端可能仍在等待上游；但 Cloudflare 在约 120 秒时可能已经对客户端返回 `524`。此时即使 `response_header_timeout` 设置为 600，也不能改变 Cloudflare 的行为。

可选策略：

- 保持 `response_header_timeout: 600`：让后端继续等待上游，有利于直连或非 Cloudflare 场景。
- 设置为 `110` 左右：让后端在 Cloudflare 之前失败并返回自己的错误，便于日志和重试控制。但这只是把 Cloudflare `524` 换成后端错误，不是让长请求成功。

## OpenAI 自动透传路径的风险

OpenAI 账号如果开启了自动透传，会进入 `forwardOpenAIPassthrough` 分支：

- 账号开关判断：`backend/internal/service/openai_gateway_service.go:2028`
- 透传分支调用：`backend/internal/service/openai_gateway_service.go:2032`
- 账号字段说明：`backend/internal/service/account.go:1123` 到 `backend/internal/service/account.go:1128`

透传流式路径是 `handleStreamingResponsePassthrough`：

- 透传流式调用：`backend/internal/service/openai_gateway_service.go:2996`
- 透传流式函数：`backend/internal/service/openai_gateway_service.go:3448`
- 透传流式也设置 SSE 头：`backend/internal/service/openai_gateway_service.go:3460` 到 `backend/internal/service/openai_gateway_service.go:3463`

风险点在于：透传流式路径会在首个 client-output 之前缓存部分上游行：

- `pendingLines` 定义：`backend/internal/service/openai_gateway_service.go:3484`
- 未开始 client output 时追加到 `pendingLines` 并继续等待：`backend/internal/service/openai_gateway_service.go:3551` 到 `backend/internal/service/openai_gateway_service.go:3553`
- 真正开始输出后才写出并 flush：`backend/internal/service/openai_gateway_service.go:3560` 到 `backend/internal/service/openai_gateway_service.go:3565`

当前没有看到该函数使用 `StreamKeepaliveInterval` 的下游 keepalive ticker。因此，如果上游长时间没有产生可转发给客户端的有效输出，Cloudflare 仍可能看到源站长时间无响应并返回 `524`。

临时规避：

- 对容易超时的 OpenAI 账号关闭“自动透传（仅替换认证）”。
- 让请求走普通 OpenAI gateway 流式路径，由 `stream_keepalive_interval` 下游 keepalive 承担保活。

长期修复：

- 给 `handleStreamingResponsePassthrough` 增加与普通 `handleStreamingResponse` 一致的下游 keepalive ticker。
- 对首个 client-output 前的缓存阶段，也应周期性写出 SSE comment `:\n\n` 并 flush，避免 Cloudflare 空等。

## `openai_ws` 配置的实际影响

项目确实有 OpenAI Responses WebSocket 相关配置，但要区分两类连接：

- 客户端到本服务：你的 `https://okmcode.com/responses` 是 HTTP 请求，经 Cloudflare HTTP 代理。
- 本服务到 OpenAI/ChatGPT：项目内部可以使用 HTTP/SSE 或 WebSocket 上游传输。

对于普通 HTTP `/responses`，handler 会标记当前客户端入站协议为 HTTP：

- HTTP `/responses` handler：`backend/internal/handler/openai_gateway_handler.go:89`
- 标记 HTTP transport：`backend/internal/handler/openai_gateway_handler.go:95`

WebSocket ingress 是另一个入口：

- WS `/responses` handler：`backend/internal/handler/openai_gateway_handler.go:1034`
- 标记 WS transport：`backend/internal/handler/openai_gateway_handler.go:1039`

协议决策逻辑中，如果客户端入站协议是 HTTP，会强制把 OpenAI WS 决策降回 HTTP：

- `resolveOpenAIWSDecisionByClientTransport`：`backend/internal/service/openai_client_transport.go:63`
- HTTP 入站强制返回 `client_protocol_http`：`backend/internal/service/openai_client_transport.go:67` 到 `backend/internal/service/openai_client_transport.go:69`

因此，对 `https://okmcode.com/responses` 这种 HTTP 请求，单纯打开：

```yaml
gateway:
  openai_ws:
    enabled: true
    responses_websockets_v2: true
```

通常不能解决 Cloudflare `524`。它不会自动把客户端到源站这一段变成 WebSocket，也不会绕过 Cloudflare 对 HTTP 响应的 read timeout。

只有当客户端本身走项目的 WebSocket ingress，并且 Cloudflare/WebSocket 链路可用时，`openai_ws` 才可能从整体链路上减少 HTTP 524 风险。

## Caddy 与中间代理注意事项

项目示例 `deploy/Caddyfile` 中有：

- `transport http { keepalive 120s }`：`deploy/Caddyfile:57`
- 全局 `encode` 匹配 `Content-Type text/*`：`deploy/Caddyfile:73` 到 `deploy/Caddyfile:78`

`transport http keepalive 120s` 是 Caddy 到后端的连接复用/空闲连接参数，不是 Cloudflare 的 Proxy Read Timeout，不能解决 Cloudflare `524`。

SSE 响应的 `Content-Type` 是 `text/event-stream`，属于 `text/*`。生产部署中应确认 `text/event-stream` 不被压缩、不被缓冲，确保后端写出的 `:\n\n` 能尽快到达 Cloudflare。

建议：

- 对 `text/event-stream` 禁用压缩。
- 对 SSE 禁用 buffering。
- 确认每次 SSE keepalive 后中间代理会立即 flush。
- 如使用 Nginx，通常需要 `proxy_buffering off`，并注意不要对 SSE gzip。
- 项目已经写了 `X-Accel-Buffering: no`，但仍应在实际反代配置中确认行为。

## 推荐配置

如果继续使用 Cloudflare 橙云，建议至少保持：

```yaml
gateway:
  response_header_timeout: 600
  stream_keepalive_interval: 10
  stream_data_interval_timeout: 180
  openai_passthrough_allow_timeout_headers: false
  openai_ws:
    enabled: true
    responses_websockets_v2: true
    force_http: false
```

对应环境变量：

```env
GATEWAY_RESPONSE_HEADER_TIMEOUT=600
GATEWAY_STREAM_KEEPALIVE_INTERVAL=10
GATEWAY_STREAM_DATA_INTERVAL_TIMEOUT=180
GATEWAY_OPENAI_PASSTHROUGH_ALLOW_TIMEOUT_HEADERS=false
GATEWAY_OPENAI_WS_ENABLED=true
GATEWAY_OPENAI_WS_RESPONSES_WEBSOCKETS_V2=true
GATEWAY_OPENAI_WS_FORCE_HTTP=false
```

说明：

- `stream_keepalive_interval` 不建议设为 `0`，否则下游 keepalive 禁用。
- `stream_keepalive_interval` 推荐 `10`，如 Cloudflare/反代仍出现空闲断开，可尝试 `5`。
- `stream_data_interval_timeout` 是上游流数据间隔超时，不是 Cloudflare keepalive。调大到 `300` 可以容忍更久的上游沉默，但如果下游 keepalive 不工作，仍不能解决 524。
- `openai_passthrough_allow_timeout_headers` 建议保持 `false`，避免客户端透传 `x-stainless-timeout` 等头导致上游更早断流。

## 分场景处理方案

### 场景 1：客户端可控制请求参数

推荐：

1. 强制使用 `stream: true`。
2. 关闭 OpenAI 自动透传。
3. 保持 `gateway.stream_keepalive_interval: 10`。
4. 检查反代不压缩、不缓冲 `text/event-stream`。

效果：

普通流式路径会每 10 秒写出 `:\n\n`，能显著降低 Cloudflare 因下游空闲而返回 `524` 的概率。

限制：

如果卡在上游响应头之前，仍可能 524。

### 场景 2：客户端必须使用非流式请求

不建议继续走 Cloudflare 橙云承载超长非流式请求。

可选方案：

1. 改为流式接口。
2. 改成异步任务：提交请求后立即返回任务 ID，后台执行，客户端轮询结果。
3. 使用 DNS-only 灰云 API 子域名直连源站。
4. 使用 Cloudflare Enterprise 调高 Proxy Read Timeout。

### 场景 3：必须保留 OpenAI 自动透传

当前配置不能完整解决。

短期：

- 对高延迟模型、高延迟图片生成或复杂 Responses 请求关闭自动透传。
- 保留透传给低延迟、稳定输出的请求。

长期：

- 修改 `handleStreamingResponsePassthrough`，加入下游 keepalive。
- 在 `pendingLines` 缓存阶段也允许写 SSE comment。
- 增加测试覆盖：模拟上游 130 秒无 client-output，确认下游每 10 秒收到 `:\n\n`。

### 场景 4：必须继续使用 Cloudflare 橙云

推荐组合：

1. API 统一要求 `stream: true`。
2. 禁用 OpenAI 自动透传或修复透传 keepalive。
3. Caddy/Nginx 禁止 SSE 压缩和缓冲。
4. 保持 keepalive 间隔 5-10 秒。
5. 观察服务日志，确认响应开始后下游确实有周期性 flush。

仍需接受的限制：

- Cloudflare 对 HTTP 橙云长请求仍有平台限制。
- 项目内配置不能改变 Cloudflare 的全局代理限制。

### 场景 5：追求稳定性优先

建议开独立 API 域名：

```text
api.example.com -> DNS-only 灰云 -> 源站 Caddy/Nginx -> sub2api
```

保留主站域名走 Cloudflare 橙云，API 长请求走灰云直连。这样不会受 Cloudflare HTTP 代理 read timeout 影响，但需要自己处理源站安全：

- 源站防火墙限制入站端口。
- TLS 证书正常配置。
- API 鉴权必须启用。
- 可按需加 WAF、IP allowlist、速率限制或独立反代层。

## 排查清单

发生 `524` 时，按以下顺序确认：

1. 请求是否为 `stream: true`。
2. 命中的账号是否开启 OpenAI 自动透传。
3. 请求是否走 HTTP `/responses`，而不是 WebSocket ingress。
4. 后端日志中，上游响应头是否在 120 秒内返回。
5. 后端是否进入 `handleStreamingResponse` 普通流式路径。
6. 响应开始后是否每 5-10 秒向客户端写出字节。
7. Caddy/Nginx 是否压缩或缓冲 `text/event-stream`。
8. Cloudflare 是否橙云代理该域名。
9. 客户端或 SDK 是否设置了额外超时头。
10. 如果使用 Codex CLI，确认重连日志中的 URL 是否仍是 Cloudflare 橙云域名。

## 建议的代码改造方向

如果希望从项目层面进一步降低 524，建议按优先级改造：

### 优先级 P0：透传流式路径补下游 keepalive

目标：

- `handleStreamingResponsePassthrough` 使用 `gateway.stream_keepalive_interval`。
- 当没有真实下游输出时，周期性写 `:\n\n` 并 flush。
- 不破坏已有 `pendingLines` 和 failover 行为。

关键点：

- keepalive 只写 SSE comment，不写 `data:`，避免影响 OpenAI SDK 对事件的解析。
- keepalive 应基于最后一次下游写出时间，而不是上游读取时间。
- 客户端断开后仍要继续 drain 上游以完成 usage/billing 统计，保持现有语义。

### 优先级 P1：HTTP 上游等待阶段的早期 SSE 响应策略

目标：

- 对 `stream: true` 的请求，在等待上游响应头期间也能尽早向下游返回 SSE headers 并 keepalive。

风险：

- 一旦提前写出 HTTP 200 和 SSE headers，后续如果上游请求失败，就不能再改成普通 JSON 错误或触发某些未写出前的 failover 语义。
- 需要重新设计 pre-output failover 和错误事件格式。

建议：

- 不要轻易全局开启。
- 可以作为可配置实验开关，例如 `gateway.preflush_stream_headers`。
- 仅在明确接受错误事件替代 HTTP 错误码的客户端上启用。

### 优先级 P2：异步任务模式

目标：

- 对图片生成、复杂 Responses、超长 reasoning 等请求，提供提交任务接口和查询接口。
- 初始请求快速返回任务 ID，避免 HTTP 长连接。

适用：

- 非交互式任务。
- 图片生成或长时间批处理。
- 客户端可以轮询或接收 webhook 的场景。

## 最终建议

短期建议：

1. API 请求尽量使用 `stream: true`。
2. 确认 `gateway.stream_keepalive_interval` 为 `10` 或 `5`，不要禁用。
3. 对经常 524 的 OpenAI 账号关闭自动透传。
4. 检查 Caddy/Nginx/Cloudflare 链路，确保 `text/event-stream` 不被压缩、不被缓冲。

中期建议：

1. 新增一个 DNS-only 灰云 API 子域名专门承载长请求。
2. 保留主站走 Cloudflare 橙云，API 长连接走灰云或其他支持长连接的代理。
3. 给透传路径补齐下游 keepalive。

长期建议：

1. 对超长任务提供异步任务/轮询模式。
2. 如果必须全部走 Cloudflare 橙云且必须单连接长时间等待，评估 Cloudflare Enterprise 的 Proxy Read Timeout 调整。
3. 对所有流式路径做统一 keepalive 抽象，避免普通路径和透传路径行为不一致。

