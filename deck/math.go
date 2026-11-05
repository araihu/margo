package deck

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"strconv"
	"strings"
	"unicode"

	"github.com/araihu/margo"
	"github.com/araihu/margo/internal/mathsyntax"
	"github.com/yuin/goldmark/ast"
	nethtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Directive chrome retains its existing literal math behavior even when body
// compilation enables math. Protect dollars without changing other Markdown.
func literalDirectiveMath(value string) (string, string) {
	if !strings.Contains(value, "$") {
		return value, ""
	}
	// A unique non-Markdown punctuation character preserves delimiter flanking
	// without adding visible backslashes. Restore it after semantic rendering.
	decoded := stdhtml.UnescapeString(value)
	for candidate := rune(0x20A0); ; candidate++ {
		if (unicode.IsPunct(candidate) || unicode.IsSymbol(candidate)) && !strings.ContainsRune(decoded, candidate) && !strings.ContainsRune(value, candidate) {
			marker := string(candidate)
			return strings.ReplaceAll(value, "$", marker), marker
		}
	}
}

// mathSourceLine maps a fragment line to the authored file. column is the
// number of bytes trimmed from that line's beginning during slot extraction.
type mathSourceLine struct{ line, column int }

func cloneMathSlotLines(input map[string][]mathSourceLine) map[string][]mathSourceLine {
	if input == nil {
		return nil
	}
	result := make(map[string][]mathSourceLine, len(input))
	for name, positions := range input {
		result[name] = append([]mathSourceLine(nil), positions...)
	}
	return result
}

// Deck comments are removed before fragment compilation. Neutralize them for
// span discovery as well: otherwise CommonMark's HTML-block rule can swallow
// a following math opener when the author omits a blank line after a marker.
func protectDeckMath(body []byte) ([]bool, []mathsyntax.Issue) {
	first, _ := mathsyntax.Mask(body, true)
	neutral := append([]byte(nil), body...)
	var fence *fenceState
	offset := 0
	for _, line := range bytes.SplitAfter(first, []byte{'\n'}) {
		if fence != nil {
			if closesFence(line, *fence) {
				fence = nil
			}
		} else if marker, length, ok := opensFence(line); ok {
			fence = &fenceState{marker: marker, length: length}
		} else if _, ok := htmlComment(line); ok {
			for i := offset; i < offset+len(line); i++ {
				if neutral[i] != '\n' && neutral[i] != '\r' {
					neutral[i] = ' '
				}
			}
		}
		offset += len(line)
	}
	root := mathsyntax.Parse(neutral)
	protected := make([]bool, len(bytes.SplitAfter(body, []byte{'\n'})))
	var issues []mathsyntax.Issue
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if data := mathsyntax.Of(node); data != nil && data.Display {
				if !data.Closed {
					issues = append(issues, mathsyntax.Issue{Code: "math.delimiter_unclosed", Offset: data.Offset, Message: "math block is not closed before its container ends"})
				}
				start := bytes.Count(body[:data.Offset], []byte{'\n'})
				end := bytes.Count(body[:max(data.Offset, data.End-1)], []byte{'\n'})
				for i := start; i <= end; i++ {
					protected[i] = true
				}
			}
		}
		return ast.WalkContinue, nil
	})
	return protected, issues
}

// Structural comments can hide content from a whole-file Markdown parse.
// Validate the actual slide/slot inputs with one shared source budget.
func validateDeckMath(name string, slides []Slide) error {
	validator := new(mathsyntax.Validator)
	validate := func(markdown []byte, positions []mathSourceLine) error {
		issues := validator.Validate(mathsyntax.Parse(markdown))
		if len(issues) == 0 {
			return nil
		}
		issue := issues[0]
		line := bytes.Count(markdown[:issue.Offset], []byte{'\n'})
		column := issue.Offset - bytes.LastIndexByte(markdown[:issue.Offset], '\n')
		if line >= len(positions) {
			return fmt.Errorf("math.source_mapping_invalid: invalid source position")
		}
		position := positions[line]
		return &margo.DiagnosticError{Diagnostics: []margo.Diagnostic{{Code: issue.Code, Severity: margo.SeverityError,
			Source: name, Line: position.line, Column: column + position.column, Pointer: "/math", Message: issue.Message,
			Hint: "Check math delimiters and the equation source limits."}}}
	}
	for _, slide := range slides {
		if slide.layout == nil {
			if err := validate(slide.markdown, slide.sourceLines); err != nil {
				return err
			}
		} else {
			for _, slot := range slide.layout.Slots {
				if err := validate(slot.Markdown, slide.slotSourceLines[slot.Name]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// remapMathSources uses scanner-owned positions rather than matching equation
// text. Repeated payloads and footnote reordering cannot change their locations.
func remapMathSources(input RenderInput, slides []Slide, article []byte) ([]byte, error) {
	sources := map[string][]mathSourceLine{}
	for _, slide := range slides {
		if slide.layout == nil {
			sources[fmt.Sprintf("%s#slide-%d", input.Name, slide.Ordinal())] = slide.sourceLines
		} else {
			for _, slot := range slide.layout.Slots {
				sources[fmt.Sprintf("%s#slide-%d-slot-%s", input.Name, slide.Ordinal(), slot.Name)] = slide.slotSourceLines[slot.Name]
			}
		}
	}
	contextNode := &nethtml.Node{Type: nethtml.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := nethtml.ParseFragment(bytes.NewReader(article), contextNode)
	if err != nil {
		return nil, err
	}
	var walk func(*nethtml.Node) error
	walk = func(node *nethtml.Node) error {
		if htmlAttribute(node, "data-margo-runtime-task") == "math" {
			positions := sources[htmlAttribute(node, "data-margo-source")]
			line, err := strconv.Atoi(htmlAttribute(node, "data-margo-line"))
			if err != nil || line < 1 || line > len(positions) {
				return fmt.Errorf("math.source_mapping_invalid: rendered equation has no original source")
			}
			column, err := strconv.Atoi(htmlAttribute(node, "data-margo-column"))
			if err != nil || column < 1 {
				return fmt.Errorf("math.source_mapping_invalid: invalid column")
			}
			position := positions[line-1]
			setMathAttribute(node, "data-margo-source", input.Name)
			setMathAttribute(node, "data-margo-line", strconv.Itoa(position.line))
			setMathAttribute(node, "data-margo-column", strconv.Itoa(column+position.column))
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	for _, node := range nodes {
		if err := walk(node); err != nil {
			return nil, err
		}
	}
	var out bytes.Buffer
	for _, node := range nodes {
		if err := nethtml.Render(&out, node); err != nil {
			return nil, err
		}
	}
	return out.Bytes(), nil
}

func htmlAttribute(node *nethtml.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}
func setMathAttribute(node *nethtml.Node, key, value string) {
	for i := range node.Attr {
		if node.Attr[i].Key == key {
			node.Attr[i].Val = value
			return
		}
	}
	node.Attr = append(node.Attr, nethtml.Attribute{Key: key, Val: value})
}
