# Margo

<p align="center">
  <img src="assets/margo-mascot.png" alt="Margo, the pink Go gopher mascot, holding a rendered document in a Brazilian publishing atelier." width="480">
</p>

Margo will compile Markdown into Goshtoso-styled documents, standalone HTML,
PDFs, and static slide decks.

The repository includes a deterministic browser preflight for standalone HTML
before PDF review. See [contrast lint](docs/CONTRAST_LINT.md) to check custom
themes and styling in print media under both light and dark color modes.

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
