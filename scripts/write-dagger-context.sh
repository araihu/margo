#!/usr/bin/env bash
set -euo pipefail

: "${MARGO_CACHE_DOMAIN:?MARGO_CACHE_DOMAIN is required}"
: "${MARGO_RUN_ID:?MARGO_RUN_ID is required}"
: "${MARGO_RUN_ATTEMPT:?MARGO_RUN_ATTEMPT is required}"

if [[ ! "$MARGO_CACHE_DOMAIN" =~ ^(local|trusted-main|trusted-release|untrusted-pr-[0-9]+)$ ]]; then
  echo "invalid Dagger cache trust domain" >&2
  exit 2
fi
if [[ ! "$MARGO_RUN_ID" =~ ^[0-9]+$ || ! "$MARGO_RUN_ATTEMPT" =~ ^[0-9]+$ ]]; then
  echo "invalid Dagger run nonce" >&2
  exit 2
fi

release_tag="${MARGO_RELEASE_TAG:-}"
if [[ -n "$release_tag" && ! "$release_tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "invalid root Margo release tag" >&2
  exit 2
fi

umask 077
if [[ -n "$release_tag" ]]; then
  printf '{"cacheDomain":"%s","nonce":"%s-%s","releaseTag":"%s"}\n' \
    "$MARGO_CACHE_DOMAIN" "$MARGO_RUN_ID" "$MARGO_RUN_ATTEMPT" "$release_tag" \
    > .dagger-ci-context.json
else
  printf '{"cacheDomain":"%s","nonce":"%s-%s"}\n' \
    "$MARGO_CACHE_DOMAIN" "$MARGO_RUN_ID" "$MARGO_RUN_ATTEMPT" \
    > .dagger-ci-context.json
fi
