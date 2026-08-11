# Host policy and trusted embeds

Margo treats the document and the host as different authorities. Markdown can
request a capability, but only the application or CLI operator can grant it.
Without an explicit host policy, raw HTML is denied and `trusted-embed` fences
remain ordinary code blocks.

## CLI policy file

`margo check`, `html`, `pdf`, `site`, and `deck` accept
`--policy PATH`. The path must name a JSON file; `-` is rejected so document
stdin cannot also become policy authority. Unknown fields, duplicate
capabilities, non-HTTPS origins, origins containing paths, unsupported values,
multiple JSON documents, and files larger than 64 KiB fail closed.

Select the policy file in deployment or application configuration. Do not
automatically trust a policy file merely because it is beside untrusted
Markdown.

```json
{
  "schemaVersion": "margo-policy/v1",
  "rawHTML": "deny",
  "outputBytes": 67108864,
  "trustedEmbeds": {
    "allowedKinds": ["iframe"],
    "allowedOrigins": [
      "https://media.example.com",
      "https://video.example.com"
    ],
    "iframeSandbox": ["allow-presentation"],
    "referrerPolicy": "no-referrer",
    "projections": {
      "html": "interactive",
      "pdf": "static-link",
      "site": "interactive",
      "deck": "deny"
    }
  }
}
```

Omitted `rawHTML` defaults to `deny`, omitted `outputBytes` defaults to the
64 MiB maximum, and every omitted target projection defaults to `deny`.
Non-deny projections require at least one allowed kind and origin.

The complete normalized CLI policy is hashed as a `sha256:...` digest.
`margo check INPUT --diagnostics json` reports that identity. A site JSON report
and its `margo-manifest.json` record the same value, binding the published
output to the operator policy. This whole-policy digest is distinct from a
library extension's unprefixed
`ExtensionRegistration.Identity.ConfigurationHash`, which binds that
extension's normalized configuration into compiler and artifact fingerprints.

## Resource limits

The CLI bounds each Markdown input read at 16 MiB before parsing or compilation;
`site` applies that read limit independently to every discovered Markdown
file. The Go library's `Compiler.Compile` enforces the same ceiling after source
normalization and Markdown parsing. The library's read-only `margo.Check`
applies the ceiling after parsing when the caller supplies `WithCheckPolicy`;
without that option, `Check` does not enforce the 16 MiB policy ceiling.

In CLI JSON, omitted `outputBytes` and explicit `0` both select the 64 MiB
default; a nonzero value must be between 1 byte and 64 MiB. The resulting limit
bounds the semantic document HTML produced by each renderer invocation, before
a target adds a standalone HTML shell or packages that rendering as PDF or site
artifacts. A deck compiles and renders each slide separately, so the limit
applies per slide rather than to their aggregate semantic HTML. It is not a
limit on the final standalone HTML file, PDF file, deck file, or aggregate site
size.

Each trusted-embed request additionally limits its URL to 4096 UTF-8 bytes,
its accessible title to 256 UTF-8 bytes, and each dimension to the inclusive
range 1 through 4096.

## Target projections

Each target independently chooses one of three projections:

| Projection | Result |
| --- | --- |
| `interactive` | Emit a sandboxed iframe. |
| `static-link` | Emit only a no-referrer HTTPS link using the required accessible title. |
| `deny` | Reject the typed embed for that target. |

A typical public-site policy uses `interactive` for `html` and `site`,
`static-link` for `pdf`, and `deny` or `static-link` for `deck`. Static-link is
the explicit degradation path; Margo never silently inserts interactive
content into a target configured otherwise.

`iframe` and `video` are the only kinds in `margo-policy/v1`. Interactive v1
projections accept only `iframe`: native video elements cannot enforce the
required no-referrer policy. A policy that allows `video` must therefore use
only `static-link` or `deny` projections. Origins match exactly after IDNA
domain conversion, IP-literal normalization, lowercasing the HTTPS host,
removing the default `:443` port, and normalizing a trailing slash. Wildcards
and noncanonical numeric IP shorthand are rejected rather than interpreted as
browser-specific host syntax.
Paths, wildcards, credentials, queries, and fragments are not valid allowed
origins. Document URLs may contain a path, query, and fragment, but their exact
origin must be present in the allowlist. Every request also needs a nonempty
title. Omitted dimensions default to 640 by 360; the exact request limits are
listed under [Resource limits](#resource-limits).

The iframe sandbox starts empty. The only v1 tokens a host can add are
`allow-presentation` and `allow-scripts`. Enabling scripts does not add
`allow-same-origin`, popups, forms, navigation, or downloads.

## Document request

An authorized document uses a typed fence:

````markdown
```trusted-embed
kind: iframe
url: https://video.example.com/watch/123
title: Architecture overview
width: 800
height: 450
```
````

Only `kind`, `url`, `title`, `width`, and `height` are accepted. Unknown YAML
fields and additional YAML documents are rejected. Values are validated before
rendering and escaped when written to HTML. There is no arbitrary HTML payload,
generic `unsafe` mode, or `--allow-unsafe-html` flag.

## Sanitized raw HTML

Raw HTML is a separate, deliberately narrow two-key handshake. The host policy
must set `"rawHTML": "sanitized"`, and the document must declare:

```yaml
---
goshtoso:
  security:
    rawHTML: sanitized
---
```

Margo parses the fragment using HTML5 parsing rules, including the parser's
normal error recovery, and validates the resulting tree against the closed,
versioned `margo-html-v1` profile. A profile-invalid tree is rejected. On
success, Margo emits the accepted original source bytes rather than rewriting
or sanitizing them. The complete v1 profile is:

- Elements: `a`, `abbr`, `b`, `blockquote`, `br`, `cite`, `code`, `dd`, `del`,
  `details`, `dfn`, `dl`, `dt`, `em`, `h1` through `h6`, `hr`, `i`, `kbd`,
  `li`, `mark`, `ol`, `p`, `pre`, `q`, `s`, `samp`, `small`, `span`, `strong`,
  `sub`, `summary`, `sup`, `table`, `tbody`, `td`, `tfoot`, `th`, `thead`,
  `tr`, `u`, `ul`, and `var`.
- Global attributes: `title`, `lang`, and `dir`; `dir` is exactly `ltr`, `rtl`,
  or `auto`.
- Element attributes: `a[href]`; `details[open]`; `ol[start,reversed,type]`;
  `li[value]`; and `td`/`th` with `abbr`, `colspan`, `headers`, `rowspan`, or
  `scope`.
- `start`, `value`, `colspan`, and `rowspan` are positive integers. Boolean
  `open` and `reversed` values are empty or repeat the attribute name. Ordered
  list `type` is one of `1`, `a`, `A`, `i`, or `I`; table-cell `scope` is one
  of `row`, `col`, `rowgroup`, or `colgroup`; `headers` cannot be blank.
- `href` may be a relative reference or use `http`, `https`, `mailto`, or
  `tel`. Network-path references such as `//example.com` and every other URL
  scheme are rejected.
- Text is allowed. Comments, namespaces, and namespaced or duplicate attributes
  are rejected. At the fragment level, Margo rejects NUL, U+0001 through
  U+0008, U+000B through U+000C, U+000E through U+001F, and U+007F; TAB, LF,
  and CR may occur in text. Attribute values reject every Unicode control
  character. Any element or attribute not listed above is rejected; notably
  this excludes `img`, `class`, `id`, `style`, event handlers, and arbitrary
  `data-*` or `aria-*` attributes.

This path is for those allowlisted semantic fragments; use `trusted-embed` for
remote media. Neither mechanism enables scripts, event attributes, or
arbitrary iframe HTML.

## Consumer security headers

Interactive output is only one layer of the consumer's security boundary. The
HTTP application serving the page still owns Content Security Policy and
related headers. For an interactive v1 iframe, its CSP should grant `frame-src`
only for the exact iframe origins selected in the Margo policy, while retaining
the application's existing script, style, image, font, media, and connection
rules. A v1 `video` request can produce only a static link or a denial, so it
loads no media subresource and requires no `media-src` widening. Do not widen
`default-src` to make an embed work.

Remote media also changes privacy, availability, and offline behavior. Use
`static-link` or `deny` for self-contained/offline artifacts, and review the
remote provider's cookies, tracking, and retention rules before adding its
origin.
