# OpenAI Token 缓存异常分析

检查日期：2026-05-06

## 背景

用户反馈：使用 Context7 MCP 做分布式上下文管理后，每次请求的上下文缓存都在约 270k token 左右，并出现类似 `max_output_tokens limit xxx` 的报错。

本次分析聚焦两个问题：

1. OpenAI 相关账号接入路径，包括 OAuth/auth 和 APIKey，是否会对输入上下文中的特定关键词做特殊处理，导致 token 缓存异常。
2. 项目内是否存在其他可能导致 token 缓存异常的代码路径。

检查范围主要包括：

- `backend/internal/service/openai*`
- `backend/internal/handler/openai*`
- `backend/internal/pkg/openai`
- OpenAI gateway、sticky session、prompt cache key、usage 解析与计费链路
- 账号 access token cache、API key auth cache、billing cache 相关路径

## 总结论

没有发现 OpenAI 账号接入路径会因为用户输入上下文中出现某些自然语言关键词，就特殊处理、改写或触发 prompt token cache，从而导致 `cached_tokens` 异常升高的证据。

当前更可能的原因是：Context7 MCP 或客户端每轮实际向上游发送了约 270k token 的稳定重复上下文前缀，例如分布式上下文、MCP resources、工具结果、系统提示词、工具 schema 或历史上下文。上游将这部分重复前缀计入 `cached_tokens`，说明 prompt cache 命中；但这些 token 仍然占用模型上下文窗口，因此会挤压可用输出预算，最终触发 `max_output_tokens limit xxx`。

换句话说，`cached_tokens ~= 270k` 更像是上游真实看到了 270k token 的重复输入，不是本项目本地伪造或按关键词触发出来的 usage 数值。

## 1. 是否存在用户关键词特殊处理

结论：OpenAI 主路径没有发现按用户 prompt 关键词触发缓存、改写、转发切换或 usage 异常的逻辑。

OpenAI 路径里确认存在的分支条件主要是：

- 账号类型：OAuth 或 APIKey
- endpoint：`/v1/responses`、`/v1/chat/completions`、`/v1/messages`
- 模型名和模型族
- header：`session_id`、`conversation_id`、`User-Agent`、`originator`
- 结构化字段：`tools`、`tool_choice`、`input`、`messages`、`instructions`、`prompt_cache_key`、`previous_response_id`
- group/account 配置
- 上游错误状态

这些分支不是扫描用户自然语言内容里的特定关键词。

### 关键证据

- `backend/internal/service/openai_compat_prompt_cache_key.go:12`
  - `shouldAutoInjectPromptCacheKeyForCompat` 只根据模型族判断是否自动注入兼容 `prompt_cache_key`，主要检查 `gpt-5` 和 `codex`。
  - 这不是按用户 prompt 关键词触发。

- `backend/internal/service/openai_compat_prompt_cache_key.go:27`
  - `deriveCompatPromptCacheKey` 的 seed 来自 model、reasoning effort、tools、functions、system message、第一条 user message。
  - 它会用请求内容生成稳定 hash，但没有关键词匹配或关键词策略。

- `backend/internal/service/openai_content_session_seed.go:14`
  - `deriveOpenAIContentSessionSeed` 在没有显式 session 信号时，从 model、tools/functions、instructions、system/developer、第一条 user message 派生 session seed。
  - 用途是 sticky routing/session affinity，不是关键词过滤或缓存伪造。

- `backend/internal/service/openai_gateway_service.go:1160`
  - `GenerateSessionHash` 的优先级是 `session_id`、`conversation_id`、body `prompt_cache_key`、内容 fallback。
  - 这是会话稳定性逻辑，不是关键词处理。

- `backend/internal/service/openai_gateway_service.go:3824`
  - `buildUpstreamRequest` 对允许透传的 header 做白名单处理，并在 OAuth 路径设置 ChatGPT Codex upstream 所需 header。
  - OAuth 下会清理客户端传入的 `session_id` 和 `conversation_id`，再用隔离后的 session 标识重设。

- `backend/internal/service/openai_gateway_chat_completions.go:86`
  - Chat Completions OAuth 兼容路径在缺少 `prompt_cache_key` 时，可能为 GPT-5/Codex 模型自动派生 prompt cache key。
  - 触发条件是账号类型和模型族，不是用户输入关键词。

- `backend/internal/service/openai_gateway_messages.go:174`
  - Anthropic `/v1/messages` 兼容到 OpenAI 时，会把 `prompt_cache_key` 转成隔离后的上游 `session_id`。
  - 该路径已混入 API key ID 做隔离。

### 容易混淆的非 OpenAI 路径

项目中确实存在基于消息内容关键词的 warmup/suggestion 拦截，但它位于通用 Anthropic/Gateway `Messages` 路径，不是 OpenAI group 的主路径。

相关代码：

- `backend/internal/handler/gateway_handler.go:1579`
  - `detectInterceptType` 会检查 `"[SUGGESTION MODE:"`、`"Warmup"`、title 相关请求。

- `backend/internal/handler/gateway_handler.go:1627`
  - 匹配 `Please write a 5-10 word title for the following conversation:` 或 `Warmup`。

但 OpenAI group 的 `/v1/messages` 会分流到 OpenAI handler，不走该通用 handler。因此这段不能解释 OpenAI 账号路径下的 `270k cached_tokens`。

## 2. 与 Context7 MCP 现象最相关的机制

### 2.1 prompt_cache_key 与 session_id 稳定化

OpenAI OAuth/Codex 路径会透传或派生 `prompt_cache_key`，并设置上游 `session_id` 或 `conversation_id`，以提升同一上下文的 sticky session 和 prompt cache 命中。

相关代码：

- `backend/internal/service/openai_gateway_service.go:3851`
  - OAuth upstream request 中，如果存在 `promptCacheKey`，会设置隔离后的 `conversation_id` 和 `session_id`。

- `backend/internal/service/openai_gateway_chat_completions.go:171`
  - Chat Completions 兼容路径在 OAuth 下执行 Codex transform，并注入或保留 `prompt_cache_key`。

- `backend/internal/service/openai_gateway_messages.go:176`
  - `/v1/messages` 兼容路径根据 `prompt_cache_key` 设置上游 `session_id`。

这类逻辑会让稳定上下文更容易命中上游 prompt cache。若 Context7 每轮都发送同一大段上下文，`cached_tokens` 会稳定保持高位。

### 2.2 内容 fallback 会参与 sticky session 派生

当客户端没有显式传入 `session_id`、`conversation_id` 或 `prompt_cache_key` 时，项目会从请求内容派生 session seed。

相关代码：

- `backend/internal/service/openai_gateway_service.go:1172`
  - 没有显式 session 时调用 `deriveOpenAIContentSessionSeed(body)`。

- `backend/internal/service/openai_content_session_seed.go:41`
  - `instructions` 会进入 seed。

- `backend/internal/service/openai_content_session_seed.go:48`
  - `messages` 中的 system/developer 和第一条 user 会进入 seed。

- `backend/internal/service/openai_content_session_seed.go:69`
  - Responses API 的 `input` 也会参与 seed。

这不是异常，但在 Context7 把巨大上下文放进稳定前缀时，会使后续请求持续走同一 sticky/cache 轨道。

### 2.3 cached_tokens 仍占上下文窗口

项目只负责读取上游返回的 cached token usage，并按 cache read 价格计费。它不能把这些 token 从模型上下文窗口中移除。

相关代码：

- `backend/internal/service/openai_gateway_service.go:4599`
  - SSE 路径读取 `response.usage.input_tokens_details.cached_tokens`。

- `backend/internal/service/openai_gateway_service.go:4609`
  - 非流式 JSON 路径读取 `usage.input_tokens_details.cached_tokens`。

- `backend/internal/service/openai_gateway_service.go:5177`
  - 计费时将 `InputTokens - CacheReadInputTokens` 作为普通输入 token，并把 cache read token 单独计费。

因此，即使 270k token 是 cached tokens，它们仍然会占用上下文窗口。若模型上下文窗口接近被输入占满，`max_output_tokens` 就会受到限制。

## 3. usage/cached_tokens 主链路检查

结论：没有发现项目本地伪造 OpenAI `cached_tokens` 的逻辑。主链路是读取上游 usage 字段、记录、计费。

### Responses API

- `backend/internal/service/openai_gateway_service.go:4585`
  - `parseSSEUsageBytes` 从 `response.usage.input_tokens_details.cached_tokens` 写入 `CacheReadInputTokens`。

- `backend/internal/service/openai_gateway_service.go:4605`
  - `extractOpenAIUsageFromJSONBytes` 从 `usage.input_tokens_details.cached_tokens` 读取缓存 token。

### Chat Completions raw passthrough

- `backend/internal/service/openai_gateway_chat_completions_raw.go:349`
  - 流式 Chat Completions usage 解析读取 `usage.prompt_tokens_details.cached_tokens`。

- `backend/internal/service/openai_gateway_chat_completions_raw.go:385`
  - 非流式 Chat Completions usage 解析读取 `PromptTokensDetails.CachedTokens`。

### WS v2 passthrough

- `backend/internal/service/openai_ws_v2/passthrough_relay.go:672`
  - WS v2 passthrough 读取 `response.usage.input_tokens_details.cached_tokens`。

- `backend/internal/service/openai_ws_v2/passthrough_relay.go:693`
  - 解析后累加到 `state.usage.CacheReadInputTokens`。

### 计费

- `backend/internal/service/openai_gateway_service.go:5177`
  - 普通 input tokens 会扣掉 cache read tokens。

- `backend/internal/service/openai_gateway_service.go:5185`
  - `UsageTokens` 中单独传入 `CacheReadTokens`。

## 4. 发现的真实风险点

### 4.1 Chat Completions OAuth session 隔离不一致

该风险与 `270k cached_tokens` 不一定直接相关，但属于真实的 session/cache 隔离问题。

问题点：

- `backend/internal/service/openai_gateway_chat_completions.go:216`
  - Chat Completions OAuth 兼容路径在 `promptCacheKey != ""` 时设置：
    - `session_id = generateSessionUUID(promptCacheKey)`
  - 这里没有混入 API key ID。

对比其他路径：

- `backend/internal/service/openai_gateway_messages.go:176`
  - `/v1/messages` 兼容路径使用：
    - `generateSessionUUID(isolateOpenAISessionID(apiKeyID, promptCacheKey))`

- `backend/internal/service/openai_gateway_service.go:3851`
  - 通用 OAuth upstream request 使用 `isolateOpenAISessionID(apiKeyID, promptCacheKey)` 设置 `session_id` 和 `conversation_id`。

影响：

如果不同用户或不同 API key 使用相同显式 `prompt_cache_key`，或自动派生出相同 key，Chat Completions OAuth 兼容路径理论上可能映射到同一个上游 session。

建议修复：

```go
apiKeyID := getAPIKeyIDFromContext(c)
upstreamReq.Header.Set("session_id", generateSessionUUID(isolateOpenAISessionID(apiKeyID, promptCacheKey)))
```

### 4.2 管理员编辑 OAuth credentials 后 token cache 可能未失效

该问题属于 access token cache，不是 prompt `cached_tokens`。

风险：

管理员直接编辑 OAuth 账号 credentials 后，部分路径没有统一清 access token cache，可能继续使用旧 token 到 TTL 过期。

相关代码：

- `backend/internal/handler/admin/account_handler.go:615`
  - `AccountHandler.Update` 将 `req.Credentials` 传给 `adminService.UpdateAccount`。

- `backend/internal/service/admin_service.go:2438`
  - `UpdateAccount` 在 `len(input.Credentials) > 0` 时直接更新 `account.Credentials`。

- `backend/internal/service/openai_token_provider.go:138`
  - OpenAI token provider 会先读 token cache。

- `backend/internal/handler/admin/account_handler.go:936`
  - `RefreshAccountToken` 路径会调用 `InvalidateToken`，说明项目已有 invalidation 机制，但没有覆盖所有 credentials 修改入口。

建议：

将 token cache invalidation 下沉到 `adminService.UpdateAccount` 和批量更新 credentials 的 service 层，避免不同 handler 遗漏。

### 4.3 Vertex service account token cache invalidator 覆盖不完整

该问题属于 service account access token cache。

相关代码：

- `backend/internal/service/vertex_service_account.go:133`
  - Vertex service account token cache key 格式为 `vertex:service_account:{fingerprint}`。

- `backend/internal/service/token_cache_invalidator.go:23`
  - `CompositeTokenCacheInvalidator` 只处理 `AccountTypeOAuth`，会跳过 service account。

影响：

如果 service account 凭证轮换后 `private_key_id` 改变，会自然换 key；但如果同一 `private_key_id` 下替换或修复私钥内容，旧 access token 可能继续命中直到 TTL 到期。

### 4.4 legacy postUsageBilling 降级路径 cache 同步不足

该问题属于 billing/auth cache，不是 OpenAI prompt cache。

相关代码：

- `backend/internal/service/gateway_service.go:7909`
  - legacy fallback 直接扣订阅或余额。

- `backend/internal/service/gateway_service.go:7944`
  - 函数尾部没有调用统一 finalize cache 同步。

- `backend/internal/service/gateway_service.go:8078`
  - 正常 repo path 会通过 `finalizePostUsageBilling` 更新余额、订阅、API key rate limit cache。

影响：

如果生产中进入 legacy fallback，余额、订阅或 API key auth cache 可能在 TTL 内滞后。

## 5. 建议排查 Context7 MCP 请求

优先抓一条出现 `cached_tokens ~= 270k` 的入站请求 body，按字段统计大小和 token 来源：

- `instructions`
- `input`
- `messages`
- `tools`
- MCP resources
- MCP tool results
- tool schema
- `prompt_cache_key`
- `session_id`
- `conversation_id`
- `previous_response_id`
- `max_output_tokens`

重点确认 Context7 是否把完整分布式上下文、文档片段、资源索引、工具返回结果反复塞到稳定前缀里。

建议处理方向：

1. 限制 Context7/MCP 每轮物化到 prompt 的上下文大小。
2. 只注入命中的资源片段，不注入完整资源或完整索引。
3. 对历史上下文和 MCP resources 做摘要或截断。
4. 避免把完整 distributed context 每轮作为 `instructions`、system message 或第一条 user message 重复发送。
5. 检查客户端是否固定发送相同 `prompt_cache_key`、`session_id` 或 `conversation_id`。
6. 根据输入 token 和模型上下文窗口动态设置 `max_output_tokens`，避免请求进入上游前已经没有足够输出预算。

## 6. 后续建议

优先级建议：

1. 抓取并统计 Context7 MCP 触发问题的真实入站请求，确认 270k token 分布在哪些字段。
2. 修复 Chat Completions OAuth 兼容路径的 session 隔离不一致问题。
3. 将 OAuth credentials 更新后的 token cache invalidation 下沉到 service 层。
4. 补齐 Vertex service account token cache invalidation。
5. 如果 legacy billing fallback 允许在生产运行，补齐 billing/auth cache 同步；如果不允许，增加保护或明确注释。

