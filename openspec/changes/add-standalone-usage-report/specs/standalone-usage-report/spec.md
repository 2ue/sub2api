## ADDED Requirements

### Requirement: Standalone report service
The system SHALL provide a standalone usage report service with its own binary, API routes, and page.
The report service SHALL not require changes to the current application's navigation or user-facing usage pages.

#### Scenario: report service runs independently
- **WHEN** the report service is deployed and started
- **THEN** it SHALL expose its own page and API without depending on the current system's usage page routes

### Requirement: Reuse existing login state
The report service SHALL accept the existing bearer access token from the browser and SHALL authenticate the request with the same JWT secret and user status checks used by the current system.
The report page SHALL not require a second login when it is served on the same origin as the current application.

#### Scenario: bootstrap with valid token
- **WHEN** the browser sends a bootstrap request with a valid bearer token
- **THEN** the report service SHALL return the current user's identity and query permissions

#### Scenario: missing token is rejected
- **WHEN** the browser sends a bootstrap request without a bearer token
- **THEN** the report service SHALL reject the request with an authentication error

### Requirement: Self query and privileged cross-user query
The report service SHALL allow any authenticated user to query their own usage.
The report service SHALL allow querying another user only for authorized callers.
The report service SHALL enforce the authorization decision on the server and SHALL not rely on browser-held privileged credentials.

#### Scenario: self query is allowed
- **WHEN** an authenticated user submits a summary request without a target user id
- **THEN** the report service SHALL query that user's own usage

#### Scenario: privileged cross-user query is allowed
- **WHEN** an authorized caller submits a summary request for a different user id
- **THEN** the report service SHALL query the requested user's usage

#### Scenario: unauthorized cross-user query is rejected
- **WHEN** a non-authorized caller submits a summary request for a different user id
- **THEN** the report service SHALL reject the request with a forbidden error

### Requirement: Explicit time range and timezone-aware parsing
The report service SHALL require a user-selected time range for usage queries.
The browser MAY prefill the range to today, but the report page SHALL NOT auto-run a query on page load.
The report service SHALL interpret the selected dates using the timezone provided by the browser request and SHALL default to the service timezone only when the timezone is missing or invalid.

#### Scenario: no auto query on load
- **WHEN** the report page finishes bootstrap loading
- **THEN** it SHALL render controls without issuing a usage summary query

#### Scenario: timezone-aware date boundaries
- **WHEN** a query is submitted with a date range and a valid timezone
- **THEN** the report service SHALL use that timezone to compute the query boundaries

#### Scenario: empty range is rejected
- **WHEN** a summary request omits the start date or end date
- **THEN** the report service SHALL reject the request as invalid

### Requirement: Summary response includes totals, group breakdown, and platform breakdown
The report service SHALL return a single response that contains overall totals, a group breakdown, and a platform breakdown for the selected user and time range.
The report service SHALL omit rows whose `actual_cost` is not greater than zero.
The platform dimension SHALL use the effective platform derived from the group platform when present and the account platform otherwise.

#### Scenario: group and platform breakdown are returned together
- **WHEN** a summary query completes successfully
- **THEN** the response SHALL contain total usage, group rows, and platform rows in one payload

#### Scenario: zero-cost placeholder rows are hidden
- **WHEN** usage logs contain rows with `actual_cost <= 0`
- **THEN** those rows SHALL NOT appear in the returned breakdowns

#### Scenario: effective platform is used
- **WHEN** a usage row belongs to a group with a platform value
- **THEN** the platform breakdown SHALL use that group platform value

### Requirement: Privileged user search and bootstrap metadata
The report service SHALL expose bootstrap metadata for the current user and SHALL expose a user search endpoint for authorized callers.
The search response SHALL return only minimal user fields needed for selection.

#### Scenario: bootstrap identifies current user
- **WHEN** the browser loads the bootstrap endpoint
- **THEN** the response SHALL include the current user's id, email, role, and whether cross-user search is available

#### Scenario: user search returns minimal records
- **WHEN** an authorized caller searches for users by keyword
- **THEN** the response SHALL return a capped list of minimal user records suitable for selection

### Requirement: Browser-side credential isolation
The report service SHALL keep privileged credentials on the server side only.
The report page, bootstrap response, and summary response SHALL NOT expose admin keys, service tokens, bearer refresh tokens, or other reusable secrets.

#### Scenario: no privileged secret in response
- **WHEN** the browser requests bootstrap or summary data
- **THEN** the response SHALL contain no admin credential or reusable server secret

