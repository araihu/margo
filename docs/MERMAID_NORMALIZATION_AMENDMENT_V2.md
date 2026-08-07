# Proposed Mermaid normalization amendment v2

Status: `PROPOSED`; not yet part of the accepted v0.0.1 design.

This amendment closes one contradiction between the accepted Mermaid contract
and the bytes produced by the pinned Mermaid 11.16.1 runtime. It does not
authorize implementation until the product owner explicitly accepts it.

## Problem

The accepted contract requires two things that cannot both hold for the pinned
positive corpus:

1. M4 must reject every attribute selector, at-rule, and `filter` declaration
   before M5.
2. Every pinned positive Mermaid fixture must normalize and validate.

The eight positive fixtures produce 444 style rules. Of those, 118 match the
detached SVG and 326 do not. The four sequence fixtures also produce:

- live `[id$="-arrowhead"] path`, `[id$="-crosshead"] path`, and
  `[id$="-sequencenumber"]` selectors;
- unreferenced `@keyframes dash` and
  `@keyframes edge-animation-frame` rules;
- `filter: none` on `.labelBox` in the conditional and style-heavy fixtures.

Rejecting these bytes rejects the required positive corpus. Admitting attribute
selectors, at-rules, or `filter` in the final validator would widen the security
profile. Neither outcome is acceptable.

## Decision proposed

Replace `margo-mermaid-svg-normalization/v1` with
`margo-mermaid-svg-normalization/v2`. Version 2 adds one closed reduction phase
inside M4, after CSS parsing and ID-map construction but before the current
selector normalization and M5 validation.

The reduction phase may perform only four operations recorded in the immutable
profile:

1. remove a selector branch that matches no element;
2. expand one of three exact sequence suffix selectors to a normalized ID
   selector;
3. remove one exact unreferenced keyframes rule;
4. remove one exact computed no-op `filter: none` declaration.

After reduction, the existing M5 contract remains unchanged: no attribute
selector, at-rule, or `filter` property may survive.

## Closed reduction profile

M1 adds a required `normalizationReductions` object to
`profiles/margo-mermaid-svg-v1.json`. It is part of the existing RFC 8785
profile preimage and therefore changes `ValidatorProfileFingerprint`.

The object has exactly these fields; unknown or missing fields fail:

```json
{
  "algorithm": "margo-mermaid-svg-normalization/v2",
  "cssTreeVersion": "3.1.0",
  "deadSelectorRules": [],
  "sequenceSelectorRewrites": [],
  "discardedAtRules": [],
  "discardedDeclarations": []
}
```

Every array is sorted by the bytewise UTF-8 order of its canonical RFC 8785
row. Duplicate rows fail profile verification.

### CSS node identity

M0's pinned css-tree 3.1.0 parses each stylesheet. Before hashing a selector or
declaration block, M4 clones its AST and constructs an execution-independent
reduction form:

1. the leading ID selector equal to `OriginalRootID` becomes the literal valid
   CSS ID `#margo-reduction-root`;
2. every descendant source ID is resolved through the already frozen descendant
   map and becomes `#margo-reduction-id-<eight-digit-document-order-ordinal>`;
3. same-SVG local fragment values use those same reduction IDs; and
4. any original ID that cannot be mapped fails instead of entering the hash.

`sourceCSS` is the UTF-8 result of `csstree.generate(node)` from that cloned
reduction AST. At-rules and declarations without ID references are generated
directly from their parsed AST. Node identity is:

```text
SHA-256("margo/mermaid-css-node/v1\n" || sourceCSS)
```

No runtime `OriginalRootID`, `NormalizedRootID`, raw substring, regular
expression, browser-normalized `cssText`, or computed style string can enter or
substitute for this identity. Two render instances of the same parsed CSS must
therefore produce the same row bytes.

### Dead selector rows

Each `deadSelectorRules` row is exactly:

```json
{"family":"sequence","selectorSHA256":"<lowercase hex>","declarationsSHA256":"<lowercase hex>"}
```

M4 may remove a branch only when:

- the complete row is present in the profile;
- `querySelectorAll` against the detached SVG returns zero elements;
- parsing and selector evaluation succeed within the resource limits; and
- the selector branch contains no external URL or unsupported reference site.

An unlisted dead branch fails with `mermaid.svg_css_reduction_unknown`. This
keeps output drift visible instead of treating any dead CSS as harmless.

### Sequence selector rewrite rows

`sequenceSelectorRewrites` contains exactly these three pattern IDs:

```json
[
  {"pattern":"sequence-arrowhead","suffix":"-arrowhead","tail":"path"},
  {"pattern":"sequence-crosshead","suffix":"-crosshead","tail":"path"},
  {"pattern":"sequence-number","suffix":"-sequencenumber","tail":""}
]
```

The source branch must already be anchored once by `OriginalRootID` and must
otherwise be exactly `[id$="<suffix>"] <tail>`. M4 requires exactly one carrier
ID and exactly one matched target. It rewrites the carrier through the existing
descendant ID map and emits an exact normalized ID selector. Zero matches,
multiple matches, a different suffix/operator/tail, or a source ID absent from
the map fails with `mermaid.svg_css_sequence_selector_invalid`.

### Discarded at-rule rows

Each `discardedAtRules` row is exactly:

```json
{"family":"sequence","name":"dash","nodeSHA256":"<lowercase hex>"}
```

The reviewed v2 profile may contain only `dash` and
`edge-animation-frame`. M4 removes a listed keyframes rule only after the
remaining parsed declarations contain no reference to its name through either
`animation-name` or the `animation` shorthand. A reference, an unlisted digest,
or any other at-rule fails with `mermaid.svg_css_at_rule_forbidden`.

### Discarded declaration rows

Each `discardedDeclarations` row is exactly:

```json
{"family":"sequence","selector":".labelBox","property":"filter","value":"none","nodeSHA256":"<lowercase hex>"}
```

M4 may remove the declaration only when the complete row is present, every
matched element computes `filter` to `none` before removal, and every matched
element still computes `filter` to `none` after removal. Zero matches or any
different before/after value fails with `mermaid.svg_css_noop_unproven`.

## Required order

For each detached SVG, M4 performs these steps in order:

1. parse SVG and every CSS node with the locked parsers;
2. verify runtime, profile, family, and resource identities;
3. build the root and descendant ID maps without mutation leakage;
4. apply only profile-listed dead-branch reductions;
5. apply only the three profile-listed sequence rewrites;
6. remove only profile-listed, unreferenced keyframes;
7. remove only profile-listed, computed no-op declarations;
8. run the existing v1 ID/reference and selector normalization;
9. canonically serialize and reparse;
10. run M5 unchanged against the closed v1 SVG/CSS grammar.

Any failure returns no accepted SVG bytes and no insertion occurs.

## Required tests

The amendment is not implemented until RED fixtures cover:

- all eight positive fixtures;
- each of the three sequence rewrites;
- zero and multiple carrier matches;
- wrong suffix, operator, tail, or unlisted rewrite pattern;
- listed and unlisted dead selector rows;
- listed, referenced, unlisted, and renamed keyframes;
- proven and unproven `filter: none` removal;
- unknown at-rules, attribute selectors, and `filter` after reduction;
- profile fingerprint mismatch and css-tree version mismatch;
- zero insertion and zero non-local requests for every failure.

A mutation that removes any reduction row from the profile must fail the
corresponding positive fixture. A mutation that adds an unreviewed row must fail
the profile golden and fingerprint checks.

## Ownership and sequencing

- M1 owns the human-reviewed profile correction and new fingerprint.
- M4 owns the reduction implementation and browser/Go normalization fixtures.
- M5 owns only the unchanged post-reduction validator and negative corpus.
- M6 and M7 remain blocked until M1, M4, and M5 receive fresh acceptance.
- I2 and every PDF/deck/CLI task remain blocked until M7 and the other I2
  predecessors are accepted.

No automatic corpus scanner may write the profile. The audit may propose rows;
the committed rows and their fingerprints require human review.

## Evidence for the decision

The proposal is derived from these preserved, read-only audit artifacts:

- render audit SHA-256
  `d051d53d2dd7a47e48ff2956d43dd915f933b94b08605e3f219f515d0bd227c1`;
- live CSS audit SHA-256
  `55c4e5a85b528a159da8b68a2265e31f54a2828564b058571d29b68428859294`;
- bounded correction proof SHA-256
  `adc80c85cc487328f70d8193fbcabd8ce73234439d1bf63b3cff3f9b5552d686`.

The exact proposed `normalizationReductions` bytes are preserved at
`docs/proposals/MERMAID_NORMALIZATION_REDUCTIONS_V2.proposed.json`, SHA-256
`cd703d58c45b3e7f0ae5ab23f4d4d7ee023c419420925674855dcd8785790826`.
They contain 120 unique dead-selector rows covering 427 dead selector branches
across all 326 observed dead style rules, three sequence rewrites, four
family/keyframe rows covering 16 at-rule occurrences, and one no-op declaration
row covering two rule occurrences and three matched elements.

The correction proof shows one carrier and one target for each listed sequence
rewrite, zero live animation names for the two keyframes, preserved computed
`filter: none` after declaration removal, and zero non-local request.
Generating the proposal again after replacing every fixture's
`OriginalRootID` with a different valid ID produced byte-identical output and
the same SHA-256. The proposed row identity is therefore independent of render
instance naming.

## Approval gate

Implementation requires the product owner to accept this exact statement:

> Approve `margo-mermaid-svg-normalization/v2` as specified in
> `docs/MERMAID_NORMALIZATION_AMENDMENT_V2.md`, including the human-reviewed
> reduction profile and M1 -> M4 -> M5 replay.

Until that approval exists, this file is evidence of a proposed correction,
not authority to change the accepted design or runtime.
