# Contributing to Margo

Thanks for helping improve Margo. Start from a clean branch and keep changes
focused on one behavior, documentation contract, or dependency update.

## Prerequisites

- Go 1.27.0 or newer (`go version`)
- Chromium only when running browser-backed PDF or site tests
- Dagger v0.21.8 for the full local pipeline (optional; CI pins this version)

The repository intentionally keeps the CLI and library in one root Go module.
Build the executable with `go build -o margo ./cmd/margo`; `go build .` builds
the root library archive.

## Verify a change

Run the focused package tests first, then the complete checks:

```sh
GOWORK=off go test ./... -count=1
GOWORK=off go test -race ./site -count=1
GOWORK=off go vet ./site
CGO_ENABLED=0 GOWORK=off go build -o margo ./cmd/margo
```

If the change touches the site or generated references, also run:

```sh
GOWORK=off go generate ./...
GOWORK=off go run ./cmd/margo site showcase.yaml --output-dir showcase/dist --assets local
```

The Dagger adapters expose the same gates used in CI. Run `dagger call
test`, `dagger call lint`, and `dagger call pages-site export` when Dagger is
available. PDF and deck tests need an explicitly selected standalone
Chromium; Margo never downloads a browser.

## Documentation and generated files

The public contract spans `README.md`, `showcase/content/`, CLI `--help`, and
the generated references under `docs/reference/`. Edit source schemas in
`schema/v1/`, then run `go generate ./...` so the reference Markdown remains
in sync. Do not hand-edit generated files. Keep examples copyable with the
current binary and quote RFC 3339 YAML timestamps.

When adding a CLI command or flag, update its long help and example, the
README command map, and the matching page under `showcase/content/cli/`.
When changing a site route or manifest contract, update the site tests and
the sitemap/`llms.txt` expectations. For upstream or dependency changes,
document the compatibility boundary and update the relevant version pin.

## Pull requests

Explain the user-facing contract, security implications, and verification in
the pull request description. Include focused tests for new diagnostics or
validation and attach browser/PDF evidence when a visual projection changes.
Keep generated output out of commits unless it is a checked-in reference
produced by `go generate ./...`.

Ordinary bugs and documentation issues belong in a public GitHub issue. Do not
include credentials, private URLs, or sensitive source content. Security
vulnerabilities must follow [`SECURITY.md`](SECURITY.md) and must not be
reported publicly.
