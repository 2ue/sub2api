# 生图计费 OpenSpec 解决方案索引

日期：2026-04-29  
变更名：`add-image-generation-billing-controls`  
OpenSpec 状态：`openspec validate add-image-generation-billing-controls --strict` 已通过。

## 2026-04-29 实施结果

本提案已按 `add-image-generation-billing-controls` 实施完成，当前实现没有引入“强制图片分组”这一新业务前提，而是按提案支持“普通编码组可关闭/可开启生图、默认共享现有分组倍率、显式开启后使用图片独立倍率”。

### 已落地代码

1. 数据模型与迁移：
   - `backend/ent/schema/group.go` 新增 `allow_image_generation`、`image_rate_independent`、`image_rate_multiplier`。
   - `backend/migrations/134_image_generation_group_controls.sql` 新增三列并按兼容策略回填。
   - Ent 生成代码已更新，分组查询、创建、更新和 predicate 均包含新字段。

2. 后端配置链路：
   - Admin create/update、DTO、repository、auth cache、API key group snapshot 均传递新字段。
   - `image_rate_multiplier` 允许显式 `0`，但拒绝负数；负数清除价格的旧行为只保留给 `image_price_1k/2k/4k`。
   - 更新请求未传新字段时保留原值，避免编辑旧分组时覆盖历史行为。

3. 权限控制：
   - `backend/internal/service/image_generation_intent.go` 统一识别 `/v1/images/*`、`gpt-image-*`、`tools[].type=image_generation`、显式 image `tool_choice`。
   - `/v1/images/*` 在解析请求后、余额/账号调度前按 `allow_image_generation` 拒绝。
   - `/v1/responses` 在请求改写前和改写后均检查生图意图。
   - 禁用生图的 Codex 分组不会自动注入 `image_generation` tool，也不会追加图片桥接 instructions。
   - WebSocket 入站的首轮和后续 turn 均在上游发送前检查生图意图。

4. 图片数量归因：
   - `backend/internal/service/image_output_accounting.go` 统一统计最终图片产物，去重规则为 `id`、`call_id`、`result hash`。
   - HTTP Responses 非流式、流式、透传、WS 上游、WS 入站均统计 `image_generation_call.result`。
   - Images API 流式统计 `data[]`、`image_generation.completed`、`response.output_item.done`、`response.completed`。
   - `partial_image` 不计费。

5. 计费：
   - `backend/internal/service/image_billing_multiplier.go` 统一解析图片倍率。
   - 共享模式：图片倍率 = 当前有效分组倍率，包含用户专属分组倍率覆盖。
   - 独立模式：图片倍率 = `group.image_rate_multiplier`，普通 `rate_multiplier` 不参与图片扣费。
   - OpenAI 和非 OpenAI Gateway 的渠道 `billing_mode=image` 均使用真实 `ImageCount` 作为 `RequestCount`。
   - `ImageCount > 0` 且 token usage 为 0 时仍写 usage log 并按图片计费。
   - Responses 图片工具请求记录 token usage，但默认仍按 image 模式计费，不引入混合计费。

6. 前端：
   - `frontend/src/views/admin/GroupsView.vue` 增加“允许当前分组生图”“生图倍率独立”“生图独立倍率”。
   - 创建/编辑表单回显服务端保存值。
   - 增加 1K/2K/4K 最终单张价格预览。
   - `frontend/src/types/index.ts` 与中英文 i18n 已更新。

### 本次反思结论

1. 未越界：
   - 没有改变现有 `image_price_1k/2k/4k` 字段含义。
   - 没有强制所有生图必须走单独图片分组。
   - 没有引入预扣费、混合计费或用户级图片独立倍率。

2. 符合原始需求：
   - 普通编码组关闭生图时，文本请求继续可用，显式生图请求 403。
   - 普通编码组开启生图时，默认继续按 `图片价格 * 当前有效分组倍率` 扣费。
   - 管理员开启独立倍率后，图片按 `图片价格 * image_rate_multiplier` 扣费。
   - `gpt-5.4` / `gpt-5.5 + image_generation` 通过最终图片结果写入 `ImageCount` 并进入 image 计费。

3. 漏洞收敛：
   - Codex 自动注入不再绕过分组生图开关。
   - HTTP stream、HTTP non-stream、passthrough、WS 上游、WS 入站都不再因 `ImageCount=0` 退回 token 计费。
   - 渠道 image 模式多图请求不再固定按 1 次计费。
   - 未知显式 OpenAI 图片尺寸不再静默按 `2K` 收费。

4. 兼容保留：
   - 现有分组迁移到 `image_rate_independent=false`，因此历史最终图片扣费公式保持不变。
   - 新建分组默认不允许生图，避免新增普通编码组意外获得生图能力。
   - 现有 `openai`、`gemini`、`antigravity` 分组迁移为允许生图，避免升级时直接中断已有图片业务。

### 已执行验证

```bash
openspec validate add-image-generation-billing-controls --strict
CGO_ENABLED=0 go test ./internal/service
CGO_ENABLED=0 go test ./internal/handler -run TestOpenAIGatewayHandlerImages_DisabledGroupRejectsBeforeScheduling
CGO_ENABLED=0 go test ./internal/service ./internal/handler/... ./internal/repository/...
cd frontend && pnpm typecheck
```

验证结果：以上命令均通过。

## 规范化产物

- `openspec/config.yaml`：已初始化 OpenSpec，schema 为 `spec-driven`。
- `openspec/changes/add-image-generation-billing-controls/proposal.md`：定义变更原因、范围、影响和两个新增能力。
- `openspec/changes/add-image-generation-billing-controls/design.md`：定义当前代码约束、目标、非目标、技术决策、迁移策略和风险控制。
- `openspec/changes/add-image-generation-billing-controls/specs/image-generation-access-control/spec.md`：定义分组生图开关、生图意图识别、Codex 注入控制和渠道限制场景。
- `openspec/changes/add-image-generation-billing-controls/specs/image-generation-billing-accounting/spec.md`：定义图片倍率模式、Responses 图片数量归因、Images API 图片数量归因、渠道图片计费、尺寸档位和 usage log 语义。
- `openspec/changes/add-image-generation-billing-controls/tasks.md`：定义实现任务清单。

## 纠偏结论

上一版方案把“图片倍率独立”设计成默认必然使用，这不符合澄清后的需求。正确方案是：

- 保留现有分组倍率。
- 保留现有 `image_price_1k/2k/4k` 图片价格设置。
- 新增“是否支持生图”开关。
- 新增“生图倍率是否独立”开关，默认 `false`。
- 新增“图片独立倍率”输入框，仅在“生图倍率是否独立”为 `true` 时参与图片扣费。

## 反思后的兼容性原则

为了最大程度不修改现有已经设置好的分组行为，实现必须遵守以下原则：

1. 迁移不改写现有 `image_price_1k/2k/4k`。
2. 迁移把现有分组统一设置为 `image_rate_independent=false`，继续共享当前有效分组倍率。
3. 迁移把现有分组的 `image_rate_multiplier` 设置为 `1`，但该值在共享模式下不参与扣费。
4. 管理员更新分组时，如果请求没有传 `allow_image_generation`、`image_rate_independent`、`image_rate_multiplier`，服务端不得覆盖旧值。
5. 前端编辑旧分组时必须回显服务端保存值，不能用表单默认值把旧分组误改成关闭生图或独立倍率。
6. 新建分组可以默认 `allow_image_generation=false`，因为这是新配置；现有分组必须按迁移策略保持现状。

## 最终方案

结论：采用“分组生图能力开关 + 图片倍率模式开关 + 可选图片独立倍率 + 可选图片分组”的组合方案。

该方案不是只创建图片分组。普通编码分组可以关闭生图，也可以开启生图；开启后默认继续共享现有分组倍率，保持当前代码的扣费行为。如果该普通编码分组需要图片最终价格直观可配，则打开“生图倍率是否独立”，再配置图片独立倍率和现有图片价格。

## 必须实现的代码行为

1. 分组新增 `allow_image_generation`：
   - 新建分组默认 `false`。
   - 现有 `openai`、`gemini`、`antigravity` 分组迁移为 `true` 以保持兼容。
   - 现有 `anthropic` 分组迁移为 `false`。

2. 分组新增 `image_rate_independent`：
   - 默认 `false`。
   - `false` 表示图片计费共享当前有效分组倍率。
   - `true` 表示图片计费使用图片独立倍率。

3. 分组新增 `image_rate_multiplier`：
   - 默认 `1`。
   - 仅当 `image_rate_independent=true` 时参与图片扣费。
   - 用于图片单独打折或图片最终价直观配置。

4. 保留现有 `image_price_1k/2k/4k`：
   - 不改名。
   - 不强制迁移成新语义。
   - 迁移不改写历史图片价格。

5. 图片计费公式：

```text
unit_price = 渠道 image 价格 或 分组 image_price_* 或 默认图片价格

如果 image_rate_independent=false:
  image_multiplier = 当前有效分组倍率

如果 image_rate_independent=true:
  image_multiplier = image_rate_multiplier

total_cost = unit_price * image_count
actual_cost = total_cost * image_multiplier
```

6. “当前有效分组倍率”必须沿用当前代码实际行为：
   - 默认配置倍率。
   - 分组 `rate_multiplier`。
   - 用户专属分组倍率覆盖。
   - 也就是说，`image_rate_independent=false` 时完全保留当前图片扣费行为。

7. 示例：
   - 当前普通编码分组 `rate_multiplier=0.15`。
   - 希望代码按 `0.15` 倍率。
   - 希望 1K 图片最终扣 `0.2/张`。
   - 设置：

```text
allow_image_generation = true
image_rate_independent = true
image_rate_multiplier = 1
image_price_1k = 0.2
```

8. 如果不打开独立倍率：

```text
allow_image_generation = true
image_rate_independent = false
rate_multiplier = 0.15
image_price_1k = 0.2
最终 1K 图片扣费 = 0.2 * 0.15 = 0.03
```

这就是当前行为的兼容模式。

9. 生图意图统一识别：
   - `/v1/images/generations`
   - `/v1/images/edits`
   - `/images/generations`
   - `/images/edits`
   - `/v1/responses` 中 `model` 以 `gpt-image-` 开头
   - `/v1/responses` 中 `tools[].type == "image_generation"`
   - `/v1/responses` 中 `tool_choice` 指向 `image_generation`

10. 分组关闭生图时：
    - 显式生图请求返回 HTTP 403，错误类型为 `permission_error`。
    - 不选择上游账号。
    - 不写 usage log。
    - Codex CLI 不自动注入 `image_generation` tool，也不追加图片桥接指令。
    - 普通 `gpt-5.4` / `gpt-5.5` 文本请求继续正常转发。

11. 分组开启生图时：
    - `/v1/images/*` 生图允许走现有 OpenAI Images 路径。
    - `gpt-5.4` / `gpt-5.5 + image_generation` 允许走 OpenAI Responses tool 路径。
    - `gpt-image-*` 发到 `/v1/responses` 时允许按现有方向改写为 Responses 文本模型 + `image_generation` tool。
    - 所有成功产图路径必须写 `ImageCount` 并按图片数量计费。

12. Responses 图片数量归因：
    - 非流式 JSON 解析 `output[]` 中的 `image_generation_call.result`。
    - 流式 SSE 解析 `response.output_item.done`。
    - 流式完成事件解析 `response.completed.response.output[]`。
    - `partial_image` 不计费。
    - 上游 usage 为 0 但有最终图片结果时，仍写 usage log 并按图片计费。

13. Images API 图片数量归因：
    - 非流式继续统计顶层 `data[]`。
    - 流式除顶层 `data[]` 外，还要支持 `image_generation.completed`、`response.output_item.done`、`response.completed`。

14. 渠道图片计费：
    - `billing_mode=image` 的 `RequestCount` 必须使用 `ImageCount`。
    - OpenAI 路径不能再固定传 `RequestCount: 1`。
    - 非 OpenAI Gateway 图片路径不能再固定传 `RequestCount: 1`。
    - 账号统计定价的 image 模式也必须使用实际图片数量。
    - 渠道图片价格同样使用上面的图片倍率模式：默认共享有效分组倍率，独立模式使用 `image_rate_multiplier`。

15. 尺寸档位：
    - `1024x1024 => 1K`
    - `1536x1024`、`1024x1536`、`1792x1024`、`1024x1792`、空值、`auto => 2K`
    - 当前 OpenAI 路径没有可证明的 `4K` 映射，未知显式 OpenAI 尺寸必须返回 HTTP 400，不能静默按 `2K` 收费，也不能臆造 `4K`。

## 反思后的潜在漏洞与处理

| 风险 | 当前判断 | 处理 |
| --- | --- | --- |
| 迁移误改历史图片价格 | 会直接改变已配置分组最终扣费 | 明确禁止迁移改写 `image_price_1k/2k/4k` |
| 编辑旧分组时表单默认值覆盖新字段 | 会把已有分组行为从共享倍率误改成独立倍率或关闭生图 | 服务端 omitted 字段不更新，前端必须回显保存值 |
| 只在请求改写前判断生图意图 | Codex 自动注入或 `gpt-image-*` 改写后可能漏检 | 改写前和改写后都执行生图意图判断 |
| `tool_choice="required"` 被误判为生图 | 会错误阻止普通工具调用 | 只有明确指向 `image_generation` 才按 tool_choice 命中；`tools[]` 中有 image tool 仍命中 |
| `gpt-5.4` / `gpt-5.5` 非 image 主模型调用图片工具 | 当前会退化成 token 或漏计图片 | 按最终 `image_generation_call.result` 数量写 `ImageCount` 并走 image 计费 |
| 流式图片工具调用 | 当前通用 Responses 只解析 usage，不稳定计图片数量 | 解析 `response.output_item.done` 和 `response.completed`，只统计最终图片 |
| `ImageCount > 0` 但上游 usage 为 0 | 当前会因为 `ImageCount==0` 才跳过；修复后不能漏写 | 只要 `ImageCount > 0` 就写 usage log 并扣图片费用 |
| 渠道 image 模式多图请求 | 当前固定 `RequestCount=1` 会少扣 | 改为 `RequestCount=ImageCount` |
| 未知 OpenAI 图片尺寸 | 当前静默落到 `2K`，可能少扣 | 未知显式尺寸返回 400，不臆造 `4K` |
| Responses 图片工具同时输出大量文本 | 默认 image-only 计费会让文本 token 不额外收费 | 为兼容本次不引入混合计费；完整记录 token，后续可新增 `image_plus_token` 模式 |

## 更友好的实现建议

1. 后台分组表增加生图状态标识：
   - 未开启生图。
   - 已开启，共享分组倍率。
   - 已开启，独立图片倍率。

2. 分组编辑页增加图片最终价预览：
   - 共享模式预览：`图片价格 * 当前分组倍率`。
   - 独立模式预览：`图片价格 * 图片独立倍率`。
   - 如果存在用户专属倍率，需要提示共享模式下不同用户最终图片价可能不同。

3. 独立倍率输入框只在 `image_rate_independent=true` 时启用：
   - 避免管理员误以为默认会使用该字段。
   - 当输入为 `0` 时显示“图片免费”的明显提示。

4. 上线后提供一次性巡检 SQL 或后台筛选：
   - 找出 `allow_image_generation=true` 的普通编码分组。
   - 找出 `image_rate_independent=false` 且 `rate_multiplier` 很低的分组。
   - 找出设置了图片价格但没有独立倍率的分组。

5. usage log 展示需要区分倍率来源：
   - `shared_group_rate`：共享当前有效分组倍率。
   - `independent_image_rate`：图片独立倍率。
   - 当前数据库可继续写 `usage_logs.rate_multiplier`，但前端文案要显示“本次图片倍率来源”。

## 场景结论

| 场景 | 结论 |
| --- | --- |
| 普通编码分组关闭生图，用户调用 `gpt-5.4` 文本 | 允许，按普通 token 计费 |
| 普通编码分组关闭生图，用户显式带 `image_generation` tool | 拒绝，HTTP 403 |
| 普通编码分组关闭生图，Codex CLI 普通请求 | 不注入图片工具，按文本请求继续 |
| 普通编码分组开启生图，`image_rate_independent=false` | 图片继续按 `图片价格 * 当前有效分组倍率` 扣费 |
| 普通编码分组开启生图，`image_rate_independent=true` | 图片按 `图片价格 * image_rate_multiplier` 扣费 |
| 普通编码分组 `rate_multiplier=0.15`，图片价 `0.2`，独立=false | 1 张 1K 图扣 `0.03`，兼容当前行为 |
| 普通编码分组 `rate_multiplier=0.15`，图片价 `0.2`，独立=true，图片倍率=1 | 1 张 1K 图扣 `0.2` |
| 图片分组开启生图 | 作为推荐运营承载，按同一套生图开关和倍率模式执行 |
| `/v1/responses` 流式生图且上游 usage 为 0 | 只要有最终图片结果，就写 usage log 并按图片计费 |
| `/v1/images/*` 多图请求 | 按实际图片数量计费 |
| 渠道 `billing_mode=image` 多图请求 | 按 `ImageCount` 计数，不再固定 1 |
| 非 image 主模型调用 `image_generation` tool | 按图片结果数量走生图计费 |
| 未知 OpenAI 图片尺寸 | 400 拒绝，不静默落到 `2K` |

## 不在本次变更内解决

- 不实现图片预扣费或资金冻结；继续沿用当前成功后扣费模型。
- 不实现用户级图片独立倍率覆盖；用户专属普通倍率只在共享倍率模式下继续影响图片。
- 不把图片分组做成强制唯一入口；因为当前代码和业务都需要支持普通编码分组按需开启生图。

## 后续改进项：Responses 混合文本 + 图片组合计费

### 触发背景

当前实现已经需要把通用 `/v1/responses`、OpenAI passthrough、OpenAI WS 中的 `image_generation` 工具调用识别为生图请求，避免 `gpt-5.4` / `gpt-5.5` 这类非 image 主模型通过图片工具生图时按普通 token 漏计图片费用。

但是当前代码路径在成本计算上是互斥分支：`backend/internal/service/openai_gateway_service.go` 的 `calculateOpenAIRecordUsageCost` 中，只要 `result.ImageCount > 0` 就直接调用 `calculateOpenAIImageCost` 返回图片费用；后续不会再叠加普通 token 费用。`backend/internal/service/billing_service.go` 的 `CalculateImageCost` 也只按 `image_unit_price * image_count * rate_multiplier` 计算图片成本，不包含文本 token 成本。

因此，当同一次 Responses 请求既输出文本又生成图片时，当前行为是：

```text
一次客户端请求
→ 一次上游转发
→ 一个 ForwardResult
→ 一次 RecordUsage
→ 一条 UsageLog
→ 一次账务扣费
→ 只按图片费用扣费，文本 token 只记录但不额外扣费
```

### 准确结论

该场景不应该拆成两次 HTTP 请求，也不应该拆成两条 usage log 或两次独立账务记录。

当前代码和数据库把一次上游请求视为一个幂等账务单元：

- `backend/internal/handler/openai_gateway_handler.go` 对一次客户端请求只执行一次 `Forward`，成功后只调用一次 `RecordUsage`。
- `backend/internal/service/openai_gateway_service.go` 的 `RecordUsage` 只构造一条 `UsageLog`，并只调用一次 `applyUsageBilling`。
- `backend/internal/repository/usage_log_repo.go` 写 usage log 时使用 `(request_id, api_key_id)` 去重。
- `backend/migrations/071_add_usage_billing_dedup.sql` 为 `usage_billing_dedup` 建立 `(request_id, api_key_id)` 唯一索引。
- `backend/internal/repository/usage_billing_repo.go` 也按 `(request_id, api_key_id)` 做账务幂等。

如果为了混合计费强行生成两条记录，需要派生 `request_id`，例如 `req_x#text` 和 `req_x#image`。这会破坏当前请求追踪、账务幂等、对账、退款、统计聚合的一致口径，不符合当前代码结构。

### 改进目标

新增“混合文本 + 图片组合计费”能力，保持一条请求链路和一条 usage log，只在同一次账务扣费中合并成本。

目标语义：

```text
一次客户端请求
→ 一次上游转发
→ 一个 ForwardResult
→ 一次 RecordUsage
→ 一条 UsageLog
→ 一次账务扣费
→ 扣费 = 文本 token 费用 + 图片费用
```

建议计费模式命名为 `image_plus_token` 或 `mixed_image_text`，用于和现有 `image` 模式区分。

### 建议触发条件

该能力只用于通用 Responses 类入口，不直接改变专用 `/v1/images/generations` 和 `/v1/images/edits` 的语义。

建议同时满足以下条件时进入组合计费：

1. `result.ImageCount > 0`。
2. 本次响应存在非图片 token 成本：
   - `InputTokens > 0`
   - 或 `CacheCreationInputTokens > 0`
   - 或 `CacheReadInputTokens > 0`
   - 或 `OutputTokens - ImageOutputTokens > 0`
3. 入站入口是 `/v1/responses`、passthrough Responses 或 OpenAI WS 的单次 `response.create` turn。
4. 分组、渠道或全局配置显式开启组合计费，不能默认改变现有 `billing_mode=image` 的历史语义。

### 计费公式

```text
text_output_tokens = max(output_tokens - image_output_tokens, 0)

text_total_cost = 按现有普通 token 计费路径计算：
  input_tokens
  text_output_tokens
  cache_creation_tokens
  cache_read_tokens

image_total_cost = 按现有图片计费路径计算：
  image_unit_price * image_count

mixed_total_cost = text_total_cost + image_total_cost
mixed_actual_cost = text_actual_cost + image_actual_cost
```

倍率规则必须延续本提案的图片倍率设计：

- 文本 token 费用继续使用现有分组倍率和用户专属倍率逻辑。
- 图片费用继续遵守“图片倍率是否独立”开关。
- `image_rate_independent=false` 时，图片费用共享当前有效分组倍率，保持历史兼容。
- `image_rate_independent=true` 时，图片费用使用图片独立倍率，不再用普通编码倍率。

### 兼容性策略

默认不改变当前 `ImageCount > 0 => image-only` 的扣费行为，避免已有分组和渠道的图片单价从“图片全包价”被突然改成“图片价 + token 价”。

推荐新增显式配置后再启用：

1. 全局开关：`enable_image_plus_token_billing`，默认关闭。
2. 或分组级开关：`enable_image_plus_token_billing`，默认关闭。
3. 或渠道级计费模式：新增 `billing_mode=image_plus_token`。
4. 管理后台必须明确展示：
   - 纯图片：按图片价格计费。
   - 混合文本 + 图片：按文本 token 费用 + 图片费用计费。
5. usage log 的 `billing_mode` 写为 `image_plus_token` 或 `mixed_image_text`，同时保留 token、image_count、image_size、rate_multiplier、cost_breakdown 等现有字段，便于统计和回溯。

### 实施位置建议

后端建议改动点：

1. `backend/internal/service/openai_gateway_service.go`
   - 在 `calculateOpenAIRecordUsageCost` 增加混合计费分支。
   - 保留当前 `ImageCount > 0` 的 image-only 分支作为默认兼容路径。
   - 组合计费分支中先计算文本成本，再计算图片成本，最后合并 `CostBreakdown`。
2. `backend/internal/service/billing_service.go`
   - 增加合并 `CostBreakdown` 的 helper。
   - 文本成本复用现有 `CalculateCostUnified` / `CalculateCostWithServiceTier` 路径。
   - 图片成本复用现有 `CalculateImageCost` 路径。
3. `backend/internal/service/channel.go`
   - 如果采用渠道级模式，新增 `BillingModeImagePlusToken`。
4. `frontend/src/views/admin/GroupsView.vue`
   - 如果暴露分组开关，新增组合计费说明和最终价预览。
5. 测试必须覆盖：
   - 非流式 Responses：文本输出 + `image_generation_call.result`。
   - 流式 Responses：文本 delta + 图片完成事件。
   - passthrough Responses：文本 + 图片工具结果。
   - OpenAI WS：单个 `response.create` turn 内文本 + 图片。
   - 专用 `/v1/images/*`：继续只按图片计费，不叠加 token。

### 不采用的方案

| 方案 | 结论 | 原因 |
| --- | --- | --- |
| 拆成两次上游请求 | 不采用 | 当前实际只有一次客户端请求和一次上游转发，强行拆分会改变业务行为 |
| 拆成两条 usage log | 不采用 | `(request_id, api_key_id)` 唯一约束会冲突，派生 request_id 会破坏对账口径 |
| 在现有 `image` 模式中默认叠加 token | 不采用 | 会改变历史图片价格语义，已有分组可能被无感涨价 |
| 忽略文本 token | 仅作为当前兼容行为保留 | 存在大文本夹带生图时文本 token 不扣费的收入风险 |

### 场景结论

| 场景 | 当前行为 | 改进后行为 |
| --- | --- | --- |
| `/v1/responses` 只输出文本 | 按普通 token 计费 | 不变 |
| `/v1/responses` 只生成图片 | 按图片计费 | 不变 |
| `/v1/responses` 同时输出文本和图片，未开启组合计费 | 按图片计费，文本 token 只记录 | 不变，保证兼容 |
| `/v1/responses` 同时输出文本和图片，开启组合计费 | 当前不支持 | 一条 usage log 内扣 `文本 token 费用 + 图片费用` |
| `/v1/images/*` 专用生图接口 | 按图片计费 | 不叠加 token，保持图片接口语义 |
| OpenAI WS 单个 `response.create` 同时文本+图片 | 当前每个 turn 单独记录，图片分支不叠加文本 | 每个 turn 仍只写一条记录，开启组合计费后在该条记录内合并成本 |
