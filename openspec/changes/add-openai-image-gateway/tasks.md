## 1. Routing And Request Foundations

- [x] 1.1 Add `POST /v1/images/generations` and `POST /v1/images/edits` to `backend/internal/server/routes/gateway.go` and dispatch them only for OpenAI-platform groups.
- [x] 1.2 Extend `backend/internal/handler/endpoint.go` with canonical image endpoint constants and OpenAI upstream-endpoint derivation for `/v1/images/generations` and `/v1/images/edits`.
- [x] 1.3 Add OpenAI image request/response DTOs plus validation helpers for JSON generations and multipart edits in a dedicated handler/service file pair.
- [x] 1.4 Transplant the portable multipart field-collection and validation patterns from the sibling repos into `sub2api`-style helpers without importing donor task/proxy behavior.

## 2. OpenAI Image Forwarding

- [x] 2.1 Create a dedicated OpenAI image handler flow on `OpenAIGatewayHandler` that reuses auth context loading, billing eligibility checks, user/account concurrency, scheduler selection, failover, and async usage submission.
- [x] 2.2 Implement dedicated OpenAI image upstream URL builders for `/v1/images/generations` and `/v1/images/edits` instead of reusing the current `/v1/responses` request builders.
- [x] 2.3 Implement JSON upstream forwarding for generations and multipart upstream forwarding for edits, reusing existing OpenAI auth/header/base-URL validation helpers where possible.
- [x] 2.4 Normalize upstream image success and error payloads into OpenAI-compatible responses, including support for `n` multi-image outputs and optional `url` or `b64_json` response items.

## 3. Billing And Usage

- [x] 3.1 Extend the OpenAI forward-result model with image metadata such as `ImageCount` and normalized `ImageSize`.
- [x] 3.2 Update `OpenAIGatewayService.RecordUsage` so successful image requests are recorded even when token usage is zero.
- [x] 3.3 Add image-aware cost calculation in the OpenAI usage path using normalized output count, normalized size tier, unified pricing when available, and existing image-price fallbacks otherwise.
- [x] 3.4 Persist image-specific usage metadata including `billing_mode=image`, `image_count`, `image_size`, inbound endpoint, upstream endpoint, and mapped/upstream model details.
- [x] 3.5 Verify that the image path continues to use the existing usage-billing idempotency and async worker flow.

## 4. Model Discovery And Operator Configuration

- [x] 4.1 Update `GET /v1/models` for OpenAI groups so mapping-derived model lists return OpenAI-compatible model objects instead of Claude-shaped objects.
- [x] 4.2 Ensure configured image models from schedulable OpenAI account mappings are surfaced through `/v1/models`.
- [x] 4.3 Decide whether the deployment needs an explicit OpenAI default image model list in addition to mapping-driven discovery, and implement only if required for non-mapped deployments.
- [x] 4.4 Document the operator prerequisites for enabling image routing: OpenAI group platform, image-capable account mappings, and channel/group pricing readiness.

## 5. Verification Coverage

- [x] 5.1 Add handler and service tests for image generations and image edits request validation, including malformed JSON and missing multipart image cases.
- [x] 5.2 Add forwarding tests covering OpenAI image endpoint selection, multipart forwarding, retryable failover, and normalized multi-image responses.
- [x] 5.3 Add usage and billing tests covering zero-token image requests, `n`-aware charging, size-tier normalization, and usage-log field persistence.
- [x] 5.4 Add `/v1/models` tests covering mapping-driven image-model exposure, non-exposure when unconfigured, and OpenAI-shaped model object responses.
