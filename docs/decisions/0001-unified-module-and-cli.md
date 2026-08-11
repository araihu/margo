# ADR 0001: unified module and CLI

## Status

Accepted.

## Context

Margo previously described nested modules for charts, PDF, and the command.
That arrangement required consumers to coordinate separate module requirements
and tags even though the packages and binary evolve together. A nested module
can also shadow the package supplied by a newer root release.

## Decision

Margo uses one root Go module, `github.com/araihu/margo`, one root release tag,
and one CLI, `margo`. `charts`, `deck`, `embed`, `pdf`, and related packages are
packages within that module. `cmd/margo` builds the one supported command.

### Current release capability

Current releases use an installed Chromium-family executable for PDF output.
The CLI accepts `--engine auto|chromium|native`; `auto` can discover installed
Chromium after an explicit path and `MARGO_CHROMIUM_PATH`. Native backends are
compiled out in current releases. `margo doctor`, `margo version`, and
`margo --version` report this state honestly. Margo does not download a browser
or native runtime.

### Target native capabilities

WKWebView, WebView2, and WebKitGTK are future native targets behind their
matching platform runners. They remain compiled out until a release contains a
verifiable backend and its declared platform evidence. This target does not
claim that a current release provides a native backend.

## Migration

Consumers migrate from historical nested module requirements by removing
requirements for `github.com/araihu/margo/pdf`,
`github.com/araihu/margo/charts`, and `github.com/araihu/margo/cmd/margo`, then
selecting one root Margo release:

```sh
go get github.com/araihu/margo@vX.Y.Z
go mod tidy
```

Historical nested-module tags remain unchanged. They are historical release
coordinates, not the versioning scheme for current packages.

## Consequences

One version selects compatible library packages and the CLI together. CI and
developer commands test the repository as one module with
`GOWORK=off GOFLAGS=-mod=readonly`. A release must describe only current
compiled capabilities; future platform work stays outside the current release
contract.
