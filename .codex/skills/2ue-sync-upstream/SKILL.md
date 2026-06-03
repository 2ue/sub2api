---
name: 2ue-sync-upstream
description: Use when maintaining the 2ue/sub2api fork by updating origin/main from upstream/main, then merging the latest main into 2ue-main. Trigger on requests about 同步上游, 更新 main 从 fork 源仓库/上游仓库, 合并 main 到 2ue-main, sync upstream into fork main, or merge upstream main into 2ue-main. Includes branch invariants, safe git commands, conflict handling, validation, and reporting.
---

# 2ue Upstream Sync

This skill maintains the `2ue/sub2api` fork workflow:

```text
upstream/main (Wei-Shaw/sub2api)
  -> origin/main (2ue/sub2api mirror)
  -> origin/2ue-main (2ue integration and release branch)
```

## Invariants

- `main` mirrors `upstream/main`.
- Do not create 2ue commits on `main`.
- Do not merge `2ue-main` into `main`.
- Do not force push `main`, `2ue-main`, or tags.
- Do not run `git reset --hard` unless the user explicitly asks for that exact destructive operation.
- Do not push upstream release tags to `origin`.
- `2ue-main` may diverge from `main`; that is expected.

## Preflight

Run and report these before changing branches:

```bash
git status --short --branch
git branch --show-current
git remote -v
```

Expected remotes for this repository:

```text
origin   git@github.com:2ue/sub2api.git
upstream git@github.com:Wei-Shaw/sub2api.git
```

If tracked files are dirty, stop unless the user explicitly asked to include those changes. Ignored local files under `.codex/` or `docs/` do not block branch switching, but mention them if they are relevant to the task.

Fetch both remotes:

```bash
git fetch upstream --prune
git fetch origin --prune
```

## Step 1: Update `origin/main` From Upstream

Use fast-forward only:

```bash
git switch main
git pull --ff-only origin main
git merge --ff-only upstream/main
git push origin main
```

If `git pull --ff-only` or `git merge --ff-only upstream/main` fails, stop and report that `main` has diverged. Do not reset or force push.

After updating, capture the new upstream commit:

```bash
git rev-parse --short upstream/main
git rev-parse --short origin/main
```

## Step 2: Merge Latest Main Into `2ue-main`

Switch back and update the integration branch:

```bash
git switch 2ue-main
git pull --ff-only origin 2ue-main
```

If `2ue-main` has no fork-only commits and can fast-forward, this is acceptable:

```bash
git merge --ff-only origin/main
```

When `2ue-main` already has fork-specific commits, use an explicit merge commit:

```bash
git merge --no-ff origin/main -m "merge: upstream main into 2ue-main"
```

If conflicts happen:

```bash
git status --short
git diff --name-only --diff-filter=U
```

Then inspect each conflicted file semantically. Preserve upstream fixes and 2ue-specific changes when both are still valid. Do not use blanket `--ours` or `--theirs`. If the correct resolution is unclear, stop and report the files and competing meanings.

## Post-Merge Analysis

After a clean merge, analyze and report:

```bash
git status --short --branch
git log --oneline --decorate -n 12
git diff --stat ORIG_HEAD..HEAD
git merge-base --is-ancestor origin/main HEAD
```

Interpretation:

- `git merge-base --is-ancestor origin/main HEAD` must exit `0`; otherwise `2ue-main` does not contain latest `main`.
- `2ue-main` being ahead of `origin/2ue-main` is expected until pushed.
- The diff from `ORIG_HEAD..HEAD` is the upstream change set brought into `2ue-main`.

## Validation

Choose validation by touched areas:

- Backend changes: `cd backend && make test-unit`
- Backend integration-sensitive changes: `cd backend && make test-integration`
- Frontend changes: `make test-frontend`
- Release, Docker, or workflow changes: inspect `.github/workflows/*`, `.goreleaser*.yaml`, Dockerfiles, and deployment docs; run relevant build or dry-run commands when practical.

If tests are too expensive or fail due to environment, report exactly what was skipped or failed.

## Push Policy

`origin/main` should be pushed as part of Step 1 because its job is to mirror upstream.

For `2ue-main`:

- Push if the user asked to sync/update the current repository remotely, asked to push, or the task clearly requires remote update.
- Otherwise leave the merge local and say it is ahead of `origin/2ue-main`.

Use:

```bash
git push origin 2ue-main
```

Never push with `--force`.

## Final Report Template

Use a concise Chinese report:

```text
已完成：
- origin/main 已 fast-forward 到 upstream/main: <sha>
- 2ue-main 已合入 origin/main，merge commit: <sha>
- 冲突：无 / 有，列出文件
- 验证：运行了 <commands> / 未运行，原因
- 推送：origin/main 已推送；2ue-main 已推送 / 未推送

注意：
- 当前分支状态：<git status summary>
- 后续建议：<only if useful>
```

