# Tour layout remediation report

## Scope

First visual-review remediation loop for the configured Margo tour profile. The
live `127.0.0.1:8080` page and supplied review evidence were inspected before
editing. The old landing frame exposed the inherited navigation track, the
brand/search row could collide, and the showcase hook was not emitted.

## Changes

- `site/config_build.go`: landing now explicitly uses a one-column
  `top-nav / main-content / footer` grid; docs keeps its three-column frame;
  profile navbar sizing prevents brand/search/Repository collisions; the
  showcase wrapper and compact 4:3 mascot presentation are targeted through
  public hooks; the profile interaction asset is staged.
- `site/site_navigation.go` and `site/profile_interactions.go`: profile search
  markup now emits combobox/listbox relationships, stable option IDs and
  selection state, live result/no-result status, a clear action, and delayed
  Escape/close focus restoration to the invoking trigger. Goshtoso remains the
  owner of search rendering and behavior.
- `site/build.go`: local image assets receive intrinsic dimensions when the
  decoder can read them, preserving the offline fallback.
- `showcase/content/index.md`: concise CLI/module primary and secondary routes
  were added near the proposition and after fit guidance; technical claims are
  unchanged.
- `site/layout_profile_browser_test.go`: added emitted-DOM/computed-geometry
  coverage for landing/header/hero/CTA, docs grid templates, and search ARIA,
  active result, announcements, clear, and focus return. Existing legacy and
  light/dark 390/1440 matrix coverage remains.

## Visual evidence

Chromium geometry checks passed for landing at 390, 719, 720, 900, 1280, and
1775px. Every landing sample emitted one grid track with the three named areas;
the 390px search measured 44px and desktop search stayed clear of Repository.
The mascot measured 326x244.5px at 390px and 384x288px at desktop widths,
with computed `aspect-ratio: 4 / 3` and `object-fit: cover`. CTA links had
non-zero geometry in the first viewport at every sample. Docs measured one
track below 720px and three rendered tracks from 720px through 1775px.

The fresh post-edit fixture server was exercised on port 8092 because port 8080
was already owned by the shared live review server.

## Verification

- `go test ./site`
- `go test ./...`
- `go test -race ./site`
- `go vet ./site`
- `git diff --check`
- `TestLayoutProfileBrowser`: light/dark, mobile/desktop, Tour/Module/CLI,
  including the existing 390/1440 matrix and navigation checks.
- `TestLandingProfileVisualGeometry`, `TestDocsProfileGridTemplate`, and
  `TestProfileSearchSemanticsAndFocusReturn` pass; the search test was repeated
  ten times after hardening close timing.

## Detector

Ran once after UI edits:

```text
node /Users/guilhermecastro/.agents/skills/impeccable/scripts/detect.mjs --json site/config_build.go site/site_navigation.go site/profile_interactions.go showcase/content/index.md
```

Result: no blocking findings. Advisory `design-system-font-size` findings
remain in `site/config_build.go` for the existing/documented showcase type
scale (`0.875rem`, `2.25rem`, `4.5rem`, `1.15rem`, `1.5rem`).

## Remaining caveats

The review used Chromium geometry and semantic assertions rather than a manual
screen-reader session. The shared 8080 process was not restarted or mutated.
