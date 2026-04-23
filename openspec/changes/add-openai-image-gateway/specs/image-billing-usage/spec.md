## ADDED Requirements

### Requirement: Successful image requests are recorded even without token usage
The system SHALL persist usage and billing records for successful OpenAI image requests even when the upstream response contains zero text-token usage.

#### Scenario: Zero-token image generation is still billed and logged
- **WHEN** an image generation request succeeds and the upstream response contains no token usage block
- **THEN** the system still records a usage entry and applies image billing instead of skipping the request

#### Scenario: Image usage log persists image metadata
- **WHEN** a successful image request is recorded
- **THEN** the usage record includes the requested model, upstream model when applicable, image count, normalized image size, inbound endpoint, upstream endpoint, and `billing_mode=image`

### Requirement: Image billing uses normalized result count and size tier
The system SHALL calculate image request cost from normalized image metadata, including output image count and normalized size tier, and SHALL use existing channel pricing or image-price fallback rules already supported by the billing service.

#### Scenario: Multi-image generation charges by returned image count
- **WHEN** a request succeeds with `n=4` and four normalized output images
- **THEN** billing uses an image request count of four instead of one

#### Scenario: Image size is mapped into a billing tier
- **WHEN** a successful image request includes a size that maps to a known tier such as `1K`, `2K`, or `4K`
- **THEN** the billing calculation uses that normalized tier when resolving per-request or image-mode pricing

### Requirement: Image requests continue to use async usage submission and billing idempotency
The system SHALL continue submitting OpenAI image usage through the existing async usage-record worker flow and SHALL preserve the existing duplicate-protection behavior of the usage billing pipeline.

#### Scenario: Image usage uses the async worker path
- **WHEN** an OpenAI image request succeeds
- **THEN** the handler submits usage persistence through the same async worker model used by current gateway requests

#### Scenario: Duplicate image billing is not applied twice
- **WHEN** the same image usage record is replayed through the billing pipeline with the same billing fingerprint inputs
- **THEN** the billing repository does not apply the same charge twice
