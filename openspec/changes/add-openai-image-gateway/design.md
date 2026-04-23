## Context

`sub2api` already has most of the infrastructure needed for image gateway support, but it is split across the wrong execution paths for this use case.

- Gateway routing, API-key auth, group-platform dispatch, request body limiting, client request IDs, ops logging, and group assignment already run through the shared gateway stack in `backend/internal/server/routes/gateway.go`.
- OpenAI-compatible request execution already has mature handler flow for auth context loading, billing eligibility checks, user/account concurrency, account scheduling, failover, and async usage submission in `backend/internal/handler/openai_gateway_handler.go`.
- The current OpenAI service layer is centered on `/v1/responses` and chat/responses conversion in `backend/internal/service/openai_gateway_service.go`. Its request builders and URL helpers assume `/responses`, and its `RecordUsage` path skips zero-token requests.
- The generic gateway billing path already supports image billing primitives: `ForwardResult.ImageCount`, `ForwardResult.ImageSize`, `CostInput.RequestCount`, `CostInput.SizeTier`, and `BillingModeImage`.
- `/v1/models` is already driven by schedulable account `model_mapping` aggregation in `backend/internal/service/gateway_service.go`, but the OpenAI fallback list in `backend/internal/pkg/openai/constants.go` does not include image models, and the mapped-model response shape is not OpenAI-specific today.

The sibling repositories provide useful implementation patterns, but not a drop-in architecture:

- `../gpt2api` is useful for request validation, multipart parsing, and OpenAI-compatible image endpoint shape.
- `../chatgpt2api` is useful for simple `n` handling and response normalization.
- Neither project should be transplanted wholesale because their image stacks also include ChatGPT reverse-protocol logic, task/history storage, and image proxy behavior that do not match this change scope or `sub2api`'s current gateway design.

## Goals / Non-Goals

**Goals:**

- Add first-class `POST /v1/images/generations` and `POST /v1/images/edits` support for OpenAI-platform groups.
- Keep image requests inside the existing `sub2api` gateway lifecycle: auth, group routing, account scheduling, concurrency, failover, channel mapping, billing eligibility, and async usage recording.
- Build an image-specific upstream path instead of forcing images through `/v1/responses` or `/v1/chat/completions`.
- Support OpenAI-compatible JSON generations and multipart edits, including `n`, model selection, prompt handling, optional `mask`, and OpenAI-style success/error payloads.
- Make image requests billable with existing channel pricing and group image pricing rules, including `n`-aware per-request charging and normalized image size tiers.
- Expose configured image models such as `gpt-image-1` and `gpt-image-2` through `GET /v1/models` without falsely advertising unsupported defaults.

**Non-Goals:**

- No `/v1/responses` image-generation tool bridge in this change.
- No `/v1/chat/completions` image-scene bridge in this change.
- No async image task center, task polling API, image history UI, or first-party image proxy.
- No transplant of ChatGPT reverse image protocol, sentinel/POW flow, or other donor-specific upstream execution stacks.
- No changes to Anthropic, Gemini, or Antigravity request behavior outside shared endpoint normalization or model-list shaping.

## Decisions

### 1. Add dedicated OpenAI image endpoints beside the existing OpenAI gateway routes

We will add `POST /v1/images/generations` and `POST /v1/images/edits` under the same `/v1` gateway group used by the existing OpenAI-compatible routes. These routes will use the same middleware chain as `/v1/responses` and `/v1/chat/completions`, and they will only dispatch to the new image handler path for OpenAI-platform groups.

The handler implementation will stay on `OpenAIGatewayHandler`, but live in a dedicated file such as `backend/internal/handler/openai_gateway_images.go` so image validation and multipart logic do not inflate the already large responses/chat handlers.

Why this approach:

- It preserves all existing gateway characteristics automatically.
- It avoids creating a parallel auth and scheduling stack.
- It keeps the public API surface explicit and easy to reason about.

Alternatives considered:

- Reusing `/v1/responses` or `/v1/chat/completions` for image work. Rejected because current conversion code is text/tool oriented, does not model image billing metadata, and does not handle multipart edits.
- Creating a completely separate image gateway handler/service pair outside the existing OpenAI gateway. Rejected because it would duplicate auth, concurrency, failover, and usage worker logic.

### 2. Build an image-specific OpenAI upstream path instead of extending the current `/responses` builders

We will add a separate image forwarding path in the OpenAI service layer, implemented in a dedicated companion file such as `backend/internal/service/openai_gateway_images.go`. That path will:

- reuse account selection and common auth/header helpers already used by OpenAI passthrough requests,
- build upstream URLs for `/v1/images/generations` and `/v1/images/edits`,
- support JSON forwarding for generations,
- support multipart forwarding for edits,
- normalize OpenAI-compatible success and error payloads,
- return an image-aware forward result carrying `ImageCount`, `ImageSize`, and endpoint metadata.

The implementation will transplant only the portable donor patterns:

- multipart parsing, field collection, and input validation patterns from `../gpt2api/internal/gateway/images.go`,
- simple `n` handling and response normalization patterns from `../chatgpt2api/services/api.py` and `../chatgpt2api/services/chatgpt_service.py`.

We will not transplant donor upstream execution stacks tied to ChatGPT reverse APIs.

Why this approach:

- Existing OpenAI builders are `/responses`-centric and set JSON-only assumptions.
- Image edits need multipart support, which does not fit the current `buildUpstreamRequest*` flow.
- This path keeps the code compatible with third-party OpenAI-style image backends, which is the primary business goal.

Alternatives considered:

- Extending `buildUpstreamRequest` and `buildOpenAIResponsesURL` with special image branches. Rejected because it would further entangle unrelated request types inside already complex responses logic.
- Transplanting all of `gpt2api` or `chatgpt2api` image execution. Rejected because their upstream protocols, task systems, and proxy behavior do not match `sub2api`'s scope.

### 3. Make OpenAI usage recording image-aware inside `OpenAIGatewayService`

Instead of injecting `GatewayService` into the OpenAI handler just to reuse generic image billing, we will extend `OpenAIForwardResult` and `OpenAIGatewayService.RecordUsage` so the OpenAI path can record image requests correctly.

Concretely, the OpenAI image path will:

- stop skipping usage when token counts are zero if `ImageCount > 0`,
- carry `ImageCount` and normalized `ImageSize` in the OpenAI forward result,
- derive image billing with the same semantics already used by the generic gateway path,
- persist `image_count`, `image_size`, `billing_mode=image`, inbound endpoint, and upstream endpoint in the usage log,
- continue using the existing usage billing repository and async worker pool.

`OpenAIGatewayService.RecordUsage` will mirror the generic gateway image-cost behavior:

- prefer unified channel pricing when available,
- otherwise fall back to group image price tiers or default image price resolution,
- use actual returned image count for billing and persistence,
- use normalized size tiers such as `1K`, `2K`, and `4K`.

Why this approach:

- The OpenAI handler already depends on `OpenAIGatewayService`, not on `GatewayService`.
- The OpenAI service already owns the required repositories, billing service, pricing resolver, and usage billing path.
- Extending one service is less invasive than cross-injecting another platform service only for one request type.

Alternatives considered:

- Calling `GatewayService.RecordUsage` from the OpenAI image path. Rejected because it introduces awkward cross-service wiring and mixes OpenAI-specific routing with generic gateway request models.
- Leaving the current zero-token short-circuit and treating image requests as unbilled. Rejected because it breaks the requirement for per-image charging.

### 4. Normalize image request fields into billing-friendly internal metadata

The image gateway will normalize client request fields into internal execution metadata before forwarding and billing:

- `n` will default to `1` and be bounded by a gateway-side safety limit.
- `size` values such as `1024x1024`, `1024x1536`, `1536x1024`, `2048x2048`, and similar provider-compatible variants will be mapped into billing tiers consumed by the pricing resolver and fallback image pricing.
- `mask` will be accepted as an optional multipart part for edits and forwarded when present.
- Unsupported or malformed image fields will be rejected with OpenAI-style `invalid_request_error` responses before any upstream call.

Billing will use normalized output metadata, not raw client input alone:

- `RequestCount` will be derived from the normalized result image count.
- `SizeTier` will be derived from the normalized image size tier.
- If the pricing resolver lacks a model-specific image price, the existing group-tier or default image fallback path remains valid, so `gpt-image-2` is not blocked on a catalog entry.

Why this approach:

- It keeps cost calculation consistent with current billing abstractions.
- It prevents direct coupling between arbitrary client `size` strings and internal pricing tiers.
- It avoids overcharging when upstream returns fewer images than requested.

Alternatives considered:

- Charging from raw requested `n` and raw requested `size` only. Rejected because partial upstream results and provider-specific size aliases can make that inaccurate.
- Blocking models without exact pricing catalog entries. Rejected because the current billing system already has safe fallbacks.

### 5. Keep model discovery mapping-driven and OpenAI-shaped

`GET /v1/models` will remain primarily driven by schedulable account `model_mapping`, but the OpenAI branch will be tightened in two ways:

- image models will appear when they are present in the aggregated schedulable mappings for the OpenAI group,
- OpenAI model-list responses built from mappings will use OpenAI-compatible model object shape instead of the current Claude-shaped fallback object.

The change will not blindly append image models to every OpenAI deployment. If a deployment wants `gpt-image-1` or `gpt-image-2` to appear, it must make those models available through account mappings or an explicitly maintained OpenAI default list.

Why this approach:

- It matches current operator-controlled model exposure rules.
- It avoids advertising image models on deployments that cannot actually route them.
- It fixes a shape inconsistency for OpenAI groups without broadening the model surface unnecessarily.

Alternatives considered:

- Always adding `gpt-image-1` and `gpt-image-2` to `openai.DefaultModels`. Rejected because it creates false-positive capability discovery on deployments with no image-capable accounts.

## Risks / Trade-offs

- [Provider field variance] Different OpenAI-compatible backends vary in how strictly they support `size`, `quality`, `style`, `mask`, and response-format behavior. → Mitigation: validate the common subset locally, pass through known-safe fields, and return OpenAI-style `invalid_request_error` for unsupported combinations.
- [Large payload pressure] Multipart edits and multi-image `b64_json` responses can significantly increase request and response memory footprint. → Mitigation: reuse existing body-size middleware, enforce a gateway-side max `n`, avoid extra copies where possible, and keep the first release synchronous only.
- [Billing mismatch risk] Some providers may omit usage and return fewer images than requested. → Mitigation: bill from normalized output metadata (`ImageCount`, `ImageSize`) instead of assuming token usage or requested `n`.
- [Discovery misconfiguration] Operators may enable image endpoints but forget to expose image models via `model_mapping`. → Mitigation: document mapping requirements in the rollout steps and keep exposure rules explicit instead of silently broadening defaults.
- [Implementation duplication] Image-aware cost logic will partially mirror generic gateway behavior inside `OpenAIGatewayService`. → Mitigation: keep the logic narrow, reuse existing billing primitives, and extract shared helpers later only if a second image-capable platform needs the same path.

## Migration Plan

1. Add the new backend route registrations, handler methods, endpoint-normalization constants, and OpenAI image forwarding path.
2. Extend OpenAI forward-result and usage-recording models to persist image metadata and charge zero-token image requests correctly.
3. Update `/v1/models` so OpenAI groups return OpenAI-shaped mapped-model responses and include configured image models.
4. Before enabling the feature in production, ensure the target OpenAI group has image-capable account mappings such as `gpt-image-1` and `gpt-image-2`, and ensure channel pricing or group image pricing is configured for the intended billing behavior.
5. Deploy without database migration work. Existing usage-log schema already has the necessary image-related columns.
6. Roll back by removing or reverting the backend change. If operational rollback is needed without code revert, operators can also remove the image model mappings so discovery stops exposing the endpoints' intended models.

## Open Questions

None for the MVP scope defined by this change.
