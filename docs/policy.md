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

The policy is normalized and hashed as `sha256:...`. `check --diagnostics json`
reports that identity. A site JSON report and its `margo-manifest.json` record
the same value, binding the published output to the operator policy. Library
compiler and artifact fingerprints bind the normalized extension
configuration directly.

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
origin must be present in the allowlist. Every request also needs a URL of at
most 4096 UTF-8 bytes, a nonempty title, and dimensions between 1 and 4096;
defaults are 640 by 360.

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

Margo then validates the fragment against the versioned `margo-html-v1`
allowlist. It rejects invalid markup instead of rewriting it. This path is for
allowlisted semantic fragments; use `trusted-embed` for remote media. Neither
mechanism enables scripts, event attributes, or arbitrary iframe HTML.

## Consumer security headers

Interactive output is only one layer of the consumer's security boundary. The
HTTP application serving the page still owns Content Security Policy and
related headers. Its CSP should grant `frame-src` only for iframe origins and
`media-src` only for video origins selected in the Margo policy, while retaining
the application's existing script, style, image, font, and connection rules.
Do not widen `default-src` to make an embed work.

Remote media also changes privacy, availability, and offline behavior. Use
`static-link` or `deny` for self-contained/offline artifacts, and review the
remote provider's cookies, tracking, and retention rules before adding its
origin.
