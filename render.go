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
	extensionSlots, err := executeRenderPlanSlots(ctx, document.plan.clone())
	if err != nil {
		return nil, err
	}
	body := templ.ComponentFunc(func(renderCtx context.Context, out io.Writer) error {
		renderer := markdownRenderer{
			ctx:                 renderCtx,
			out:                 out,
			source:              parsed.frontmatter.body,
			policy:              document.effectivePolicy,
			tableSort:           tableSortMode(renderOptions),
			runtimeTaskOrdinals: make(map[string]uint32),
			contextLabels:       make(map[string]string),
			extensionSlots:      extensionSlots,
			target:              renderTarget(renderOptions),
			idAllocator:         renderIDAllocator(renderOptions),
		}
		return renderer.renderBlock(parsed.root)
	})
	var rendered bytes.Buffer
	if err := semanticDocument(body).Render(ctx, &rendered); err != nil {
		return nil, err
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
	contextLabels       map[string]string
	extensionSlots      [][]byte
	target              RenderTarget
	idAllocator         RenderIDAllocator
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
	if _, paragraph := node.(*goldast.Paragraph); !paragraph && r.contextLabels["lead-pending"] == "true" {
		delete(r.contextLabels, "lead-pending")
	}
	switch value := node.(type) {
	case *goldast.Heading:
		r.contextLabels["heading"] = plainInlineText(value, r.source)
		renderLevel := value.Level
		if value.Level == 1 {
			if r.contextLabels["document-title-rendered"] == "true" {
				renderLevel = 2
			} else {
				r.contextLabels["document-title-rendered"] = "true"
				r.contextLabels["lead-pending"] = "true"
			}
		}
		id := ""
		if attr, ok := value.AttributeString("id"); ok {
			switch typed := attr.(type) {
			case []byte:
				id = string(typed)
			case string:
				id = typed
			}
		}
		if _, err := fmt.Fprintf(r.out, `<h%d`, renderLevel); err != nil {
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
		_, err := fmt.Fprintf(r.out, `</h%d>`, renderLevel)
		return err
	case *goldast.Paragraph:
		if image, ok := value.FirstChild().(*goldast.Image); ok && image.NextSibling() == nil && len(image.Title) > 0 {
			delete(r.contextLabels, "lead-pending")
			if _, err := io.WriteString(r.out, `<figure class="margo-figure">`); err != nil {
				return err
			}
			if err := r.renderImage(image); err != nil {
				return err
			}
			_, err := fmt.Fprintf(r.out, `<figcaption class="margo-figure-caption">%s</figcaption></figure>`, html.EscapeString(string(image.Title)))
			return err
		}
		opening := `<p>`
		if r.contextLabels["lead-pending"] == "true" {
			opening = `<p class="margo-document__lead">`
			delete(r.contextLabels, "lead-pending")
		}
		if _, err := io.WriteString(r.out, opening); err != nil {
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
		if slot, ok := extensionSlot(value); ok {
			if language == "mermaid" {
				return r.renderRuntimeFence(language, value.Lines().Value(r.source))
			}
			if int(slot) >= len(r.extensionSlots) {
				return fmt.Errorf("extension.slot_missing: %d", slot)
			}
			_, err := r.out.Write(r.extensionSlots[slot])
			return err
		}
		if language == "mermaid" {
			return r.renderRuntimeFence(language, value.Lines().Value(r.source))
		}
		return renderCodeBlock(r.ctx, r.out, language, value.Lines().Value(r.source))
	case *goldast.CodeBlock:
		return renderCodeBlock(r.ctx, r.out, "", value.Lines().Value(r.source))
	case *tableast.Table:
		ordinal := r.runtimeTaskOrdinals["table"]
		r.runtimeTaskOrdinals["table"] = ordinal + 1
		tableID := fmt.Sprintf("margo-table-%d", ordinal)
		if r.idAllocator != nil {
			tableID = r.idAllocator.Allocate("table", strconv.FormatUint(uint64(ordinal), 10))
		}
		return renderMarkdownTable(r.ctx, r.out, value, r.source, r.tableSort, tableID)
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
		footnoteID := fmt.Sprintf("fn:%d", value.Index)
		if r.idAllocator != nil {
			footnoteID = r.idAllocator.Allocate("footnote", strconv.Itoa(value.Index))
		}
		if _, err := fmt.Fprintf(r.out, `<li id="%s">`, html.EscapeString(footnoteID)); err != nil {
			return err
		}
		if err := r.renderBlock(value); err != nil {
			return err
		}
		_, err := io.WriteString(r.out, `</li>`)
		return err
	case *goldast.HTMLBlock:
		return r.renderRawHTML(value.Text(r.source))
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
		footnoteID := fmt.Sprintf("fn:%d", value.Index)
		if r.idAllocator != nil {
			refID = r.idAllocator.Allocate("footnote-ref", refID)
			footnoteID = r.idAllocator.Allocate("footnote", strconv.Itoa(value.Index))
		}
		_, err := fmt.Fprintf(r.out, `<sup id="%s"><a href="#%s" role="doc-noteref" aria-label="Footnote %d">%d</a></sup>`, html.EscapeString(refID), html.EscapeString(footnoteID), value.Index, value.Index)
		return err
	case *tableast.FootnoteBacklink:
		refID := "fnref:" + strconv.Itoa(value.Index)
		if value.RefIndex > 0 {
			refID = "fnref" + strconv.Itoa(value.RefIndex) + ":" + strconv.Itoa(value.Index)
		}
		if r.idAllocator != nil {
			refID = r.idAllocator.Allocate("footnote-ref", refID)
		}
		_, err := fmt.Fprintf(r.out, `&#160;<a href="#%s" role="doc-backlink" aria-label="Back to footnote reference %d">↩</a>`, html.EscapeString(refID), value.Index)
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
	captionID := fmt.Sprintf("margo-mermaid-caption-%d", ordinal)
	sourceID := fmt.Sprintf("margo-mermaid-source-%d", ordinal)
	if r.idAllocator != nil {
		captionID = r.idAllocator.Allocate("mermaid-caption", strconv.FormatUint(uint64(ordinal), 10))
		sourceID = r.idAllocator.Allocate("mermaid-source", strconv.FormatUint(uint64(ordinal), 10))
	}
	contextLabel := strings.TrimSpace(r.contextLabels["heading"])
	if contextLabel == "" {
		contextLabel = fmt.Sprintf("diagram %d", ordinal+1)
	}
	caption, printLayout := mermaidFigureDescription(source, contextLabel)
	printLayoutAttribute := ""
	if printLayout != "" {
		printLayoutAttribute = fmt.Sprintf(` data-margo-print-layout="%s"`, html.EscapeString(printLayout))
	}
	if _, err := fmt.Fprintf(r.out, `<figure class="margo-runtime-task margo-mermaid" data-margo-runtime-task="%s" data-margo-runtime-task-ordinal="%d"%s><div class="margo-mermaid__canvas" role="img" aria-labelledby="%s" aria-describedby="%s"></div><span id="%s" class="margo-mermaid__accessible-source">Complete Mermaid source: %s</span><span class="margo-mermaid__overflow-cue">Scroll diagram horizontally to inspect all labels.</span><details class="margo-mermaid__source"><summary>Mermaid source for %s</summary><pre><code>`, html.EscapeString(kind), ordinal, printLayoutAttribute, captionID, sourceID, sourceID, html.EscapeString(strings.TrimSpace(string(source))), html.EscapeString(contextLabel)); err != nil {
		return err
	}
	if _, err := io.WriteString(r.out, html.EscapeString(string(source))); err != nil {
		return err
	}
	_, err := fmt.Fprintf(r.out, `</code></pre></details><figcaption id="%s" class="margo-figure-caption">%s</figcaption></figure>`, captionID, html.EscapeString(caption))
	return err
}

func mermaidFigureDescription(source []byte, contextLabel string) (string, string) {
	lines := strings.Split(string(source), "\n")
	first := ""
	for _, line := range lines {
		if first = strings.TrimSpace(line); first != "" {
			break
		}
	}
	fields := strings.Fields(first)
	if len(fields) > 0 && (strings.EqualFold(fields[0], "flowchart") || strings.EqualFold(fields[0], "graph")) {
		labels := mermaidDelimitedLabels(lines)
		if len(labels) > 0 {
			return "Flowchart connecting " + naturalLanguageList(labels) + ".", ""
		}
		return "Flowchart for " + contextLabel + ", with its Mermaid source available as a text fallback.", ""
	}
	if len(fields) > 0 && strings.EqualFold(fields[0], "sequenceDiagram") {
		participants := mermaidParticipants(lines)
		if len(participants) > 0 {
			connector := " among "
			if len(participants) == 2 {
				connector = " between "
			}
			return "Sequence diagram showing interactions" + connector + naturalLanguageList(participants) + ".", ""
		}
		return "Sequence diagram for " + contextLabel + ", with its Mermaid source available as a text fallback.", ""
	}
	return "Mermaid diagram for " + contextLabel + ", with its source available as a text fallback.", ""
}

func mermaidDelimitedLabels(lines []string) []string {
	labels := make([]string, 0, 6)
	seen := make(map[string]struct{})
	for _, line := range lines[1:] {
		for index := 0; index < len(line); index++ {
			opening := line[index]
			closing := byte(0)
			switch opening {
			case '[':
				closing = ']'
			case '{':
				closing = '}'
			default:
				continue
			}
			end := strings.IndexByte(line[index+1:], closing)
			if end < 0 {
				continue
			}
			label := strings.TrimSpace(line[index+1 : index+1+end])
			index += end + 1
			if label == "" {
				continue
			}
			if _, duplicate := seen[label]; duplicate {
				continue
			}
			seen[label] = struct{}{}
			labels = append(labels, label)
		}
	}
	return labels
}

func mermaidParticipants(lines []string) []string {
	participants := make([]string, 0, 4)
	seen := make(map[string]struct{})
	for _, line := range lines[1:] {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 || (!strings.EqualFold(fields[0], "participant") && !strings.EqualFold(fields[0], "actor")) {
			continue
		}
		label := fields[1]
		for index := 2; index+1 < len(fields); index++ {
			if strings.EqualFold(fields[index], "as") {
				label = strings.Join(fields[index+1:], " ")
				break
			}
		}
		if _, duplicate := seen[label]; duplicate {
			continue
		}
		seen[label] = struct{}{}
		participants = append(participants, label)
	}
	return participants
}

func naturalLanguageList(values []string) string {
	if len(values) > 5 {
		return strings.Join(values[:4], ", ") + fmt.Sprintf(", and %d more nodes", len(values)-4)
	}
	switch len(values) {
	case 0:
		return ""
	case 1:
		return values[0]
	case 2:
		return values[0] + " and " + values[1]
	default:
		return strings.Join(values[:len(values)-1], ", ") + ", and " + values[len(values)-1]
	}
}

func (r markdownRenderer) renderRawHTML(fragment []byte) error {
	remaining, err := stripHTMLComments(fragment)
	if err != nil {
		return fmt.Errorf("source.html_comment_malformed: %w", err)
	}
	if strings.TrimSpace(string(remaining)) == "" {
		return nil
	}
	if embed, recognized, embedErr := parseIframeFragment(remaining); recognized {
		if embedErr != nil {
			return fmt.Errorf("source.iframe_invalid: %w", embedErr)
		}
		return renderIframe(r.out, embed, r.policy.Iframe, r.target)
	}
	if r.policy.RawHTML != RawHTMLSanitized {
		return fmt.Errorf("policy.raw_html.denied: raw HTML is not allowed during rendering")
	}
	normalized, err := normalizeHTML(remaining)
	if err != nil {
		return err
	}
	_, err = r.out.Write(normalized)
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
