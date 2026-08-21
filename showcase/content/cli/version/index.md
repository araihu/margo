---
title: version
description: Inspect the release identity and compiled capabilities of a Margo binary.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# `margo version`

## Purpose

`margo version` prints release identity and capabilities compiled into the
binary. `margo --version` is an exact alias. Neither form searches for a browser
or probes an external engine.

## Input and output

The command accepts no input. Its text report goes to stdout and includes the
Margo version, module, commit, Go version, platform, compiled engines, and a
reminder to run `doctor` for external discovery. Argument errors go to stderr.

## Examples

```sh
margo version > build/margo-version.txt
cat build/margo-version.txt
```

```sh
margo --version
```

## Failures and diagnostics

Extra positional arguments use Cobra's text error and exit `1`. A successful
report exits `0`; there are no command-specific numeric exit codes.

## Limitations and care

A source build can report `dev`, `unknown`, and `compiled engines none`; those
values describe build metadata, not whether Chromium is installed. Use
`margo doctor` to inspect external engine candidates.
