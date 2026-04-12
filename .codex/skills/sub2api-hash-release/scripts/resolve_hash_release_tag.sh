#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
version_file="$repo_root/backend/cmd/server/VERSION"

if [[ ! -f "$version_file" ]]; then
  echo "version file not found: $version_file" >&2
  exit 1
fi

base_version="$(tr -d '[:space:]' < "$version_file")"
base_version="${base_version#v}"

if [[ ! "$base_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "invalid numeric base version in $version_file: $base_version" >&2
  exit 1
fi

short_hash="$(git rev-parse --short=8 HEAD)"
tag="v${base_version}-${short_hash}"

case "${1:-}" in
  --tag-only)
    printf '%s\n' "$tag"
    ;;
  *)
    printf 'BASE_VERSION=%s\n' "$base_version"
    printf 'SHORT_HASH=%s\n' "$short_hash"
    printf 'TAG=%s\n' "$tag"
    ;;
esac
