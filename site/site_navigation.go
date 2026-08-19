package site

import (
	"bytes"
	"context"
	stdhtml "html"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/a-h/templ"
	goshtosoassets "github.com/araihu/goshtoso/assets"
	"github.com/araihu/goshtoso/components/head"
	"github.com/araihu/goshtoso/components/navbar"
	"github.com/araihu/goshtoso/components/search"
	"github.com/araihu/goshtoso/components/sidebar"
)

// siteNavigationFragment renders Margo's public site chrome for a configured
// profile page. The component owns only semantic navigation data; Goshtoso
// remains the owner of its internal markup and responsive behavior.
func (b *builder) siteNavigationFragment(page Page) (string, error) {
	searchConfig := b.siteSearchConfig(page.Locale)
	brand := templ.Raw(`<span class="margo-site-brand"><img src="` + stdhtml.EscapeString(relativeAssetPath(path.Dir(page.Output), b.config.Site.Logo)) + `" alt="` + stdhtml.EscapeString(b.config.Site.Name) + `"><span>` + stdhtml.EscapeString(b.config.Site.Name) + `</span></span>`)
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

	primaryLinks := make([]navbar.NavLink, 0, 1)
	if repository := strings.TrimSpace(b.config.Site.RepositoryURL); repository != "" {
		primaryLinks = append(primaryLinks, navbar.NavLink{
			Label: "Repository",
			Href:  repository,
			LinkAttrs: templ.Attributes{
				"target": "_blank",
				"rel":    "noopener noreferrer",
			},
		})
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
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		if err := navbar.Navbar(navbar.Config{
			Brand:     brand,
			BrandHref: b.siteHomeHref(page),
			Links:     primaryLinks,
			Actions: []navbar.ActionItem{{
				Content: searchAction,
				// Navbar duplicates right actions in the mobile menu. Keep the
				// global search field in one left action slot;
				// SearchModal still provides the keyboard shortcut globally.
				Position: navbar.ActionLeft,
			}},
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
	return string(markup), nil
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
			Keywords:    []string{page.Source, page.Output},
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
	route := "/" + strings.TrimPrefix(output, "/")
	if home {
		route = "/"
	}
	basePath := normalizedBasePath(b.config.BasePath)
	if basePath != "/" {
		route = strings.TrimSuffix(basePath, "/") + route
	}
	return route
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

func withoutGoshtosoStylesheet(markup []byte) []byte {
	start := bytes.Index(markup, []byte(`<link rel="stylesheet"`))
	if start < 0 {
		return markup
	}
	relativeEnd := bytes.Index(markup[start:], []byte(`/>`))
	if relativeEnd < 0 {
		return markup
	}
	end := start + relativeEnd + len([]byte(`/>`))
	result := make([]byte, 0, len(markup)-(end-start))
	result = append(result, markup[:start]...)
	result = append(result, markup[end:]...)
	return result
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
