#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
MODE=""
ENV_FILE=""
GREP=""
CONTRAST_HTML=""
CONTRAST_MODE="both"
CONTRAST_FORMAT="json"
CONTRAST_OUTPUT=""
CONTRAST_ONLY=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --check) MODE=check ;;
    --env-file) shift; ENV_FILE=${1:?environment file required} ;;
    --grep) shift; GREP=${1:?grep required} ;;
    --contrast-html) shift; CONTRAST_HTML=${1:?contrast HTML required} ;;
    --contrast-mode) shift; CONTRAST_MODE=${1:?contrast mode required} ;;
    --contrast-format) shift; CONTRAST_FORMAT=${1:?contrast format required} ;;
    --contrast-output) shift; CONTRAST_OUTPUT=${1:?contrast output required} ;;
    --contrast-only) CONTRAST_ONLY=1 ;;
    *) printf '%s\n' "margo.harness_argument_unknown:$1" >&2; exit 1 ;;
  esac
  shift
done

test "$MODE" = check || { printf '%s\n' "margo.harness_check_required" >&2; exit 1; }
case "$ENV_FILE" in /*) ;; *) printf '%s\n' "margo.harness_env_absolute_required" >&2; exit 1 ;; esac
test -f "$ENV_FILE" || { printf '%s\n' "margo.harness_env_missing" >&2; exit 1; }
if [ -n "$CONTRAST_HTML" ]; then
  case "$CONTRAST_HTML" in /*) ;; *) printf '%s\n' "margo.contrast_lint_html_absolute_required" >&2; exit 1 ;; esac
  test -f "$CONTRAST_HTML" || { printf '%s\n' "margo.contrast_lint_html_missing" >&2; exit 1; }
else
  test "$CONTRAST_ONLY" -eq 0 || { printf '%s\n' "margo.contrast_lint_html_required" >&2; exit 1; }
fi
case "$CONTRAST_MODE" in light|dark|both) ;; *) printf '%s\n' "margo.contrast_lint_mode_invalid:$CONTRAST_MODE" >&2; exit 1 ;; esac
case "$CONTRAST_FORMAT" in json|text) ;; *) printf '%s\n' "margo.contrast_lint_format_invalid:$CONTRAST_FORMAT" >&2; exit 1 ;; esac
if [ -n "$CONTRAST_OUTPUT" ]; then
  case "$CONTRAST_OUTPUT" in /*) ;; *) printf '%s\n' "margo.contrast_lint_output_absolute_required" >&2; exit 1 ;; esac
fi

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
test "$MARGO_CHROMIUM_REVISION" = 1193 && test "$MARGO_CHROMIUM_VERSION" = 140.0.7339.186

MARGO_NODE_BIN="$MARGO_NODE_BIN" MARGO_NPM_BIN="$MARGO_NPM_BIN" \
  "$MARGO_NODE_BIN" "$SCRIPT_DIR/populate-npm-cache.mjs" --check \
  --lock "$SCRIPT_DIR/package-lock.json" --cache "$MARGO_NPM_CACHE" --receipt "$MARGO_NPM_CACHE_RECEIPT"

cd "$SCRIPT_DIR"
PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 PLAYWRIGHT_BROWSERS_PATH=0 \
  "$MARGO_NODE_BIN" "$MARGO_NPM_BIN" ci --offline --cache "$MARGO_NPM_CACHE" \
  --ignore-scripts --no-audit --no-fund --logs-max=0 --update-notifier=false
test -f "$MARGO_PLAYWRIGHT_CLI" || { printf '%s\n' "margo.harness_playwright_cli_missing" >&2; exit 1; }

if [ -n "$CONTRAST_HTML" ]; then
  if [ -n "$CONTRAST_OUTPUT" ]; then
    "$MARGO_NODE_BIN" "$SCRIPT_DIR/lint-contrast.mjs" \
      --html "$CONTRAST_HTML" --mode "$CONTRAST_MODE" --format "$CONTRAST_FORMAT" --output "$CONTRAST_OUTPUT"
  else
    "$MARGO_NODE_BIN" "$SCRIPT_DIR/lint-contrast.mjs" \
      --html "$CONTRAST_HTML" --mode "$CONTRAST_MODE" --format "$CONTRAST_FORMAT"
  fi
fi

if [ "$CONTRAST_ONLY" -eq 1 ]; then
  printf '%s\n' "margo.harness_contrast_ok node=$($MARGO_NODE_BIN --version) chromium_revision=$MARGO_CHROMIUM_REVISION chromium_version=$MARGO_CHROMIUM_VERSION network=0"
  exit 0
fi

PW_DISABLE_TS_ESM=1 PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 PLAYWRIGHT_BROWSERS_PATH=0 \
  "$MARGO_NODE_BIN" "$MARGO_PLAYWRIGHT_CLI" test --config "$SCRIPT_DIR/playwright.config.mjs" --grep "$GREP"

printf '%s\n' "margo.harness_ok node=$($MARGO_NODE_BIN --version) npm=$($MARGO_NODE_BIN $MARGO_NPM_BIN --version) chromium_revision=$MARGO_CHROMIUM_REVISION chromium_version=$MARGO_CHROMIUM_VERSION network=0"
