# Margo Development Server and Live Reload Design

Date: 2026-08-18
Status: approved in chat; awaiting written-spec review

## Purpose

Add a development-only server for Margo static sites. A writer can point Margo
at a Markdown directory and start immediately, while an established site can
use its existing YAML configuration. The server rebuilds after local file
changes and reloads connected browsers after successful builds.

This is not a production web server. It provides no TLS, authentication,
authorization, rate limiting, hardened request limits, or deployment contract.
Command help, startup output, README documentation, and non-loopback warnings
must call this out explicitly.

## Command Contract

```text
margo serve [INPUT_DIR|CONFIG] [--host HOST] [--port PORT] [--open]
```

- Input defaults to the current directory.
- An explicit `.yaml` or `.yml` file is loaded as a Margo site config.
- A directory containing `site.yaml` uses that config automatically.
- Any other directory is recursively built using the same strong defaults as
  `margo site INPUT_DIR`: local assets and Margo's default linked-site output.
- `--host` defaults to `127.0.0.1`.
- `--open` opens the selected URL after the HTTP server starts. It is opt-in.
- Ctrl+C or context cancellation shuts down the watcher, reload streams, and
  HTTP server.

### Port Selection

When `--port` is absent, Margo attempts these ports in order:

```text
8080, 8000, 3000, 1313, 4000
```

Each attempt binds a listener directly, avoiding a check-then-bind race. If all
baked-in ports are unavailable, Margo binds port `0` and uses the valid port
selected by the operating system.

When `--port` is explicitly supplied, Margo validates the range and attempts
only that port. An unavailable explicit port is an error; no fallback occurs.

## Development Workflow

Startup resolves the project, selects and retains a TCP listener, starts the
watcher, and performs the initial build. The selected URL is printed with a
clear `development server` label. When a non-loopback host is selected, Margo
warns that the unauthenticated development server is exposed to the network.

The initial build may fail because content or configuration is incomplete.
For a resolvable input path, the server remains available, serves an escaped
diagnostic page, and watches for a correcting edit. A missing or unreadable
input path is fatal because no safe watch root can be established.

After a successful build, the server publishes an immutable in-memory snapshot.
It never writes or mutates the configured `output` directory. Later successful
builds atomically replace the snapshot. Later failed builds retain the last
good snapshot, emit diagnostics to the terminal, and do not reload browsers.

HTTP responses use `Cache-Control: no-store`. Requests map to generated
artifacts using static-site conventions, including `index.html` resolution and
the configured `base_path`. The root redirects to a non-root configured base
path. Missing artifacts return 404 rather than falling through to local files.

## Live Reload

The design follows two established development-tool patterns:

- Hugo builds, watches, serves, injects a reload client, and refreshes browsers
  after changes.
- Air debounces filesystem events, serializes rebuilds, queues a follow-up when
  changes arrive during a build, and injects a small browser reload client.

Margo uses a private Server-Sent Events endpoint under `/.margo/`. HTML
responses receive a small development-only script immediately before
`</body>`. Non-HTML artifacts and generated snapshot bytes remain unchanged.
If an HTML document has no closing body tag, it is served unchanged.

Each successful snapshot has a monotonically increasing generation. A new SSE
client receives a ready event for the current generation without reloading.
Later generations trigger `location.reload()`. The browser reconnects using
native `EventSource` behavior. Failed builds never advance the generation.

## Watching and Rebuild Scheduling

The real watcher uses `fsnotify`, already present in Margo's dependency graph
and promoted to a direct dependency. It recursively watches the resolved local
project tree, including Markdown, YAML, CSS, images, and other local assets.
It adds newly created directories while running.

The watcher excludes `.git` and the configured output directory. This avoids
irrelevant repository traffic and output-triggered rebuild loops. Symlinked
directories are not traversed, matching Margo's current source discovery
boundary.

Filesystem event bursts are debounced before a build. Builds are serialized.
When one or more relevant changes arrive during a build, the coordinator queues
exactly one follow-up build, ensuring the latest filesystem state is eventually
observed without unbounded build queues.

The first implementation always performs a complete site rebuild. Incremental
dependency graphs, CSS-only refreshes, configurable watch filters, polling
fallbacks, and browser error overlays are outside this scope.

## Internal Architecture

No public Margo library API is added. A new internal development-server package
owns orchestration and HTTP behavior. Cobra remains an adapter that parses user
input, resolves command dependencies, and reports errors.

Core test seams are intentionally small:

```go
type Builder interface {
	Build(context.Context) (Snapshot, error)
}

type ChangeSource interface {
	Changes() <-chan struct{}
	Errors() <-chan error
	Close() error
}

type ListenFunc func(network, address string) (net.Listener, error)
```

`Snapshot` contains an immutable artifact map plus route/build metadata. The
coordinator depends on `Builder` and `ChangeSource`; production adapters connect
them to `site.Build`, `site.BuildConfig`, and `fsnotify`. Port selection depends
on `ListenFunc`, letting tests prove exact bind behavior without real ports.

HTTP serving is a handler over an atomic snapshot store. SSE fan-out is a small
generation broker. Reload injection is a response transformation independent
of the builder. Each unit can be tested without starting the full CLI.

The existing CLI dependency container receives one serve function seam. Cobra
tests can capture the resolved serve request and return without opening sockets,
watching files, or waiting for cancellation.

## Error Handling

- Invalid flags, missing inputs, unreadable watch roots, and explicit-port bind
  failures return normal CLI errors.
- Automatic candidate bind failures are silent until all baked-in candidates
  fail; OS-assigned fallback failure returns one actionable bind error.
- Watcher errors are reported. A terminal watcher failure stops the command;
  transient event-level errors do not discard the last good snapshot.
- Build diagnostics preserve existing Margo diagnostic codes and remediation
  text. Browser error pages HTML-escape all diagnostic content.
- HTTP server failures cancel the coordinator and close owned resources.
- Shutdown is idempotent and bounded.

## Testing Strategy

Development follows test-driven implementation.

- Cobra tests cover default input, directory/config discovery, flags,
  development-only help text, and request construction with no real sockets.
- Port tests use fake `ListenFunc` values to prove candidate order, ephemeral
  fallback, explicit-port exclusivity, validation, and listener retention.
- Coordinator tests use fake builders and change sources to prove initial
  success, recoverable initial failure, atomic successful replacement,
  last-good retention, debounce, serialized builds, one queued follow-up, and
  context shutdown.
- Handler tests use `httptest` to verify routes, base paths, MIME types,
  no-store headers, 404s, escaped error pages, HTML-only injection, unchanged
  snapshot bytes, SSE readiness, and generation reload events.
- Watcher integration tests use temporary directories to verify recursive
  changes, newly created directories, output exclusion, `.git` exclusion, and
  close behavior.
- A focused end-to-end test runs the server on an OS-selected loopback port,
  edits Markdown, observes rebuilt content and an SSE generation, then cancels
  cleanly.
- `go test -race` covers concurrent snapshot reads, swaps, and SSE clients.

## Documentation

README CLI reference gains `margo serve`, example commands for both directory
and config workflows, automatic port behavior, live reload semantics, and the
development-only warning. Root help lists `serve` as a development server, not
a deployment or production-hosting command.

## Non-Goals

- Production hosting or deployment
- TLS or automatic certificates
- Authentication or remote collaboration
- Writing generated files to `dist`
- Incremental page-level builds
- CSS hot replacement without a page reload
- Configurable port candidate lists
- Configurable debounce or watch filters
- Polling fallback for filesystems without reliable notifications
