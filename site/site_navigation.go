package site

import (
	"bytes"
	"context"
	"fmt"
	stdhtml "html"
	"io"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/a-h/templ"
	goshtosoassets "github.com/araihu/goshtoso/assets"
	"github.com/araihu/goshtoso/components/head"
	"github.com/araihu/goshtoso/components/navbar"
	"github.com/araihu/goshtoso/components/search"
	"github.com/araihu/goshtoso/components/sidebar"
	margo "github.com/araihu/margo"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// siteNavigationFragment renders Margo's public site chrome for a configured
// profile page. The component owns only semantic navigation data; Goshtoso
// remains the owner of its internal markup and responsive behavior.
func (b *builder) siteNavigationFragment(page Page) (string, error) {
	searchConfig := b.siteSearchConfig(page.Locale)
	brand := templ.Raw(`<span class="margo-site-brand"><img src="` + stdhtml.EscapeString(relativeAssetPath(path.Dir(page.Output), b.config.Site.Logo)) + `" alt=""><span>` + stdhtml.EscapeString(b.config.Site.Name) + `</span></span>`)
	secondaryLinks := make([]navbar.SecondaryLink, 0, len(b.config.Navigation.Families))
	for _, family := range b.config.Navigation.Families {
		overview, ok := b.familyOverviewPage(page, family)
		if !ok {
			continue
		}
		current := navbar.SecondaryCurrentNone
		if family.ID == page.Family {
			current = navbar.SecondaryCurrentLocation
		}
		secondaryLinks = append(secondaryLinks, navbar.SecondaryLink{
			Label:   family.Label,
			Href:    b.sitePageHref(overview),
			Current: current,
			LinkAttrs: templ.Attributes{
				"data-margo-family-link": family.ID,
			},
		})
	}

	var repositoryAction templ.Component
	if repository := strings.TrimSpace(b.config.Site.RepositoryURL); repository != "" {
		repositoryAction = githubRepositoryAction(repository)
	}
	searchAction := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		if _, err := io.WriteString(writer, `<div class="margo-site-search">`); err != nil {
			return err
		}
		if err := search.SearchField(searchConfig).Render(ctx, writer); err != nil {
			return err
		}
		_, err := io.WriteString(writer, `</div>`)
		return err
	})
	actions := []navbar.ActionItem{{
		Content: searchAction,
		// Navbar duplicates right actions in the mobile menu. Keep the
		// global search field in one left action slot;
		// SearchModal still provides the keyboard shortcut globally.
		Position: navbar.ActionLeft,
	}}
	if repositoryAction != nil {
		actions = append(actions, navbar.ActionItem{Content: repositoryAction, Position: navbar.ActionRight})
	}
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		if err := navbar.Navbar(navbar.Config{
			Brand:     brand,
			BrandHref: b.siteHomeHref(page),
			Actions:   actions,
			Secondary: &navbar.SecondaryConfig{
				Links:      secondaryLinks,
				AriaLabel:  "Documentation families",
				Scrollable: true,
				RootClass:  "margo-site-family-links",
				RootAttrs:  templ.Attributes{"data-margo-family-navigation": "true"},
			},
			NavClass: "margo-site-navbar",
			NavAttrs: templ.Attributes{"data-margo-global-navigation": "true"},
		}).Render(ctx, writer); err != nil {
			return err
		}
		return search.SearchModal(searchConfig).Render(ctx, writer)
	})
	markup, err := renderComponentBytes(component)
	if err != nil {
		return "", err
	}
	return string(hardenSearchMarkup(markup, searchConfig)), nil
}

func githubRepositoryAction(repository string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, `<a class="margo-site-repository" data-margo-repository-link="true" href="`+stdhtml.EscapeString(repository)+`" target="_blank" rel="noopener noreferrer" aria-label="Repository" title="Repository"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path fill="currentColor" d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.757 3.633 18.4 3.633 18.4c-1.087-.744.084-.729.084-.729 1.205.084 1.84 1.237 1.84 1.237 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 15.592 24 12.297c0-6.627-5.373-12-12-12"/></svg><span class="sr-only">Repository</span></a>`)
		return err
	})
}

func hardenSearchMarkup(markup []byte, config search.Config) []byte {
	root, err := html.ParseFragment(bytes.NewReader(markup), &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div"})
	if err != nil {
		return markup
	}
	id := configID(config)
	fieldID := id + "-input"
	resultsID := id + "-results"
	statusID := id + "-status"
	var field, modal, input, listbox *html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			if hasAttribute(node, "data-search-field") && attributeValue(node, "data-search-id") == id {
				field = node
			}
			if hasAttribute(node, "data-search-modal") && attributeValue(node, "data-search-id") == id {
				modal = node
			}
			if attributeValue(node, "id") == fieldID {
				input = node
			}
			if node.Data == "div" && attributeValue(node, "role") == "listbox" && modal != nil {
				listbox = node
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	for _, node := range root {
		walk(node)
	}
	if field == nil || modal == nil || input == nil || listbox == nil {
		return markup
	}
	setHTMLAttribute(input, "role", "combobox")
	setHTMLAttribute(input, "aria-controls", resultsID)
	setHTMLAttribute(input, "aria-autocomplete", "list")
	setHTMLAttribute(input, "aria-haspopup", "listbox")
	setHTMLAttribute(input, "aria-expanded", "false")
	setHTMLAttribute(listbox, "id", resultsID)
	setHTMLAttribute(listbox, "aria-live", "off")
	setHTMLAttribute(modal, "data-margo-search-a11y", "true")
	setHTMLAttribute(field, "data-margo-search-a11y", "true")
	for index, child := range childElements(listbox) {
		if child.Type != html.ElementNode || attributeValue(child, "role") != "option" {
			continue
		}
		if attributeValue(child, "id") == "" {
			setHTMLAttribute(child, "id", fmt.Sprintf("%s-result-%d", id, index))
		}
		setHTMLAttribute(child, "aria-selected", "false")
	}
	if firstElementByAttribute(modal, "data-margo-search-status") == nil {
		status := &html.Node{Type: html.ElementNode, DataAtom: atom.P, Data: "p", Attr: []html.Attribute{
			{Key: "id", Val: statusID},
			{Key: "class", Val: "margo-search-status"},
			{Key: "data-margo-search-status", Val: ""},
			{Key: "role", Val: "status"},
			{Key: "aria-live", Val: "polite"},
			{Key: "aria-atomic", Val: "true"},
		}}
		modal.AppendChild(status)
	}
	if firstElementByAttribute(modal, "data-margo-search-clear") == nil {
		clear := &html.Node{Type: html.ElementNode, DataAtom: atom.Button, Data: "button", Attr: []html.Attribute{
			{Key: "type", Val: "button"},
			{Key: "class", Val: "margo-search-clear"},
			{Key: "data-margo-search-clear", Val: ""},
			{Key: "aria-label", Val: "Clear search"},
			{Key: "x-cloak", Val: ""},
			{Key: "x-show", Val: "query.trim().length > 0"},
		}}
		clear.AppendChild(&html.Node{Type: html.TextNode, Data: "Clear"})
		container := input.Parent
		if container != nil {
			container.InsertBefore(clear, input.NextSibling)
		}
	}
	var output bytes.Buffer
	for _, node := range root {
		if err := html.Render(&output, node); err != nil {
			return markup
		}
	}
	return output.Bytes()
}

func configID(config search.Config) string {
	if strings.TrimSpace(config.ID) == "" {
		return "search"
	}
	return strings.TrimSpace(config.ID)
}

func hasAttribute(node *html.Node, key string) bool {
	return attributeIndex(node, key) >= 0
}

func childElements(node *html.Node) []*html.Node {
	children := make([]*html.Node, 0)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		children = append(children, child)
	}
	return children
}

func firstElementByAttribute(root *html.Node, key string) *html.Node {
	if root.Type == html.ElementNode && hasAttribute(root, key) {
		return root
	}
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if found := firstElementByAttribute(child, key); found != nil {
			return found
		}
	}
	return nil
}

// familyNavigationFragment renders only the active family's local pages.
func (b *builder) familyNavigationFragment(page Page) (string, error) {
	pages := b.familyPages(page)
	items := make([]sidebar.Item, 0, len(pages))
	for _, candidate := range pages {
		items = append(items, sidebar.Item{
			ID:     "margo-family-" + strings.NewReplacer("/", "-", ".", "-").Replace(candidate.Source),
			Label:  candidate.Title,
			Href:   b.sitePageHref(candidate),
			Active: candidate.Source == page.Source,
		})
	}
	familyLabel := page.Family
	for _, family := range b.config.Navigation.Families {
		if family.ID == page.Family {
			familyLabel = family.Label
			break
		}
	}
	component := sidebar.Sidebar(sidebar.Config{
		Sections: []sidebar.Section{{
			Title: familyLabel,
			Items: items,
		}},
		RootClass:       "margo-family-sidebar",
		DisableSkipLink: true,
	})
	markup, err := renderComponentBytes(component)
	if err != nil {
		return "", err
	}
	return string(markup), nil
}

// tocFragment builds a Margo-owned table of contents from the already-rendered
// article. The renderer owns deterministic heading IDs; this fragment only
// projects those public IDs into links and never depends on App Shell markup.
func (b *builder) tocFragment(article []byte, locale string) string {
	root, err := html.ParseFragment(bytes.NewReader(article), &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div"})
	if err != nil {
		return ""
	}
	type heading struct {
		level int
		id    string
		label string
	}
	headings := make([]heading, 0)
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && len(node.Data) == 2 && node.Data[0] == 'h' && node.Data[1] >= '1' && node.Data[1] <= '6' {
			id := strings.TrimSpace(attributeValue(node, "id"))
			label := strings.Join(strings.Fields(htmlText(node)), " ")
			if id != "" && label != "" {
				headings = append(headings, heading{level: int(node.Data[1] - '0'), id: id, label: label})
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	for _, node := range root {
		walk(node)
	}
	if len(headings) == 0 {
		return ""
	}
	label := localizedLabel(locale, "toc")
	var builder strings.Builder
	builder.WriteString(`<nav class="margo-toc" aria-label="` + stdhtml.EscapeString(label) + `" data-margo-toc="true"><p class="margo-toc-title">` + stdhtml.EscapeString(label) + `</p><ol data-margo-toc-list="true">`)
	for _, item := range headings {
		builder.WriteString(`<li data-margo-toc-level="` + stdhtml.EscapeString(fmt.Sprint(item.level)) + `"><a data-margo-toc-link="` + stdhtml.EscapeString(item.id) + `" href="#` + stdhtml.EscapeString(item.id) + `">` + stdhtml.EscapeString(item.label) + `</a></li>`)
	}
	builder.WriteString(`</ol></nav>`)
	return builder.String()
}

func htmlText(node *html.Node) string {
	if node.Type == html.TextNode {
		return node.Data
	}
	var builder strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		builder.WriteString(htmlText(child))
	}
	return builder.String()
}

func (b *builder) siteSearchConfig(locale string) search.Config {
	locale = strings.TrimSpace(locale)
	if locale == "" {
		locale = b.config.Locales.Default
	}
	items := make([]search.Item, 0, len(b.configPages))
	for _, page := range b.configPages {
		if page.Locale != locale {
			continue
		}
		items = append(items, search.Item{
			ID:          "margo-site-search-" + strings.NewReplacer("/", "-", ".", "-").Replace(page.Source),
			Title:       page.Title,
			Description: page.Description,
			Href:        b.sitePageHref(page),
			Kind:        "Page",
			Path:        b.sitePageHref(page),
			Section:     page.Family,
			Scope:       page.Family,
			Keywords:    []string{page.Source, b.sitePageHref(page)},
		})
	}
	return search.Config{
		ID:             "margo-site-search",
		Label:          "Search documentation",
		Placeholder:    "Search documentation",
		ShortcutText:   "⌘ K",
		GlobalShortcut: true,
		Items:          items,
		MatchMode:      search.MatchModeFuzzy,
		MaxResults:     8,
		EmptyText:      "No matching pages.",
		RootClass:      "margo-site-search-control",
		TriggerClass:   "margo-site-search-trigger",
	}
}

func (b *builder) familyPages(page Page) []Page {
	pages := make([]Page, 0, len(b.configPages))
	for _, candidate := range b.configPages {
		if candidate.Locale == page.Locale && candidate.Family == page.Family {
			pages = append(pages, candidate)
		}
	}
	sort.SliceStable(pages, func(left, right int) bool {
		return pageRouteLess(pages[left], pages[right])
	})
	if len(pages) < 2 {
		return pages
	}
	family, ok := b.familyConfig(page.Family)
	if !ok {
		return pages
	}
	overviewRoute := routeKey(family.Overview, b.config.Locales)
	overviewIndex := -1
	for index, candidate := range pages {
		if routeKey(candidate.Source, b.config.Locales) == overviewRoute {
			overviewIndex = index
			break
		}
	}
	if overviewIndex > 0 {
		overview := pages[overviewIndex]
		copy(pages[1:overviewIndex+1], pages[0:overviewIndex])
		pages[0] = overview
	}
	return pages
}

func (b *builder) familyConfig(id string) (FamilyConfig, bool) {
	for _, family := range b.config.Navigation.Families {
		if family.ID == id {
			return family, true
		}
	}
	return FamilyConfig{}, false
}

func (b *builder) familyOverviewPage(page Page, family FamilyConfig) (Page, bool) {
	overviewRoute := routeKey(family.Overview, b.config.Locales)
	for _, candidate := range b.configPages {
		if candidate.Locale == page.Locale && routeKey(candidate.Source, b.config.Locales) == overviewRoute {
			return candidate, true
		}
	}
	return Page{}, false
}

func (b *builder) siteHomeHref(page Page) string {
	output := b.localeHomeOutput(page)
	if page.Locale == b.config.Locales.Default {
		return b.siteOutputHref(output, true)
	}
	return b.siteOutputHref(output, false)
}

func (b *builder) sitePageHref(page Page) string {
	isHome := page.Source == b.config.Site.Home && page.Locale == b.config.Locales.Default
	return b.siteOutputHref(page.Output, isHome)
}

func (b *builder) siteOutputHref(output string, home bool) string {
	return b.publicOutputPath(output, home)
}

// stageGoshtosoNavigationAssets stages the public Goshtoso runtime required
// by navbar, sidebar, and search. The profile path never imports demo/private
// App Shell assets.
func (b *builder) stageGoshtosoNavigationAssets() error {
	handler := goshtosoassets.Handler()
	manifest := goshtosoassets.DefaultRuntimeManifest()
	publicURLs := []string{manifest.Stylesheet.LocalURL}
	for _, dependency := range manifest.Dependencies {
		if dependency.Enabled && dependency.LocalURL != "" {
			publicURLs = append(publicURLs, dependency.LocalURL)
		}
	}
	for _, publicURL := range publicURLs {
		if err := b.stageHandlerAsset(handler, publicURL); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) configuredGoshtosoDependencyBytes() ([]byte, error) {
	if !b.profileMode {
		return nil, nil
	}
	return renderComponentBytes(head.Dependencies(head.WithLocalRuntime()))
}

func withoutGoshtosoStylesheet(markup []byte, requirements margo.HTMLRequirements) []byte {
	styles := make(map[string]struct{})
	identities := make(map[string]struct{})
	for _, requirement := range requirements.List() {
		if requirement.Kind != margo.HTMLStylesheet {
			continue
		}
		identities[requirement.ID] = struct{}{}
		if value := canonicalResourceURL(requirement.LocalURL); value != "" {
			styles[value] = struct{}{}
		}
	}
	root, err := html.ParseFragment(bytes.NewReader(markup), &html.Node{Type: html.ElementNode, DataAtom: atom.Head, Data: "head"})
	if err != nil {
		return markup
	}
	var output bytes.Buffer
	for _, node := range root {
		if node.Type == html.ElementNode && node.Data == "link" && strings.EqualFold(attributeValue(node, "rel"), "stylesheet") {
			identity := attributeValue(node, "data-margo-requirement")
			resource := canonicalResourceURL(attributeValue(node, "href"))
			if _, sameIdentity := identities[identity]; sameIdentity {
				continue
			}
			if _, sameResource := styles[resource]; sameResource {
				continue
			}
		}
		if err := html.Render(&output, node); err != nil {
			return markup
		}
	}
	return output.Bytes()
}

func canonicalResourceURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return value
	}
	parsed.Fragment = ""
	if parsed.Path != "" {
		parsed.Path = path.Clean("/" + strings.TrimPrefix(parsed.Path, "/"))
		parsed.RawPath = ""
	}
	return parsed.String()
}

func addProfileLayoutHook(fragment []byte, layout string) []byte {
	if strings.TrimSpace(layout) == "" {
		return fragment
	}
	marker := []byte(`<div class="margo-frame`)
	index := bytes.Index(fragment, marker)
	if index < 0 {
		return fragment
	}
	attribute := []byte(`<div data-margo-layout="` + stdhtml.EscapeString(layout) + `" class="margo-frame`)
	result := make([]byte, 0, len(fragment)+len(attribute)-len(marker))
	result = append(result, fragment[:index]...)
	result = append(result, attribute...)
	result = append(result, fragment[index+len(marker):]...)
	return result
}
