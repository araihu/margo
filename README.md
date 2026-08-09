# Margo

<p align="center">
  <img src="assets/margo-mascot.png" alt="Margo, the pink Go gopher mascot, holding a rendered document in a Brazilian publishing atelier." width="480">
</p>

Margo compiles Markdown into semantic Goshtoso-compatible HTML. The root module
supports a host-owned fragment and a generic complete HTML page from the same
immutable render result. Consumer applications own routing, publication
metadata, site identity, and deployment. PDF and static slide projections
remain separate modules.

The repository includes a deterministic browser preflight for standalone HTML
before PDF review. See [contrast lint](docs/CONTRAST_LINT.md) to check custom
themes and styling in print media under both light and dark color modes.

## HTML

Compile and render Markdown once, then project the result as a fragment owned
by the surrounding application:

```go
compiler := margo.New()
document, err := compiler.Compile(ctx, margo.Source{Name: "description.md", Content: source})
if err != nil {
	return err
}
rendered, err := compiler.Render(ctx, document)
if err != nil {
	return err
}
htmlResult, err := margo.RenderHTML(rendered)
if err != nil {
	return err
}
return htmlResult.Fragment().Render(ctx, writer)
```

`HTMLResult.Fragment()` contains one `article.margo-document`; it does not
own the document shell, theme selection, or color mode. A Manja-style host can
place it inside its existing `.manja-markdown` layout. The article inherits the
host's Goshtoso `data-theme` and `.dark` state without regeneration. Inspect
`htmlResult.Requirements()` when the host assembles its own head.

For a complete generic page, use the technical page contract. It does not
infer routes, canonical URLs, social metadata, or article semantics:

```go
page, err := margo.RenderHTMLPage(htmlResult, margo.HTMLPageInput{
	Theme:           margo.ThemeModern,
	ColorMode:       margo.ColorModeLight,
	DependencyMode: margo.HTMLDependenciesLocal,
})
```

Mount the handlers directly at their owning prefixes for local requirements:

```go
mux.Handle("/assets/", goshtosoassets.Handler())
mux.Handle("/margo-assets/", margo.HTMLAssetHandler())
mux.Handle("/charts/assets/", chartassets.Handler())
```

Do not strip those prefixes a second time. `HTMLAssetHandler` serves the
scoped Margo stylesheet and progressive table-sort runtime. Goshtoso and Charts
retain ownership of their own embedded bytes.

Applications can compose their own metadata and chrome through the generic
page seams. The components below belong to the consumer, not Margo:

```go
page, err := margo.RenderHTMLPage(htmlResult, margo.HTMLPageInput{
	Theme:           consumerTheme,
	ColorMode:       margo.ColorModeLight,
	DependencyMode: margo.HTMLDependenciesLocal,
	ThemeStylesheet: consumerStylesheet,
	Head:            consumerMetadata(htmlResult.Metadata()),
	Header:          consumerNavigation(),
	BeforeContent:   consumerArticleDetails(htmlResult.Metadata()),
	Footer:          consumerFooter(),
})
if err != nil {
	return err
}
return page.Render(ctx, writer)
```

Use `HTMLDependenciesLocal` with the three mounts above, or
`HTMLDependenciesInline` for a private/offline page with reviewed embedded
bytes. Both modes preserve requirement order. Custom theme CSS loads after
Goshtoso and Margo styles.

Markdown tables and charts are readable before JavaScript. Table header buttons
are progressive client controls. For charts in the current-root HTML
path, register
`charts.Extension(charts.WithExternalizedControlRuntime(true))`; this declares
one ordered runtime set and leaves static SVG plus accessible data tables in the
initial HTML. Provenance-marked chart component styles may remain inside the
trusted chart output; unowned styles and all fragment scripts are rejected.

Manja continues to own its documentation layout and routing. An araihu.com
consumer continues to own localization, verified route authority, custom theme,
brand chrome, and deployment. No external consumer is modified by this API.
Generated-HTML unit and browser gates are documented in
[HTML browser evidence](docs/testing/editorial-html.md). The generic HTML
contract does not impose public-web metadata on PDF consumers. PDF visual review
is deferred; this HTML slice does not claim PDF acceptance.

The reproducible [blog example](examples/blog/README.md) generates a landing
page and a public article with AVIF, WebP, JPEG, PNG, and GIF assets.

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
