#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 || -z "$1" ]]; then
  echo "usage: install-dagger.sh INSTALL_DIRECTORY" >&2
  exit 2
fi

readonly dagger_version="0.21.8"
readonly archive="dagger_v${dagger_version}_linux_amd64.tar.gz"
readonly expected_sha256="53e226c7da8fb75171e58c35759d736d961ce8b3a12db0baa7b7107954fccc5a"
readonly release_url="https://github.com/dagger/dagger/releases/download/v${dagger_version}/${archive}"
readonly install_directory="$1"

mkdir -p "$install_directory"

if embedded_dagger="$(command -v dagger 2>/dev/null)"; then
  embedded_version="$("$embedded_dagger" version | awk 'NR == 1 {print $2}')"
  if [[ "$embedded_version" == "v${dagger_version}" ]]; then
    ln -sf "$embedded_dagger" "$install_directory/dagger"
    exit 0
  fi
  echo "ignoring mismatched embedded Dagger CLI: $embedded_version" >&2
fi

temporary_archive="$(mktemp "${install_directory}/.${archive}.XXXXXX")"
trap 'rm -f "$temporary_archive"' EXIT

curl --proto '=https' --tlsv1.2 --fail --silent --show-error --location \
  --output "$temporary_archive" "$release_url"
actual_sha256="$(sha256sum "$temporary_archive" | awk '{print $1}')"
if [[ "$actual_sha256" != "$expected_sha256" ]]; then
  echo "Dagger archive checksum mismatch" >&2
  exit 1
fi

tar -xzf "$temporary_archive" -C "$install_directory" dagger
chmod 0755 "$install_directory/dagger"
actual_version="$("$install_directory/dagger" version | awk 'NR == 1 {print $2}')"
if [[ "$actual_version" != "v${dagger_version}" ]]; then
  echo "unexpected Dagger CLI version: $actual_version" >&2
  exit 1
fi
