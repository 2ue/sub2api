# 图片独立并发与外部图片网关方案

## 结论

当前阶段只在主服务内实现“图片独立并发开关”，外部独立图片网关不做运行时代码改造。这样可以先保护普通文本流式请求不被图片长连接无限挤压，同时避免引入新的请求转发链路、鉴权链路和计费绕行风险。

## 当前落地范围

- 新增 `gateway.image_concurrency.enabled`，默认 `false`。
- 新增 `gateway.image_concurrency.max_concurrent_requests`，默认 `0`。
- 新增 `gateway.image_concurrency.overflow_mode`，默认 `reject`，可选 `reject` 或 `wait`。
- 新增 `gateway.image_concurrency.wait_timeout_seconds`，默认 `30`，仅 `overflow_mode=wait` 时生效。
- 新增 `gateway.image_concurrency.max_waiting_requests`，默认 `100`，仅 `overflow_mode=wait` 时生效，限制当前进程内图片等待队列长度。
- 只有当 `enabled=true` 且 `max_concurrent_requests>0` 时，才启用图片独立并发限制。
- 限制覆盖明确图片生成意图：
  - `/v1/images/generations`
  - `/v1/images/edits`
  - `/v1/responses` 中 image 模型、显式 `tools[].type=image_generation`、显式 `tool_choice=image_generation`
- 普通 coding/text 请求不受图片并发限制影响。

## 与现有等待/超时配置的区别

当前项目已有几类“等待/超时”配置，但它们不能直接等同于图片并发等待：

- `gateway.image_stream_data_interval_timeout` 是图片流式请求已经进入上游后，等待上游下一段 SSE 数据的空闲超时，不是排队等待图片并发槽位的超时。
- `gateway.stream_data_interval_timeout` 是普通文本流式上游数据间隔超时，不应该用于图片排队等待。
- `gateway.scheduling.sticky_session_wait_timeout` 和 `gateway.scheduling.fallback_wait_timeout` 是账号调度/账号并发等待计划，不是图片全局资源保护。
- `gateway.user_message_queue.wait_timeout_ms` 是用户消息串行队列等待，不是图片生成并发控制。

因此图片独立并发需要自己的溢出策略和等待超时，避免复用现有配置导致语义混乱。

## 溢出策略建议

- `overflow_mode=reject`：图片并发满时立即返回 `429 rate_limit_error`。这是默认值，最能保护普通文本流式接口。
- `overflow_mode=wait`：图片并发满时在当前进程内等待槽位，最多等待 `wait_timeout_seconds`，等待队列最多 `max_waiting_requests`。该模式对用户更友好，但会增加等待连接数量，建议只在图片并发上限较低且前置代理连接容量充足时启用。

## 为什么外部图片网关先不做代码

独立图片网关只有在独立部署为单独进程、单独容器或单独机器时，才能真正隔离 CPU、内存、文件描述符、HTTP 连接池和长连接资源。如果只在当前进程里新增一个“图片转发 handler”，仍然共享同一个服务进程和连接池，不能解决资源隔离问题。

本次不做外部网关代码，原因：

- 避免新增内部反向代理链路导致鉴权、限流、计费、错误透传重复实现。
- 避免 `/v1/responses` image tool 请求被 path-only 路由漏判。
- 避免把普通 `/v1/responses` coding 请求误转发到图片网关。
- 保留主服务的图片 intent 检测和计费兜底，防止直连主服务绕过图片规则。

## 后续外部网关推荐方案

### 路径分流

前置代理可以安全按 path 分流：

- `/v1/images/generations` → 图片服务
- `/v1/images/edits` → 图片服务
- `/images/generations` → 图片服务
- `/images/edits` → 图片服务

这些接口天然是图片生成接口，path-only 分流不会误伤文本请求。

### `/v1/responses` 分流

`/v1/responses` 不能只靠 path 分流，因为普通 coding 请求和 image tool 请求都使用同一个 path。

可选方案：

1. 前置网关读取 JSON body，命中以下任一条件时转发图片服务：
   - `model` 是 `gpt-image-*` 或其他 image 模型
   - `tools` 数组包含 `{ "type": "image_generation" }`
   - `tool_choice` 明确选择 `image_generation`
2. 如果前置网关不能读取 body，则不要把 `/v1/responses` 整体转给图片服务，继续由主服务兜底识别和计费。
3. 即使未来实现 body-aware 分流，主服务仍必须保留图片开关、图片计费和图片并发兜底，防止用户绕过前置网关直连主服务。

## 多实例容量计算

当前图片独立并发限制是进程级限制。多实例部署时，总图片并发上限约等于：

```text
总图片并发上限 = 实例数 × gateway.image_concurrency.max_concurrent_requests
```

如果需要跨实例严格全局上限，后续需要在 Redis 中新增图片专用并发槽位，例如 `concurrency:image:global`，并配套启动清理和 TTL 语义。本次不扩展 Redis 并发接口，避免影响已有用户/账号并发行为。

## 与计费关系

图片独立并发只决定请求是否允许进入图片生成流程，不改变图片计费：

- 图片生成开关仍由分组 `allow_image_generation` 控制。
- 图片价格仍使用现有 1K/2K/4K 设置。
- 图片倍率仍使用现有共享/独立图片倍率逻辑。
- 流式图片仍以最终图片输出计数，客户端断开后的上游 drain 逻辑保持不变。
