# OpenAI 图片网关接入与上线说明

本文档用于整理本次图片网关相关改动，并给出在“**当前版本不继续改代码**”前提下的上线配置方案。

文档目标有两类：

- 让人能快速看懂：这次到底改了什么
- 让运维和管理员能直接照着配置上线

---

## 1. 最终想达到的业务目标

本次改动围绕下面这组上线目标展开：

- 订阅组不能调用 `gpt-image-*`
- 指定的余额组可以调用 `gpt-image-*`
- 图片能力不能自动下放给所有 OpenAI 分组
- 生图请求可以在配置完成后按张/按图收费
- 模型暴露、权限控制、计费入口三者保持一致

如果不再继续改代码，当前版本已经可以实现上面这组目标，但推荐按本文档的方式配置。

---

## 2. 本次修改整理

### 2.1 新增了哪些能力

当前项目已经支持面向 OpenAI 平台分组的同步图片接口：

- `POST /v1/images/generations`
- `POST /v1/images/edits`
- `GET /v1/models` 在满足条件时暴露图片模型

其中：

- `generations` 支持 JSON 请求
- `edits` 支持 multipart/form-data
- 支持 `n` 返回多张图
- 支持 `url` / `b64_json` 响应项

### 2.2 新增了哪些权限控制

本次核心新增了一个**分组级图片能力开关**：

- `allow_image_generation`

它的作用不是“定价”，而是“准入”。

只有显式开启该开关的 OpenAI 分组，才允许使用图片能力。没有开启的分组，即使底层账号本身支持生图，也不会自动获得图片能力。

### 2.3 权限控制现在覆盖了哪些入口

当前版本下，图片权限不只拦专用图片端点，也会拦所有最终落到 `gpt-image-*` 的 OpenAI 入口：

- `/v1/images/generations`
- `/v1/images/edits`
- `/v1/responses`
- `/v1/chat/completions`
- `/v1/messages`
- Responses WebSocket

并且当前版本已经补上了“映射后绕过”的问题，权限判断会看**最终生效模型**，不只看客户端原始请求里的 `model`。

也就是说，即使请求先写普通模型，再通过：

- 渠道模型映射
- 账号 `model_mapping`
- 分组 `default_mapped_model`

最终变成 `gpt-image-*`，未开启 `allow_image_generation` 的分组仍然会被拦截。

### 2.4 `/v1/models` 的行为

`/v1/models` 不会默认伪造图片模型能力。

只有同时满足下面两个条件时，图片模型才会被对外暴露：

- 当前分组 `allow_image_generation=true`
- 可调度 **API Key** 账号的 `model_mapping` 中显式出现了图片模型

如果分组没有开启图片能力，即使底层账号映射了 `gpt-image-1` / `gpt-image-2`，`/v1/models` 也不会返回这些模型。

如果只有 OAuth/ChatGPT 账号映射了图片模型，但没有可调度的 API Key 图片账号，`/v1/models` 也不会返回这些模型，避免客户端看到能用但实际调度不到。

### 2.5 图片计费链路现在是什么样

对专用图片端点 `/v1/images/*`，当前版本已经接通图片 usage 与计费链路，会记录：

- `billing_mode=image`
- `image_count`
- `image_size`
- `inbound_endpoint`
- `upstream_endpoint`

并且支持通过渠道定价做按张收费。

这里需要特别说明：

- 当前图片计费的真实语义是“按输出图片张数计费”
- 不是“每个 HTTP 请求固定扣 1 次”
- 例如一次请求 `n=4`，最终返回 4 张图，则按 4 张计费

---

## 3. 当前版本的边界

### 3.1 当前最稳的生图入口

如果你的目标是：

- 权限可控
- 计费可控
- 运维可控

那么**当前最推荐的正式入口**是：

- `POST /v1/images/generations`
- `POST /v1/images/edits`

### 3.2 当前已经做好的部分

- 图片专用端点权限控制
- 图片模型在 `responses/chat/messages/ws` 里的最终模型拦截
- `/v1/models` 模型暴露控制
- `/v1/images/*` 的图片 usage 与按张计费

### 3.3 当前不建议作为稳定按张计费入口的部分

`/v1/responses` 和 `/v1/chat/completions` 的图片场景，当前版本已经做了**权限控制**，但不建议作为“稳定按实际出图张数收费”的正式入口。

原因是：

- 这两条通用 OpenAI 转发路径目前没有完整回填 `ImageCount/ImageSize`
- 因此不适合作为“严格按 `n` 和图片尺寸结算”的正式计费入口

结论很简单：

- 要稳定按张收费，请优先使用 `/v1/images/*`
- `responses/chat` 图片场景当前更适合先看作“权限已控”，不适合当正式图片计费入口

---

## 4. 上线时需要理解的三层控制

当前版本里，和图片能力最相关的控制不是一层，而是三层：

### 4.1 分组层：决定“能不能用”

分组层最关键的是：

- `subscription_type`
- `allow_image_generation`

其中：

- `subscription_type` 决定这个分组是订阅组还是余额组
- `allow_image_generation` 决定这个分组有没有图片能力

### 4.2 渠道层：决定“允许哪些模型、如何计费”

渠道层最关键的是：

- `restrict_models`
- `billing_model_source`
- `model_pricing`

其中：

- `restrict_models=true` 表示只允许渠道定价表里出现的模型
- `model_pricing` 里可以给图片模型配置 `billing_mode=image` 或 `per_request`

### 4.3 账号层：决定“实际向上游怎么调”

账号层最关键的是：

- 账号平台必须是 OpenAI
- 当前图片执行只支持 OpenAI **API Key** 账号
- 账号要可调度
- 账号 `model_mapping` 要显式暴露图片模型

---

## 5. 不改代码情况下的推荐上线配置

下面这套是当前版本最推荐的上线方式。

### 5.1 分组规划

建议至少拆成三类分组：

| 分组类型 | subscription_type | allow_image_generation | 是否绑定图片渠道 |
|---|---|---:|---:|
| 订阅组 | `subscription` | `false` | 否 |
| 普通余额组 | `standard` | `false` | 否 |
| 图片余额组 | `standard` | `true` | 是 |

这套配置的效果是：

- 订阅组不能调用 `gpt-image-*`
- 普通余额组也不能调用 `gpt-image-*`
- 只有“图片余额组”可以调用 `gpt-image-*`

如果你现在只想达成：

- 订阅组不能用
- 余额组可以用

那么最小配置就是：

- 订阅组：`allow_image_generation=false`
- 图片余额组：`allow_image_generation=true`

### 5.2 图片账号配置

图片请求当前只支持 OpenAI API Key 类型账号，不支持 OAuth/ChatGPT 反代账号。

并且本次实现已经把 `/v1/images/*` 的正式图片路由收口为 **仅 API 风格上游**：

- `/v1/images/generations`
- `/v1/images/edits`

不会再为这些图片端点回退到 ChatGPT 网页 `backend-api` 生图链路，避免出现网页版协议抖动、模型暴露错误或计费链路不一致的问题。

建议给图片账号配置显式的 `model_mapping`，至少把图片模型暴露出来：

```json
{
  "model_mapping": {
    "gpt-image-1": "gpt-image-1",
    "gpt-image-2": "gpt-image-2"
  }
}
```

如果你的第三方 OpenAI 兼容底座内部模型名不是这个，也可以映射到内部真实模型名，例如：

```json
{
  "model_mapping": {
    "gpt-image-1": "your-upstream-image-model-a",
    "gpt-image-2": "your-upstream-image-model-b"
  }
}
```

账号还需要满足：

- 可调度
- 未被暂停
- 有效的 `base_url`
- 有效的 API Key / token

### 5.3 图片专用渠道配置

如果你的目标是**稳定按张收费**，建议创建一个**图片专用渠道**，只绑定给“图片余额组”。

推荐字段如下：

- `group_ids`: 只绑定图片余额组
- `restrict_models=true`
- `billing_model_source=requested`
- `model_pricing`: 只放允许的图片模型

最简单的渠道配置示例：

```json
{
  "name": "openai-image-billing",
  "group_ids": [123],
  "billing_model_source": "requested",
  "restrict_models": true,
  "model_pricing": [
    {
      "platform": "openai",
      "models": ["gpt-image-1", "gpt-image-2"],
      "billing_mode": "image",
      "per_request_price": 0.1
    }
 ]
}
```

这表示：

- 只要请求的是 `gpt-image-1` 或 `gpt-image-2`
- 且来自绑定到该渠道的图片余额组
- 就按每张 `0.1 USD` 计费

注意：

- 配置字段虽然叫 `per_request_price`
- 但在当前 `billing_mode=image` 的图片模式里，应把它理解为“单张价格”
- 如果一次请求最终返回 3 张图，且这里配置为 `0.1`，那么本次费用就是 `0.3 USD`

如果你希望按尺寸分层收费，可以使用 `intervals`：

```json
{
  "platform": "openai",
  "models": ["gpt-image-1"],
  "billing_mode": "image",
  "intervals": [
    { "tier_label": "1K", "per_request_price": 0.05, "sort_order": 10 },
    { "tier_label": "2K", "per_request_price": 0.10, "sort_order": 20 },
    { "tier_label": "4K", "per_request_price": 0.20, "sort_order": 30 }
  ]
}
```

### 5.4 是否一定要配置渠道

不是。

如果一个余额组已经开启了 `allow_image_generation=true`，即使你不配图片渠道，图片也仍然可以调用，并回退到：

1. 分组图片价格
2. 内置图片回退价格

但如果你的目标是：

- 定价清晰
- 行为稳定
- 只允许指定模型

那么**强烈建议配置图片专用渠道**，不要完全依赖回退价。

### 5.5 订阅组是否还要额外做模型排除

如果你只是想让订阅组不能用图片模型，当前版本下：

- 只要订阅组 `allow_image_generation=false`

就已经足够。

如果你还想做更强的额外限制，可以再给订阅组绑定一个“订阅专用渠道”，并配置：

- `restrict_models=true`
- 不把 `gpt-image-*` 放进 `model_pricing`

但这是**增强项**，不是达成“订阅组不能用图片”的必需项。

### 5.6 API Key 分配建议

如果用户会同时用文本和图片，建议不要共用一个“泛用分组”的 Key。

更推荐：

- 普通文本使用普通余额组或订阅组的 Key
- 图片能力使用绑定到“图片余额组”的 Key

这样权限边界最清晰，也最容易排查问题。

---

## 6. 上线后的预期行为

按照上面的配置上线后，行为应当是：

### 6.1 订阅组

- 调用 `gpt-image-*` 会被拒绝
- 访问 `/v1/images/*` 会被拒绝
- `/v1/models` 不会看到图片模型

### 6.2 普通余额组

- 如果 `allow_image_generation=false`，行为和订阅组一样
- 即使底层账号本身支持生图，也不会自动获得图片能力

### 6.3 图片余额组

- 可以调用 `gpt-image-*`
- 可以使用 `/v1/images/generations`
- 可以使用 `/v1/images/edits`
- 如果配置了图片专用渠道，会按渠道定价按张收费

---

## 7. 当前版本下“按张/按图收费”能做到什么程度

### 7.1 当前已经可以稳定做到的

对 `/v1/images/generations` 和 `/v1/images/edits`：

- 可以按输出张数收费
- 可以按图片模型收费
- 可以按尺寸层级收费
- 可以通过 `n` 计入图片数量

这是当前版本下最稳的正式方案。

### 7.2 当前不建议承诺的

对 `/v1/responses` 和 `/v1/chat/completions` 的图片场景：

- 当前版本已经做了权限控制
- 但不建议承诺“严格按实际出图张数和尺寸结算”

所以在对外使用说明里，建议明确：

- 图片正式能力入口是 `/v1/images/*`
- `responses/chat` 图片场景当前不是推荐的正式计费入口

---

## 8. 一次性上线清单

建议按下面顺序操作：

1. 准备 OpenAI API Key 图片账号，并确认账号可调度
2. 在图片账号上配置 `model_mapping`，显式暴露 `gpt-image-*`
3. 创建或更新订阅组，确保 `allow_image_generation=false`
4. 创建或更新普通余额组，确保 `allow_image_generation=false`
5. 创建“图片余额组”，设置 `subscription_type=standard`、`allow_image_generation=true`
6. 创建“图片专用渠道”，绑定到图片余额组
7. 在该渠道中配置 `gpt-image-*` 的 `billing_mode=image/per_request`
8. 如需更严格限制，开启 `restrict_models=true`
9. 给需要生图的用户分配绑定到图片余额组的 API Key
10. 用图片余额组的 Key 验证 `/v1/models`、`/v1/images/generations`
11. 用订阅组的 Key 验证调用 `gpt-image-*` 被拒绝

---

## 9. 一句话结论

在当前版本且**不继续改代码**的前提下，推荐这样上线：

- 订阅组：不开图片能力
- 普通余额组：不开图片能力
- 单独建立“图片余额组”
- 单独建立“图片专用渠道”
- 对外正式只开放 `/v1/images/*` 作为图片能力入口

这样就可以同时满足：

- 订阅组不能调用 `gpt-image-*`
- 指定余额组可以调用 `gpt-image-*`
- 生图能力按张/按图收费
- 上线后只靠配置即可落地
