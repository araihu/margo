#!/usr/bin/env bash
set -euo pipefail

if [[ $# -gt 1 ]]; then
  echo "usage: prepare-dagger-git.sh [OUTPUT_BUNDLE]" >&2
  exit 2
fi

output_bundle="${1:-.dagger-git.bundle}"
repository_root="$(git rev-parse --show-toplevel)"
case "$output_bundle" in
  /*) ;;
  *) output_bundle="$repository_root/$output_bundle" ;;
esac

temporary_bundle="$(mktemp "${output_bundle}.XXXXXX")"
portable_repository="$(mktemp -d "${TMPDIR:-/tmp}/margo-dagger-git.XXXXXX")"
trap 'rm -f "$temporary_bundle"; rm -rf "$portable_repository"' EXIT

git -C "$repository_root" rev-parse --verify 'refs/remotes/origin/main^{commit}' >/dev/null
git init --quiet --bare "$portable_repository"
git --git-dir="$portable_repository" fetch --quiet --no-write-fetch-head "$repository_root" \
  'HEAD:refs/heads/dagger-head' \
  'refs/remotes/origin/main:refs/heads/main' \
  '+refs/tags/*:refs/tags/*'
git --git-dir="$portable_repository" symbolic-ref HEAD refs/heads/dagger-head
git --git-dir="$portable_repository" rev-parse --verify 'refs/heads/main^{commit}' >/dev/null
git --git-dir="$portable_repository" bundle create "$temporary_bundle" --all HEAD
git bundle verify "$temporary_bundle" >/dev/null
mv "$temporary_bundle" "$output_bundle"
chmod 0600 "$output_bundle"
