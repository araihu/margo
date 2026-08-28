---
title: doctor
description: Check the PDF renderers available in the current Margo environment.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# `margo doctor`

## Purpose

`doctor` checks renderer candidates without rendering a document.

## Input and output

The command accepts no input. Text or JSON output goes to stdout; command
failures go to stderr. The JSON envelope is described from the same versioned
schema that ships with Margo, so this page does not duplicate the output
contract:

```jsonschema ref=margo://schema/v1/output/doctor-report.json
```

See the [doctor report schema reference](../../schemas/doctor-report/index.md)
for the full tree and its other consumers.

## Examples

```sh
margo doctor
margo doctor --diagnostics json > build/margo-doctor.json
```

## Failures and diagnostics

Read `available`, not merely the presence of a row. A missing known path can
appear with `pdf.engine_path_invalid`; the native slot can appear with
`pdf.native.compiled_out`. Invalid flags or diagnostic formats exit `1` and
write the command failure to stderr.

`doctor` can exit `0` while listing only unavailable candidates, because the
report itself was produced successfully.

## Limitations and care

Discovery follows automatic PDF selection: explicit path where applicable,
`MARGO_CHROMIUM_PATH`, `PATH` and known locations in platform order, then the
native slot. `doctor` does not download, install, or launch a document render.
It cannot guarantee that a later document-specific export will succeed.
