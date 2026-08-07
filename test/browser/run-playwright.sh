#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
MODE=""
ENV_FILE=""
GREP=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    --check) MODE=check ;;
    --env-file) shift; ENV_FILE=${1:?environment file required} ;;
    --grep) shift; GREP=${1:?grep required} ;;
    *) printf '%s\n' "margo.harness_argument_unknown:$1" >&2; exit 1 ;;
  esac
  shift
done

test "$MODE" = check || { printf '%s\n' "margo.harness_check_required" >&2; exit 1; }
case "$ENV_FILE" in /*) ;; *) printf '%s\n' "margo.harness_env_absolute_required" >&2; exit 1 ;; esac
test -f "$ENV_FILE" || { printf '%s\n' "margo.harness_env_missing" >&2; exit 1; }

EXPECTED_KEYS='MARGO_NODE_BIN MARGO_NPM_BIN MARGO_PLAYWRIGHT_CLI MARGO_CHROMIUM_EXECUTABLE MARGO_NPM_CACHE MARGO_NPM_CACHE_RECEIPT MARGO_NODE_SHA256 MARGO_NPM_SHA256 MARGO_CHROMIUM_SHA256 MARGO_CHROMIUM_REVISION MARGO_CHROMIUM_VERSION'
for key in $EXPECTED_KEYS; do
  count=$(grep -c "^export $key=" "$ENV_FILE" || true)
  test "$count" -eq 1 || { printf '%s\n' "margo.harness_env_key_invalid:$key" >&2; exit 1; }
  eval "caller_$key=\${$key-}"
done
test "$(wc -l < "$ENV_FILE" | tr -d ' ')" -eq 11 || { printf '%s\n' "margo.harness_env_unknown_key" >&2; exit 1; }

# The file is bootstrap-owned and restricted above to one assignment per known key.
. "$ENV_FILE"
for key in $EXPECTED_KEYS; do
  eval "before=\${caller_$key-}"
  eval "after=\${$key-}"
  if [ -n "$before" ] && [ "$before" != "$after" ]; then
    printf '%s\n' "margo.harness_ambient_conflict:$key" >&2
    exit 1
  fi
done

for path in "$MARGO_NODE_BIN" "$MARGO_NPM_BIN" "$MARGO_CHROMIUM_EXECUTABLE" "$MARGO_NPM_CACHE" "$MARGO_NPM_CACHE_RECEIPT"; do
  case "$path" in /*) ;; *) printf '%s\n' "margo.harness_path_not_absolute:$path" >&2; exit 1 ;; esac
done
test -x "$MARGO_NODE_BIN" || { printf '%s\n' "margo.harness_node_missing" >&2; exit 1; }
test -f "$MARGO_NPM_BIN" || { printf '%s\n' "margo.harness_npm_missing" >&2; exit 1; }
test -x "$MARGO_CHROMIUM_EXECUTABLE" || { printf '%s\n' "margo.harness_chromium_missing" >&2; exit 1; }
test "$(shasum -a 256 "$MARGO_NODE_BIN" | awk '{print $1}')" = "$MARGO_NODE_SHA256" || { printf '%s\n' "margo.harness_node_digest_mismatch" >&2; exit 1; }
test "$(shasum -a 256 "$MARGO_NPM_BIN" | awk '{print $1}')" = "$MARGO_NPM_SHA256" || { printf '%s\n' "margo.harness_npm_digest_mismatch" >&2; exit 1; }
test "$(shasum -a 256 "$MARGO_CHROMIUM_EXECUTABLE" | awk '{print $1}')" = "$MARGO_CHROMIUM_SHA256" || { printf '%s\n' "margo.harness_chromium_digest_mismatch" >&2; exit 1; }
test "$MARGO_CHROMIUM_REVISION" = 1169 && test "$MARGO_CHROMIUM_VERSION" = 136.0.7103.25

MARGO_NODE_BIN="$MARGO_NODE_BIN" MARGO_NPM_BIN="$MARGO_NPM_BIN" \
  "$MARGO_NODE_BIN" "$SCRIPT_DIR/populate-npm-cache.mjs" --check \
  --lock "$SCRIPT_DIR/package-lock.json" --cache "$MARGO_NPM_CACHE" --receipt "$MARGO_NPM_CACHE_RECEIPT"

cd "$SCRIPT_DIR"
PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 PLAYWRIGHT_BROWSERS_PATH=0 \
  "$MARGO_NODE_BIN" "$MARGO_NPM_BIN" ci --offline --cache "$MARGO_NPM_CACHE" \
  --ignore-scripts --no-audit --no-fund --logs-max=0 --update-notifier=false
test -f "$MARGO_PLAYWRIGHT_CLI" || { printf '%s\n' "margo.harness_playwright_cli_missing" >&2; exit 1; }

PW_DISABLE_TS_ESM=1 PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 PLAYWRIGHT_BROWSERS_PATH=0 \
  "$MARGO_NODE_BIN" "$MARGO_PLAYWRIGHT_CLI" test --config "$SCRIPT_DIR/playwright.config.mjs" --grep "$GREP"

printf '%s\n' "margo.harness_ok node=$($MARGO_NODE_BIN --version) npm=$($MARGO_NODE_BIN $MARGO_NPM_BIN --version) chromium_revision=$MARGO_CHROMIUM_REVISION chromium_version=$MARGO_CHROMIUM_VERSION network=0"
