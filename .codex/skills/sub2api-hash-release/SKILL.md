---
name: sub2api-hash-release
description: Use when the user asks to 发版本, 发版, 发布版本, 重新发版, or 打 tag 发版 for this Sub2API repository and the release policy is hash-based: keep `backend/cmd/server/VERSION` unchanged, do not bump the numeric version, derive the base version from that file, create a new annotated tag in the form `v<base-version>-<8-char-git-hash>`, then push the current branch and tag to `origin` so the existing tag-triggered GitHub Actions release workflow runs.
---

# Sub2API Hash Release

Use this skill only for this repository's hash-release flow.

## Rules

- Do not modify `backend/cmd/server/VERSION` for a hash release.
- Do not increment the numeric version for a hash release.
- Always fetch `origin` tags before computing the release tag.
- The release tag format must be `v<base-version>-<8-char-git-hash>`.
- Use an annotated git tag. The tag message becomes the GitHub release body.
- Push both the branch and the new tag to `origin`.
- If the worktree is dirty, commit the pending changes before tagging.
- If the computed tag already exists locally or on `origin`, stop and report it instead of moving or overwriting tags.

## Workflow

1. Inspect git state:
   - `git status --short`
   - `git branch --show-current`
   - `git rev-parse --abbrev-ref --symbolic-full-name @{u}`
2. Commit pending work if needed. Do not tag uncommitted changes.
3. Sync remote refs:
   - `git fetch origin --prune --tags`
4. If the upstream branch is ahead, fast-forward or rebase onto it before tagging.
5. Compute the release tag with:
   - `.codex/skills/sub2api-hash-release/scripts/resolve_hash_release_tag.sh`
6. Verify the computed tag does not already exist:
   - `git rev-parse -q --verify "refs/tags/$TAG"`
   - `git ls-remote --tags --refs origin "$TAG"`
7. Create the annotated tag:
   - `git tag -a "$TAG" -m "$TAG" -m "<release notes>"`
8. Push the branch and tag:
   - `git push origin <branch>`
   - `git push origin "$TAG"`
9. Confirm that `.github/workflows/release.yml` is triggered by the `v*` tag push.

## Helper

- `.codex/skills/sub2api-hash-release/scripts/resolve_hash_release_tag.sh`
  - Reads `backend/cmd/server/VERSION`
  - Normalizes it to `X.Y.Z`
  - Reads `git rev-parse --short=8 HEAD`
  - Prints `BASE_VERSION=...`, `SHORT_HASH=...`, `TAG=...`
