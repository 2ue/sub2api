## Why

`sub2api` currently centers its OpenAI-compatible gateway around `/v1/responses` and `/v1/chat/completions`, which is not enough to support real image-generation workloads. The project now needs first-class image gateway support so third-party OpenAI-compatible image backends can be integrated without breaking existing `sub2api` traits such as API-key auth, account scheduling, channel pricing, and usage accounting.

## What Changes

- Add dedicated OpenAI-compatible image endpoints to the gateway:
  - `POST /v1/images/generations`
  - `POST /v1/images/edits`
- Introduce an independent image request path for OpenAI-platform groups instead of forcing image traffic through the existing `/v1/responses` and `/v1/chat/completions` conversion pipeline.
- Preserve current project behavior for authentication, group/platform routing, account selection, concurrency control, failover, channel restrictions, and usage-record worker submission.
- Add image-specific request/response normalization for:
  - model selection
  - prompt + multipart image edit inputs
  - `n` multi-image generation
  - OpenAI-compatible response payloads
- Add image-specific billing and usage recording so image requests can be charged by request count and size tier while still fitting the existing usage log and billing architecture.
- Extend OpenAI model discovery so configured image models such as `gpt-image-1` and `gpt-image-2` can be surfaced through `GET /v1/models` when the current deployment enables them.
- Define an MVP migration boundary:
  - backend gateway support is in scope
  - frontend image studio, history UI, and async image task center are out of scope for this change

## Capabilities

### New Capabilities
- `openai-image-gateway`: Independent OpenAI-compatible image generation and image editing gateway behavior, including request validation, upstream forwarding, failover, and response normalization.
- `image-billing-usage`: Image-specific usage extraction, `n`-aware request billing, size-tier billing, and usage log persistence aligned with current `sub2api` billing architecture.
- `image-model-discovery`: Exposure rules for image-capable models in `GET /v1/models`, including deployment-controlled visibility for configured image models.

### Modified Capabilities

None.

## Impact

- Affected backend gateway APIs:
  - `/v1/images/generations`
  - `/v1/images/edits`
  - `GET /v1/models`
- Affected backend areas:
  - OpenAI gateway route registration
  - OpenAI handler and service layers
  - upstream OpenAI/base URL request builders
  - billing resolver inputs and usage recording
  - usage log metadata for image requests
- External dependencies:
  - third-party OpenAI-compatible image backends that expose image endpoints
  - existing pricing data and channel pricing configuration
- Explicitly not part of this change:
  - full frontend image workspace
  - long-lived async image task orchestration
  - unrelated text-chat or Anthropic/Gemini behavior changes
