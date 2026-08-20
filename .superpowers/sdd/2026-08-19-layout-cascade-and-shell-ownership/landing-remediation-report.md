# Landing remediation report

Base commit: `8f3bf16`

## Result

Implemented the approved Tour landing composition without changing the typed
layout schema or the plain-HTML boundary. Landing pages keep one generated
`article.margo-document`, remain outside the documentation shell, and now gain
semantic hero and section grouping only at the landing renderer boundary.

## Changes

- `site/config_build.go`
  - transforms the rendered landing fragment into one `header` hero and one
    `section` per H2;
  - moves the first leading media block into the hero visual region and keeps
    all other leading nodes in the copy region;
  - carries H2 IDs to `aria-labelledby` and rejects fragments that are not one
    `article.margo-document` root;
  - styles generic landing classes, with one-column narrow/intermediate flow,
    a two-column hero from 900px, 44px-safe action links, bounded reading
    measure, locally wider media, and no mascot-alt selector;
  - removes obsolete landing selectors from the shared legacy stylesheet.
- `site/config_build_test.go`
  - covers semantic hero/visual/section output, one article/H1, public route
    rewriting, shell exclusion, text-only/no-H2 fallback, malformed roots, and
    landing dependency isolation.
- `site/layout_browser_test.go`
  - covers 390, 719, 720, 900, 1493, and 1775px;
  - asserts hero column count and DOM/visual order, two first-viewport actions
    at least 44px tall, intentional 4:3 media, reading measure, local media
    widening, one article, and no horizontal overflow;
  - retains the existing docs-shell mobile interaction coverage.
- `showcase/content/index.md`
  - keeps ordinary Markdown and the approved mascot, Mermaid, Goshtoso chart,
    output tour, trust, fit, CLI, and Module markers;
  - promotes CLI and Module choices into descriptive Markdown list links in the
    hero and final decision section;
  - tightens copy and combines Good fit / Not a fit under one decision group.

No new schema field, dependency, runtime, breakpoint patch, `landingshell`, or
landing-specific asset was added. Article, docs, and legacy shared styles do not
receive landing classes or selectors.

## TDD evidence

RED:

```text
GOWORK=off go test ./site -run 'TestBuildConfigLandingGroupsMarkdownIntoSemanticComposition|TestBuildConfigLandingSupportsTextOnlyMarkdownWithoutSections' -count=1
```

Both tests failed against the original raw article binding. The decisive
failure was the missing `<header class="margo-landing-hero">`; the text-only
case likewise lacked the semantic copy wrapper.

GREEN:

```text
GOWORK=off go test ./site -run 'TestBuildConfigLandingGroupsMarkdownIntoSemanticComposition|TestBuildConfigLandingSupportsTextOnlyMarkdownWithoutSections' -count=1
ok github.com/araihu/margo/site
```

The first browser run then exposed an over-strict numeric test ceiling for the
existing `75ch` token at 900px. The assertion was corrected to test the actual
65–75ch contract and local media widening; production CSS was unchanged for
that correction. The responsive suite then passed.

## Gates

```text
GOWORK=off go test ./site -run 'Test(BuildConfig.*Landing|LandingLayout|LayoutBrowserTour|LayoutBrowserDocsShell)' -count=1
ok github.com/araihu/margo/site

GOWORK=off go test ./site -run TestBuildConfiguredShowcasePublicationContract -count=1
ok github.com/araihu/margo/site

GOWORK=off go test ./site -count=1
ok github.com/araihu/margo/site

GOWORK=off go vet ./site
PASS

git diff --check
PASS
```

Chromium browser tests ran rather than skipping.

## Concerns and boundaries

- Browser geometry uses the repository's established generated-site fixture;
  the real showcase publication contract separately compiles the full Mermaid
  and Goshtoso chart content.
- No screenshot-only aesthetic judgment or screen-reader session was added;
  semantic landmarks, keyboard targets, focus styles, DOM order, and overflow
  are automated.
- The existing running server was not touched or restarted. No push, merge,
  tag, release, publication, or deployment was performed.
- All pre-existing `.impeccable/critique/*` files and unrelated worktree state
  were preserved and excluded from the implementation commit.
