# Product

## Register

brand

## Users

Engineers, reviewers, and documentation readers who need to understand one
Markdown source as readable HTML or PDF. Authors work in small feature-focused
Markdown files; reviewers also use one broad optimistic document to judge the
whole composition.

## Product Purpose

Margo compiles Markdown once into deterministic, Goshtoso-compatible editorial
HTML that can be embedded in a host page or composed into a complete
publication. PDF and static slide decks are later projections over that source.
Success means semantic source, metadata, dependencies, and policy behavior
remain inspectable while the generated artifact is comfortable to read.

## Brand Personality

Precise, quiet, technical. The document should feel like maintained project
documentation, with clear hierarchy and trustworthy component behavior.

## Anti-references

- Unstyled browser-default Markdown.
- A second visual system that duplicates or fights Goshtoso.
- Dashboard chrome, decorative cards, or marketing treatments around ordinary
  long-form content.
- Remote assets required for first paint or print.

## Design Principles

- Reuse Goshtoso CSS, themes, tokens, and components as the visual authority.
- Keep document-owned CSS limited to prose rhythm, pagination, print, and
  bounded brand adjustments.
- Preserve semantic HTML and make hierarchy visible without JavaScript.
- Let host applications own page theme, dark mode, routing, and surrounding
  information architecture while Margo owns scoped prose rhythm.
- Declare browser dependencies explicitly, deduplicate them, and retain static
  table/chart fallbacks in initial HTML.
- Keep small focused fixtures as the first visual debugging surface; use the
  broad fixture for composition and pagination regressions.
- Render offline deterministically from reviewed embedded assets.

## Current acceptance boundary

The editorial HTML slice is accepted through generated-HTML unit tests and
Chromium-family E2E journeys shaped like Manja documentation and an araihu.com
article. The recorded browser is evidence of the tested environment, not a
version policy for users. PDF generation and visual PDF correctness are
deferred to later human review and are not implied by HTML acceptance.

## Accessibility & Inclusion

Preserve semantic headings, lists, tables, links, code, and text alternatives.
Maintain keyboard-safe controls, readable line lengths, visible focus, AA text
contrast, and useful output when runtime behavior, color, or animation is
unavailable. Print must omit controls that have no paper meaning.
