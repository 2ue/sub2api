---
name: sub2api-doc-sync
description: Safely fetch the latest Sub2API git refs, audit the real API surface and related implementation details, then sync `docs/api_full_reference_v2/`, `docs/api_inventory/`, `docs/ADMIN_PAYMENT_INTEGRATION_API.md`, and `docs/codex-auth-import-cn.md` so they match the code exactly. Use when working on Sub2API API docs, route inventories, admin-key support notes, payment integration docs, or Codex/OpenAI OAuth import docs.
---

# Sub2API Doc Sync

Use this skill when the Sub2API documentation must be brought back into exact alignment with the current implementation.

Treat code as the only source of truth. Existing docs are reference material that may be stale or partially updated.

## Safety Rules

- Never discard or overwrite local changes just to sync docs.
- Start by syncing git refs before analyzing behavior.
- If the current checkout is dirty, prefer a temporary analysis worktree over mutating the active working tree.
- Verify every referenced file, script, config key, field name, and route in the repository before keeping it in docs.
- Do not assume a documented helper script still exists. A referenced script may have been intentionally removed; if so, document it as removed/legacy or drop the claim instead of presenting it as a current tool.

## Workflow

### 1. Sync Git State First

- On a clean checkout, run `scripts/safe_git_sync.sh --ff-only`.
- On a dirty checkout, run `scripts/safe_git_sync.sh --analysis-worktree`.
- Analyze the printed temporary worktree path when the current checkout cannot be fast-forwarded safely.
- After the doc work is complete, remove the temporary worktree with the cleanup command printed by the script.

### 2. Rebuild the Route Inventory from Code

- Run `python3 scripts/export_routes.py --tsv-output /tmp/sub2api-routes.tsv --summary-json /tmp/sub2api-endpoints.json --unresolved-output /tmp/sub2api-unresolved.txt`.
- Diff `/tmp/sub2api-routes.tsv` against `docs/api_inventory/routes.tsv`.
- Use the generated counts to update:
  - `docs/api_inventory/INDEX.md`
  - `docs/api_full_reference_v2/INDEX.md`
  - `docs/api_full_reference_v2/00-文档说明.md`
  - `docs/api_full_reference_v2/endpoints.json`
  - `docs/api_full_reference_v2/UNRESOLVED_HANDLERS.txt`
- The generated route inventory is the fastest way to catch missing endpoints, wrong handler bindings, stale counts, or stale section assignments.

### 3. Audit Behavior Behind the Routes

- Read [references/source-of-truth.md](references/source-of-truth.md) before editing docs.
- For route docs, start from registration files and middleware, then read handlers and related DTO/service logic when the prose claims behavior beyond method/path/auth.
- For special-topic docs, verify both backend behavior and any frontend/helper code that changes observable behavior.

### 4. Update Documentation in Dependency Order

Update these artifacts in this order so summaries are always derived from detailed source material:

1. `docs/api_inventory/routes.tsv`
2. `docs/api_full_reference_v2/01-common-setup.md` through `07-admin-misc.md`
3. `docs/api_full_reference_v2/08-admin-key-supported.md`
4. `docs/api_full_reference_v2/INDEX.md`
5. `docs/api_full_reference_v2/00-文档说明.md`
6. `docs/api_full_reference_v2/endpoints.json`
7. `docs/api_full_reference_v2/UNRESOLVED_HANDLERS.txt`
8. `docs/api_inventory/00-auth-model.md` through `05-admin-key-supported-endpoints.md`
9. `docs/api_inventory/INDEX.md`
10. `docs/ADMIN_PAYMENT_INTEGRATION_API.md`
11. `docs/codex-auth-import-cn.md`

### 5. Project-Specific Rules

- `docs/api_full_reference_v2/` is the detailed API reference set.
- `docs/api_inventory/` is the compact index plus the machine-readable `routes.tsv`.
- All `/api/v1/admin/**` routes support admin-key because they go through `AdminAuthMiddleware`; gateway keys are not admin-keys.
- `inline route func` is a valid handler label for wrapper routes such as `/health`, `/setup/status` in normal mode, `/v1/messages`, and `/v1/messages/count_tokens`.
- `docs/ADMIN_PAYMENT_INTEGRATION_API.md` must verify idempotency semantics, payment-facing admin routes, and iframe/query forwarding behavior from frontend code.
- `docs/codex-auth-import-cn.md` must verify account creation fields, OpenAI/Sora OAuth endpoints, and referenced import tooling. If a tool was intentionally removed, say so explicitly or remove the operational guidance instead of pretending the file still exists.

## Validation

- No route appears in generated inventory without being reflected in the detailed reference docs.
- Counts match across `routes.tsv`, both `INDEX.md` files, `00-文档说明.md`, and `endpoints.json`.
- Admin-key claims only cover `/api/v1/admin/**`.
- Payment doc examples match real route paths, idempotency behavior, and current frontend embed/query forwarding behavior.
- Codex Auth doc fields match `CreateAccountRequest`, OAuth handlers, and real repo files.
- Every code/file path named in docs exists in the repository.

## Resources

- `scripts/safe_git_sync.sh`: fetch refs, fast-forward safely when possible, or create an analysis worktree when the checkout is dirty.
- `scripts/export_routes.py`: export the current route inventory from the Go route-registration files and optionally write summary artifacts used by the docs.
- `references/source-of-truth.md`: project-specific map of route sources, auth sources, document responsibilities, and hotspot files for the two special-topic docs.

Use the scripts first. Use the reference file when you need project-specific guidance on which code paths drive which documents.
