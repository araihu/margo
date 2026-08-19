// Package deck parses Margo Markdown and renders accessible HTML presentation
// decks.
//
// After optional opening YAML frontmatter, an exact thematic break (---)
// outside a fenced code block separates slides. Frontmatter supplies the deck
// title and description. Parse returns the immutable slide model when a caller
// needs to inspect it directly. Render performs parsing, compiles every slide
// with the supplied margo.Compiler, and returns a complete HTML document:
//
//	result, err := deck.Render(ctx, compiler, deck.RenderInput{
//		Name:     "talk.md",
//		Markdown: source,
//	})
//	if err != nil {
//		return err
//	}
//	html := result.HTML()
//
// Result also reports the slide count, merged HTML requirements, document
// fingerprint, and runtime descriptor needed by later projections. Theme and
// color-mode defaults are Margo's modern theme and light mode.
// Detect inspects only opening frontmatter and returns true for `marp: true`,
// allowing a host to route a source into this package without changing the
// root compiler's ordinary Markdown behavior.
//
// The implementation follows the versioned Margo Marpit-compatible v0.0.1
// profile. It intentionally does not claim universal Marpit or Marp Core
// compatibility: only the built-in themes, directives, layouts, and extension
// contracts documented by this profile are accepted.
package deck
