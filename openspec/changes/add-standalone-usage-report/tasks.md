## 1. Backend service

- [x] 1.1 Create the standalone `backend/cmd/report` entrypoint and minimal wiring for config, database, auth, and report handlers.
- [x] 1.2 Implement server-side JWT authentication, current-user bootstrap, and admin-gated cross-user selection.
- [x] 1.3 Implement the direct SQL summary query for group and platform breakdowns with timezone parsing and `actual_cost > 0` filtering.

## 2. Standalone page

- [x] 2.1 Add the embedded report page with user bootstrap, current-user default state, optional privileged user search, date inputs, query action, and result tables.
- [x] 2.2 Add static file serving and route handling so the report page is reachable from the standalone service root.

## 3. Verification

- [x] 3.1 Add backend tests covering authentication, permission boundaries, query parsing, and summary shaping.
- [x] 3.2 Run build and test verification for the new report service and confirm the current system is unchanged.
