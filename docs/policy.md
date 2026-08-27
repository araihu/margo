# Host policy and natural iframe embeds

Margo separates author content from host authority. Documents never declare,
request, grant, or negotiate capabilities. With no policy, ordinary Markdown
renders deterministically and offline; raw HTML, iframe embeds, and remote
Markdown images fail closed.

The canonical contract is the shipped
[`schema/v1/policy.json`](../schema/v1/policy.json). The exhaustive generated
[policy reference](reference/policy.md) lists every field, default, target,
limit, precedence rule, and security effect. Print the exact embedded schema
offline with `margo schema policy` and attach the result to the policy file in
an IDE. The same binary emits `margo schema document` for Markdown frontmatter
and `margo schema site` for `site.yaml`; schemas provide local completion while
the CLI remains authoritative for cross-file and asset validation.

## Library API

The host constructs a `margo.Policy` and supplies it before compilation:

```go
policy := margo.Policy{
	SchemaVersion: "margo-policy/v1",
	RawHTML:       margo.RawHTMLSanitized,
	InputBytes:    margo.MaxDocumentBytes,
	OutputBytes:   margo.MaxOutputBytes,
	Iframe: &margo.IframePolicy{
		AllowedOrigins: []string{"https://video.example.com"},
		Sandbox:        []margo.SandboxToken{margo.SandboxAllowPresentation},
		ReferrerPolicy: margo.ReferrerNoReferrer,
		Projections: margo.TargetProjections{
			HTML: margo.ProjectionInteractive,
			Site: margo.ProjectionInteractive,
			PDF:  margo.ProjectionStaticLink,
			Deck: margo.ProjectionDeny,
		},
	},
}

compiler := margo.New(margo.WithHostPolicy(policy))
document, err := compiler.Compile(ctx, source)
```

Use the same policy with `margo.WithCheckPolicy(policy)` for preflight. The
document cannot add an origin, sandbox capability, projection, or raw-HTML
grant.

## CLI policy file

`margo check`, `html`, `pdf`, `site`, and `deck` accept `--policy FILE`.
Policy input is JSON, capped at 64 KiB, schema-validated before conversion, and
rejects duplicate keys, unknown properties, non-HTTPS origins, wildcard
syntax, unsupported values, and multiple JSON documents. The path `-` is
rejected so document stdin cannot also become policy authority.

```json
{
  "schemaVersion": "margo-policy/v1",
  "rawHTML": "sanitized",
  "inputBytes": 16777216,
  "outputBytes": 67108864,
  "iframe": {
    "allowedOrigins": ["https://video.example.com"],
    "sandbox": ["allow-presentation"],
    "referrerPolicy": "no-referrer",
    "projections": {
      "html": "interactive",
      "site": "interactive",
      "pdf": "static-link",
      "deck": "deny"
    }
  }
}
```

Omission selects schema-documented defaults. Explicit `0` is invalid for byte
ceilings. Unknown properties fail. The normalized policy is hashed; CLI JSON
reports and site manifests record its `sha256:...` identity.

### IDE and external validator workflow

Capture schemas from the same binary that CI runs and associate them externally;
do not add a `$schema` property to the policy JSON because the closed runtime
contract rejects unknown root fields:

```sh
mkdir -p .schemas
margo schema policy > .schemas/margo-policy.schema.json
margo schema document > .schemas/margo-document.schema.json
margo schema site > .schemas/margo-site.schema.json
```

For VS Code's YAML extension, a workspace association can point `site.yaml` at
the site schema:

```json
{
  "yaml.schemas": {
    "./.schemas/margo-site.schema.json": ["site.yaml"]
  },
  "json.schemas": [
    {"fileMatch": ["**/*.policy.json"], "url": "./.schemas/margo-policy.schema.json"}
  ]
}
```

Attach the document schema to the frontmatter block through an editor that
supports Markdown frontmatter (or validate an extracted YAML object). The
schemas include `x-margo-*` annotations; strict JSON Schema validators must
register those annotation names or disable unknown-keyword checks. Generic
schema validation also cannot model Margo's duplicate-key check, byte limits,
or exact HTTPS-origin canonicalization: paths, queries, fragments, credentials,
wildcards, and noncanonical IP forms can satisfy a generic URI format but are
rejected by Margo. Run `margo check guide.md --policy policy.json` as the final
gate.

## Raw HTML

`rawHTML: sanitized` enables the closed `margo-html-v1` semantic profile. No
frontmatter declaration is needed or read. Margo parses the fragment, rejects
unknown elements and attributes, and serializes a fresh canonical fragment;
original HTML bytes are never passed through.

The profile includes common semantic text, heading, list, details, table, and
link elements. It excludes scripts, styles, images, iframes, event handlers,
classes, IDs, arbitrary `data-*`, arbitrary `aria-*`, namespaces, and unknown
attributes. Links may be relative or use `http`, `https`, `mailto`, or `tel`;
network-path URLs and other schemes fail.

Well-formed HTML comments are not raw HTML capability use. They are discarded
before policy enforcement and never appear in fragment, standalone, site,
deck, or PDF-input HTML. A malformed comment receives a positioned diagnostic;
a comment cannot hide adjacent real HTML.

## Natural iframe embeds

Authors use standard HTML, not a Margo fence:

```html
<iframe
  src="https://video.example.com/watch/123"
  title="Architecture overview"
  width="800"
  height="450">
</iframe>
```

Document attributes are closed to `src`, `title`, `width`, and `height`.
Unsupported attributes fail rather than pass through. `src` must be absolute
HTTPS without credentials and its canonical exact origin must appear in
`allowedOrigins`. Width and height are integers from 1 through 4096 and default
to 640 by 360. Titles are at most 256 UTF-8 bytes; a missing title produces an
accessibility warning.

Margo emits a new canonical element. It supplies `sandbox`,
`referrerpolicy="no-referrer"`, lazy loading, and an empty Permissions Policy
surface; authors cannot supply or widen them. The only v1 sandbox capabilities
are `allow-presentation` and `allow-scripts`. The profile never grants
same-origin, popups, forms, navigation, or downloads.

Each target is independent:

| Projection | Behavior |
| --- | --- |
| `interactive` | Canonical sandboxed iframe; allowed for HTML, site, and deck. |
| `static-link` | HTTPS link only; no iframe or remote subresource. |
| `deny` | Target rendering fails. |

Defaults are `deny` for HTML, site, and deck, and `static-link` for PDF when an
iframe policy exists. PDF cannot be interactive. A static link uses the title
or the normalized URL as a deterministic fallback.

The removed `trusted-embed` fence produces `source.trusted_embed_removed` with
a standard-iframe migration example. It has no renderer.

## Origin normalization and privacy

Allowed origins contain only scheme, host, and optional port. Runtime
normalization applies IDNA lookup, lowercase hostnames, IP-literal
normalization, removal of default port 443, and trailing-dot normalization.
Paths, queries, fragments, credentials, wildcards, backslashes, scoped IP
addresses, and noncanonical numeric-IP shorthand fail.

Interactive remote content changes privacy, availability, and offline
behavior. Prefer `static-link` or `deny` for self-contained artifacts. Margo
does not perform oEmbed discovery or any provider network lookup.

## Consumer security headers

The serving application still owns Content Security Policy and related HTTP
headers. For an interactive iframe, add `frame-src` only for the exact policy
origins while retaining existing script, style, image, font, media, and
connection rules. Do not widen `default-src`. Static-link and deny projections
need no `frame-src` widening.

## Document metadata ownership

Generic metadata stays at the frontmatter root. The only Margo-owned document
namespace is closed `margo`; it currently contains optional page size,
orientation, and per-side margin preferences for PDF documents and PDF decks.
An explicit zero margin requests full bleed on that side; an omitted side keeps
the target default. Security, brand, theme, table behavior, and Mermaid
configuration are not document metadata. See the generated
[document metadata reference](reference/document-metadata.md) and print the
matching bytes with `margo schema document`.

The universal preference order is:

```text
explicit CLI/API option -> document preference -> built-in default
```

Legacy `goshtoso` frontmatter fails with `frontmatter.goshtoso_removed` and
targeted migration guidance.
