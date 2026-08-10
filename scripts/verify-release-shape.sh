#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 || -z "$1" || ! -d "$1" ]]; then
  echo "usage: verify-release-shape.sh OUTPUT_DIRECTORY" >&2
  exit 2
fi

output_directory="$(cd "$1" && pwd -P)"
artifacts=()
while IFS= read -r artifact; do
  artifacts+=("$artifact")
done < <(find "$output_directory" -mindepth 1 -maxdepth 1 -type f -print | LC_ALL=C sort)
if [[ ${#artifacts[@]} -eq 0 ]]; then
  echo "release shape contains no artifacts" >&2
  exit 1
fi

root_version=""
for artifact in "${artifacts[@]}"; do
  name="$(basename "$artifact")"
  if [[ "$name" != "margo" && "$name" != "margo.exe" ]]; then
    echo "unexpected release artifact name: $name" >&2
    exit 1
  fi

  metadata="$(go version -m "$artifact")"
  if ! grep -Eq $'^\tpath\tgithub.com/araihu/margo/cmd/margo$' <<<"$metadata"; then
    echo "artifact does not contain the margo command path: $name" >&2
    exit 1
  fi
  module_line="$(grep -E $'^\tmod\tgithub.com/araihu/margo\t' <<<"$metadata" || true)"
  if [[ -z "$module_line" ]]; then
    echo "artifact does not derive from the root margo module: $name" >&2
    exit 1
  fi
  if grep -Eq $'^\tmod\tgithub.com/araihu/margo/(charts|deck|pdf|cmd/margo)\t' <<<"$metadata"; then
    echo "artifact contains nested margo module metadata: $name" >&2
    exit 1
  fi
  version="$(awk '{print $3}' <<<"$module_line")"
  if [[ -n "$root_version" && "$version" != "$root_version" ]]; then
    echo "artifacts derive from different root versions" >&2
    exit 1
  fi
  root_version="$version"

  command_version="$("$artifact" version)"
  if ! grep -Fq "module github.com/araihu/margo" <<<"$command_version"; then
    echo "margo version reports a different module: $name" >&2
    exit 1
  fi
done

echo "verified ${#artifacts[@]} margo artifact(s) from root version $root_version"
