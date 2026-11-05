# Security Policy

Margo is early-stage software. Responsible disclosure is appreciated.

## Supported versions

The current supported release line is
[`v0.0.17`](https://github.com/araihu/margo/releases/tag/v0.0.17). Upgrade to
the latest release before reporting a vulnerability. Development snapshots on
`main` can contain unreleased changes and are not a substitute for a supported
release; fixes are applied to `main` first and backported to the supported
release when appropriate.

## Reporting a vulnerability

Do not open a public issue for security problems.

Report privately through GitHub's
[Report a vulnerability](https://github.com/araihu/margo/security/advisories/new)
page or email **security@araihu.dev**.

Please include:

- the affected module and commit;
- impact and expected security boundary;
- minimal reproduction steps or proof of concept; and
- suggested remediation, if available.

Remove credentials, private URLs, and sensitive content from reports sent
through channels you do not fully trust.

## Math rendering

The opt-in math capability runs bundled KaTeX with trust-sensitive commands
disabled, strict expression handling, bounded source/output sizes and macro
expansion, and fresh macro state for each expression. Enabling unsafe HTML
does not relax the math profile. Source fallbacks and exception messages are
inserted as text. Margo's parser identifies math; the engine does not scan
arbitrary host HTML for delimiters.

KaTeX assets and fonts have pinned acquisition hashes in `muamba.yaml`.
Generated self-contained CSS embeds the locked WOFF2 files. Updating the engine
requires regenerating the asset registry/CSS and rerunning browser, font-lock,
and PDF checks. HTML builds perform Go preflight; expression validation happens
in the browser. PDF export requires successful runtime and font evidence.

## What to expect

- Acknowledgement within five business days.
- Initial impact and severity assessment after reproduction.
- Coordinated disclosure and credit, unless anonymity is preferred, after a
  fix is available.
