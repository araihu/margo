---
title: CLI workflows
description: Drive the same module through a small command surface with explicit outputs.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# CLI workflows

The `margo` command is the operational face of the root Go module. Commands
write artifacts and reports to stdout, send diagnostics to stderr, and keep
destructive output replacement explicit.

## The core loop

```sh
margo check input.md
margo html input.md --output input.html
margo pdf input.md --output input.pdf
# Experimental deck projection.
margo deck input.md --format html --output input-deck.html
margo site ./docs --output-dir ./dist
```

## Command map

| Command | Purpose |
| --- | --- |
| `check` | Validate Markdown compatibility without rendering |
| `html` | Render one standalone HTML page |
| `site` | Build linked HTML pages from a directory |
| `pdf` | Export a document PDF |
| `deck` | Experimental HTML or PDF presentation projection |
| `doctor` | Report PDF engine candidates and reasons |
| `schema` | Emit the embedded policy or document schema |
| `version` | Report release and compiled capability information |

## Explicit output boundaries

`html` defaults to stdout. `deck` defaults to experimental HTML output and
stdout. PDF commands require an explicit output path or `-` for binary stdout.
Existing files are protected unless `--force` is present, and a site destination
must not already exist.

That small amount of friction is intentional: the command tells you which
artifact it is about to write and gives automation a stable place to inspect
the result.
