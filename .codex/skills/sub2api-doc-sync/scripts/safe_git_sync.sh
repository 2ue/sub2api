#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  safe_git_sync.sh [--ref <git-ref>] [--ff-only] [--analysis-worktree]

Behavior:
  1. Always runs: git fetch --all --prune --tags
  2. Chooses target ref in this order:
     - explicit --ref
     - current upstream (@{u})
     - origin/main
  3. Optional actions:
     --ff-only           Fast-forward the current checkout to the target ref.
                         Refuses to run when the working tree has local changes.
     --analysis-worktree Create a temporary detached worktree at the target ref.

Examples:
  safe_git_sync.sh --ff-only
  safe_git_sync.sh --analysis-worktree
  safe_git_sync.sh --ref origin/main --analysis-worktree
EOF
}

ref=""
ff_only=false
analysis_worktree=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --ref)
      ref="${2:-}"
      shift 2
      ;;
    --ff-only)
      ff_only=true
      shift
      ;;
    --analysis-worktree)
      analysis_worktree=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

git fetch --all --prune --tags

if [[ -z "$ref" ]]; then
  if upstream_ref="$(git rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>/dev/null)"; then
    ref="$upstream_ref"
  else
    ref="origin/main"
  fi
fi

if ! git rev-parse --verify "$ref^{commit}" >/dev/null 2>&1; then
  echo "Target ref does not resolve to a commit: $ref" >&2
  exit 1
fi

head_commit="$(git rev-parse HEAD)"
target_commit="$(git rev-parse "$ref")"
merge_base="$(git merge-base HEAD "$ref")"

relationship="diverged"
if [[ "$head_commit" == "$target_commit" ]]; then
  relationship="up-to-date"
elif [[ "$merge_base" == "$head_commit" ]]; then
  relationship="behind"
elif [[ "$merge_base" == "$target_commit" ]]; then
  relationship="ahead"
fi

dirty=false
if ! git diff --quiet --ignore-submodules -- || ! git diff --cached --quiet --ignore-submodules --; then
  dirty=true
fi

echo "repo_root=$repo_root"
echo "target_ref=$ref"
echo "head_commit=$(git rev-parse --short "$head_commit")"
echo "target_commit=$(git rev-parse --short "$target_commit")"
echo "relationship=$relationship"
echo "dirty_worktree=$dirty"

if $ff_only; then
  if $dirty; then
    echo "Refusing --ff-only because the working tree has local changes." >&2
    echo "Use --analysis-worktree instead, or clean/stash changes yourself first." >&2
    exit 2
  fi

  if git symbolic-ref --quiet HEAD >/dev/null 2>&1; then
    if upstream_ref="$(git rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>/dev/null)"; then
      if [[ "$ref" == "$upstream_ref" ]]; then
        git pull --ff-only
      else
        git merge --ff-only "$ref"
      fi
    else
      git merge --ff-only "$ref"
    fi
  else
    git merge --ff-only "$ref"
  fi

  echo "updated_head=$(git rev-parse --short HEAD)"
fi

if $analysis_worktree; then
  tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/sub2api-doc-sync.XXXXXX")"
  git worktree add --detach "$tmpdir" "$ref" >/dev/null
  echo "analysis_worktree=$tmpdir"
  echo "cleanup_command=git -C '$repo_root' worktree remove '$tmpdir'"
fi
