# 完善 GPT Image 访问控制、计费、并发与流式稳定性

1. 新增分组级图片生成开关，支持按用户分组控制是否允许使用 `/v1/images/*` 和 `/v1/responses` 的 `image_generation` 能力。

2. 新增图片生成独立倍率配置，支持图片费用选择沿用分组倍率或使用独立图片倍率，避免图片计费和文本 token 计费强耦合。

3. 新增 1K、2K、4K 图片价格配置和尺寸档位识别，支持按最终图片数量和图片尺寸进行更准确的图片计费。

4. 完善 OpenAI Images API 支持，覆盖 `/v1/images/generations`、`/v1/images/edits` 以及无 `/v1` 前缀别名。

5. 支持 OpenAI API Key 账号直连上游 Images API，客户端 `stream:true` 时保留流式上游调用并透传 SSE。

6. 支持 OpenAI OAuth 账号通过 Responses `image_generation` tool 承载 Images API，请求会转换为上游流式 Responses 调用。

7. 支持 Images API 非流式客户端请求在 OAuth 分支内部使用上游流式调用后聚合为标准 Images JSON 响应。

8. 支持 `/v1/responses` 中显式使用 `image_generation` tool 的图片生成、图生图和改图请求。

9. 支持 `/v1/responses` 中直接使用 `gpt-image-*` 主模型时自动规范化为文本主模型加 `image_generation` tool，兼容 Responses API 调用方式。

10. 新增 Codex 客户端图片生成桥接指令，在支持图片生成的模型上自动注入 `image_generation` tool，降低 Codex 生图调用失败率。

11. 新增 Spark/Codex 不支持图片生成时的输入校验和提示，避免不支持图片能力的模型错误接收图片输入或图片生成工具。

12. 新增图片生成专用并发隔离配置，支持独立限制图片请求最大并发，避免长耗时图片生成占满普通文本请求并发槽。

13. 支持图片并发溢出时选择直接拒绝或等待排队，并支持等待超时和最大等待队列限制。

14. 新增图片流式专用上游数据间隔超时配置，避免直接套用普通文本流式超时导致长耗时图片任务被误杀。

15. 新增图片流式 keepalive 配置，降低代理、浏览器、SDK 或中间网络在长时间无图片事件时断开连接的概率。

16. 优化图片流式请求的客户端断开处理，客户端断开后服务端继续 drain 上游，尽量保留最终 usage 和图片数量用于计费。

17. 修复图片流式请求在已经产生最终图片后遇到尾部错误时跳过用量记录的问题，避免图片已生成但未扣费。

18. 修复 usage worker 队列满时图片用量记录任务可能被丢弃的问题，图片结果会走强制记录 fallback，降低漏计费风险。

19. 修复多行 SSE `data:` 事件解析问题，确保 OpenAI/兼容上游返回多行 SSE JSON 时仍能正确解析 usage、终止事件和图片输出。

20. 优化 Images API API Key 流式返回的图片输出统计，支持从 Images API 风格 SSE、Responses SSE 和 fallback JSON 中识别最终图片数量。

21. 优化 OAuth Images 分支的流式转换逻辑，支持 partial image、output item done、response completed 等多种 Responses 图片事件形态。

22. 修复 OAuth Images 非流式聚合场景对 Responses SSE 的解析，确保最终图片、usage、tool usage 和元信息能正确转换为 Images API JSON。

23. 新增图片输出去重计数逻辑，避免 partial image 或重复 done/completed 事件导致图片数量重复计算。

24. 新增图片用量在 Usage Log 中的记录字段，包括 `ImageCount`、`ImageSize` 和图片计费模型，便于后续账单和审计。

25. 优化用量页和管理端展示，支持显示图片单位价格、图片数量、图片尺寸和图片相关费用信息。

26. 新增 `gpt-5.4-nano` OpenAI 模型别名归一化，提升模型映射、计费模型识别和统计展示准确性。

27. 修复 OpenAI compact / model alias 场景下的计费模型归一化问题，避免未知模型 fallback 造成错误计费。

28. 删除未使用的 OpenAI Codex helper，减少无效代码和维护噪音。

29. 补充图片生成访问控制、计费、并发、流式断连、多行 SSE、usage fallback 和模型 alias 的后端测试覆盖。

30. 补充部署配置示例，暴露图片生成并发隔离、图片流式超时、图片 keepalive 和图片计费相关配置项。
