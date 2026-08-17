package site

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/araihu/goshtoso/components/button"
	"github.com/araihu/goshtoso/components/dropdown"
	"github.com/araihu/goshtoso/components/icon"
	"github.com/araihu/goshtoso/components/splitbutton"
	margo "github.com/araihu/margo"
	"github.com/araihu/margo/pdf"
	"github.com/araihu/margo/pdf/engines"
	"github.com/araihu/margo/site/appicons"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The iconpack sprite is embedded so inline builds remain self-contained. Local
// builds publish the same bytes under pageActionsIconSpritePath.
//
//go:embed appicons/sprite.svg
var pageActionsIconSprite []byte

const pageActionsScript = `(function () {
  "use strict";

  function copyText(value) {
    if (navigator.clipboard && window.isSecureContext) return navigator.clipboard.writeText(value);
    var input = document.createElement("textarea");
    input.value = value;
    input.setAttribute("readonly", "");
    input.style.position = "fixed";
    input.style.opacity = "0";
    document.body.appendChild(input);
    input.select();
    var copied = document.execCommand("copy");
    input.remove();
    return copied ? Promise.resolve() : Promise.reject(new Error("clipboard unavailable"));
  }

  document.addEventListener("click", function (event) {
    var printButton = event.target && event.target.closest ? event.target.closest("[data-margo-print-page]") : null;
    if (printButton) {
      event.preventDefault();
      window.print();
      return;
    }
    var button = event.target && event.target.closest ? event.target.closest("[data-margo-copy-page]") : null;
    if (!button) return;
    event.preventDefault();
    var actions = button.closest(".margo-page-actions");
    var status = actions ? actions.querySelector("[data-margo-copy-status]") : null;
    var markdownURL = button.dataset.margoMarkdownUrl || button.getAttribute("href");
    if (!markdownURL) return;
    button.setAttribute("aria-disabled", "true");
    fetch(markdownURL, { credentials: "same-origin" })
      .then(function (response) {
        if (!response.ok) throw new Error("Markdown source unavailable");
        return response.text();
      })
      .then(copyText)
      .then(function () {
        if (status) status.textContent = "Copied";
      })
      .catch(function () {
        if (status) status.textContent = "Copy failed";
      })
      .then(function () { button.removeAttribute("aria-disabled"); });
  });
})();`

const pageActionsCSS = `.margo-page-heading {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  column-gap: 1rem;
  row-gap: 0.75rem;
  align-items: start;
}
.margo-page-heading__title {
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
  min-inline-size: 0;
  grid-column: 1;
  grid-row: 1;
}
.margo-page-heading__title h1 {
  min-inline-size: 0;
  margin-block-end: 0 !important;
}
.margo-page-heading__anchor {
  position: absolute;
  inline-size: 1px;
  block-size: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  white-space: nowrap;
  border: 0;
}
.margo-page-heading__anchor:focus-visible {
  position: static;
  inline-size: auto;
  block-size: auto;
  margin: 0;
  overflow: visible;
  clip: auto;
  white-space: normal;
  color: var(--margo-accent, var(--color-primary, #155eef));
  font-size: 1rem;
  font-weight: 700;
  text-decoration: none;
  opacity: 1;
}
.margo-page-actions {
  position: relative;
  z-index: 1;
  grid-column: 2;
  grid-row: 1 / span 2;
  align-self: start;
  justify-self: end;
  margin-block-start: 0.25rem;
  color: var(--margo-text-strong, var(--color-on-surface-strong, #0b1220));
}
.margo-page-actions [data-split-button] > a {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--color-on-primary, #ffffff);
  text-decoration: none;
}
.dark .margo-page-actions [data-split-button] > a { color: var(--color-on-primary-dark, #ffffff); }
.margo-page-actions [data-split-button] { max-inline-size: 100%; }
.margo-page-actions [data-popover-panel] { z-index: 40; }
.margo-page-actions__status {
  display: block;
  min-block-size: 1.1rem;
  margin-block-start: 0.15rem;
  padding: 0.15rem 0.6rem 0.3rem;
  color: var(--margo-text, var(--color-on-surface, #17202a));
  font-size: 0.75rem;
}
@media (max-width: 42rem) {
  .margo-page-heading { grid-template-columns: minmax(0, 1fr); }
  .margo-page-heading__title { grid-column: 1; grid-row: 1; }
  .margo-page-actions {
    grid-column: 1;
    grid-row: auto;
    justify-self: start;
    margin-block-start: 0;
  }
  .margo-page-actions [data-popover-panel] { left: 0 !important; right: auto !important; }
}
@media print {
  .margo-page-actions, .margo-page-heading__anchor { display: none !important; }
  .margo-page-heading__title { display: block; }
}
/* Goshtoso owns split-button and menu visuals; Margo only owns placement. */
.margo-document .margo-page-actions {
  max-inline-size: none;
  margin-block: 0.25rem;
  padding: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
}
.margo-document .margo-page-heading > .margo-document__lead {
  grid-column: 1;
  grid-row: 2;
  margin-block: 0;
}
.margo-document .margo-page-actions__status {
  margin: 0;
  padding: 0.15rem 0.6rem 0.3rem;
}`

const pageActionsScriptPath = "margo-assets/page-actions.js"
const pageActionsStylePath = "margo-assets/page-actions.css"
const pageActionsIconSpritePath = "margo-assets/icons/page-actions.svg"

const (
	pageActionRootID         = "margo-page-actions"
	pageActionCopyIDSuffix   = "copy"
	pageActionMarkdownSuffix = "view-markdown"
	pageActionDownloadSuffix = "download-pdf"
	pageActionPrintSuffix    = "print"
)

type pageActionIDs struct {
	root     string
	copy     string
	markdown string
	download string
	print    string
}

func pageActionIDsFor(page Page) pageActionIDs {
	token := strings.TrimSuffix(page.Output, path.Ext(page.Output))
	token = strings.NewReplacer("/", "-", "_", "-").Replace(token)
	token = strings.Trim(token, "-")
	if token == "" {
		token = "page"
	}
	prefix := pageActionRootID + "-" + token
	return pageActionIDs{
		root:     prefix,
		copy:     prefix + "-" + pageActionCopyIDSuffix,
		markdown: prefix + "-" + pageActionMarkdownSuffix,
		download: prefix + "-" + pageActionDownloadSuffix,
		print:    prefix + "-" + pageActionPrintSuffix,
	}
}

func pageActionsForMetadata(metadata margo.Metadata) *margo.PageActions {
	if metadata.Margo.Actions == nil || (!metadata.Margo.Actions.Markdown && !metadata.Margo.Actions.PDF) {
		return nil
	}
	actions := *metadata.Margo.Actions
	if actions.PDF {
		actions.Markdown = true
	}
	return &actions
}

func pageImageOverflowForMetadata(metadata margo.Metadata) string {
	if metadata.Margo.Page == nil {
		return ""
	}
	return metadata.Margo.Page.ImageOverflow
}

func pageMarkdownOutput(output string) string {
	return strings.TrimSuffix(output, path.Ext(output)) + ".md"
}

func pagePDFOutput(output string) string {
	return strings.TrimSuffix(output, path.Ext(output)) + ".pdf"
}

func (b *builder) ensurePageActionAssets() error {
	if err := b.addArtifact(pageActionsScriptPath, []byte(pageActionsScript)); err != nil {
		return err
	}
	if b.request.Assets != AssetsInline {
		if err := b.addArtifact(pageActionsIconSpritePath, pageActionsIconSprite); err != nil {
			return err
		}
	}
	if b.config == nil && b.request.Assets != AssetsInline {
		if err := b.addArtifact(pageActionsStylePath, []byte(pageActionsCSS)); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) addDeclaredPageArtifacts(ctx context.Context, source Source, page Page, document *margo.Document) error {
	if page.Actions == nil {
		return nil
	}
	if page.Actions.Markdown {
		if err := b.addArtifact(pageMarkdownOutput(page.Output), source.Content); err != nil {
			return err
		}
	}
	if !page.Actions.PDF || page.Actions.UsesClientPDF() {
		return nil
	}
	if document == nil {
		return diagnostic("site.pdf_source_invalid", "PDF publication has no compiled document", "Compile the page before generating its PDF artifact.", page.Source)
	}
	pdfBytes, err := b.pdfArtifact(ctx, page, document)
	if err != nil {
		return err
	}
	return b.addArtifact(pagePDFOutput(page.Output), pdfBytes)
}

func (b *builder) injectPageActions(ctx context.Context, document []byte, page Page) ([]byte, error) {
	hasActions := page.Actions != nil && (page.Actions.Markdown || page.Actions.PDF)
	if !hasActions && page.ImageOverflow != string(pdf.ImageOverflowAllow) {
		return document, nil
	}
	root, err := html.Parse(bytes.NewReader(document))
	if err != nil {
		return nil, diagnostic("site.html_invalid", err.Error(), "Report the generated page action defect.", page.Source)
	}
	if page.ImageOverflow == string(pdf.ImageOverflowAllow) {
		documentRoot := firstElement(root, "html")
		if documentRoot == nil {
			return nil, diagnostic("site.html_invalid", "generated page has no html root", "Render a complete HTML document.", page.Source)
		}
		setHTMLAttribute(documentRoot, "data-margo-image-overflow", string(pdf.ImageOverflowAllow))
	}
	if !hasActions {
		var output bytes.Buffer
		if err := html.Render(&output, root); err != nil {
			return nil, diagnostic("site.html_invalid", err.Error(), "Report the generated page defect.", page.Source)
		}
		return output.Bytes(), nil
	}
	if err := b.ensurePageActionAssets(); err != nil {
		return nil, err
	}
	article := findDocumentArticle(root)
	if article == nil {
		return nil, diagnostic("site.page_actions_invalid", "page action toolbar has no document article", "Keep one article.margo-document in the generated page.", page.Source)
	}
	h1 := firstElement(article, "h1")
	if h1 == nil || h1.Parent == nil {
		return nil, diagnostic("site.page_actions_invalid", "page action toolbar has no document h1", "Keep one h1 in the generated page.", page.Source)
	}
	parent := h1.Parent
	next := h1.NextSibling
	lead := pageHeadingLead(h1)
	if lead != nil {
		if next == lead {
			next = lead.NextSibling
		}
		parent.RemoveChild(lead)
	}
	parent.RemoveChild(h1)

	heading := &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div", Attr: []html.Attribute{{Key: "class", Val: "margo-page-heading"}}}
	title := &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div", Attr: []html.Attribute{{Key: "class", Val: "margo-page-heading__title"}}}
	title.AppendChild(h1)
	if id := attributeValue(h1, "id"); id != "" {
		anchor := &html.Node{Type: html.ElementNode, DataAtom: atom.A, Data: "a", Attr: []html.Attribute{
			{Key: "class", Val: "margo-page-heading__anchor"}, {Key: "href", Val: "#" + id},
			{Key: "aria-label", Val: "Link to page heading"},
		}}
		anchor.AppendChild(&html.Node{Type: html.TextNode, Data: "#"})
		title.AppendChild(anchor)
	}
	heading.AppendChild(title)
	if lead != nil {
		heading.AppendChild(lead)
	}

	head := firstElement(root, "head")
	if head == nil {
		return nil, diagnostic("site.head_missing", "generated page has no head for page action assets", "Render a complete HTML document.", page.Source)
	}
	iconSpriteURL := "/" + pageActionsIconSpritePath
	iconMode := icon.ModeExternal
	if b.request.Assets == AssetsInline {
		iconSpriteURL = ""
		iconMode = icon.ModeInline
		if err := appendInlinePageActionIconSprite(head); err != nil {
			return nil, diagnostic("site.page_actions_invalid", err.Error(), "Keep the iconpack sprite embeddable.", page.Source)
		}
	} else if b.config == nil {
		iconSpriteURL, err = relativeSitePath(path.Dir(page.Output), pageActionsIconSpritePath)
		if err != nil {
			return nil, diagnostic("site.page_actions_invalid", err.Error(), "Keep the iconpack sprite path relative to the page.", page.Source)
		}
	}
	toolbar, err := pageActionsNode(ctx, page, iconSpriteURL, iconMode)
	if err != nil {
		return nil, err
	}
	heading.AppendChild(toolbar)
	if next == nil {
		parent.AppendChild(heading)
	} else {
		parent.InsertBefore(heading, next)
	}

	if b.config == nil {
		if b.request.Assets == AssetsInline {
			head.AppendChild(styleNode(pageActionsCSS))
			head.AppendChild(scriptNode(pageActionsScript))
		} else {
			styleURL, _ := relativeSitePath(path.Dir(page.Output), pageActionsStylePath)
			scriptURL, _ := relativeSitePath(path.Dir(page.Output), pageActionsScriptPath)
			head.AppendChild(linkNode(styleURL))
			head.AppendChild(scriptNodeURL(scriptURL))
		}
	}
	var output bytes.Buffer
	if err := html.Render(&output, root); err != nil {
		return nil, diagnostic("site.html_invalid", err.Error(), "Report the generated page action defect.", page.Source)
	}
	return output.Bytes(), nil
}

func pageHeadingLead(h1 *html.Node) *html.Node {
	for sibling := h1.NextSibling; sibling != nil; sibling = sibling.NextSibling {
		if sibling.Type == html.TextNode && strings.TrimSpace(sibling.Data) == "" {
			continue
		}
		if sibling.Type == html.ElementNode && hasClass(sibling, "margo-document__lead") {
			return sibling
		}
		break
	}
	return nil
}

func findDocumentArticle(root *html.Node) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if found != nil {
			return
		}
		if node.Type == html.ElementNode && node.Data == "article" && hasClass(node, "margo-document") {
			found = node
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return found
}

func hasClass(node *html.Node, expected string) bool {
	return strings.Contains(" "+attributeValue(node, "class")+" ", " "+expected+" ")
}

func pageActionsNode(ctx context.Context, page Page, iconSpriteURL string, iconMode icon.Mode) (*html.Node, error) {
	ids := pageActionIDsFor(page)
	markdownURL := relativeAssetPath(path.Dir(page.Output), pageMarkdownOutput(page.Output))
	items := make([]dropdown.Item, 0, 2)
	if page.Actions != nil && page.Actions.Markdown {
		items = append(items, dropdown.Item{
			ID:           ids.markdown,
			Label:        "View as Markdown",
			Caption:      "View this page as plain text",
			Href:         markdownURL,
			Target:       "_blank",
			Rel:          "noopener noreferrer",
			Icon:         pageActionIcon(iconSpriteURL, iconMode, appicons.IconHeroiconsDocumentText16SolidDocumentText),
			TrailingIcon: pageActionIcon(iconSpriteURL, iconMode, appicons.IconHeroiconsArrowTopRight16SolidArrowTopRightOnSquare),
		})
	}
	if page.Actions != nil && page.Actions.PDF {
		if page.Actions.UsesClientPDF() {
			items = append(items, dropdown.Item{
				ID:      ids.print,
				Label:   "Print / Save PDF",
				Caption: "Open the browser print dialog",
				OnClick: "$event.preventDefault()",
				Icon:    pageActionIcon(iconSpriteURL, iconMode, appicons.IconHeroiconsPrinter16SolidPrinter),
			})
		} else {
			items = append(items, dropdown.Item{
				ID:      ids.download,
				Label:   "Download PDF",
				Caption: "Download the pre-rendered document",
				Href:    relativeAssetPath(path.Dir(page.Output), pagePDFOutput(page.Output)),
				Icon:    pageActionIcon(iconSpriteURL, iconMode, appicons.IconHeroiconsArrowDownTray16SolidArrowDownTray),
			})
		}
	}
	component := splitbutton.SplitButton(splitbutton.Config{
		ID:        ids.root,
		Primary:   splitbutton.Action{ID: ids.copy, Label: pageActionSummary(page), Href: markdownURL, Icon: pageActionIcon(iconSpriteURL, iconMode, appicons.IconHeroiconsCopy16SolidClipboard)},
		MenuLabel: "More page actions",
		MenuAlign: dropdown.AlignEnd,
		Tone:      button.TonePrimary,
		Size:      button.SizeSmall,
		Sections:  []dropdown.Section{{Items: items}},
	})
	var markup bytes.Buffer
	if err := component.Render(ctx, &markup); err != nil {
		return nil, diagnostic("site.page_actions_invalid", err.Error(), "Keep the Goshtoso split button renderable.", page.Source)
	}
	contextNode := &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := html.ParseFragment(bytes.NewReader(markup.Bytes()), contextNode)
	if err != nil || len(nodes) != 1 {
		return nil, diagnostic("site.page_actions_invalid", "Goshtoso split button markup is malformed", "Keep the generated action controls valid HTML.", page.Source)
	}
	root := nodes[0]
	primary := elementByID(root, ids.copy)
	if primary == nil {
		return nil, diagnostic("site.page_actions_invalid", "Goshtoso split button has no copy action", "Keep the split button primary action renderable.", page.Source)
	}
	setHTMLAttribute(primary, "data-margo-copy-page", "")
	setHTMLAttribute(primary, "data-margo-markdown-url", markdownURL)
	if page.Actions != nil && page.Actions.PDF {
		if page.Actions.UsesClientPDF() {
			printAction := elementByID(root, ids.print)
			if printAction == nil {
				return nil, diagnostic("site.page_actions_invalid", "Goshtoso split button has no print action", "Keep the client PDF menu item renderable.", page.Source)
			}
			setHTMLAttribute(printAction, "data-margo-print-page", "")
		} else {
			downloadAction := elementByID(root, ids.download)
			if downloadAction == nil {
				return nil, diagnostic("site.page_actions_invalid", "Goshtoso split button has no PDF download action", "Keep the pre-rendered PDF menu item renderable.", page.Source)
			}
			setHTMLAttribute(downloadAction, "download", "")
		}
	}
	panel := firstElementWithAttribute(root, "data-popover-panel")
	if panel == nil {
		return nil, diagnostic("site.page_actions_invalid", "Goshtoso split button has no menu panel", "Keep the split button dropdown panel renderable.", page.Source)
	}
	status := &html.Node{Type: html.ElementNode, DataAtom: atom.Span, Data: "span", Attr: []html.Attribute{
		{Key: "class", Val: "margo-page-actions__status"}, {Key: "data-margo-copy-status", Val: ""}, {Key: "role", Val: "status"}, {Key: "aria-live", Val: "polite"},
	}}
	panel.AppendChild(status)
	toolbar := &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div", Attr: []html.Attribute{{Key: "class", Val: "margo-page-actions"}}}
	toolbar.AppendChild(root)
	return toolbar, nil
}

func pageActionIcon(spriteURL string, mode icon.Mode, symbol icon.Symbol) icon.Instance {
	return icon.Icon(icon.Config{
		SpriteURL:  spriteURL,
		Symbol:     symbol,
		Size:       icon.SizeSM,
		Decorative: true,
		Mode:       mode,
	})
}

func appendInlinePageActionIconSprite(head *html.Node) error {
	// Parse the SVG as ordinary body content. A <head> parsing context may
	// discard foreign SVG content before the fragment reaches the document.
	contextNode := &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := html.ParseFragment(bytes.NewReader(pageActionsIconSprite), contextNode)
	if err != nil {
		return err
	}
	var sprite *html.Node
	for _, node := range nodes {
		if node.Type == html.TextNode && strings.TrimSpace(node.Data) == "" {
			continue
		}
		if node.Type != html.ElementNode || node.Data != "svg" || sprite != nil {
			return fmt.Errorf("iconpack sprite must contain one SVG root")
		}
		sprite = node
	}
	if sprite == nil {
		return fmt.Errorf("iconpack sprite must contain one SVG root")
	}
	setHTMLAttribute(sprite, "hidden", "")
	setHTMLAttribute(sprite, "aria-hidden", "true")
	head.AppendChild(sprite)
	return nil
}

func elementByID(root *html.Node, id string) *html.Node {
	if root.Type == html.ElementNode && attributeValue(root, "id") == id {
		return root
	}
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if found := elementByID(child, id); found != nil {
			return found
		}
	}
	return nil
}

func firstElementWithAttribute(root *html.Node, key string) *html.Node {
	if root.Type == html.ElementNode {
		for _, attr := range root.Attr {
			if attr.Key == key {
				return root
			}
		}
	}
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if found := firstElementWithAttribute(child, key); found != nil {
			return found
		}
	}
	return nil
}

func pageActionSummary(page Page) string {
	if page.Actions != nil && page.Actions.Markdown {
		return "Copy page"
	}
	return "Page actions"
}

func styleNode(content string) *html.Node {
	node := &html.Node{Type: html.ElementNode, DataAtom: atom.Style, Data: "style", Attr: []html.Attribute{{Key: "data-margo-page-actions", Val: ""}}}
	node.AppendChild(&html.Node{Type: html.TextNode, Data: content})
	return node
}

func scriptNode(content string) *html.Node {
	node := &html.Node{Type: html.ElementNode, DataAtom: atom.Script, Data: "script", Attr: []html.Attribute{{Key: "data-margo-page-actions", Val: ""}}}
	node.AppendChild(&html.Node{Type: html.TextNode, Data: content})
	return node
}

func scriptNodeURL(source string) *html.Node {
	return &html.Node{Type: html.ElementNode, DataAtom: atom.Script, Data: "script", Attr: []html.Attribute{{Key: "defer", Val: ""}, {Key: "src", Val: source}}}
}

func linkNode(source string) *html.Node {
	return &html.Node{Type: html.ElementNode, DataAtom: atom.Link, Data: "link", Attr: []html.Attribute{{Key: "rel", Val: "stylesheet"}, {Key: "href", Val: source}}}
}

func (b *builder) pdfArtifact(ctx context.Context, page Page, document *margo.Document) ([]byte, error) {
	if err := b.ensurePDFEngine(ctx); err != nil {
		return nil, err
	}
	rendered, err := b.request.Compiler.Render(ctx, document, margo.WithTableSort(margo.TableSortClient), margo.WithRenderTarget(margo.TargetPDF))
	if err != nil {
		return nil, err
	}
	brandName, logo, err := b.pdfBrandIdentity()
	if err != nil {
		return nil, err
	}
	component, err := margo.RenderStandalone(rendered, margo.WithPDFBrand(brandName, logo))
	if err != nil {
		return nil, err
	}
	var htmlBytes bytes.Buffer
	if err := component.Render(ctx, &htmlBytes); err != nil {
		return nil, err
	}
	materializedHTML, err := b.materializePDFImages(htmlBytes.Bytes(), page)
	if err != nil {
		return nil, err
	}
	instance, err := b.pdfInstances.Next()
	if err != nil {
		return nil, err
	}
	descriptor, err := rendered.RuntimeDescriptor(instance)
	if err != nil {
		return nil, err
	}
	executionID := margo.ExecutionID("site-" + margo.ArtifactDigestOf([]byte(page.Source+"\x00"+page.Output)).String())
	result, err := b.pdfEngine.Export(ctx, pdf.Request{
		HTML: materializedHTML, Runtime: descriptor, ExecutionID: executionID,
		Page: pagePDFConfig(document.Metadata()), RelativeLinks: pdf.RelativeLinksStrip,
	})
	if err != nil {
		return nil, err
	}
	if !bytes.HasPrefix(result.PDF, []byte("%PDF-")) {
		return nil, diagnostic("site.pdf_invalid", "selected PDF engine returned invalid bytes", "Use a working Chromium or native PDF engine.", page.Source)
	}
	if err := margo.ValidateRuntimeReport(descriptor, executionID, result.Runtime); err != nil {
		return nil, fmt.Errorf("site.pdf_runtime_invalid: %w", err)
	}
	if err := result.Engine.Validate(); err != nil {
		return nil, fmt.Errorf("site.pdf_engine_invalid: %w", err)
	}
	return append([]byte(nil), result.PDF...), nil
}

func (b *builder) materializePDFImages(document []byte, page Page) ([]byte, error) {
	root, err := html.Parse(bytes.NewReader(document))
	if err != nil {
		return nil, diagnostic("site.pdf_asset_invalid", err.Error(), "Use local images that can be embedded in the pre-rendered PDF.", page.Source)
	}
	var visit func(*html.Node) error
	visit = func(node *html.Node) error {
		if node.Type == html.ElementNode && (node.Data == "img" || node.Data == "source") {
			for index := range node.Attr {
				attribute := &node.Attr[index]
				switch attribute.Key {
				case "src":
					attribute.Val, err = b.materializePDFImageURL(attribute.Val, page)
					if err != nil {
						return err
					}
				case "srcset":
					attribute.Val, err = b.materializePDFSourceSet(attribute.Val, page)
					if err != nil {
						return err
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(root); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := html.Render(&output, root); err != nil {
		return nil, diagnostic("site.pdf_asset_invalid", err.Error(), "Use local images that can be embedded in the pre-rendered PDF.", page.Source)
	}
	return output.Bytes(), nil
}

func (b *builder) materializePDFImageURL(value string, page Page) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || strings.HasPrefix(value, "//") {
		if parsed != nil && strings.EqualFold(parsed.Scheme, "data") {
			return value, nil
		}
		return "", diagnostic("site.pdf_asset_external", fmt.Sprintf("image %q is not a local asset", value), "Use a local image so the pre-rendered PDF stays offline.", page.Source)
	}
	if parsed.Path == "" {
		return "", diagnostic("site.pdf_asset_invalid", "image source is empty", "Use a local image so the pre-rendered PDF stays offline.", page.Source)
	}
	assetPath := path.Clean(path.Join(path.Dir(page.Source), parsed.Path))
	if assetPath == ".." || strings.HasPrefix(assetPath, "../") {
		return "", diagnostic("site.pdf_asset_external", fmt.Sprintf("image %q escapes the site source root", value), "Use an image below the site source directory.", page.Source)
	}
	asset, ok := b.assets[assetPath]
	if !ok {
		return "", diagnostic("site.pdf_asset_unreadable", fmt.Sprintf("image %q was not materialized before PDF rendering", assetPath), "Use a readable local image below the site source directory.", page.Source)
	}
	return "data:" + asset.mediaType + ";base64," + base64.StdEncoding.EncodeToString(asset.content), nil
}

func (b *builder) materializePDFSourceSet(value string, page Page) (string, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "data:") {
		return value, nil
	}
	parts := strings.Split(value, ",")
	for index, part := range parts {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) == 0 || len(fields) > 2 {
			return "", diagnostic("site.pdf_asset_invalid", fmt.Sprintf("srcset %q is malformed", value), "Use local image sources with optional width descriptors.", page.Source)
		}
		materialized, err := b.materializePDFImageURL(fields[0], page)
		if err != nil {
			return "", err
		}
		fields[0] = materialized
		parts[index] = strings.Join(fields, " ")
	}
	return strings.Join(parts, ", "), nil
}

func (b *builder) pdfBrandIdentity() (string, margo.AssetRef, error) {
	if b.config == nil {
		logo, err := margo.EmbeddedAsset("logo.svg")
		if err != nil {
			return "", margo.AssetRef{}, fmt.Errorf("site.pdf_brand_logo: %w", err)
		}
		return "Margo", logo, nil
	}

	logoAsset, ok := b.assets[b.config.Site.Logo]
	if !ok {
		return "", margo.AssetRef{}, fmt.Errorf("site.pdf_brand_logo: configured logo %q was not staged", b.config.Site.Logo)
	}
	return b.config.Site.Name, margo.AssetRef{
		Path:      b.config.Site.Logo,
		MediaType: logoAsset.mediaType,
		Content:   append([]byte(nil), logoAsset.content...),
	}, nil
}

func (b *builder) ensurePDFEngine(ctx context.Context) error {
	if b.pdfEngine != nil {
		return nil
	}
	discovery, err := engines.Discover(ctx, engines.Request{Mode: engines.ModeAuto}, engines.Probe{})
	if err != nil {
		return err
	}
	b.pdfEngine, _, err = discovery.Select()
	return err
}

func pagePDFConfig(metadata margo.Metadata) pdf.PageConfig {
	config := pdf.PageConfig{Size: pdf.PageA4, Orientation: pdf.Portrait, Margins: pdf.Margins{Top: 24, Right: 22, Bottom: 26, Left: 22}}
	if metadata.Margo.Page == nil {
		return config
	}
	if metadata.Margo.Page.Size != "" {
		config.Size = pdf.PageSize(metadata.Margo.Page.Size)
	}
	if metadata.Margo.Page.Orientation != "" {
		config.Orientation = pdf.Orientation(metadata.Margo.Page.Orientation)
	}
	if metadata.Margo.Page.ImageOverflow != "" {
		config.ImageOverflow = pdf.ImageOverflowPolicy(metadata.Margo.Page.ImageOverflow)
	}
	if margins := metadata.Margo.Page.Margins; margins != nil {
		if margins.Top != nil {
			config.Margins.Top = pdf.Millimeters(*margins.Top)
		}
		if margins.Right != nil {
			config.Margins.Right = pdf.Millimeters(*margins.Right)
		}
		if margins.Bottom != nil {
			config.Margins.Bottom = pdf.Millimeters(*margins.Bottom)
		}
		if margins.Left != nil {
			config.Margins.Left = pdf.Millimeters(*margins.Left)
		}
	}
	return config
}
