# 生图计费风险分析

日期：2026-04-29  
范围：当前 `sub2api` 代码库的分组控制、OpenAI/Gemini/Antigravity 生图转发、用量记录与扣费逻辑；对照只读参考项目 `~/Desktop/project/image2images`。

## 2026-04-29 更新结论

本文件记录的是实施前风险分析与推导过程。规范化解决方案和实施记录已经沉淀到：

- `2ue/image-billing-openspec-solution.md`
- `openspec/changes/add-image-generation-billing-controls/proposal.md`
- `openspec/changes/add-image-generation-billing-controls/design.md`
- `openspec/changes/add-image-generation-billing-controls/specs/image-generation-access-control/spec.md`
- `openspec/changes/add-image-generation-billing-controls/specs/image-generation-billing-accounting/spec.md`

最终实现采用兼容优先方案：

1. 保留现有分组倍率和 `image_price_1k/2k/4k`。
2. 新增 `allow_image_generation` 控制分组是否允许生图。
3. 新增 `image_rate_independent` 控制图片是否共享当前有效分组倍率，默认 `false`。
4. 新增 `image_rate_multiplier`，仅在独立模式下参与图片扣费。
5. 所有 Responses/Images/WS 可产图路径均按最终图片数量写 `ImageCount` 并进入 image 计费。

因此，本文件下方“图片价格解耦”章节中的“图片价格为最终用户单价”只在 `image_rate_independent=true 且 image_rate_multiplier=1` 时成立；默认兼容模式仍是 `图片价格 * 当前有效分组倍率`。

## 结论

当前实现存在确定的生图计费与权限风险。单独设置“图片分组”可以降低运营混乱，但不能单独解决问题；必须同时增加“生图能力开关”和“图片计费独立倍率/独立最终单价”。原因如下：

1. 代码中没有分组级 `allow_image_generation` 或等价字段，普通 OpenAI 编码分组只要能使用 `/v1/responses` 的 `gpt-5.4` / `gpt-5.5`，就能携带或被注入 `image_generation` tool。
2. 图片价格最终扣费仍乘以分组/用户分组倍率，无法把“普通 token 折扣倍率”和“图片最终单价”解耦。
3. 专用 `/v1/images/*` 路径在部分场景会进入图片计费；通用 `/v1/responses` 的 `image_generation` tool 路径不会把产图数量写入 `OpenAIForwardResult.ImageCount`，因此按 token 模式计费，甚至在上游不返回 usage 时不写用量。
4. `gpt-5.4` / `gpt-5.5` 在当前代码中是普通 Responses 文本模型；它们通过 `tools:[{"type":"image_generation"}]` 生图时，当前计费入口只看模型和 token usage，不看 tool 产物数量。
5. 渠道 `RestrictModels` 不能作为生图能力开关；它只限制模型名，无法限制 `gpt-5.4` / `gpt-5.5` 上的 `image_generation` tool，并且在 OpenAI 高级调度开启时存在绕过路径。

最终建议：采用“图片能力开关 + 分组级独立图片定价 + 可选图片分组”的组合方案。普通编码分组不是必须关闭生图能力；它可以按业务需要开启，但开启后必须能在该分组内直接配置图片最终单价或图片独立倍率。图片分组只用于运营隔离和默认承载，不是唯一方案；计费逻辑必须对所有实际产图路径统一按图片数量计费。

## 当前代码链路

### 专用 OpenAI Images API

入口只允许 OpenAI 平台分组：

- `backend/internal/server/routes/gateway.go:91`：`/v1/images/generations` 非 OpenAI 平台直接返回 `Images API is not supported for this platform`。
- `backend/internal/server/routes/gateway.go:103`：`/v1/images/edits` 非 OpenAI 平台直接返回 404。
- `backend/internal/server/routes/gateway.go:158`：无 `/v1` 前缀的 `/images/generations` 别名执行同样平台检查。
- `backend/internal/server/routes/gateway.go:170`：无 `/v1` 前缀的 `/images/edits` 别名执行同样平台检查。

处理与转发链路：

- `backend/internal/handler/openai_images.go:23`：`Images` handler 负责两个 Images endpoint。
- `backend/internal/handler/openai_images.go:71`：解析 OpenAI Images 请求。
- `backend/internal/handler/openai_images.go:91`：只解析渠道映射；没有分组生图开关检查。
- `backend/internal/handler/openai_images.go:110`：只执行通用计费资格检查，不做预计图片成本检查。
- `backend/internal/handler/openai_images.go:130`：调度 OpenAI 账号。
- `backend/internal/handler/openai_images.go:180`：转发到上游。
- `backend/internal/handler/openai_images.go:263`：异步调用 `RecordUsage` 记录用量。

Images 请求约束：

- `backend/internal/service/openai_images.go:153`：解析后应用默认值。
- `backend/internal/service/openai_images.go:390`：未指定模型时默认 `gpt-image-2`。
- `backend/internal/service/openai_images.go:393`：`isOpenAIImageGenerationModel` 只接受 `gpt-image-` 前缀。
- `backend/internal/service/openai_images.go:397`：专用 Images endpoint 拒绝非图片模型。
- `backend/internal/service/openai_images.go:470`：尺寸只把 `1024x1024` 归为 `1K`，把 `1536x1024`、`1024x1536`、`1792x1024`、`1024x1792`、空值、`auto` 归为 `2K`，其余全部归为 `2K`。

专用 Images 转发后的图片计数：

- `backend/internal/service/openai_images.go:597`：API Key 图片路径先把 `imageCount` 设为请求的 `n`。
- `backend/internal/service/openai_images.go:600`：API Key 流式路径调用 `handleOpenAIImagesStreamingResponse`。
- `backend/internal/service/openai_images.go:605`：API Key 流式路径会用 `streamCount` 覆盖请求 `n`。
- `backend/internal/service/openai_images.go:608`：API Key 非流式路径调用 `handleOpenAIImagesNonStreamingResponse`。
- `backend/internal/service/openai_images.go:626`：API Key 图片路径返回 `OpenAIForwardResult.ImageCount`。
- `backend/internal/service/openai_images_responses.go:229`：OAuth 图片路径使用 Responses API 构造请求，主模型固定为 `gpt-5.4-mini`。
- `backend/internal/service/openai_images_responses.go:245`：OAuth 图片路径把 `image_generation` tool 的 `model` 设为请求的 `gpt-image-*`。
- `backend/internal/service/openai_images_responses.go:603`：OAuth 流式路径识别 `response.image_generation_call.partial_image`。
- `backend/internal/service/openai_images_responses.go:624`：OAuth 流式路径识别 `response.output_item.done`。
- `backend/internal/service/openai_images_responses.go:644`：OAuth 流式路径识别 `response.completed`。
- `backend/internal/service/openai_images_responses.go:850`：OAuth 图片路径返回 `OpenAIForwardResult.ImageCount`。

### 通用 OpenAI Responses API

通用 `/v1/responses` 当前会为 Codex CLI 注入或保留图片工具：

- `backend/internal/service/openai_codex_transform.go:9`：模型映射包含 `gpt-5.5`。
- `backend/internal/service/openai_codex_transform.go:11`：模型映射包含 `gpt-5.4`。
- `backend/internal/service/openai_codex_transform.go:51`：桥接指令明确告诉 Codex 使用 Responses 原生 `image_generation` tool。
- `backend/internal/service/openai_codex_transform.go:540`：`ensureOpenAIResponsesImageGenerationTool` 会为非 Spark Codex 请求注入 `{"type":"image_generation","output_format":"png"}`。
- `backend/internal/service/openai_codex_transform.go:578`：只要请求有 `image_generation` tool，就追加桥接指令。
- `backend/internal/service/openai_codex_transform.go:618`：校验只拒绝“`image_generation` tool + `gpt-image-*` 作为 Responses 主模型”的组合；不会拒绝 `gpt-5.4` / `gpt-5.5` + `image_generation` tool。
- `backend/internal/service/openai_codex_transform.go:629`：`/v1/responses` 中如果主模型是 `gpt-image-*`，会被改写为 `gpt-5.4-mini` + `image_generation` tool。

转发和用量解析：

- `backend/internal/service/openai_gateway_service.go:2111`：Codex CLI 请求会注入 `image_generation` tool。
- `backend/internal/service/openai_gateway_service.go:2117`：会规范化 `image_generation` tool 参数。
- `backend/internal/service/openai_gateway_service.go:2122`：会追加 image generation bridge instructions。
- `backend/internal/service/openai_gateway_service.go:2162`：检测到 `image_generation` tool 时只打日志。
- `backend/internal/service/openai_gateway_service.go:2699`：流式 Responses 调用 `handleStreamingResponse`。
- `backend/internal/service/openai_gateway_service.go:2706`：非流式 Responses 调用 `handleNonStreamingResponse`。
- `backend/internal/service/openai_gateway_service.go:2726`：通用 Responses 返回 `OpenAIForwardResult` 时没有设置 `ImageCount` 或 `ImageSize`。
- `backend/internal/service/openai_gateway_service.go:2930`：OpenAI 透传路径返回 `OpenAIForwardResult` 时也没有设置 `ImageCount` 或 `ImageSize`。
- `backend/internal/service/openai_gateway_service.go:4025`：`openaiStreamingResult` 只有 usage 和首 token 时间，没有图片数量字段。
- `backend/internal/service/openai_gateway_service.go:4273`：流式 Responses 只解析 SSE usage。
- `backend/internal/service/openai_gateway_service.go:4477`：流式 usage 只解析 `response.usage.output_tokens_details.image_tokens`。
- `backend/internal/service/openai_gateway_service.go:4480`：非流式 JSON usage 只解析 token 与 `image_tokens`，不解析产图数量。
- `backend/internal/service/openai_gateway_service.go:4686`：SSE 转 JSON 时可以识别 `image_generation_call` 输出。
- `backend/internal/service/openai_gateway_service.go:4718`：`extractImageGenerationOutputFromSSEData` 确认 `response.output_item.done` 且 `item.type == image_generation_call` 且 `result` 非空。
- `backend/internal/service/openai_gateway_service.go:4711`：重建的 output 只写回响应体，不返回图片数量给计费结果。

测试也体现了“能重建图片输出，但只得到 image token”的现状：

- `backend/internal/service/openai_gateway_service_test.go:2181`：测试名称为 `TestHandleSSEToJSON_ReconstructsImageGenerationOutputItemDone`。
- `backend/internal/service/openai_gateway_service_test.go:2193`：测试输入包含 `image_generation_call.result`。
- `backend/internal/service/openai_gateway_service_test.go:2201`：断言解析到 `ImageOutputTokens == 4`。
- `backend/internal/service/openai_gateway_service_test.go:2203`：断言响应体里有 `image_generation_call` 输出。

该测试没有断言任何图片数量，因为当前函数签名没有返回图片数量。

### 计费入口

OpenAI 用量记录只在 `ImageCount > 0` 时走图片计费：

- `backend/internal/service/openai_gateway_service.go:236`：`OpenAIForwardResult` 有 `ImageCount` 字段。
- `backend/internal/service/openai_gateway_service.go:237`：`OpenAIForwardResult` 有 `ImageSize` 字段。
- `backend/internal/service/openai_gateway_service.go:5033`：当 token 与 `ImageCount` 全为 0 时直接跳过用量记录。
- `backend/internal/service/openai_gateway_service.go:5053`：把 OpenAI usage 转为统一 token 输入。
- `backend/internal/service/openai_gateway_service.go:5090`：调用 `calculateOpenAIRecordUsageCost`。
- `backend/internal/service/openai_gateway_service.go:5130`：用量日志记录 `ImageCount`。
- `backend/internal/service/openai_gateway_service.go:5131`：用量日志记录 `ImageSize`。
- `backend/internal/service/openai_gateway_service.go:5154`：优先使用 `CostBreakdown.BillingMode`。
- `backend/internal/service/openai_gateway_service.go:5157`：只有 `result.ImageCount > 0` 才兜底设置 `billing_mode=image`。
- `backend/internal/service/openai_gateway_service.go:5228`：只有 `result.ImageCount > 0` 才调用 `calculateOpenAIImageCost`。

图片价格公式：

- `backend/internal/service/billing_service.go:787`：图片价格配置只有 `Price1K`、`Price2K`、`Price4K`。
- `backend/internal/service/billing_service.go:800`：`CalculateImageCost(model, imageSize, imageCount, groupConfig, rateMultiplier)` 接收倍率。
- `backend/internal/service/billing_service.go:806`：先取图片单价。
- `backend/internal/service/billing_service.go:809`：`TotalCost = unitPrice * imageCount`。
- `backend/internal/service/billing_service.go:815`：`ActualCost = TotalCost * rateMultiplier`。
- `backend/internal/service/billing_service.go:820`：图片计费模式写为 `image`。
- `backend/internal/service/billing_service.go:848`：默认图片价格来自 LiteLLM `output_cost_per_image`，否则用硬编码默认值。
- `backend/internal/service/billing_service.go:860`：硬编码默认基础价格为 `0.134`。
- `backend/internal/service/billing_service.go:866`：`2K` 默认价格为基础价 `1.5` 倍。
- `backend/internal/service/billing_service.go:869`：`4K` 默认价格为基础价 `2` 倍。

测试确认倍率会继续乘到图片价格上：

- `backend/internal/service/billing_service_image_test.go:65`：`2K` 默认价格为 `0.201` 时，倍率 `1.5`。
- `backend/internal/service/billing_service_image_test.go:67`：实际费用断言为 `0.3015`。

扣费使用 `ActualCost`：

- `backend/internal/service/gateway_service.go:7617`：余额/订阅扣费使用倍率后的 `ActualCost`。
- `backend/internal/service/gateway_service.go:7623`：订阅消耗写入 `SubscriptionCost = ActualCost`。
- `backend/internal/service/gateway_service.go:7625`：余额扣费写入 `BalanceCost = ActualCost`。
- `backend/internal/service/gateway_service.go:7629`：API Key quota 消耗写入 `ActualCost`。
- `backend/internal/service/gateway_service.go:7635`：账号 quota 消耗使用 `TotalCost * accountRateMultiplier`。

## 用户提出的四个风险逐项结论

### 1. “无法为分组关闭生图能力”

结论：成立。

代码证据：

- `backend/ent/schema/group.go:77`：分组只有 `image_price_1k`。
- `backend/ent/schema/group.go:81`：分组只有 `image_price_2k`。
- `backend/ent/schema/group.go:85`：分组只有 `image_price_4k`。
- `backend/ent/schema/group.go:120`：`supported_model_scopes` 是模型系列配置。
- `backend/internal/service/group.go:48`：`SupportedModelScopes` 注释限定为 Antigravity 模型系列。
- `backend/internal/handler/admin/group_handler.go:95`：创建分组 DTO 只有图片价格字段，没有生图开关字段。
- `backend/internal/handler/admin/group_handler.go:132`：更新分组 DTO 只有图片价格字段，没有生图开关字段。
- `backend/internal/service/admin_service.go:188`：创建分组输入只有图片价格字段。
- `backend/internal/service/admin_service.go:225`：更新分组输入只有图片价格字段。
- `backend/internal/service/admin_service.go:1417`：创建分组只保存图片价格。
- `backend/internal/service/admin_service.go:1593`：更新分组只更新图片价格。

当前可用控制项不是生图开关：

- 平台限制只能阻止非 OpenAI 分组访问 `/v1/images/*`，不能阻止 OpenAI 编码分组在 `/v1/responses` 使用 `image_generation` tool。
- 渠道 `RestrictModels` 只能按模型名限制，不能表达“同一个 `gpt-5.4` 模型允许写代码但禁止调用图片工具”。
- `supported_model_scopes` 在当前 Go 代码中只出现在分组保存、DTO、支付配置展示等路径；没有请求入口依据它拒绝生图请求。

影响：

- 普通 OpenAI 编码分组只要允许 `gpt-5.4` / `gpt-5.5`，用户就可以显式携带 `tools:[{"type":"image_generation"}]`。
- Codex CLI 请求还会在服务端自动注入 `image_generation` tool，普通编码分组没有能力关闭该注入。

### 2. “分组设置的生图单价、倍率无法独立”

结论：成立。

当前最终扣费公式：

```text
图片最终扣费 = 图片单价 * 图片数量 * 分组/用户分组倍率
```

代码证据：

- `backend/internal/service/openai_gateway_service.go:5067`：OpenAI 用量记录会通过用户分组倍率解析器取倍率。
- `backend/internal/service/user_group_rate_resolver.go:79`：用户分组倍率存在时覆盖分组默认倍率。
- `backend/internal/service/openai_gateway_service.go:5281`：OpenAI 图片 fallback 计费把同一个 `multiplier` 传给 `CalculateImageCost`。
- `backend/internal/service/billing_service.go:809`：图片 `TotalCost` 是单价乘数量。
- `backend/internal/service/billing_service.go:815`：图片 `ActualCost` 再乘倍率。

示例：

```text
分组倍率 = 0.15
期望图片最终扣费 = 0.2 / 张
当前必须配置图片单价 = 0.2 / 0.15 = 1.3333333333
```

该配置不是“图片单价”，而是“为了抵消分组倍率而倒推的倍率前单价”。这会导致后台价格不可读、不可审计，并且用户分组倍率覆盖时同一图片价格继续变化。

澄清后的需求结论：

- 普通编码分组可以开启生图能力。
- 普通编码分组开启生图能力后，也必须支持在本分组设置图片价格。
- 管理员在本分组里设置的“0.2 / 张”必须最终扣 `0.2 / 张`，不能再被普通 token 的 `rate_multiplier=0.15` 改写成 `0.03 / 张`。
- 如果仍需给图片也做折扣，必须使用图片自己的 `image_rate_multiplier` 或等价字段，而不是复用普通 token 的 `rate_multiplier`。
- 当前实现不能满足该需求，因为 `CalculateImageCost` 与渠道 `image/per_request` 计费都会继续乘普通分组/用户分组倍率。

渠道图片定价同样被倍率影响：

- `backend/internal/service/billing_service.go:570`：统一按次/图片计费的 `totalCost = unitPrice * count`。
- `backend/internal/service/billing_service.go:571`：统一按次/图片计费的 `actualCost = totalCost * input.RateMultiplier`。

### 3. “调用生图模型时，如果是流式调用会当成普通模型来计费”

结论：分两类；通用 `/v1/responses` 生图成立，专用 `/v1/images/*` 不能笼统成立。

专用 `/v1/images/*`：

- OAuth 图片路径的流式处理会识别 Responses 生图事件并设置 `ImageCount`。
- API Key 图片路径流式处理只通过 `data` 数组计算图片数量；如果上游流式事件是 `image_generation.completed` / `b64_json` 形态，当前计数为 0。

代码证据：

- `backend/internal/service/openai_images_responses.go:678`：OAuth 图片流式路径把已发出的图片数量设置为 `imageCount`。
- `backend/internal/service/openai_images_responses.go:850`：OAuth 图片路径返回 `ImageCount`。
- `backend/internal/service/openai_images.go:812`：API Key 图片流式路径初始 `imageCount := 0`。
- `backend/internal/service/openai_images.go:830`：API Key 图片流式路径只调用 `extractOpenAIImageCountFromJSONBytes`。
- `backend/internal/service/openai_images.go:865`：`extractOpenAIImageCountFromJSONBytes` 只统计 JSON 顶层 `data` 数组长度。
- `backend/internal/service/openai_images.go:842`：如果流式事件里没有顶层 `data` 数组，返回的图片数量为 0。

参考项目对 Images API 流式事件的解析不是顶层 `data` 数组：

- `/Users/yuanfeijie/Desktop/project/image2images/server/src/providers/openai-images-shared.ts:85`：解析 `image_generation.partial_image`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/providers/openai-images-shared.ts:94`：解析 `image_generation.completed`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/providers/openai-images-shared.ts:117`：如果没有 `image_generation.completed.b64_json` 就报错。

通用 `/v1/responses`：

- 该路径可以产出图片，但 `OpenAIForwardResult.ImageCount` 不会被设置。
- 只要 `ImageCount == 0`，`RecordUsage` 不进入图片计费分支。
- 如果上游返回 `image_tokens`，会按 token 模式计算；如果上游没有 usage，直接不写用量。

代码证据：

- `backend/internal/service/openai_gateway_service.go:2726`：普通 Responses 返回结果未设置 `ImageCount`。
- `backend/internal/service/openai_gateway_service.go:2930`：透传 Responses 返回结果未设置 `ImageCount`。
- `backend/internal/service/openai_gateway_service.go:5033`：token 与 `ImageCount` 全为 0 时跳过用量记录。
- `backend/internal/service/openai_gateway_service.go:5228`：图片计费分支只在 `ImageCount > 0` 时执行。
- `backend/internal/service/billing_service.go:506`：token 模式下 `ImageOutputTokens > 0` 只会走图片 token 价格。
- `backend/internal/service/billing_service.go:509`：未配置图片 token 价格时回退到普通输出 token 价格。

### 4. “5.4 / 5.5 也能调用 image tool 生图；如何避免或如何收费”

结论：成立。当前代码已把 `gpt-5.4` / `gpt-5.5` 当作普通 Responses 模型支持，且当前 Codex 转换逻辑会鼓励或注入 `image_generation` tool。

代码证据：

- `backend/internal/pkg/openai/constants.go:18`：默认 OpenAI 模型列表包含 `gpt-5.5`。
- `backend/internal/pkg/openai/constants.go:19`：默认 OpenAI 模型列表包含 `gpt-5.4`。
- `backend/internal/service/openai_codex_transform.go:10`：Codex 模型映射包含 `gpt-5.5`。
- `backend/internal/service/openai_codex_transform.go:11`：Codex 模型映射包含 `gpt-5.4`。
- `backend/internal/service/openai_codex_transform.go:540`：服务端会自动注入 `image_generation` tool。
- `backend/internal/service/openai_codex_transform.go:578`：有图片工具时会追加使用该工具的桥接说明。
- `backend/internal/service/openai_codex_transform.go:618`：当前校验不会拒绝 `gpt-5.4` / `gpt-5.5` + `image_generation` tool。

`image2images` 的实现方式：

- `/Users/yuanfeijie/Desktop/project/image2images/server/src/presets.ts:34`：定义 `responses_generate` 能力模式。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/presets.ts:39`：该模式默认模型是 `gpt-5.4`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/presets.ts:40`：该模式模型 allowlist 只有 `gpt-5.4`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/catalog.ts:24`：`gpt-5.4` 对应 provider 是 `openai-compatible-responses`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/catalog.ts:37`：`gpt-5.4` 支持流式预览。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/providers/openai-responses-image-adapter.ts:29`：构造 `image_generation` tool。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/providers/openai-responses-image-adapter.ts:50`：请求主模型使用参数中的 `model`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/providers/openai-responses-image-adapter.ts:53`：请求包含 `tools: [tool]`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/providers/openai-responses-image-adapter.ts:54`：请求强制 `tool_choice: { type: "image_generation" }`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/providers/openai-responses-image-adapter.ts:85`：流式解析 `response.output_item.done`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/providers/openai-responses-image-adapter.ts:87`：确认 `item.type === "image_generation_call"` 并读取 `item.result`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/providers/openai-responses-image-adapter.ts:94`：流式解析 `response.image_generation_call.partial_image`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/providers/openai-responses-image-adapter.ts:103`：流式解析 `response.completed`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/providers/openai-responses-image-adapter.ts:191`：非流式解析 `payload.output`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/providers/openai-responses-image-adapter.ts:192`：非流式查找 `image_generation_call` 并读取 `result`。

如何避免：

1. 增加分组级 `allow_image_generation`。
2. 在通用 `/v1/responses` 注入工具之前检查该开关；关闭时不注入。
3. 关闭时拒绝用户显式传入的 `tools[].type == "image_generation"`。
4. 关闭时拒绝 `/v1/responses` 主模型为 `gpt-image-*` 的请求，因为当前代码会把它改写成工具生图。
5. 关闭时拒绝 `/v1/images/generations` 和 `/v1/images/edits`。

如何收费：

1. 把 `image_generation_call.result` 视为产出一张图。
2. 流式 Responses 统计 `response.output_item.done` 中 `item.type == image_generation_call` 且 `result` 非空的去重数量。
3. 非流式 Responses 统计 `output[]` 中 `type == image_generation_call` 且 `result` 非空的去重数量。
4. `response.completed.response.output`、`response.output_item.done`、`response.tool_usage.image_gen.images` 三者都要作为计数来源；优先以实际 `result` 数量为准，`tool_usage.image_gen.images` 只能作为 fallback。
5. 将统计结果写入 `OpenAIForwardResult.ImageCount` 和 `ImageSize`，让 `RecordUsage` 进入 `calculateOpenAIImageCost`。

## 额外风险

### A. 渠道图片定价忽略多图数量

结论：成立。

OpenAI 图片路径如果命中渠道 `image` / `per_request` 定价，当前固定按 1 次计费：

- `backend/internal/service/openai_gateway_service.go:5254`：OpenAI 图片计费优先使用渠道定价。
- `backend/internal/service/openai_gateway_service.go:5261`：调用统一计费时固定 `RequestCount: 1`。
- `backend/internal/service/openai_gateway_service.go:5262`：传入了尺寸层级。
- `backend/internal/service/openai_gateway_service.go:5281`：只有回退到分组图片价格时才传 `result.ImageCount`。

通用网关也有同类问题：

- `backend/internal/service/gateway_service.go:8083`：图片计费命中渠道定价。
- `backend/internal/service/gateway_service.go:8095`：调用统一计费时固定 `RequestCount: 1`。
- `backend/internal/service/gateway_service.go:8115`：只有回退到分组图片价格时才传 `result.ImageCount`。

影响：

- 专用 OpenAI Images API 支持请求字段 `n`。
- `backend/internal/service/openai_images.go:176`：解析请求中的 `n`。
- `backend/internal/service/openai_images.go:180`：把 `n` 写入 `req.N`。
- 当实际产出多张图且渠道采用图片/按次定价时，当前代码只扣 1 次。

修正结论：

- 渠道图片计费必须传 `RequestCount: result.ImageCount`。
- 账号统计自定义规则也必须传真实图片数量；当前 `applyAccountStatsCost` 固定传 1。
- `backend/internal/service/account_stats_pricing.go:233`：账号统计成本计算固定 `requestCount=1`。

### B. OpenAI 高级调度会绕过渠道 `RestrictModels`

结论：成立。

代码证据：

- `backend/internal/service/openai_gateway_service.go:1329`：传统 `selectAccountForModelWithExclusions` 会调用 `checkChannelPricingRestriction`。
- `backend/internal/service/openai_gateway_service.go:1530`：传统 `selectAccountWithLoadAwareness` 会调用 `checkChannelPricingRestriction`。
- `backend/internal/service/openai_account_scheduler.go:1011`：`SelectAccountWithScheduler` 进入统一调度入口。
- `backend/internal/service/openai_account_scheduler.go:1055`：当高级 scheduler 存在时，不进入传统 `selectAccountWithLoadAwareness`。
- `backend/internal/service/openai_account_scheduler.go:1116`：直接调用 `scheduler.Select`。
- `backend/internal/service/openai_account_scheduler.go:891`：高级 scheduler 的请求兼容性检查只看账号。
- `backend/internal/service/openai_account_scheduler.go:895`：只检查 `account.IsModelSupported`。
- `backend/internal/service/openai_account_scheduler.go:898`：只检查 `account.SupportsOpenAIImageCapability`。

影响：

- 即使渠道开启 `RestrictModels` 并且没有把 `gpt-image-*` 放进定价列表，在高级调度启用时仍不会在 scheduler 层执行渠道限制。
- 即使修复了模型限制，`RestrictModels` 也仍然不能限制 `gpt-5.4` / `gpt-5.5` 的 `image_generation` tool，因为模型名本身仍是允许的文本模型。

### C. OpenAI Images API Key 流式计数格式过窄

结论：成立。

当前 API Key 直连 Images 流式路径只统计顶层 `data` 数组：

- `backend/internal/service/openai_images.go:812`：流式图片计数从 0 开始。
- `backend/internal/service/openai_images.go:827`：逐行读取 SSE data。
- `backend/internal/service/openai_images.go:830`：用 `extractOpenAIImageCountFromJSONBytes` 统计图片数。
- `backend/internal/service/openai_images.go:865`：该函数只处理顶层 `data` 数组。

参考项目的 Images API 流式解析使用 `image_generation.completed`：

- `/Users/yuanfeijie/Desktop/project/image2images/server/src/providers/openai-images-shared.ts:85`：处理 partial image。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/providers/openai-images-shared.ts:94`：处理 completed image。

影响：

- API Key 直连 `/v1/images/*` 如果上游返回 `image_generation.completed`，当前 `streamCount` 为 0。
- `backend/internal/service/openai_images.go:605` 会用 0 覆盖请求 `n`。
- 之后 `RecordUsage` 不会进入图片计费分支。

### D. OpenAI 尺寸分层会把高分辨率统一按 2K 收费

结论：成立。

代码证据：

- `backend/internal/service/openai_images.go:470`：OpenAI 图片尺寸分层函数。
- `backend/internal/service/openai_images.go:472`：只把 `1024x1024` 归为 `1K`。
- `backend/internal/service/openai_images.go:474`：只把少数横竖图和 `auto` 归为 `2K`。
- `backend/internal/service/openai_images.go:476`：所有其他尺寸默认归为 `2K`。

参考项目暴露的 popular sizes 包含更多尺寸：

- `/Users/yuanfeijie/Desktop/project/image2images/server/src/presets.ts:22`：定义 `OFFICIAL_GPT_IMAGE_POPULAR_SIZES`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/presets.ts:27`：包含 `2048x2048`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/presets.ts:28`：包含 `2048x1152`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/presets.ts:29`：包含 `3840x2160`。
- `/Users/yuanfeijie/Desktop/project/image2images/server/src/presets.ts:30`：包含 `2160x3840`。

影响：

- 当前分组配置有 `image_price_4k`，但 OpenAI Images 归一化函数没有任何路径返回 `4K`。
- 管理后台设置的 OpenAI `4K` 图片单价在当前 OpenAI Images 路径不会生效。

### E. 计费资格预检不校验预计图片费用

结论：成立。

代码证据：

- `backend/internal/handler/openai_images.go:110`：图片请求开始前只调用通用 `CheckBillingEligibility`。
- `backend/internal/service/billing_cache_service.go:650`：该方法注释是检查“是否有资格发起请求”。
- `backend/internal/service/billing_cache_service.go:775`：余额模式检查余额。
- `backend/internal/service/billing_cache_service.go:788`：只要余额 `> 0` 就放行。
- `backend/internal/service/billing_cache_service.go:820`：订阅模式只检查当前 usage 是否已达到限额。
- `backend/internal/repository/usage_billing_repo.go:176`：余额扣费直接 `balance = balance - amount`。
- `backend/internal/repository/usage_billing_repo.go:148`：订阅用量直接累加。

影响：

- 余额为 `0.01` 的用户可以发起最终扣费 `0.2` 的图片请求，请求完成后余额被扣成负数。
- 订阅剩余额度低于图片价格时，请求会先完成，再把 usage 累加到超过限额。
- 该风险不是图片路径独有，但图片单次成本显著高于普通 token 请求，影响更直接。

### F. 调度层的 OpenAI 图片能力不是权限控制

结论：成立。

代码证据：

- `backend/internal/service/openai_images.go:44`：定义 `OpenAIImagesCapability`。
- `backend/internal/service/openai_images.go:47`：定义 `images-basic`。
- `backend/internal/service/openai_images.go:48`：定义 `images-native`。
- `backend/internal/service/account.go:1049`：`SupportsOpenAIImageCapability` 实现。
- `backend/internal/service/account.go:1054`：`images-basic` 和 `images-native` 走同一个分支。
- `backend/internal/service/account.go:1055`：只要账号类型是 OAuth 或 API Key 就返回 true。

影响：

- `RequiredImageCapability` 不能用来表达“这个分组允许/禁止生图”。
- 它也不能精确表达账号是否真实支持某一类图片能力；失败只会在上游请求后表现为错误或 failover。

### G. 通用 Responses 图片输出在无 usage 时会完全不计费

结论：成立。

代码证据：

- `backend/internal/service/openai_gateway_service.go:4686`：SSE 转 JSON 能发现 `image_generation_call` 输出。
- `backend/internal/service/openai_gateway_service.go:4718`：确认图片输出的条件是 `image_generation_call.result` 非空。
- `backend/internal/service/openai_gateway_service.go:2726`：返回 `OpenAIForwardResult` 时不写 `ImageCount`。
- `backend/internal/service/openai_gateway_service.go:5033`：如果上游 usage 没有 token 且 `ImageCount == 0`，直接跳过用量记录。

影响：

- 上游返回图片结果但不返回 usage 时，该请求不会有 usage log，也不会扣费。
- 专用 Images OAuth 路径不会出现这个问题，因为它会从图片输出数量设置 `ImageCount`。

## 推荐方案

### 方案选择

结论：采用“能力开关 + 分组级独立图片定价 + 可选图片分组”，不要把“图片分组”作为唯一承载。

只做图片分组的缺陷：

- 普通编码分组仍然允许 `gpt-5.4` / `gpt-5.5` 文本模型。
- 用户可以在普通编码分组的 `/v1/responses` 请求里显式传 `image_generation` tool。
- Codex CLI 请求会被服务端自动注入 `image_generation` tool。
- 渠道 `RestrictModels` 不能区分“同一模型的文本能力”和“同一模型的图片工具能力”。
- 业务上允许普通编码分组开启生图时，图片分组无法表达“同一个普通分组内 token 走 0.15 倍，图片按 0.2 / 张最终扣费”的需求。

完整方案：

1. 分组新增 `allow_image_generation`，用于表达该分组是否允许生图；普通编码分组可以为 `true`，也可以为 `false`。
2. 分组新增图片独立价格语义：管理员设置 `image_price_1k/2k/4k` 时，该值必须能表示最终用户单价；如果需要折扣，再使用独立 `image_rate_multiplier`。
3. 迁移上线时为保持兼容，可以先把现有 OpenAI 分组置为 `true`，再由管理员按运营策略关闭部分普通编码分组；如果允许安全优先上线，则现有分组默认 `false`。
4. 创建独立图片分组作为推荐运营承载，`allow_image_generation=true`，并配置图片价格；但普通编码分组开启 `allow_image_generation=true` 时，也必须使用同一套图片独立定价逻辑。
5. 分组 `allow_image_generation=false` 时，服务端禁止 `/v1/images/*`、`gpt-image-*`、显式 `image_generation` tool、Codex 自动注入。
6. 分组 `allow_image_generation=true` 时，允许两类能力：专用 `gpt-image-*` Images API，以及 `gpt-5.4` / `gpt-5.5` Responses tool 生图，并且全部按实际 `ImageCount` 进入图片计费。

### 权限控制实现点

新增统一判断函数：

```text
isImageGenerationIntent(endpoint, requestBody, requestedModel) =
  endpoint 是 /v1/images/generations 或 /v1/images/edits
  OR requestedModel 是 gpt-image-*
  OR tools 中存在 type=image_generation
  OR tool_choice 指向 image_generation
```

必须接入的位置：

1. `backend/internal/handler/openai_images.go`：解析完 `OpenAIImagesRequest` 后，若分组未开启生图，直接 403。
2. `backend/internal/service/openai_gateway_service.go`：`ensureOpenAIResponsesImageGenerationTool` 之前，若分组未开启生图，不允许注入。
3. `backend/internal/service/openai_gateway_service.go`：解析原始请求后，若用户显式携带 `image_generation` tool 且分组未开启生图，直接 403。
4. `backend/internal/service/openai_codex_transform.go`：`normalizeOpenAIResponsesImageOnlyModel` 前，若分组未开启生图，不允许把 `gpt-image-*` 改写成工具生图。
5. `backend/internal/service/openai_account_scheduler.go`：高级 scheduler 进入 `scheduler.Select` 前或 `isAccountRequestCompatible` 内补充渠道限制；但这只能修复模型限制，不替代生图开关。

### 计费改造

1. 扩展通用 Responses 流式结果：
   - `openaiStreamingResult` 增加 `imageCount`、`imageSize`。
   - `handleStreamingResponse` 在 SSE 中统计 `response.output_item.done` 的 `image_generation_call.result`。
   - `handleStreamingResponsePassthrough` 做同样统计。

2. 扩展非流式结果：
   - `handleNonStreamingResponse` 解析 JSON `output[]` 中的 `image_generation_call.result`。
   - `handleSSEToJSON` 在 `reconstructResponseOutputFromSSE` 的同时返回图片数量。
   - `handleNonStreamingResponsePassthrough` 做同样统计。

3. 扩展专用 Images API Key 流式计数：
   - 继续支持顶层 `data` 数组。
   - 新增支持 `image_generation.completed` + `b64_json`。
   - 新增支持 `response.output_item.done` + `image_generation_call.result`。
   - 新增支持 `response.completed.response.output[]`。

4. 修正渠道图片计费：
   - `backend/internal/service/openai_gateway_service.go:5261` 从 `RequestCount: 1` 改为 `RequestCount: result.ImageCount`。
   - `backend/internal/service/gateway_service.go:8095` 从 `RequestCount: 1` 改为 `RequestCount: result.ImageCount`。
   - `applyAccountStatsCost` 增加 `requestCount` 参数，图片请求传真实 `ImageCount`。

5. 修正尺寸分层：
   - 明确定义 OpenAI Images 尺寸到 `1K` / `2K` / `4K` 的映射表。
   - 如果继续保留 `image_price_4k`，必须存在返回 `4K` 的尺寸。
   - 未识别尺寸不能静默归为 `2K`；应拒绝或按最高档计费。

### 图片价格解耦

推荐采用“图片价格为最终用户单价”的方案，该方案同时适用于图片分组和开启生图的普通编码分组：

```text
图片 ActualCost = image_unit_price[tier] * image_count
图片 TotalCost  = ActualCost
```

并新增单独字段控制图片折扣：

```text
image_rate_multiplier
```

如果不需要图片折扣，`image_rate_multiplier` 固定为 `1`。普通 token 的 `rate_multiplier` 不再影响图片最终扣费。

当前实现下的临时设置方式：

```text
目标图片最终单价 = P
当前普通 token 分组倍率 = R
当前必须填写的图片价格 = P / R
```

例如：

```text
P = 0.2
R = 0.15
当前必须填写 image_price = 1.3333333333
```

这个设置方式只能作为临时方案，不能作为长期方案。原因是字段名表达的是“图片价格”，但实际含义变成了“倍率前图片基价”；当用户分组倍率覆盖、分组倍率调整、渠道图片定价启用时，最终扣费会继续变化。

上线前的可选运营方案：

1. 创建图片分组。
2. 图片分组 `rate_multiplier=1`。
3. 图片分组配置当前希望展示的最终图片单价。
4. 普通编码分组按业务决定是否开启 `allow_image_generation`：如果开启，就按上面的临时公式倒推图片价格；如果关闭，就引导用户使用图片分组。

这个临时方案只有在新增生图能力开关之后才可控；没有开关时，普通编码分组仍能通过 `gpt-5.4` / `gpt-5.5` 工具生图绕过运营上设定的图片分组。

## 最终决策建议

1. 立即把“生图能力”从“模型名”中抽离出来，变成分组/渠道的一等权限。
2. 图片分组只作为运营承载，不作为唯一安全边界；普通编码分组也可以开启生图，但必须使用图片独立定价。
3. 所有能产生图片的路径都必须写 `ImageCount`；计费入口只接受“实际产图数量”作为图片计费依据。
4. 图片价格必须与普通 token 倍率解耦；后台展示的图片单价必须等于最终扣费单价。
5. 渠道图片/按次定价必须按 `ImageCount` 计数，不能固定 `RequestCount=1`。
6. 高级 OpenAI scheduler 必须补齐渠道 restriction，否则渠道模型限制在开启高级调度时失效。
7. 对 `gpt-5.4` / `gpt-5.5` 生图，服务端要么在普通组拒绝 `image_generation` tool，要么在图片组按 `image_generation_call.result` 数量收费；不能继续仅按 token usage 计费。
