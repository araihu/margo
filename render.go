package margo

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	goldast "github.com/yuin/goldmark/ast"
	tableast "github.com/yuin/goldmark/extension/ast"
)

func init() {
	renderDocumentBytes = renderSemanticDocumentBytes
}

func renderSemanticDocumentBytes(ctx context.Context, document *Document, options []RenderOption) ([]byte, error) {
	renderOptions, err := applyRenderOptions(options)
	if err != nil {
		return nil, err
	}
	parsed, ok := document.parsed.(normalizedMarkdown)
	if !ok || parsed.root == nil {
		return nil, fmt.Errorf("render.document_invalid: compiled document has no Markdown AST")
	}
	body := templ.ComponentFunc(func(renderCtx context.Context, out io.Writer) error {
		renderer := markdownRenderer{
			ctx:                 renderCtx,
			out:                 out,
			source:              parsed.frontmatter.body,
			policy:              document.effectivePolicy,
			tableSort:           tableSortMode(renderOptions),
			runtimeTaskOrdinals: make(map[string]uint32),
		}
		return renderer.renderBlock(parsed.root)
	})
	var rendered bytes.Buffer
	if err := semanticDocument(body).Render(ctx, &rendered); err != nil {
		return nil, err
	}
	extensionBytes, err := executeRenderPlan(ctx, document.plan.clone())
	if err != nil {
		return nil, err
	}
	if len(extensionBytes) > 0 {
		if _, err := rendered.Write(extensionBytes); err != nil {
			return nil, err
		}
	}
	if int64(rendered.Len()) > document.effectivePolicy.OutputBytes {
		return nil, policyDiagnostic("policy.output_bytes_exceeded", "rendered output exceeds the effective output byte limit")
	}
	return append([]byte(nil), rendered.Bytes()...), nil
}

type markdownRenderer struct {
	ctx                 context.Context
	out                 io.Writer
	source              []byte
	policy              EffectivePolicy
	tableSort           TableSortMode
	runtimeTaskOrdinals map[string]uint32
}

func (r markdownRenderer) renderBlock(node goldast.Node) error {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if err := r.renderNode(child); err != nil {
			return err
		}
	}
	return nil
}

func (r markdownRenderer) renderNode(node goldast.Node) error {
	switch value := node.(type) {
	case *goldast.Heading:
		id := ""
		if attr, ok := value.AttributeString("id"); ok {
			switch typed := attr.(type) {
			case []byte:
				id = string(typed)
			case string:
				id = typed
			}
		}
		if _, err := fmt.Fprintf(r.out, `<h%d`, value.Level); err != nil {
			return err
		}
		if id != "" {
			if _, err := fmt.Fprintf(r.out, ` id="%s"`, html.EscapeString(id)); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(r.out, `>`); err != nil {
			return err
		}
		if err := r.renderInlineChildren(value); err != nil {
			return err
		}
		_, err := fmt.Fprintf(r.out, `</h%d>`, value.Level)
		return err
	case *goldast.Paragraph:
		if _, err := io.WriteString(r.out, `<p>`); err != nil {
			return err
		}
		if err := r.renderInlineChildren(value); err != nil {
			return err
		}
		_, err := io.WriteString(r.out, `</p>`)
		return err
	case *goldast.TextBlock:
		return r.renderInlineChildren(value)
	case *goldast.Blockquote:
		if _, err := io.WriteString(r.out, `<blockquote>`); err != nil {
			return err
		}
		if err := r.renderBlock(value); err != nil {
			return err
		}
		_, err := io.WriteString(r.out, `</blockquote>`)
		return err
	case *goldast.List:
		if value.IsOrdered() {
			if _, err := fmt.Fprintf(r.out, `<ol start="%d">`, value.Start); err != nil {
				return err
			}
		} else if _, err := io.WriteString(r.out, `<ul>`); err != nil {
			return err
		}
		for child := value.FirstChild(); child != nil; child = child.NextSibling() {
			if err := r.renderListItem(child, value.IsTight); err != nil {
				return err
			}
		}
		if value.IsOrdered() {
			_, err := io.WriteString(r.out, `</ol>`)
			return err
		}
		_, err := io.WriteString(r.out, `</ul>`)
		return err
	case *goldast.ThematicBreak:
		_, err := io.WriteString(r.out, `<hr>`)
		return err
	case *goldast.FencedCodeBlock:
		language := string(value.Language(r.source))
		if language == "mermaid" {
			return r.renderRuntimeFence(language, value.Lines().Value(r.source))
		}
		return renderCodeBlock(r.ctx, r.out, language, value.Lines().Value(r.source))
	case *goldast.CodeBlock:
		return renderCodeBlock(r.ctx, r.out, "", value.Lines().Value(r.source))
	case *tableast.Table:
		return renderMarkdownTable(r.ctx, r.out, value, r.source, r.tableSort)
	case *tableast.FootnoteList:
		if _, err := io.WriteString(r.out, `<section class="footnotes" aria-label="Footnotes" role="doc-endnotes"><hr><ol>`); err != nil {
			return err
		}
		if err := r.renderBlock(value); err != nil {
			return err
		}
		_, err := io.WriteString(r.out, `</ol></section>`)
		return err
	case *tableast.Footnote:
		if _, err := fmt.Fprintf(r.out, `<li id="fn:%d">`, value.Index); err != nil {
			return err
		}
		if err := r.renderBlock(value); err != nil {
			return err
		}
		_, err := io.WriteString(r.out, `</li>`)
		return err
	case *goldast.HTMLBlock:
		return r.renderRawHTML(value.Lines().Value(r.source))
	default:
		return r.renderBlock(node)
	}
}

func (r markdownRenderer) renderListItem(node goldast.Node, tight bool) error {
	item, ok := node.(*goldast.ListItem)
	if !ok {
		return r.renderNode(node)
	}
	if _, err := io.WriteString(r.out, `<li>`); err != nil {
		return err
	}
	for child := item.FirstChild(); child != nil; child = child.NextSibling() {
		if tight {
			switch inlineBlock := child.(type) {
			case *goldast.Paragraph:
				if err := r.renderInlineChildren(inlineBlock); err != nil {
					return err
				}
				continue
			case *goldast.TextBlock:
				if err := r.renderInlineChildren(inlineBlock); err != nil {
					return err
				}
				continue
			}
		}
		if err := r.renderNode(child); err != nil {
			return err
		}
	}
	_, err := io.WriteString(r.out, `</li>`)
	return err
}

func (r markdownRenderer) renderInlineChildren(node goldast.Node) error {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if err := r.renderInline(child); err != nil {
			return err
		}
	}
	return nil
}

func (r markdownRenderer) renderInline(node goldast.Node) error {
	switch value := node.(type) {
	case *goldast.Text:
		if _, err := io.WriteString(r.out, html.EscapeString(string(value.Value(r.source)))); err != nil {
			return err
		}
		if value.HardLineBreak() {
			_, err := io.WriteString(r.out, `<br>`)
			return err
		}
		if value.SoftLineBreak() {
			_, err := io.WriteString(r.out, "\n")
			return err
		}
		return nil
	case *goldast.String:
		_, err := io.WriteString(r.out, html.EscapeString(string(value.Value)))
		return err
	case *goldast.CodeSpan:
		if _, err := io.WriteString(r.out, `<code>`); err != nil {
			return err
		}
		if err := r.renderInlineChildren(value); err != nil {
			return err
		}
		_, err := io.WriteString(r.out, `</code>`)
		return err
	case *goldast.Emphasis:
		tag := "em"
		if value.Level > 1 {
			tag = "strong"
		}
		if _, err := fmt.Fprintf(r.out, "<%s>", tag); err != nil {
			return err
		}
		if err := r.renderInlineChildren(value); err != nil {
			return err
		}
		_, err := fmt.Fprintf(r.out, "</%s>", tag)
		return err
	case *tableast.Strikethrough:
		if _, err := io.WriteString(r.out, `<del>`); err != nil {
			return err
		}
		if err := r.renderInlineChildren(value); err != nil {
			return err
		}
		_, err := io.WriteString(r.out, `</del>`)
		return err
	case *tableast.TaskCheckBox:
		if value.IsChecked {
			_, err := io.WriteString(r.out, `<input checked="" disabled="" type="checkbox" aria-label="Completed task"> `)
			return err
		}
		_, err := io.WriteString(r.out, `<input disabled="" type="checkbox" aria-label="Incomplete task"> `)
		return err
	case *tableast.FootnoteLink:
		refID := "fnref:" + strconv.Itoa(value.Index)
		if value.RefIndex > 0 {
			refID = "fnref" + strconv.Itoa(value.RefIndex) + ":" + strconv.Itoa(value.Index)
		}
		_, err := fmt.Fprintf(r.out, `<sup id="%s"><a href="#fn:%d" role="doc-noteref" aria-label="Footnote %d">%d</a></sup>`, refID, value.Index, value.Index, value.Index)
		return err
	case *tableast.FootnoteBacklink:
		refID := "fnref:" + strconv.Itoa(value.Index)
		if value.RefIndex > 0 {
			refID = "fnref" + strconv.Itoa(value.RefIndex) + ":" + strconv.Itoa(value.Index)
		}
		_, err := fmt.Fprintf(r.out, `&#160;<a href="#%s" role="doc-backlink" aria-label="Back to footnote reference %d">↩</a>`, refID, value.Index)
		return err
	case *goldast.Link:
		return r.renderLink(value.Destination, value.Title, value)
	case *goldast.Image:
		return r.renderImage(value)
	case *goldast.AutoLink:
		return r.renderLink(value.URL(r.source), nil, value)
	case *goldast.RawHTML:
		return r.renderRawHTML(value.Segments.Value(r.source))
	default:
		return r.renderInlineChildren(node)
	}
}

func (r markdownRenderer) renderLink(destination, title []byte, labelNode goldast.Node) error {
	href := string(destination)
	if err := validateRenderURL(href); err != nil {
		return fmt.Errorf("render.link_invalid: %w", err)
	}
	if _, err := fmt.Fprintf(r.out, `<a href="%s"`, html.EscapeString(href)); err != nil {
		return err
	}
	if len(title) > 0 {
		if _, err := fmt.Fprintf(r.out, ` title="%s"`, html.EscapeString(string(title))); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(r.out, `>`); err != nil {
		return err
	}
	if auto, ok := labelNode.(*goldast.AutoLink); ok {
		if _, err := io.WriteString(r.out, html.EscapeString(string(auto.Label(r.source)))); err != nil {
			return err
		}
	} else if err := r.renderInlineChildren(labelNode); err != nil {
		return err
	}
	_, err := io.WriteString(r.out, `</a>`)
	return err
}

func (r markdownRenderer) renderImage(image *goldast.Image) error {
	destination := string(image.Destination)
	if err := validateRenderURL(destination); err != nil {
		return fmt.Errorf("render.image_invalid: %w", err)
	}
	alt := plainInlineText(image, r.source)
	if _, err := fmt.Fprintf(r.out, `<img src="%s" alt="%s"`, html.EscapeString(destination), html.EscapeString(alt)); err != nil {
		return err
	}
	if len(image.Title) > 0 {
		if _, err := fmt.Fprintf(r.out, ` title="%s"`, html.EscapeString(string(image.Title))); err != nil {
			return err
		}
	}
	_, err := io.WriteString(r.out, `>`)
	return err
}

func (r markdownRenderer) renderRuntimeFence(kind string, source []byte) error {
	ordinal := r.runtimeTaskOrdinals[kind]
	r.runtimeTaskOrdinals[kind] = ordinal + 1
	if _, err := fmt.Fprintf(r.out, `<figure class="margo-runtime-task margo-mermaid" data-margo-runtime-task="%s" data-margo-runtime-task-ordinal="%d"><div class="margo-mermaid__canvas" role="img" aria-label="Mermaid diagram"></div><details open class="margo-mermaid__source"><summary>Mermaid source</summary><pre><code>`, html.EscapeString(kind), ordinal); err != nil {
		return err
	}
	if _, err := io.WriteString(r.out, html.EscapeString(string(source))); err != nil {
		return err
	}
	_, err := io.WriteString(r.out, `</code></pre></details></figure>`)
	return err
}

func (r markdownRenderer) renderRawHTML(fragment []byte) error {
	if r.policy.RawHTML != RawHTMLSanitized {
		return fmt.Errorf("policy.raw_html.denied: raw HTML is not allowed during rendering")
	}
	if err := ValidateHTML(string(fragment)); err != nil {
		return err
	}
	_, err := r.out.Write(fragment)
	return err
}

func validateRenderURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return err
	}
	if parsed.Scheme == "" {
		if strings.HasPrefix(value, "//") || parsed.Host != "" {
			return fmt.Errorf("network-path URL is not allowed")
		}
		return nil
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "mailto", "tel":
		return nil
	default:
		return fmt.Errorf("URL scheme %q is not allowed", parsed.Scheme)
	}
}
