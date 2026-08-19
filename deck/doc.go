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
//
// Deck projection is experimental. Hosts should not treat its current slide or
// publication behavior as a stable pre-v1 contract.
package deck
