package margo

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// The upstream quote parser reports Close when the current line lacks its
// marker. Goldmark then normally permits its paragraph's lazy continuation.
// Keep that decision scoped to the current parse context and source line.
var explicitQuoteBoundaryKey = parser.NewContextKey()

type explicitQuoteParser struct{ parser.BlockParser }

func (p explicitQuoteParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	state := p.BlockParser.Continue(node, reader, pc)
	if state&parser.Continue == 0 {
		line, _ := reader.Position()
		pc.Set(explicitQuoteBoundaryKey, line)
	}
	return state
}

type explicitQuoteParagraphParser struct {
	parser.BlockParser
	code parser.BlockParser
}

func explicitQuoteBoundary(reader text.Reader, pc parser.Context) bool {
	line, _ := reader.Position()
	boundary, ok := pc.Get(explicitQuoteBoundaryKey).(int)
	return ok && line == boundary
}
func (p explicitQuoteParagraphParser) CanInterruptParagraph() bool { return true }
func (p explicitQuoteParagraphParser) CanAcceptIndentedLine() bool { return true }

// Interrupt only the paragraph whose quote container just closed. The engine
// attaches this new block to the surviving parent, including a shallower quote.
// Four-space indentation begins ordinary indented code at that new boundary.
func (p explicitQuoteParagraphParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	boundary := explicitQuoteBoundary(reader, pc)
	if !boundary && (ast.IsParagraph(pc.LastOpenedBlock().Node) || pc.BlockIndent() > 3) {
		return nil, parser.NoChildren
	}
	if boundary && pc.BlockIndent() > 3 {
		return p.code.Open(parent, reader, pc)
	}
	return p.BlockParser.Open(parent, reader, pc)
}
func (p explicitQuoteParagraphParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	if node.Kind() == ast.KindCodeBlock {
		return p.code.Continue(node, reader, pc)
	}
	return p.BlockParser.Continue(node, reader, pc)
}
func (p explicitQuoteParagraphParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	if node.Kind() == ast.KindCodeBlock {
		p.code.Close(node, reader, pc)
		return
	}
	p.BlockParser.Close(node, reader, pc)
}
func configureExplicitChatQuotes(markdown parser.Parser) {
	// Precede the default quote/paragraph parsers without changing inline parsers.
	markdown.AddOptions(parser.WithBlockParsers(util.Prioritized(explicitQuoteParser{parser.NewBlockquoteParser()}, 799), util.Prioritized(explicitQuoteParagraphParser{BlockParser: parser.NewParagraphParser(), code: parser.NewCodeBlockParser()}, 999)))
}
