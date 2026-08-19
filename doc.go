// Package margo compiles Markdown into immutable semantic documents and
// projects them to HTML.
//
// The shortest library path compiles one source, renders its semantic content,
// then places that content in a standalone HTML document:
//
//	compiler := margo.New()
//	document, err := compiler.Compile(ctx, margo.Source{
//		Name:    "guide.md",
//		Content: markdown,
//	})
//	if err != nil {
//		return err
//	}
//	rendered, err := compiler.Render(ctx, document)
//	if err != nil {
//		return err
//	}
//	page, err := margo.RenderStandalone(rendered)
//	if err != nil {
//		return err
//	}
//	return page.Render(ctx, output)
//
// A Compiler freezes its options at construction and supports concurrent
// Compile and Render calls. A compiled Document remains bound to the Compiler
// configuration that created it. Pass WithExtension to New to register
// optional integrations such as charts.
//
// Check performs read-only compatibility analysis without rendering. Host
// applications own capability policy through WithHostPolicy and
// WithCheckPolicy; document metadata cannot grant capabilities. Raw HTML is
// denied by default.
//
// RenderHTML exposes a semantic fragment and its dependency requirements.
// RenderHTMLPage composes that result into a host-owned page, while
// RenderStandalone creates Margo's self-contained document shell. Host-owned
// static sites can compose RenderHTML output; PDF and deck workflows reuse the
// same compilation and runtime contracts.
//
// Package margo does not provide a production HTTP server. The margo serve CLI
// command is a local development preview with file watching and live reload.
package margo
