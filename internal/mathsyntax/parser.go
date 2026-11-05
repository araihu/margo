// Package mathsyntax owns Margo's math grammar shared by compilation and decks.
package mathsyntax

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

const (
	EngineVersion      = "0.19.1"
	Version            = "margo-math/v1"
	MaxExpressionBytes = 16 << 10
	MaxExpressions     = 1024
	MaxPayloadBytes    = 1 << 20
)

// Data is immutable after compilation. Offsets address the original parser input.
type Data struct {
	SourceName      string
	Line, Column    int
	Payload         []byte
	Offset, End     int
	Display, Closed bool
	Ordinal         uint32
	InputSHA256     string
}

type Inline struct {
	ast.BaseInline
	Data
}
type Block struct {
	ast.BaseBlock
	Data
}

var inlineKind = ast.NewNodeKind("MargoMathInline")
var blockKind = ast.NewNodeKind("MargoMathBlock")

func (n *Inline) Kind() ast.NodeKind            { return inlineKind }
func (n *Block) Kind() ast.NodeKind             { return blockKind }
func (n *Inline) Text([]byte) []byte            { return n.Payload }
func (n *Block) Text([]byte) []byte             { return n.Payload }
func (n *Inline) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }
func (n *Block) Dump(source []byte, level int)  { ast.DumpHelper(n, source, level, nil, nil) }
func (n *Block) IsRaw() bool                    { return true }

func Of(node ast.Node) *Data {
	switch n := node.(type) {
	case *Inline:
		return &n.Data
	case *Block:
		return &n.Data
	}
	return nil
}

// Options adds parsers without exposing Goldmark through Margo's public API.
func Options() []parser.Option {
	return []parser.Option{
		parser.WithBlockParsers(util.Prioritized(&blockParser{}, 650), util.Prioritized(&fenceParser{base: parser.NewFencedCodeBlockParser()}, 650)),
		parser.WithInlineParsers(util.Prioritized(&inlineParser{}, 150)),
		parser.WithASTTransformers(util.Prioritized(&transformer{}, 1000)),
	}
}

func escaped(source []byte, offset int) bool {
	n := 0
	for offset--; offset >= 0 && source[offset] == '\\'; offset-- {
		n++
	}
	return n%2 != 0
}
func spaceAt(source []byte) bool { r, _ := utf8.DecodeRune(source); return unicode.IsSpace(r) }

type inlineParser struct{}

func (*inlineParser) Trigger() []byte { return []byte{'$'} }
func (*inlineParser) Parse(parent ast.Node, reader text.Reader, pc parser.Context) ast.Node {
	line, segment := reader.PeekLine()
	start := segment.Start
	if ranges, ok := pc.Get(imageRangesKey).([]imageRange); ok {
		i := sort.Search(len(ranges), func(i int) bool { return ranges[i].end > start })
		if i < len(ranges) && ranges[i].start <= start {
			return nil
		}
	}
	source := reader.Source()
	if len(line) < 3 || line[0] != '$' || line[1] == '$' || (start > 0 && source[start-1] == '$') || escaped(source, start) || spaceAt(line[1:]) {
		return nil
	}
	for i := 1; i < len(line) && line[i] != '\n' && line[i] != '\r'; i++ {
		if line[i] != '$' || escaped(line, i) {
			continue
		}
		prev, _ := utf8.DecodeLastRune(line[:i])
		if i == 1 || unicode.IsSpace(prev) || (i+1 < len(line) && (line[i+1] == '$' || line[i+1] >= '0' && line[i+1] <= '9')) {
			return nil
		}
		n := &Inline{Data: Data{Payload: append([]byte(nil), line[1:i]...), Offset: start, End: start + i + 1, Closed: true}}
		reader.Advance(i + 1)
		return n
	}
	return nil
}

type blockParser struct{}

func (*blockParser) Trigger() []byte             { return []byte{'$'} }
func (*blockParser) CanInterruptParagraph() bool { return true }
func (*blockParser) CanAcceptIndentedLine() bool { return false }
func (*blockParser) Open(_ ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	indent := pc.BlockIndent()
	if indent < 0 || indent > 3 || indent >= len(line) {
		return nil, parser.NoChildren
	}
	content := bytes.TrimRight(line[indent:], " \t\r\n")
	if !bytes.HasPrefix(content, []byte("$$")) || (len(content) > 2 && content[2] == '$') {
		return nil, parser.NoChildren
	}
	n := &Block{Data: Data{Offset: segment.Start + indent - segment.Padding, Display: true}}
	if len(content) > 2 {
		close := -1
		for i := 2; i < len(content); i++ {
			if content[i] == '$' && !escaped(content, i) {
				close = i
				break
			}
		}
		if close < 0 || close+2 != len(content) || content[close+1] != '$' {
			return nil, parser.NoChildren
		}
		n.Payload = append([]byte(nil), bytes.Trim(content[2:close], " \t")...)
		n.Closed = true
	}
	n.End = segment.Stop
	reader.AdvanceToEOL()
	return n, parser.NoChildren
}
func (*blockParser) Continue(node ast.Node, reader text.Reader, _ parser.Context) parser.State {
	n := node.(*Block)
	if n.Closed {
		return parser.Close
	}
	line, segment := reader.PeekLine()
	n.End = segment.Stop
	indent := 0
	for indent < len(line) && line[indent] == ' ' {
		indent++
	}
	if indent <= 3 && bytes.Equal(bytes.TrimRight(line[indent:], " \t\r\n"), []byte("$$")) {
		n.Closed = true
		reader.AdvanceToEOL()
		return parser.Close
	}
	n.Lines().Append(segment)
	reader.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}
func (*blockParser) Close(node ast.Node, reader text.Reader, _ parser.Context) {
	n := node.(*Block)
	if n.Lines().Len() > 0 {
		n.Payload = normalize(n.Lines().Value(reader.Source()))
	}
}

const fenceOffset = "margo-math-fence-offset"
const fenceClosed = "margo-math-fence-closed"

type fenceParser struct{ base parser.BlockParser }

func (*fenceParser) Trigger() []byte             { return []byte{'`', '~'} }
func (*fenceParser) CanInterruptParagraph() bool { return true }
func (*fenceParser) CanAcceptIndentedLine() bool { return false }
func (p *fenceParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	_, segment := reader.PeekLine()
	n, state := p.base.Open(parent, reader, pc)
	if n != nil {
		n.SetAttributeString(fenceOffset, segment.Start+pc.BlockIndent()-segment.Padding)
	}
	return n, state
}
func (p *fenceParser) Continue(n ast.Node, reader text.Reader, pc parser.Context) parser.State {
	_, segment := reader.PeekLine()
	state := p.base.Continue(n, reader, pc)
	if state == parser.Close {
		n.SetAttributeString(fenceClosed, segment.Stop)
	}
	return state
}
func (p *fenceParser) Close(n ast.Node, reader text.Reader, pc parser.Context) {
	p.base.Close(n, reader, pc)
}

type transformer struct{}

func (*transformer) Transform(root *ast.Document, reader text.Reader, _ parser.Context) {
	source := reader.Source()
	var fences []*ast.FencedCodeBlock
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n, ok := node.(*ast.FencedCodeBlock); ok && string(n.Language(source)) == "math" {
			fences = append(fences, n)
		}
		return ast.WalkContinue, nil
	})
	for _, f := range fences {
		offset, _ := f.AttributeString(fenceOffset)
		start, _ := offset.(int)
		closed, end := f.AttributeString(fenceClosed)
		stop, _ := closed.(int)
		if !end {
			stop = len(source)
		}
		n := &Block{Data: Data{Payload: normalize(f.Lines().Value(source)), Offset: start, End: stop, Display: true, Closed: end}}
		f.Parent().ReplaceChild(f.Parent(), f, n)
	}
}
func normalize(value []byte) []byte { return bytes.ReplaceAll(value, []byte("\r\n"), []byte("\n")) }

// Issue is a stable source-level finding, independent of the rendering engine.
type Issue struct {
	Code, Message string
	Offset        int
}

// Validator retains publication totals across independently parsed fragments.
type Validator struct{ count, total int }

func Validate(root ast.Node) []Issue { return new(Validator).Validate(root) }

func (v *Validator) Validate(root ast.Node) []Issue {
	var issues []Issue
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		n := Of(node)
		if n == nil {
			return ast.WalkContinue, nil
		}
		v.count++
		v.total += len(n.Payload)
		code, message := "", ""
		switch {
		case !n.Closed:
			code, message = "math.delimiter_unclosed", "math block is not closed before its container ends"
		case strings.TrimSpace(string(n.Payload)) == "":
			code, message = "math.empty_expression", "math expression is empty"
		case len(n.Payload) > MaxExpressionBytes:
			code, message = "math.expression_too_large", fmt.Sprintf("math expression exceeds %d bytes", MaxExpressionBytes)
		case v.count > MaxExpressions || v.total > MaxPayloadBytes:
			code, message = "math.document_limit_exceeded", "combined math payload or equation count exceeds the publication limit"
		}
		if code != "" {
			issues = append(issues, Issue{code, message, n.Offset})
		}
		return ast.WalkContinue, nil
	})
	return issues
}

// NewParser keeps compilation, deck scans, and protected image text on the
// same Markdown grammar. The disabled parser never installs math transforms.
func NewParser(enabled bool) parser.Parser {
	options := []parser.Option{parser.WithAutoHeadingID()}
	if enabled {
		options = append(options, Options()...)
	}
	p := goldmark.New(goldmark.WithExtensions(extension.Table, extension.Strikethrough,
		extension.TaskList, extension.Footnote, extension.Linkify),
		goldmark.WithParserOptions(options...)).Parser()
	if enabled {
		return &imageAwareParser{Parser: p}
	}
	return p
}

// Image labels are protected before math consumes any source, so an equation
// cannot swallow an image's closing bracket or change ordinary alt semantics.
var imageRangesKey = parser.NewContextKey()

const imageEndAttribute = "margo-image-alt-end"

type imageRange struct{ start, end int }
type imageAwareParser struct{ parser.Parser }

func (p *imageAwareParser) Parse(reader text.Reader, opts ...parser.ParseOption) ast.Node {
	if !bytes.Contains(reader.Source(), []byte("![")) {
		return p.Parser.Parse(reader, opts...)
	}
	plain := NewParser(false)
	plain.AddOptions(recordImageLinks{})
	ordinary := plain.Parse(text.NewReader(reader.Source()))
	var ranges []imageRange
	_ = ast.Walk(ordinary, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if image, ok := node.(*ast.Image); ok && entering {
			if value, ok := image.AttributeString(imageEndAttribute); ok {
				if end, ok := value.(int); ok {
					ranges = append(ranges, imageRange{image.Pos() + 2, end})
				}
			}
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })
	config := &parser.ParseConfig{}
	for _, opt := range opts {
		opt(config)
	}
	if config.Context == nil {
		config.Context = parser.NewContext()
	}
	config.Context.Set(imageRangesKey, ranges)
	return p.Parser.Parse(reader, append(opts, parser.WithContext(config.Context))...)
}

type recordImageLinks struct{}

func (recordImageLinks) SetParserOption(config *parser.Config) {
	for i, value := range config.InlineParsers {
		if value.Value == parser.NewLinkParser() {
			config.InlineParsers[i].Value = &imageLinkParser{InlineParser: parser.NewLinkParser()}
		}
	}
}

type imageLinkParser struct{ parser.InlineParser }

func (p *imageLinkParser) Parse(parent ast.Node, reader text.Reader, pc parser.Context) ast.Node {
	line, segment := reader.PeekLine()
	closing := len(line) > 0 && line[0] == ']'
	node := p.InlineParser.Parse(parent, reader, pc)
	if image, ok := node.(*ast.Image); ok && closing {
		image.SetAttributeString(imageEndAttribute, segment.Start)
	}
	return node
}
func (p *imageLinkParser) CloseBlock(parent ast.Node, reader text.Reader, pc parser.Context) {
	p.InlineParser.(parser.CloseBlocker).CloseBlock(parent, reader, pc)
}

// Parse is used for protecting deck scans with the compilation grammar.
func Parse(source []byte) ast.Node {
	return NewParser(true).Parse(text.NewReader(source))
}

// Mask preserves all source offsets and newlines while hiding math from deck syntax.
func Mask(source []byte, blocksOnly ...bool) ([]byte, []Issue) {
	root := Parse(source)
	out := append([]byte(nil), source...)
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if len(blocksOnly) > 0 && blocksOnly[0] {
			if _, ok := node.(*Inline); ok {
				return ast.WalkSkipChildren, nil
			}
		}
		if entering {
			if n := Of(node); n != nil {
				for i := max(0, n.Offset); i < min(len(out), n.End); i++ {
					if out[i] != '\n' && out[i] != '\r' {
						out[i] = ' '
					}
				}
			}
		}
		return ast.WalkContinue, nil
	})
	return out, Validate(root)
}
