# Sub2API Documentation Source of Truth

## Route Sources

Start from `backend/internal/server/router.go`. It wires the route registrars in this order:

- `backend/internal/server/routes/common.go`
- `backend/internal/server/routes/auth.go`
- `backend/internal/server/routes/user.go`
- `backend/internal/server/routes/sora_client.go`
- `backend/internal/server/routes/admin.go`
- `backend/internal/server/routes/gateway.go`
- `backend/internal/setup/handler.go`

Use `scripts/export_routes.py` before editing any route-count or route-table document.

## Auth Sources

These files define the auth/admin-key model used by the docs:

- `backend/internal/server/middleware/admin_auth.go`
- `backend/internal/server/middleware/jwt_auth.go`
- `backend/internal/server/middleware/api_key_auth.go`
- `backend/internal/server/middleware/api_key_auth_google.go`

Rules that matter:

- `/api/v1/admin/**` uses `AdminAuthMiddleware`, so it supports `x-api-key: <admin-api-key>`.
- User JWT routes use `JWTAuthMiddleware`; they do not support admin-key.
- Gateway routes use user API keys, not admin-key.

## Document Responsibilities

- `docs/api_inventory/routes.tsv`
  - Machine-readable route inventory.
  - Update this first from code.
- `docs/api_full_reference_v2/01-common-setup.md` to `07-admin-misc.md`
  - Detailed route tables.
  - These are the detailed route docs the rest of the inventory refers to.
- `docs/api_full_reference_v2/08-admin-key-supported.md`
  - Admin-key support conclusion and navigation.
  - Do not turn this back into a duplicated 265-route listing unless explicitly asked.
- `docs/api_full_reference_v2/endpoints.json`
  - Lightweight manifest only.
  - It should agree with the generated route inventory counts.
- `docs/api_full_reference_v2/UNRESOLVED_HANDLERS.txt`
  - Inline-wrapper note file, not a backlog dump.
- `docs/api_inventory/*.md`
  - Compact summaries derived from the detailed reference set and route inventory.

## Section Mapping

`scripts/export_routes.py` emits one of these sections:

- `common`
  - `docs/api_full_reference_v2/01-common-setup.md`
  - `docs/api_inventory/01-common-setup.md`
- `auth-user`
  - `docs/api_full_reference_v2/02-auth-user.md`
  - `docs/api_inventory/02-auth-user.md`
- `gateway`
  - `docs/api_full_reference_v2/03-gateway.md`
  - `docs/api_inventory/04-gateway.md`
- `admin-core`
  - `docs/api_full_reference_v2/04-admin-core.md`
  - `docs/api_inventory/03-admin.md`
- `admin-accounts`
  - `docs/api_full_reference_v2/05-admin-accounts.md`
  - `docs/api_inventory/03-admin.md`
- `admin-ops`
  - `docs/api_full_reference_v2/06-admin-ops.md`
  - `docs/api_inventory/03-admin.md`
- `admin-misc`
  - `docs/api_full_reference_v2/07-admin-misc.md`
  - `docs/api_inventory/03-admin.md`

## Special-Topic Doc Hotspots

### `docs/ADMIN_PAYMENT_INTEGRATION_API.md`

Verify these files before keeping any claim:

- `backend/internal/server/routes/admin.go`
- `backend/internal/handler/admin/redeem_handler.go`
- `backend/internal/handler/admin/user_handler.go`
- `backend/internal/handler/admin/idempotency_helper.go`
- `backend/internal/handler/idempotency_helper.go`
- `backend/internal/config/config_test.go`
- `frontend/src/utils/embedded-url.ts`
- `frontend/src/views/user/PurchaseSubscriptionView.vue`
- `frontend/src/views/user/CustomPageView.vue`
- `backend/internal/service/setting_service.go`
- `backend/internal/server/router.go`

Specific facts this doc commonly gets wrong:

- Whether `Idempotency-Key` is enforced or only observed depends on deployed config.
- `create-and-redeem` defaulting `type` to `balance` is a compatibility behavior implemented in code.
- Payment/query-forwarding behavior is partly frontend behavior, not only backend API behavior.
- Query forwarding includes `src_host` and `src_url` via `buildEmbeddedUrl()`.
- CSP/embed behavior depends on `GetFrameSrcOrigins()` and the router’s security-header callback.

### `docs/codex-auth-import-cn.md`

Verify these files before keeping any claim:

- `backend/internal/server/routes/admin.go`
- `backend/internal/handler/admin/account_handler.go`
- `backend/internal/handler/admin/openai_oauth_handler.go`
- `backend/internal/service/admin_service.go`
- `backend/internal/pkg/openai/oauth.go`
- `frontend/src/components/account/CreateAccountModal.vue`
- `frontend/src/components/account/EditAccountModal.vue`

Specific facts this doc commonly gets wrong:

- `POST /api/v1/admin/accounts` field names must match `CreateAccountRequest`.
- `group_ids`, `confirm_mixed_channel_risk`, `rate_multiplier`, `load_factor`, and `auto_pause_on_expired` are real request fields and should only be described as far as code supports them.
- OpenAI and Sora OAuth endpoints share handler logic but are exposed through distinct route paths.
- Any import helper script path must be verified in the repo first. If a helper was intentionally deleted, the doc must describe it as removed/legacy or stop recommending it as an active workflow.

## Suggested Commands

Rebuild route inventory and summary artifacts:

```bash
python3 scripts/export_routes.py \
  --tsv-output /tmp/sub2api-routes.tsv \
  --summary-json /tmp/sub2api-endpoints.json \
  --unresolved-output /tmp/sub2api-unresolved.txt
```

Diff the generated inventory against docs:

```bash
diff -u docs/api_inventory/routes.tsv /tmp/sub2api-routes.tsv
```

Re-check payment/idempotency hotspots:

```bash
rg -n "Idempotency-Key|observe_only|create-and-redeem|buildEmbeddedUrl|purchase_subscription_url|custom_menu_items" \
  backend frontend docs
```

Re-check Codex/OpenAI OAuth import hotspots:

```bash
rg -n "CreateAccountRequest|create-from-oauth|generate-auth-url|exchange-code|refresh-token|group_ids|confirm_mixed_channel_risk|chatgpt_account_id" \
  backend frontend docs
find . -path '*import_openai_oauth_accounts.py' -o -path '*openai*oauth*import*'
```
