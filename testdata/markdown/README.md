# Margo Markdown corpus

`margo-full-feature-set.md` is the optimistic human acceptance document. It
describes the complete composition we want Margo to render as the feature set
grows. It may mention extensions that are not wired in the current checkout.
Its purpose is visual and product-level review, not a claim that every future
extension is already implemented.

Edge cases enter this corpus before happy paths. Add the smallest focused slice
and a failing assertion first, record the expected fail-closed diagnostic or
layout behavior there, then mirror the case in the full feature set before
regenerating its HTML/PDF. This is the human-artifact equivalent of TDD: a bug
or boundary becomes benchmark input before its implementation is considered
complete.

The `slices/` directory is the executable development corpus. Each file stays
small and isolates one feature or one deliberate composition. Every renderer
change should regenerate HTML and PDF for the affected slice before the large
document is regenerated. Inspect the slice first, then use the full document to
catch layout, pagination, and composition regressions.

Suggested local loop:

```text
slice Markdown -> Margo standalone HTML -> human review -> PDF preview
                                      -> focused test and golden update
full feature set Markdown -> integration HTML/PDF review
```
