# Deterministic PDF contrast lint

Margo's standalone HTML is the source for the PDF projection. A custom theme or
token override should be checked before human review, not discovered after a
PDF is exported. The built-in browser preflight renders the supplied complete
HTML in print media, applies both Goshtoso color-mode projections, and audits
visible text against WCAG AA contrast thresholds:

- 4.5:1 for normal text;
- 3:1 for large text;
- zero tolerance for a blocked network dependency. Local `file:` assets next to
  the supplied HTML remain offline and are allowed; remote URLs are blocked.

The same report audits print layout: protected blocks and table cells must stay
inside document width, table rows retain positive height and stay inside their
table, hidden or clipped overflow that would discard content is reported, and
root-level horizontal overflow fails the preflight.

The check uses the exact Node, npm, and Chromium receipt already provisioned by
the M0 browser harness. It does not use ambient Node, npm, Playwright, or a
network fallback.

## Run against a custom document

Provision and verify the browser harness first. Then run the contrast-only
preflight with absolute paths:

```bash
sh test/browser/run-playwright.sh \
  --check \
  --env-file "$PWD/test/browser/.cache/node-env.checked.sh" \
  --contrast-html "$PWD/output/html/custom.html" \
  --contrast-mode both \
  --contrast-format text \
  --contrast-only
```

Use `--contrast-mode light` or `--contrast-mode dark` to inspect one projection.
Use `--contrast-format json --contrast-output /absolute/path/report.json` for
machine-readable evidence. The JSON report contains the source byte count and
SHA-256, print-media mode results, each failing text node, and blocked resource
records. Exit status is:

- `0`: all requested modes pass and no resource was blocked;
- `1`: the document is not readable under the selected rule or requests a
  resource;
- `2`: the invocation or checked browser environment is invalid.

PowerShell uses the same checked receipt and report contract:

```powershell
pwsh -NoProfile -File .\test\browser\run-playwright.ps1 `
  -Check `
  -EnvironmentJson (Resolve-Path .\test\browser\.cache\node-env.checked.json) `
  -ContrastHtml (Resolve-Path .\output\html\custom.html) `
  -ContrastMode both `
  -ContrastFormat text `
  -ContrastOnly
```

## What the lint covers

The auditor walks visible text in the composed DOM, including headings, links,
stamps, table cells, blockquotes, code, Mermaid source disclosures, and footer
content. It ignores decorative or hidden content such as the standalone
watermark when it is marked `aria-hidden="true"`. Custom HTML must embed the
reviewed Goshtoso stylesheet and document CSS; external stylesheets, fonts,
images, scripts, and other remote requests are intentionally rejected by the
offline boundary. Local `file:` images can be used for a human-facing artifact
and are resolved from the supplied HTML path.

This is a deterministic preflight, not a replacement for semantic tests, PDF
text extraction, pagination review, or human inspection of diagrams and
spacing. New readability rules can be added to the versioned
`margo/contrast-lint/v1` report without changing the source artifact. Layout
findings appear under each mode as `layout.checked` and `layout.failures`.
