# Product

## Register

brand

## Users

Engineers, reviewers, and documentation readers who need to understand one
Markdown source as readable HTML or PDF. Authors work in small feature-focused
Markdown files; reviewers also use one broad optimistic document to judge the
whole composition.

## Product Purpose

Margo compiles Markdown into deterministic Goshtoso-styled documents,
standalone HTML, PDF, and static slide decks. Success means semantic source and
policy behavior remain inspectable while the generated artifact is comfortable
to read without opening its HTML.

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
- Keep small focused fixtures as the first visual debugging surface; use the
  broad fixture for composition and pagination regressions.
- Render offline deterministically from reviewed embedded assets.

## Accessibility & Inclusion

Preserve semantic headings, lists, tables, links, code, and text alternatives.
Maintain keyboard-safe controls, readable line lengths, visible focus, AA text
contrast, and useful output when runtime behavior, color, or animation is
unavailable. Print must omit controls that have no paper meaning.
