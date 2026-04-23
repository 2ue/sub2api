## ADDED Requirements

### Requirement: OpenAI image endpoints use the existing gateway lifecycle
The system SHALL expose `POST /v1/images/generations` and `POST /v1/images/edits` for OpenAI-platform groups and SHALL run them through the same authentication, group routing, billing eligibility, concurrency, account scheduling, and failover lifecycle used by the existing OpenAI gateway handlers.

#### Scenario: OpenAI group request enters the image gateway path
- **WHEN** an authenticated API key bound to an OpenAI-platform group calls `POST /v1/images/generations`
- **THEN** the request is handled by the OpenAI gateway image path instead of the Anthropic or Gemini compatibility paths

#### Scenario: Non-OpenAI group cannot use OpenAI image routes
- **WHEN** an authenticated API key bound to a non-OpenAI group calls `POST /v1/images/generations` or `POST /v1/images/edits`
- **THEN** the system returns an OpenAI-style error indicating that the endpoint is not available for that platform

### Requirement: Image generation requests are validated and normalized before forwarding
The system SHALL validate OpenAI-compatible image generation requests before any upstream call, including `model`, `prompt`, `n`, and image-related option fields, and SHALL normalize accepted requests into an internal image execution model.

#### Scenario: Multi-image generation request is accepted
- **WHEN** a client sends `POST /v1/images/generations` with a valid `model`, a non-empty `prompt`, and `n=3`
- **THEN** the request is accepted for upstream execution with normalized multi-image metadata

#### Scenario: Invalid generation payload is rejected locally
- **WHEN** a client sends `POST /v1/images/generations` without `model` or without a non-empty `prompt`
- **THEN** the system returns `400 invalid_request_error` without selecting an upstream account

### Requirement: Image edit requests accept OpenAI-compatible multipart input
The system SHALL accept `multipart/form-data` image edit requests and SHALL support at least a primary `image` part, a `prompt`, a `model`, optional `n`, and an optional `mask` part for direct upstream forwarding.

#### Scenario: Multipart edit request is normalized
- **WHEN** a client sends `POST /v1/images/edits` with `image`, `prompt`, and `model`
- **THEN** the system normalizes the multipart payload into an internal edit request and forwards it through the image-specific upstream path

#### Scenario: Edit request without an input image is rejected
- **WHEN** a client sends `POST /v1/images/edits` without any `image` part
- **THEN** the system returns `400 invalid_request_error`

### Requirement: Image requests are forwarded independently from the Responses pipeline
The system SHALL forward image requests through dedicated OpenAI image upstream builders and SHALL NOT route them through the existing `/v1/responses` request-conversion pipeline.

#### Scenario: Image generation uses a dedicated upstream endpoint
- **WHEN** an image generation request is forwarded upstream
- **THEN** the upstream target path is `/v1/images/generations` instead of `/v1/responses`

#### Scenario: Image edit uses multipart upstream forwarding
- **WHEN** an image edit request is forwarded upstream
- **THEN** the upstream request preserves multipart form semantics required by `/v1/images/edits`

### Requirement: Successful image responses remain OpenAI-compatible
The system SHALL normalize successful image responses into OpenAI-compatible payloads containing `created` and `data[]`, SHALL preserve multi-image results, and SHALL return either `b64_json` or `url` entries according to the normalized upstream response.

#### Scenario: Multi-image upstream response is preserved
- **WHEN** the upstream image request succeeds with three generated images
- **THEN** the normalized response contains `data` with exactly three image result objects

#### Scenario: Upstream error is surfaced in OpenAI error format
- **WHEN** the upstream image request fails with a non-retryable request error
- **THEN** the client receives an OpenAI-style error payload instead of a raw upstream body dump
