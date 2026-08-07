#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
LOCK="$SCRIPT_DIR/node-toolchain.lock"
MODE=""
CACHE_ROOT=""
BROWSER_RECEIPT=""
EMIT=false

while [ "$#" -gt 0 ]; do
  case "$1" in
    --provision-node) MODE=provision ;;
    --check) MODE=check ;;
    --cache-root) shift; CACHE_ROOT=${1:?cache root required} ;;
    --browser-receipt) shift; BROWSER_RECEIPT=${1:?browser receipt required} ;;
    --emit-shell-env) EMIT=true ;;
    *) printf '%s\n' "margo.node_argument_unknown:$1" >&2; exit 1 ;;
  esac
  shift
done

test -n "$MODE" || { printf '%s\n' "margo.node_mode_required" >&2; exit 1; }
case "$CACHE_ROOT" in /*) ;; *) printf '%s\n' "margo.node_cache_absolute_required" >&2; exit 1 ;; esac
case "$BROWSER_RECEIPT" in /*) ;; *) printf '%s\n' "margo.node_browser_receipt_absolute_required" >&2; exit 1 ;; esac

case "$(uname -s):$(uname -m)" in
  Darwin:arm64) RUNNER=darwin-arm64 ;;
  Darwin:x86_64) RUNNER=darwin-x64 ;;
  Linux:x86_64) RUNNER=linux-x64 ;;
  *) printf '%s\n' "margo.node_runner_unsupported" >&2; exit 1 ;;
esac

eval "$(python3 - "$LOCK" "$RUNNER" <<'PY'
import json, shlex, sys
lock = json.load(open(sys.argv[1], encoding='utf-8'))
if lock.get('schemaVersion') != 'margo/node-toolchain/v1': raise SystemExit('margo.node_lock_schema')
entry = next((row for row in lock['runners'] if row['id'] == sys.argv[2]), None)
if entry is None: raise SystemExit('margo.node_runner_unrecorded')
manifest = lock['manifest']; npm = lock['npmSource']
values = {
  'NODE_VERSION': lock['nodeVersion'], 'NPM_VERSION': lock['npmVersion'],
  'ARCHIVE_URL': entry['archiveURL'], 'ARCHIVE_SHA256': entry['archiveSHA256'],
  'ARCHIVE_ROOT': entry['archiveRoot'], 'NODE_PATH': entry['nodePath'],
  'NPM_CLI_PATH': entry['npmCLIPath'], 'NODE_SHA256': entry['nodeSHA256'],
  'NPM_SHA256': entry['npmSHA256'], 'MANIFEST_URL': manifest['url'],
  'MANIFEST_SHA256': manifest['sha256'], 'SIGNATURE_URL': manifest['signatureURL'],
  'SIGNATURE_SHA256': manifest['signatureSHA256'], 'SIGNING_FINGERPRINT': manifest['signingKeyFingerprint'],
  'KEY_URL': manifest['keySourceURL'], 'KEY_SHA256': manifest['keySourceSHA256'],
  'NPM_SOURCE_URL': npm['url'], 'NPM_SOURCE_INTEGRITY': npm['integrity'],
}
for key, value in values.items(): print(f'{key}={shlex.quote(value)}')
PY
)"

INSTALL_ROOT="$SCRIPT_DIR/.cache/node/$NODE_VERSION/$RUNNER"
DOWNLOADS="$SCRIPT_DIR/.cache/downloads/node/$NODE_VERSION"
KEYRING_DIR="$SCRIPT_DIR/.cache/node-keyring"
ARCHIVE_NAME=${ARCHIVE_URL##*/}
ARCHIVE="$DOWNLOADS/$ARCHIVE_NAME"
NODE_BIN="$INSTALL_ROOT/$ARCHIVE_ROOT/$NODE_PATH"
NPM_CLI="$INSTALL_ROOT/$ARCHIVE_ROOT/$NPM_CLI_PATH"
NPM_BIN="$NPM_CLI"

verify_provenance() {
  test -f "$KEYRING_DIR/nodejs-release-keys.kbx" || { printf '%s\n' "margo.node_keyring_missing" >&2; exit 1; }
  test -f "$DOWNLOADS/SHASUMS256.txt" || { printf '%s\n' "margo.node_manifest_missing" >&2; exit 1; }
  test -f "$DOWNLOADS/SHASUMS256.txt.sig" || { printf '%s\n' "margo.node_signature_missing" >&2; exit 1; }
  test -f "$DOWNLOADS/npm.tgz" || { printf '%s\n' "margo.npm_source_missing" >&2; exit 1; }
  test -f "$ARCHIVE" || { printf '%s\n' "margo.node_archive_missing" >&2; exit 1; }
  test "$(shasum -a 256 "$KEYRING_DIR/nodejs-release-keys.kbx" | awk '{print $1}')" = "$KEY_SHA256" || { printf '%s\n' "margo.node_keyring_digest_mismatch" >&2; exit 1; }
  test "$(shasum -a 256 "$DOWNLOADS/SHASUMS256.txt" | awk '{print $1}')" = "$MANIFEST_SHA256" || { printf '%s\n' "margo.node_manifest_digest_mismatch" >&2; exit 1; }
  test "$(shasum -a 256 "$DOWNLOADS/SHASUMS256.txt.sig" | awk '{print $1}')" = "$SIGNATURE_SHA256" || { printf '%s\n' "margo.node_signature_digest_mismatch" >&2; exit 1; }
  test "$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')" = "$ARCHIVE_SHA256" || { printf '%s\n' "margo.node_archive_digest_mismatch" >&2; exit 1; }
  expected_npm=${NPM_SOURCE_INTEGRITY#sha512-}
  actual_npm=$(openssl dgst -sha512 -binary "$DOWNLOADS/npm.tgz" | openssl base64 -A)
  test "$actual_npm" = "$expected_npm" || { printf '%s\n' "margo.npm_source_integrity_mismatch" >&2; exit 1; }
  GPG_STATUS=$(gpgv --homedir "$KEYRING_DIR" --keyring "$KEYRING_DIR/nodejs-release-keys.kbx" --status-fd 1 "$DOWNLOADS/SHASUMS256.txt.sig" "$DOWNLOADS/SHASUMS256.txt" 2>/dev/null) || { printf '%s\n' "margo.node_signature_invalid" >&2; exit 1; }
  printf '%s\n' "$GPG_STATUS" | grep -F "[GNUPG:] VALIDSIG $SIGNING_FINGERPRINT " >/dev/null || { printf '%s\n' "margo.node_signer_mismatch" >&2; exit 1; }
  printf '%s\n' "$GPG_STATUS" | grep -F "[GNUPG:] GOODSIG" >/dev/null || { printf '%s\n' "margo.node_signature_invalid" >&2; exit 1; }
  grep -F "$ARCHIVE_SHA256  $ARCHIVE_NAME" "$DOWNLOADS/SHASUMS256.txt" >/dev/null || { printf '%s\n' "margo.node_archive_not_signed" >&2; exit 1; }
}

verify_browser_receipt() {
  python3 - "$BROWSER_RECEIPT" "$RUNNER" "$SCRIPT_DIR/browser-lock.json" <<'PY'
import hashlib, json, os, sys
path, runner, lock_path = sys.argv[1:]
data = open(path, 'rb').read(); record = json.loads(data); lock = json.load(open(lock_path, encoding='utf-8'))
if set(record) != {'archiveURL','archiveSHA256','executable','executableSHA256','revision','runner','schemaVersion','version'}: raise SystemExit('margo.browser_receipt_fields')
if record.get('schemaVersion') != 'margo/browser-install/v1' or record.get('runner') != runner: raise SystemExit('margo.browser_receipt_mismatch')
if record.get('revision') != '1169' or record.get('version') != '136.0.7103.25': raise SystemExit('margo.browser_identity_mismatch')
entry = next((row for row in lock['runners'] if row['id'] == runner), None)
if entry is None or record.get('archiveURL') != entry['urls'][0] or record.get('archiveSHA256') != entry['sha256']: raise SystemExit('margo.browser_archive_identity_mismatch')
executable = record.get('executable', '')
if not os.path.isabs(executable) or os.path.realpath(executable) != executable or not os.path.isfile(executable): raise SystemExit('margo.browser_executable_missing')
digest = hashlib.sha256(open(executable, 'rb').read()).hexdigest()
if digest != record.get('executableSHA256'): raise SystemExit('margo.browser_executable_digest_mismatch')
print(record['executable']); print(record['executableSHA256']); print(record['revision']); print(record['version'])
PY
}

verify_install() {
  test -x "$NODE_BIN" || { printf '%s\n' "margo.node_executable_missing" >&2; exit 1; }
  test -f "$NPM_CLI" || { printf '%s\n' "margo.npm_cli_missing" >&2; exit 1; }
  test "$(shasum -a 256 "$NODE_BIN" | awk '{print $1}')" = "$NODE_SHA256" || { printf '%s\n' "margo.node_executable_digest_mismatch" >&2; exit 1; }
  test "$(shasum -a 256 "$NPM_CLI" | awk '{print $1}')" = "$NPM_SHA256" || { printf '%s\n' "margo.npm_executable_digest_mismatch" >&2; exit 1; }
  test "$($NODE_BIN --version)" = "$NODE_VERSION" || { printf '%s\n' "margo.node_version_mismatch" >&2; exit 1; }
  test "$($NODE_BIN "$NPM_CLI" --version)" = "$NPM_VERSION" || { printf '%s\n' "margo.npm_version_mismatch" >&2; exit 1; }
}

if [ "$MODE" = provision ]; then
  mkdir -p "$DOWNLOADS" "$KEYRING_DIR" "$INSTALL_ROOT"
  curl -fL --retry 3 --output "$KEYRING_DIR/nodejs-release-keys.kbx" "$KEY_URL"
  curl -fL --retry 3 --output "$DOWNLOADS/SHASUMS256.txt" "$MANIFEST_URL"
  curl -fL --retry 3 --output "$DOWNLOADS/SHASUMS256.txt.sig" "$SIGNATURE_URL"
  curl -fL --retry 3 --output "$DOWNLOADS/npm.tgz" "$NPM_SOURCE_URL"
  curl -fL --retry 3 --output "$ARCHIVE" "$ARCHIVE_URL"
  if [ ! -d "$INSTALL_ROOT/$ARCHIVE_ROOT" ]; then
    case "$ARCHIVE" in
      *.zip) unzip -q "$ARCHIVE" -d "$INSTALL_ROOT" ;;
      *) tar -xf "$ARCHIVE" -C "$INSTALL_ROOT" ;;
    esac
  fi
fi

verify_provenance
verify_install
BROWSER_VALUES=$(verify_browser_receipt)
MARGO_CHROMIUM_EXECUTABLE=$(printf '%s\n' "$BROWSER_VALUES" | sed -n '1p')
MARGO_CHROMIUM_SHA256=$(printf '%s\n' "$BROWSER_VALUES" | sed -n '2p')
MARGO_CHROMIUM_REVISION=$(printf '%s\n' "$BROWSER_VALUES" | sed -n '3p')
MARGO_CHROMIUM_VERSION=$(printf '%s\n' "$BROWSER_VALUES" | sed -n '4p')

if [ "$MODE" = check ]; then
  test -f "$CACHE_ROOT/receipt.json" || { printf '%s\n' "margo.npm_cache_receipt_missing" >&2; exit 1; }
  MARGO_NODE_BIN="$NODE_BIN" MARGO_NPM_BIN="$NPM_BIN" "$NODE_BIN" "$SCRIPT_DIR/populate-npm-cache.mjs" --check --lock "$SCRIPT_DIR/package-lock.json" --cache "$CACHE_ROOT" --receipt "$CACHE_ROOT/receipt.json"
fi

if [ "$EMIT" = true ]; then
  python3 - "$NODE_BIN" "$NPM_BIN" "$SCRIPT_DIR/node_modules/@playwright/test/cli.js" "$MARGO_CHROMIUM_EXECUTABLE" "$CACHE_ROOT" "$CACHE_ROOT/receipt.json" "$NODE_SHA256" "$NPM_SHA256" "$MARGO_CHROMIUM_SHA256" "$MARGO_CHROMIUM_REVISION" "$MARGO_CHROMIUM_VERSION" <<'PY'
import shlex, sys
keys = ['MARGO_NODE_BIN','MARGO_NPM_BIN','MARGO_PLAYWRIGHT_CLI','MARGO_CHROMIUM_EXECUTABLE','MARGO_NPM_CACHE','MARGO_NPM_CACHE_RECEIPT','MARGO_NODE_SHA256','MARGO_NPM_SHA256','MARGO_CHROMIUM_SHA256','MARGO_CHROMIUM_REVISION','MARGO_CHROMIUM_VERSION']
for key, value in zip(keys, sys.argv[1:]): print(f'export {key}={shlex.quote(value)}')
PY
fi
