---
title: completion
description: Generate a shell completion script for the installed Margo command surface.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# `margo completion`

## Purpose

`completion` generates a completion script for one supported shell. The shell
is a subcommand: `bash`, `zsh`, `fish`, or `powershell`.

## Input and output

The required positional input is one supported shell. Script bytes go to
stdout; command failures go to stderr. `--no-descriptions` produces a smaller
script without help text.

## Examples

Load completion for the current zsh session:

```sh
source <(margo completion zsh)
```

Generate scripts for distribution or inspection:

```sh
margo completion bash --no-descriptions > build/margo.bash
margo completion zsh --no-descriptions > build/_margo
margo completion fish --no-descriptions > build/margo.fish
margo completion powershell --no-descriptions > build/margo.ps1
```

## Failures and diagnostics

An unsupported shell or invalid option exits `1` and writes Cobra's diagnostic
to stderr. Use `margo completion SHELL --help` for that shell's supported
installation paths and loading instructions.

## Limitations and care

Generation does not edit shell startup files automatically. Redirect the output
to a location appropriate for the target shell, or source it explicitly for
the current session.
