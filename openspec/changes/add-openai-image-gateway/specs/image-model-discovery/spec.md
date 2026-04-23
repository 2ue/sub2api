## ADDED Requirements

### Requirement: Configured image models are discoverable through GET /v1/models
The system SHALL expose image-capable OpenAI models through `GET /v1/models` when those models are configured on schedulable OpenAI accounts or on an explicitly maintained OpenAI default model list for the current deployment.

#### Scenario: Aggregated mappings expose image models
- **WHEN** schedulable OpenAI accounts in the current group advertise `gpt-image-1` and `gpt-image-2` through `model_mapping`
- **THEN** `GET /v1/models` includes both image models in the returned model list

#### Scenario: Unconfigured image models are not advertised
- **WHEN** no schedulable OpenAI account mapping or explicit default list exposes an image model
- **THEN** `GET /v1/models` does not claim support for that image model

### Requirement: OpenAI groups return OpenAI-shaped model objects for mapped models
The system SHALL return OpenAI-compatible model objects for OpenAI-platform groups even when the model list is built from aggregated account mappings instead of the static default model list.

#### Scenario: Mapping-based image models use OpenAI schema
- **WHEN** an OpenAI-platform group receives `/v1/models` data sourced from aggregated account mappings
- **THEN** each returned item uses OpenAI-compatible fields such as `id`, `object`, `created`, and `owned_by`

#### Scenario: Image model discovery does not change non-OpenAI model schemas
- **WHEN** a non-OpenAI platform group calls `GET /v1/models`
- **THEN** its existing model-list behavior remains unchanged except for shared route normalization needed by the image change
