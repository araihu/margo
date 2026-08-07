# Margo Markdown corpus

`margo-full-feature-set.md` is the optimistic human acceptance document. It
describes the complete composition we want Margo to render as the feature set
grows. It may mention extensions that are not wired in the current checkout.
Its purpose is visual and product-level review, not a claim that every future
extension is already implemented.

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
