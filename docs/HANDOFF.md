# Margo v0.0.1 handoff

Status: implementation root and human HTML/PDF artifacts are locally
verifiable. Optional modules and formal integration remain blocked by missing
external authority receipts. This handoff is a candidate for `main`; it is not
a release, tag, publication, or deployment record.

## Git identity

- Repository: `https://github.com/araihu/margo`.
- Source branch: `impl/v0.0.1-core`.
- Functional source checkpoint before handoff:
  `b0ad36a7a6c706a419dbd4f9797fd3be3355ecbe`;
  tree `1463298b8ecd9f791b5280dc0723dfcdcaa249d9`.
- Handoff commit: `7a9eb0dc661ac3c6d7d4ea74a34caa3755a8f12b`;
  tree `c6466f06a59d9937a6ae2c3c6c10eeb43ce0dffb`.
- `main` local foi fast-forward para o handoff. `origin/main` permanece no
  bootstrap `608c0f41243b9adc7d8a4a41d2d13bf6d8b363b0`: push direto recusado
  pela proteção, que exige Pull Request e `Multi-module CI`.
- Worktree: `/private/tmp/margo-v001-implementation`.
- Intentional untracked path: `test/browser/.cache/`; do not stage it.
- Accepted design and plan identities remain unchanged; see `GOAL.md`.

## Delivered scope

- Root C0-C8: compiler, parser, policy, immutable render plan, semantic HTML,
  table/code adapters, Mermaid runtime, sinks, composition and tests.
- Standalone renderer: embedded Goshtoso CSS, theme `modern`, light/dark mode,
  TOC, brand/logo, watermark, stamps and offline assets.
- Browser/PDF evidence: checked Node/npm/Chromium runner, Mermaid rendering,
  contrast/layout lint, protected block pagination and table continuity.
- CI: all module commands use `GOWORK=off` and
  `GOFLAGS=-mod=readonly`; `go mod verify` and `go.mod`/`go.sum` hash guards
  replace mutating `go mod tidy`.

## Artifact receipts

| Artifact | Bytes | SHA-256 |
| --- | ---: | --- |
| `output/html/margo-v0.0.1-optimistic.html` | 343588 | `b830a04167d509368d36762bd9da181ce7213eb623f6f99bb51a2389f4b00a02` |
| `output/html/margo-v0.0.1-optimistic-dark.html` | 343591 | `6d14c24b8008c720706d35c7bcfcf3ebcac714534a7a216d44ee33f4d2de5c27` |
| `output/pdf/margo-v0.0.1-optimistic.pdf` | 443181 | `67bcf07184315e62e629f9aac7c3d830dfa2297f8bfb30b8568f4457a82f179a` |
| `output/pdf/margo-v0.0.1-optimistic-dark.pdf` | 455322 | `01cfa9839ad62ff793560e8dc35207de31ccd9c9c4a0ca9b9aeae3a5ff3e4e35` |

HTML evidence files use `margo/optimistic-browser-evidence/v4`. PDF evidence
files use `margo/pdf-print/v2`. Current PDF contract: A4, 20 pages per mode,
three tables with row counts `[5,5,21]`, six break markers, zero blocked
requests and zero console errors. Final Mermaid rejection rows remain present
across the protected continuation.

## Gates passed

```text
GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1
GOWORK=off GOFLAGS=-mod=readonly go vet ./...
node --check test/browser/print-pdf.mjs
node --check test/browser/generate-evidence.mjs
node --test test/browser/generate-evidence.test.mjs
test/browser/run-playwright.sh --check --env-file test/browser/.cache/node-env.checked.sh
```

M0 local candidate: Node `v26.5.0`, npm `11.17.0`, Chromium revision `1169`
(`136.0.7103.25`), `49 passed`, `network=0`. This is not independent M0
acceptance.

## External blockers

Do not create substitutes for these inputs:

- I1a/I1b: complete proxy, source/ZIP provenance and immutable handoff receipt.
- I2/I3: verified identities for root/charts/pdf/cmd modules and platform pins.
- H1-H6, P1-P7, D1-D5, O5: cannot start under the accepted readonly contract
  until those identities exist; optional modules are intentionally skeletons.
- T6: user moved `release/table-handoff.json` to the end of the backlog. No
  file, tag, release, or pseudo-version may be invented here.
- Independent M0 review: local green result remains candidate only.

`integration/root-module-transfer.v1.json` is the existing C0-to-C5 transfer;
SHA-256 `e1efcfbf57ba6a3dfa1adfa524a484be112248fcec667a3e041fdf4cea0fb1f1`.
It does not authorize I1a/I1b/I2/I3 or T6.

## Next operator sequence

1. Verify this handoff commit and fast-forward `main` only.
2. Keep `.cache` untracked; do not stage generated browser dependencies.
3. Obtain external I1a/I1b/I2/I3 receipts and independent M0 review.
4. Re-run readonly gates and begin optional modules only after ownership and
   provenance are recorded.
5. Defer T6 until explicit release authority exists.

Historical detail is available through `git log -- GOAL.md`; accepted plan
snapshot is `/private/tmp/margo-v001-plan-integration-r17`.
