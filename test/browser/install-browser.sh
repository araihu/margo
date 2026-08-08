#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
LOCK="$SCRIPT_DIR/browser-lock.json"
MODE=""
RUNNER=""
RECEIPT=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    --provision) MODE=provision ;;
    --runner) shift; RUNNER=${1:?runner required} ;;
    --receipt) shift; RECEIPT=${1:?receipt required} ;;
    *) printf '%s\n' "margo.browser_argument_unknown:$1" >&2; exit 1 ;;
  esac
  shift
done

test "$MODE" = provision || { printf '%s\n' "margo.browser_provision_required" >&2; exit 1; }
test -n "$RECEIPT" || { printf '%s\n' "margo.browser_receipt_required" >&2; exit 1; }
case "$RECEIPT" in /*) ;; *) printf '%s\n' "margo.browser_receipt_absolute_required" >&2; exit 1 ;; esac

if [ -z "$RUNNER" ]; then
  case "$(uname -s):$(uname -m)" in
    Darwin:arm64) RUNNER=darwin-arm64 ;;
    Darwin:x86_64) RUNNER=darwin-x64 ;;
    Linux:x86_64) RUNNER=linux-x64 ;;
    *) printf '%s\n' "margo.browser_runner_unsupported" >&2; exit 1 ;;
  esac
fi

eval "$(python3 - "$LOCK" "$RUNNER" <<'PY'
import json, shlex, sys
lock = json.load(open(sys.argv[1], encoding='utf-8'))
if lock.get('schemaVersion') != 'margo/browser-lock/v1': raise SystemExit('margo.browser_lock_schema')
entry = next((row for row in lock['runners'] if row['id'] == sys.argv[2]), None)
if entry is None: raise SystemExit('margo.browser_runner_unrecorded')
values = {
  'LOCK_REVISION': lock['revision'], 'LOCK_VERSION': lock['version'],
  'LOCK_ARCHIVE': entry['archive'], 'LOCK_SHA256': entry['sha256'],
  'LOCK_EXECUTABLE': entry['executablePath'], 'LOCK_URL': entry['urls'][0],
  'LOCK_URLS': '\n'.join(entry['urls']),
}
for key, value in values.items(): print(f'{key}={shlex.quote(value)}')
PY
)"

ROOT="$SCRIPT_DIR/.cache/playwright/$RUNNER/$LOCK_REVISION"
DOWNLOADS="$SCRIPT_DIR/.cache/downloads"
ARCHIVE="$DOWNLOADS/$LOCK_ARCHIVE"
mkdir -p "$DOWNLOADS"

if [ ! -f "$ARCHIVE" ] || [ "$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')" != "$LOCK_SHA256" ]; then
  : > "$ARCHIVE.part"
  downloaded=false
  printf '%s\n' "$LOCK_URLS" | while IFS= read -r url; do
    if curl -fL --retry 2 --output "$ARCHIVE.part" "$url" && [ "$(shasum -a 256 "$ARCHIVE.part" | awk '{print $1}')" = "$LOCK_SHA256" ]; then
      mv "$ARCHIVE.part" "$ARCHIVE"
      exit 0
    fi
  done
  test -f "$ARCHIVE" || { printf '%s\n' "margo.browser_archive_unavailable" >&2; exit 1; }
fi
test "$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')" = "$LOCK_SHA256" || { printf '%s\n' "margo.browser_archive_digest_mismatch" >&2; exit 1; }

if [ ! -d "$ROOT" ]; then
  mkdir -p "$ROOT.tmp"
  unzip -q "$ARCHIVE" -d "$ROOT.tmp"
  mv "$ROOT.tmp" "$ROOT"
fi
EXECUTABLE="$ROOT/$LOCK_EXECUTABLE"
test -f "$EXECUTABLE" || { printf '%s\n' "margo.browser_executable_missing:$EXECUTABLE" >&2; exit 1; }
chmod +x "$EXECUTABLE"
EXECUTABLE_SHA256=$(shasum -a 256 "$EXECUTABLE" | awk '{print $1}')

mkdir -p "$(dirname -- "$RECEIPT")"
python3 - "$RECEIPT" "$RUNNER" "$LOCK_REVISION" "$LOCK_VERSION" "$EXECUTABLE" "$EXECUTABLE_SHA256" "$LOCK_URL" "$LOCK_SHA256" <<'PY'
import json, os, sys
path, runner, revision, version, executable, executable_sha, archive_url, archive_sha = sys.argv[1:]
record = {
  'archiveURL': archive_url,
  'archiveSHA256': archive_sha,
  'executable': os.path.realpath(executable),
  'executableSHA256': executable_sha,
  'revision': revision,
  'runner': runner,
  'schemaVersion': 'margo/browser-install/v1',
  'version': version,
}
data = json.dumps(record, sort_keys=True, separators=(',', ':')).encode()
fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
with os.fdopen(fd, 'wb') as out: out.write(data)
PY
