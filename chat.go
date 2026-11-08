package margo

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// MaxChatBytes bounds a single RenderChat source to one MiB.
const MaxChatBytes = 1 << 20

// RenderChat renders a static, safe chat fragment using Margo's existing Markdown
// parser and delimiter behavior. It supports emphasis, strike, quotes, lists,
// links, and plain inline/fenced code. Soft newlines remain visible. Frontmatter
// is literal content; raw HTML is escaped. Images become readable links or text,
// never image elements. Only absolute HTTP(S) and mailto links are interactive.
// Document headings have no IDs, fences never invoke extensions or runtimes, and
// the fragment needs no Margo assets. Existing Compile/Render APIs are unchanged.
// The returned component contains immutable pre-rendered content and is safe to
// reuse concurrently. Source/resource failures return an error for host fallback.
func RenderChat(ctx context.Context, source string) (templ.Component, error) {
	return RenderChatWithOptions(ctx, source, ChatOptions{})
}

// ChatOptions selects additive chat syntax behavior. Its zero value preserves
// RenderChat's CommonMark syntax, including lazy blockquote continuation.
type ChatOptions struct {
	// ExplicitBlockquotes requires a quote marker at each active quote depth on
	// every quoted line. An unmarked line starts content outside that quote.
	ExplicitBlockquotes bool
}

// RenderChatWithOptions renders the same safe static fragment as RenderChat,
// with optional chat syntax behavior. It never modifies the caller's source.
func RenderChatWithOptions(ctx context.Context, source string, options ChatOptions) (templ.Component, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if len(source) > MaxChatBytes {
		return nil, fmt.Errorf("chat.source_too_large: maximum %d bytes", MaxChatBytes)
	}
	r := chatRenderer{ctx: ctx, source: []byte(source)}
	if end := chatFrontmatterEnd(r.source); end > 0 {
		r.out.WriteString("<p>")
		r.out.WriteString(strings.ReplaceAll(html.EscapeString(string(r.source[:end])), "\n", "<br>"))
		r.out.WriteString("</p>")
		r.source = r.source[end:]
	}
	markdown := newMarkdownParser(false)
	if options.ExplicitBlockquotes {
		configureExplicitChatQuotes(markdown)
	}
	root := markdown.Parse(text.NewReader(r.source))
	if err := r.node(root, 0); err != nil {
		return nil, err
	}
	rendered := r.out.String()
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if err := contextError(ctx); err != nil {
			return err
		}
		_, err := io.WriteString(w, rendered)
		return err
	}), nil
}

func chatFrontmatterEnd(source []byte) int {
	lines := bytes.SplitAfter(source, []byte("\n"))
	if len(lines) == 0 || strings.TrimSpace(string(lines[0])) != "---" {
		return 0
	}
	offset := len(lines[0])
	for _, line := range lines[1:] {
		offset += len(line)
		marker := strings.TrimSpace(string(line))
		if marker == "---" || marker == "..." {
			return offset
		}
	}
	return len(source)
}

type chatRenderer struct {
	ctx    context.Context
	source []byte
	out    strings.Builder
}

func (r *chatRenderer) node(n ast.Node, depth int) error {
	if err := contextError(r.ctx); err != nil {
		return err
	}
	if depth > 128 || r.out.Len() > 8<<20 {
		return fmt.Errorf("chat.resource_limit")
	}
	if n.Type() == ast.TypeInline {
		return r.inline(n, depth)
	}
	return r.block(n, depth)
}
func (r *chatRenderer) children(n ast.Node, depth int) error {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if err := r.node(child, depth+1); err != nil {
			return err
		}
	}
	return nil
}
func (r *chatRenderer) wrap(n ast.Node, depth int, open, close string) error {
	r.out.WriteString(open)
	if err := r.children(n, depth); err != nil {
		return err
	}
	r.out.WriteString(close)
	return nil
}
func (r *chatRenderer) block(n ast.Node, depth int) error {
	switch v := n.(type) {
	case *ast.Document, *ast.TextBlock:
		return r.children(n, depth)
	case *ast.Paragraph:
		return r.wrap(n, depth, "<p>", "</p>")
	case *ast.Blockquote:
		return r.wrap(n, depth, "<blockquote>", "</blockquote>")
	case *ast.List:
		if v.IsOrdered() {
			return r.wrap(n, depth, fmt.Sprintf(`<ol start="%d">`, v.Start), "</ol>")
		}
		return r.wrap(n, depth, "<ul>", "</ul>")
	case *ast.ListItem:
		return r.wrap(n, depth, "<li>", "</li>")
	case *ast.FencedCodeBlock, *ast.CodeBlock:
		r.out.WriteString("<pre><code>")
		r.out.WriteString(html.EscapeString(string(n.Lines().Value(r.source))))
		r.out.WriteString("</code></pre>")
		return nil
	case *ast.HTMLBlock:
		r.out.WriteString("<p>")
		r.out.WriteString(strings.ReplaceAll(html.EscapeString(string(v.Text(r.source))), "\n", "<br>"))
		r.out.WriteString("</p>")
		return nil
	case *ast.Heading:
		r.out.WriteString("<p>")
		r.out.WriteString(strings.Repeat("#", v.Level) + " ")
		if err := r.children(n, depth); err != nil {
			return err
		}
		r.out.WriteString("</p>")
		return nil
	case *ast.ThematicBreak:
		r.out.WriteString("<p>---</p>")
		return nil
	default:
		return r.readableBlock(n, depth)
	}
}

// Unsupported document structures remain text, without controls, identity IDs,
// or runtime requirements. Row/cell separators retain readable table content.
func (r *chatRenderer) readableBlock(n ast.Node, depth int) error {
	switch n.(type) {
	case *extast.Table:
		return r.wrap(n, depth, "<div>", "</div>")
	case *extast.TableRow, *extast.TableHeader:
		return r.wrap(n, depth, "<p>", "</p>")
	case *extast.TableCell:
		return r.wrap(n, depth, "", " ")
	case *extast.Footnote:
		return r.wrap(n, depth, "<p>", "</p>")
	}
	return r.children(n, depth)
}
func (r *chatRenderer) inline(n ast.Node, depth int) error {
	switch v := n.(type) {
	case *ast.Text:
		r.out.WriteString(html.EscapeString(string(v.Value(r.source))))
		if v.SoftLineBreak() || v.HardLineBreak() {
			r.out.WriteString("<br>")
		}
		return nil
	case *ast.String:
		r.out.WriteString(html.EscapeString(string(v.Value)))
		return nil
	case *ast.Emphasis:
		if v.Level > 1 {
			return r.wrap(n, depth, "<strong>", "</strong>")
		}
		return r.wrap(n, depth, "<em>", "</em>")
	case *extast.Strikethrough:
		return r.wrap(n, depth, "<del>", "</del>")
	case *ast.CodeSpan:
		r.out.WriteString("<code>")
		r.out.WriteString(html.EscapeString(inlineText(n, r.source)))
		r.out.WriteString("</code>")
		return nil
	case *ast.Link:
		return r.link(n, depth, string(v.Destination), false)
	case *ast.Image:
		return r.link(n, depth, string(v.Destination), true)
	case *ast.AutoLink:
		return r.autoLink(v)
	case *ast.RawHTML:
		r.out.WriteString(html.EscapeString(string(v.Segments.Value(r.source))))
		return nil
	case *extast.TaskCheckBox:
		if v.IsChecked {
			r.out.WriteString("[x] ")
		} else {
			r.out.WriteString("[ ] ")
		}
		return nil
	case *extast.FootnoteLink:
		r.out.WriteString(fmt.Sprintf("[%d]", v.Index))
		return nil
	case *extast.FootnoteBacklink:
		return nil
	default:
		return r.children(n, depth)
	}
}
func chatURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.Host != ""
	case "mailto":
		return u.Opaque != "" && u.Host == ""
	default:
		return false
	}
}
func (r *chatRenderer) link(n ast.Node, depth int, destination string, image bool) error {
	valid := chatURL(destination)
	if valid {
		r.out.WriteString(`<a href="` + html.EscapeString(destination) + `" target="_blank" rel="noopener noreferrer">`)
	}
	if err := r.children(n, depth); err != nil {
		return err
	}
	if image || !valid {
		r.out.WriteString(" (" + html.EscapeString(destination) + ")")
	}
	if valid {
		r.out.WriteString("</a>")
	}
	return nil
}
func (r *chatRenderer) autoLink(v *ast.AutoLink) error {
	destination := string(v.URL(r.source))
	label := html.EscapeString(string(v.Label(r.source)))
	if chatURL(destination) {
		r.out.WriteString(`<a href="` + html.EscapeString(destination) + `" target="_blank" rel="noopener noreferrer">` + label + `</a>`)
	} else {
		r.out.WriteString(label)
	}
	return nil
}
