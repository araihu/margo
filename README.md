# Margo

Margo will compile Markdown into Goshtoso-styled documents, standalone HTML,
PDFs, and static slide decks.

This repository currently contains bootstrap boilerplate only. It does not yet
provide parsing, rendering, export, deck, chart, PDF, or CLI behavior.

## Modules

| Module | Purpose |
| --- | --- |
| `github.com/araihu/margo` | Core library and the `deck` package |
| `github.com/araihu/margo/charts` | Optional chart integration |
| `github.com/araihu/margo/pdf` | Optional PDF integration |
| `github.com/araihu/margo/cmd/margo` | Command-line application |

Each module is tested independently with `GOWORK=off`.

## Security

See [SECURITY.md](SECURITY.md) for private vulnerability reporting.

## License

Margo is licensed under the [MIT License](LICENSE).
