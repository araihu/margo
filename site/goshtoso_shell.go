package site

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	stdhtml "html"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"

	"github.com/a-h/templ"
	"github.com/araihu/goshtoso-app-shells/componentdocshell"
	shellassets "github.com/araihu/goshtoso-app-shells/componentdocshell/assets"
	goshtosoassets "github.com/araihu/goshtoso/assets"
	"github.com/araihu/goshtoso/components/link"
	"github.com/araihu/goshtoso/components/search"
	"github.com/araihu/goshtoso/components/sidebar"
	internalmermaid "github.com/araihu/margo/internal/mermaid"
	"github.com/araihu/margo/ssg"
	"golang.org/x/net/html"
)

const goshtosoComponentDocShellVersion = "v0.1.6"

const componentDocShellScrollSpyAssetName = "margo-scroll-spy.js"

const componentDocShellScrollSpy = `(function () {
  "use strict";

  var observer;
  var observerGeneration = 0;
  var refreshFrame;
  var tocSyncFrame;
  var tocMutationObserver;
  var tocListNode;
  var tocSyncing = false;
  var explicitReleaseTimer;
  var activeID = "";
  var visibilitySequence = 0;
  var visibleHeadings = new Map();
  var explicitID = "";
  var explicitLock = false;
  var explicitTargetSeen = false;
  var explicitRestore = false;
  var automaticHold = false;
  var mermaidLoadPromise;
  var mermaidRunnerLoadPromise;
  var mermaidRenderQueue = Promise.resolve();
  var shellAssetBase = document.currentScript && document.currentScript.src
    ? new URL("../", document.currentScript.src)
    : new URL("margo-assets/", document.baseURI);
  var mermaidRuntimeURL = new URL("mermaid/` + internalmermaid.RuntimeVersion + `/mermaid.min.js", shellAssetBase).toString();
  var mermaidRunnerURL = new URL("runtime/mermaid-run.js", shellAssetBase).toString();

  function headings() {
    var content = document.getElementById("main-content");
    if (!content) return [];
    return Array.prototype.slice.call(content.querySelectorAll("[data-toc-heading][id]"));
  }

  function idFromHash(hash) {
    var value = (hash || "").replace(/^#/, "");
    if (!value) return "";
    try { return decodeURIComponent(value); } catch (_) { return value; }
  }

  function headingForID(id) {
    if (!id) return null;
    return headings().find(function (heading) { return heading.id === id; }) || null;
  }

  function headingIsVisible(id) {
    var scroller = document.getElementById("page-scroll");
    var heading = headingForID(id);
    if (!scroller || !heading) return false;
    var scrollerRect = scroller.getBoundingClientRect();
    var headingRect = heading.getBoundingClientRect();
    return headingRect.bottom > scrollerRect.top && headingRect.top < scrollerRect.bottom;
  }

  function tocLinks() {
    return Array.prototype.slice.call(document.querySelectorAll(
      "[data-componentdocshell-toc-list] a[data-toc-link], [data-componentdocshell-toc-list] a[href^='#']"
    ));
  }

  function watchTOC(list) {
    if (tocListNode === list) return;
    if (tocMutationObserver) tocMutationObserver.disconnect();
    tocMutationObserver = null;
    tocListNode = list;
    if (!list || !("MutationObserver" in window)) return;
    tocMutationObserver = new MutationObserver(function () {
      if (tocSyncing || !activeID) return;
      scheduleTOCSync(activeID);
    });
    tocMutationObserver.observe(list, { subtree: true, attributes: true, attributeFilter: ["class", "aria-current"] });
  }

  function syncTOC(id) {
    var list = document.querySelector("[data-componentdocshell-toc-list]");
    if (!list) return;
    watchTOC(list);
    tocSyncing = true;
    tocLinks().forEach(function (link) {
      var linkID = link.getAttribute("data-toc-link") || idFromHash(link.getAttribute("href"));
      var active = linkID === id;
      link.classList.toggle("is-active", active);
      if (active) {
        link.setAttribute("aria-current", "location");
        link.setAttribute("data-margo-toc-active", "true");
      } else {
        link.removeAttribute("aria-current");
        link.removeAttribute("data-margo-toc-active");
      }
    });
    tocSyncing = false;
  }

  function scheduleTOCSync(id) {
    if (tocSyncFrame) cancelAnimationFrame(tocSyncFrame);
    tocSyncFrame = requestAnimationFrame(function () {
      tocSyncFrame = null;
      syncTOC(id);
    });
  }

  function writeHash(id) {
    if (idFromHash(window.location.hash) === id) return;
    history.replaceState(history.state, "", window.location.pathname + window.location.search + "#" + encodeURIComponent(id));
  }

  function setActive(id, writeLocation) {
    if (!id) return;
    activeID = id;
    if (writeLocation) writeHash(id);
    scheduleTOCSync(id);
  }

  function updateHash(id) {
    if (!id || explicitLock) return;
    setActive(id, true);
  }

  function latestVisibleID() {
    var latestID = "";
    var latestTime = -Infinity;
    var latestSequence = -1;
    visibleHeadings.forEach(function (stamp, id) {
      if (stamp.time > latestTime || (stamp.time === latestTime && stamp.sequence > latestSequence)) {
        latestID = id;
        latestTime = stamp.time;
        latestSequence = stamp.sequence;
      }
    });
    return latestID;
  }

  function updateFromVisibility() {
    if (automaticHold) return;
    updateHash(latestVisibleID());
  }

  function clearExplicitReleaseTimer() {
    if (!explicitReleaseTimer) return;
    clearTimeout(explicitReleaseTimer);
    explicitReleaseTimer = null;
  }

  function releaseExplicitNavigation() {
    clearExplicitReleaseTimer();
    if (!explicitLock) return;
    var requestedID = explicitID;
    var targetWasSeen = explicitTargetSeen;
    var wasRestoredHash = explicitRestore;
    explicitID = "";
    explicitLock = false;
    explicitTargetSeen = false;
    explicitRestore = false;
    automaticHold = true;
    if (wasRestoredHash || targetWasSeen || visibleHeadings.has(requestedID) || headingIsVisible(requestedID)) {
      setActive(requestedID, true);
      return;
    }
    updateFromVisibility();
  }

  function armExplicitNavigation() {
    clearExplicitReleaseTimer();
    explicitReleaseTimer = setTimeout(releaseExplicitNavigation, explicitRestore ? 1200 : (explicitTargetSeen ? 180 : 1200));
  }

  function explicitNavigate(id) {
    if (!headingForID(id)) return;
    explicitID = id;
    explicitLock = true;
    explicitTargetSeen = false;
    explicitRestore = false;
    automaticHold = false;
    setActive(id, true);
    armExplicitNavigation();
  }

  function refresh() {
    var generation = ++observerGeneration;
    if (observer) observer.disconnect();
    observer = null;
    if (tocMutationObserver) tocMutationObserver.disconnect();
    tocMutationObserver = null;
    tocListNode = null;
    if (tocSyncFrame) cancelAnimationFrame(tocSyncFrame);
    tocSyncFrame = null;
    clearExplicitReleaseTimer();
    explicitID = "";
    explicitLock = false;
    explicitTargetSeen = false;
    automaticHold = false;
    activeID = "";
    visibilitySequence = 0;
    visibleHeadings.clear();
    var pageScroll = document.getElementById("page-scroll");
    var items = headings();
    var hashID = idFromHash(window.location.hash);
    if (headingForID(hashID)) {
      explicitID = hashID;
      explicitLock = true;
      setActive(hashID, false);
      explicitRestore = true;
      armExplicitNavigation();
    }
    if (!pageScroll || !items.length || !("IntersectionObserver" in window)) return;

    observer = new IntersectionObserver(function (entries) {
      if (generation !== observerGeneration) return;
      entries.forEach(function (entry) {
        if (entry.isIntersecting) {
          visibilitySequence += 1;
          visibleHeadings.set(entry.target.id, {
            time: typeof entry.time === "number" ? entry.time : 0,
            sequence: visibilitySequence
          });
          if (explicitLock && entry.target.id === explicitID) {
            explicitTargetSeen = true;
            armExplicitNavigation();
          }
        } else {
          visibleHeadings.delete(entry.target.id);
        }
      });
      updateFromVisibility();
    }, { root: pageScroll, rootMargin: "0px", threshold: 0 });
    items.forEach(function (heading) { observer.observe(heading); });
  }

  function scheduleRefresh() {
    if (refreshFrame) cancelAnimationFrame(refreshFrame);
    refreshFrame = requestAnimationFrame(function () {
      refreshFrame = null;
      refresh();
    });
  }

  function loadScript(url) {
    return new Promise(function (resolve, reject) {
      var script = document.createElement("script");
      script.src = url;
      script.async = true;
      script.onload = resolve;
      script.onerror = function () { reject(new Error("runtime asset unavailable: " + url)); };
      document.head.appendChild(script);
    });
  }

  function mermaidTasksNeedRender() {
    return Array.prototype.some.call(
      document.querySelectorAll('[data-margo-runtime-task="mermaid"]'),
      function (node) { return node.dataset.margoRuntimeStatus !== "succeeded" || !node.querySelector("svg"); }
    );
  }

  async function renderMermaidAfterSwap() {
    var nodes = document.querySelectorAll('[data-margo-runtime-task="mermaid"]');
    if (!nodes.length || !mermaidTasksNeedRender()) return;
    try {
      nodes.forEach(function (node) { node.dataset.margoRuntimeStatus = "pending"; });
      if (window.goshtosoDependencies && window.goshtosoDependencies.ready) {
        await window.goshtosoDependencies.ready;
      }
      if (!window.mermaid || typeof window.mermaid.render !== "function") {
        if (!mermaidLoadPromise) {
          mermaidLoadPromise = loadScript(mermaidRuntimeURL).catch(function (error) {
            mermaidLoadPromise = null;
            throw error;
          });
        }
        await mermaidLoadPromise;
      }
      var runnerWasLoaded = typeof window.margoRunMermaid === "function";
      if (typeof window.margoRunMermaid !== "function") {
        if (!mermaidRunnerLoadPromise) {
          mermaidRunnerLoadPromise = loadScript(mermaidRunnerURL).catch(function (error) {
            mermaidRunnerLoadPromise = null;
            throw error;
          });
        }
        await mermaidRunnerLoadPromise;
      }
      if (typeof window.margoRunMermaid !== "function") throw new Error("Mermaid executor unavailable");
      // The executor auto-runs when dynamically injected. Reuse that promise
      // before asking an already-loaded executor to render a new HTMX fragment.
      if (!runnerWasLoaded && window.margoRuntimeReady && typeof window.margoRuntimeReady.then === "function") {
        try { await window.margoRuntimeReady; } catch (_) {}
      }
      if (mermaidTasksNeedRender()) {
        if (runnerWasLoaded && window.margoRuntimeReady && typeof window.margoRuntimeReady.then === "function") {
          try { await window.margoRuntimeReady; } catch (_) {}
        }
        await window.margoRunMermaid();
      }
    } catch (error) {
      document.documentElement.dataset.margoRuntimeStatus = "failed";
      document.querySelectorAll('[data-margo-runtime-task="mermaid"]').forEach(function (node) {
        node.dataset.margoRuntimeStatus = "failed";
      });
    }
  }

  function queueMermaidRender() {
    mermaidRenderQueue = mermaidRenderQueue.catch(function () {}).then(renderMermaidAfterSwap);
    return mermaidRenderQueue;
  }

  document.addEventListener("click", function (event) {
    if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
    var link = event.target && event.target.closest ? event.target.closest("a[href]") : null;
    if (!link) return;
    var isTOCLink = !!link.getAttribute("data-toc-link") || !!(link.closest && link.closest("[data-componentdocshell-toc-list]"));
    var isHeadingLink = !!(link.closest && link.closest(".margo-page-heading__anchor"));
    if (!isTOCLink && !isHeadingLink) return;
    explicitNavigate(idFromHash(link.getAttribute("href")));
  }, true);
  document.addEventListener("hashchange", function () {
    var id = idFromHash(window.location.hash);
    if (id) {
      explicitNavigate(id);
    } else {
      clearExplicitReleaseTimer();
      explicitID = "";
      explicitLock = false;
      explicitTargetSeen = false;
      explicitRestore = false;
      automaticHold = false;
    }
  });
  window.addEventListener("popstate", function () {
    var id = idFromHash(window.location.hash);
    if (id) explicitNavigate(id);
  });
  document.addEventListener("scroll", function (event) {
    if (!explicitLock || event.target !== document.getElementById("page-scroll")) return;
    armExplicitNavigation();
  }, true);
  document.addEventListener("scrollend", function (event) {
    if (event.target === document.getElementById("page-scroll")) releaseExplicitNavigation();
  }, true);
  ["wheel", "touchstart", "pointerdown"].forEach(function (eventName) {
    document.addEventListener(eventName, function (event) {
      if (explicitLock) releaseExplicitNavigation();
      automaticHold = false;
    }, true);
  });
  document.addEventListener("keydown", function (event) {
    if (["ArrowDown", "ArrowUp", "PageDown", "PageUp", "Home", "End", " "].indexOf(event.key) === -1) return;
    if (explicitLock) releaseExplicitNavigation();
    automaticHold = false;
  }, true);
  document.addEventListener("DOMContentLoaded", function () {
    scheduleRefresh();
    queueMermaidRender();
  });
  window.addEventListener("componentdocshell:navigated", scheduleRefresh);
  document.addEventListener("htmx:afterSwap", function (event) {
    if (event.detail && event.detail.target && event.detail.target.id === "main-content") {
      scheduleRefresh();
    }
  });
  document.addEventListener("htmx:afterSettle", function (event) {
    if (event.detail && event.detail.target && event.detail.target.id === "main-content") {
      scheduleRefresh();
      queueMermaidRender();
    }
  });
  document.addEventListener("htmx:historyRestore", function () {
    scheduleRefresh();
    queueMermaidRender();
  });
})();`

func componentDocShellAssetPrefix(basePath string) string {
	if basePath == "" || basePath == "/" {
		return "/margo-assets/goshtoso/"
	}
	return strings.TrimSuffix(normalizedBasePath(basePath), "/") + "/margo-assets/goshtoso/"
}

func componentDocShellSchemaHash(config Config) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(ssg.ShellContract))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte("componentdocshell\x00"))
	_, _ = hash.Write([]byte(goshtosoComponentDocShellVersion))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(config.Theme.Name))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(config.Theme.ColorMode))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(normalizedBasePath(config.BasePath)))
	return hex.EncodeToString(hash.Sum(nil))
}

func (b *builder) stageGoshtosoComponentDocShellAssets() error {
	shellHandler := shellassets.Handler(b.shellAssetPrefix)
	for _, publicURL := range []string{
		shellassets.ScriptURL(b.shellAssetPrefix),
		shellassets.StylesheetURL(b.shellAssetPrefix),
		shellassets.AraiHuThemeURL(b.shellAssetPrefix),
		shellassets.GoshtosoMarkURL(b.shellAssetPrefix),
		shellassets.GoshtosoMarkReverseURL(b.shellAssetPrefix),
		shellassets.GoshtosoFaviconURL(b.shellAssetPrefix),
	} {
		if err := b.stageHandlerAsset(shellHandler, publicURL); err != nil {
			return err
		}
	}

	runtimeHandler := goshtosoassets.Handler()
	manifest := goshtosoassets.DefaultRuntimeManifest()
	publicURLs := []string{manifest.Stylesheet.LocalURL}
	for _, dependency := range manifest.Dependencies {
		if dependency.Enabled && dependency.LocalURL != "" {
			publicURLs = append(publicURLs, dependency.LocalURL)
		}
	}
	for _, publicURL := range publicURLs {
		if err := b.stageHandlerAsset(runtimeHandler, publicURL); err != nil {
			return err
		}
	}

	publicURL := b.shellAssetPrefix + componentDocShellScrollSpyAssetName
	parsed, err := url.Parse(publicURL)
	if err != nil || parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") {
		return diagnostic("site.shell_asset_invalid", "Margo emitted an invalid scroll-spy asset URL", "Keep shell assets on an absolute site path.", publicURL)
	}
	artifactPath := strings.TrimPrefix(parsed.Path, "/")
	basePath := normalizedBasePath(b.config.BasePath)
	if basePath != "/" {
		artifactPath = strings.TrimPrefix(artifactPath, strings.TrimPrefix(basePath, "/")+"/")
	}
	if err := b.addArtifact(artifactPath, []byte(componentDocShellScrollSpy)); err != nil {
		return err
	}
	b.dependencies[strings.ToLower(strings.TrimPrefix(parsed.Path, "/"))] = artifactPath
	return nil
}

func (b *builder) stageHandlerAsset(handler http.Handler, publicURL string) error {
	parsed, err := url.Parse(publicURL)
	if err != nil || parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") {
		return diagnostic("site.shell_asset_invalid", "Goshtoso emitted an invalid asset URL", "Keep shell assets on an absolute site path.", publicURL)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://margo.invalid"+publicURL, nil)
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		return diagnostic("site.shell_asset_unavailable", "Goshtoso did not serve an emitted asset", "Use the pinned Goshtoso shell release and rebuild.", publicURL)
	}
	artifactPath := strings.TrimPrefix(parsed.Path, "/")
	basePath := normalizedBasePath(b.config.BasePath)
	if basePath != "/" {
		artifactPath = strings.TrimPrefix(artifactPath, strings.TrimPrefix(basePath, "/")+"/")
	}
	if err := b.addArtifact(artifactPath, recorder.Body.Bytes()); err != nil {
		return err
	}
	// Keep the public URL path as the lookup key. The HTML shell emits absolute
	// paths, while the published artifact remains site-relative.
	b.dependencies[strings.ToLower(strings.TrimPrefix(parsed.Path, "/"))] = artifactPath
	return nil
}

func (b *builder) renderConfiguredShellSource(ctx context.Context, source Source, prepared configuredPage, dependencyBytes []byte) error {
	page := prepared.page
	content := `<div class="margo-showcase-article">` + b.breadcrumbFragment(page) + string(prepared.article) + b.paginationFragment(page) + `</div>`
	documentTitle := ""
	if page.Source == b.config.Site.Home && page.Locale == b.config.Locales.Default {
		documentTitle = page.Title + " — Markdown to durable outputs"
	}
	shellPage := componentdocshell.Page{
		Title:         page.Title,
		DocumentTitle: documentTitle,
		Description:   page.Description,
		CanonicalURL:  page.Canonical,
		SiteName:      b.config.Site.Name,
		Locale:        openGraphLocale(page.Locale),
		SocialImage: componentdocshell.SocialImage{
			URL:      b.socialURL(),
			MIMEType: b.socialMediaType,
			Width:    1280,
			Height:   640,
			Alt:      b.config.Site.SocialImage.Alt,
		},
		Active:    componentDocShellPageID(b, page),
		Content:   templ.Raw(content),
		Head:      templ.Raw(b.renderComponentDocShellHead(page)),
		EnableTOC: true,
	}
	component := componentdocshell.Layout(b.componentDocShellConfig(), shellPage)
	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		return err
	}
	withSearchModal, err := injectComponentDocShellSearchModal(rendered.Bytes(), b.componentDocShellSearchConfig(), ctx, page.Source)
	if err != nil {
		return err
	}
	withDependencies, err := injectComponentDocShellPageDependencies(withSearchModal, dependencyBytes, page.Source)
	if err != nil {
		return err
	}
	rewritten, err := b.rewriteHTML(ctx, source, withDependencies)
	if err != nil {
		return err
	}
	rewritten, err = b.injectPageActions(ctx, rewritten, page)
	if err != nil {
		return err
	}
	if err := b.addDeclaredPageArtifacts(ctx, source, page, prepared.document); err != nil {
		return err
	}
	if err := validateConfiguredShellDocument(rewritten, page); err != nil {
		return err
	}
	if err := b.addArtifact(page.Output, rewritten); err != nil {
		return err
	}
	b.pages = append(b.pages, page)
	return nil
}

func (b *builder) componentDocShellConfig() componentdocshell.Config {
	items := make([]sidebar.Item, 0, len(b.configPages))
	var home Page
	for _, page := range b.configPages {
		if page.Source == b.config.Site.Home && page.Locale == b.config.Locales.Default {
			home = page
			break
		}
	}
	for _, page := range b.configPages {
		if page.Locale != b.config.Locales.Default {
			continue
		}
		if page.Source == home.Source {
			continue
		}
		items = append(items, sidebar.Item{ID: componentDocShellPageID(b, page), Label: page.Title, Href: b.shellPageHref(page)})
	}
	brand := componentdocshell.Brand{
		Name:       b.config.Site.Name,
		HomeURL:    b.shellPageHref(home),
		FaviconURL: shellassets.GoshtosoFaviconURL(b.shellAssetPrefix),
	}
	if b.config.Site.Version != "" {
		brand.Badge = &componentdocshell.BrandBadge{
			Label:     b.config.Site.Version,
			AriaLabel: b.config.Site.Name + " version " + b.config.Site.Version,
		}
	}
	defaultTheme := "araihu"
	disableDefaultThemeStylesheet := false
	var themeStylesheets []string
	for _, theme := range b.config.Themes {
		if theme.Name != b.config.Theme.Name {
			continue
		}
		defaultTheme = theme.Name
		disableDefaultThemeStylesheet = true
		themeStylesheets = []string{b.publicationArtifactHref(theme.CSSURL)}
		break
	}
	return componentdocshell.Config{
		Brand: brand,
		Navigation: componentdocshell.Navigation{
			Items:         []sidebar.Item{{ID: componentDocShellPageID(b, home), Label: home.Title, Href: b.shellPageHref(home)}},
			SectionsTitle: "Margo features",
			Sections:      []sidebar.Section{{Title: "Features", Items: items}},
			DisableSearch: true,
		},
		Appearance: componentdocshell.AppearanceConfig{
			DefaultTheme:                  defaultTheme,
			InitialColorScheme:            componentDocShellColorScheme(b.config.Theme.ColorMode),
			PersistPreferences:            true,
			DisableThemeSelector:          true,
			DisableDarkModeToggle:         false,
			DisableDefaultThemeStylesheet: disableDefaultThemeStylesheet,
			ThemeStylesheets:              themeStylesheets,
		},
		Interactions:  componentdocshell.InteractionConfig{EnableHTMX: true, LocalRuntime: true},
		HeaderActions: b.componentDocShellSearch(),
		Footer:        b.componentDocShellFooter(),
		RepositoryURL: b.config.Site.RepositoryURL,
		AssetPrefix:   b.shellAssetPrefix,
	}
}

func (b *builder) componentDocShellSearch() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		if _, err := io.WriteString(writer, `<div class="margo-shell-search">`); err != nil {
			return err
		}
		if err := search.SearchField(b.componentDocShellSearchConfig()).Render(ctx, writer); err != nil {
			return err
		}
		_, err := io.WriteString(writer, `</div>`)
		return err
	})
}

func (b *builder) componentDocShellSearchConfig() search.Config {
	return search.Config{
		ID:             "margo-doc-search",
		Label:          "Search features",
		Placeholder:    "Search features",
		ShortcutText:   "⌘ K",
		GlobalShortcut: true,
		Items:          b.componentDocShellSearchItems(),
		MatchMode:      search.MatchModeFuzzy,
		MaxResults:     8,
		EmptyText:      "No matching pages.",
		TriggerClass:   "margo-shell-search-trigger",
	}
}

func (b *builder) componentDocShellSearchItems() []search.Item {
	items := make([]search.Item, 0, len(b.configPages))
	for _, page := range b.configPages {
		if page.Locale != b.config.Locales.Default {
			continue
		}
		href := b.shellPageHref(page)
		items = append(items, search.Item{
			ID:          "margo-search-" + componentDocShellPageID(b, page),
			Title:       page.Title,
			Description: page.Description,
			Href:        href,
			Kind:        "Page",
			Path:        href,
			Keywords:    []string{page.Source, page.Output},
		})
	}
	return items
}

func (b *builder) componentDocShellFooter() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if _, err := io.WriteString(w, `<p class="margo-shell-footer">Margo · `); err != nil {
			return err
		}

		renderLink := func(href, label string, external bool) error {
			options := make([]link.Option, 0, 1)
			if external {
				options = append(options, link.WithTarget("_blank"))
			}
			return link.Link(href, options...).Render(
				templ.WithChildren(ctx, templ.Raw(stdhtml.EscapeString(label))),
				w,
			)
		}

		if err := renderLink("https://goshtoso.araihu.com/", "Built with Goshtoso", true); err != nil {
			return err
		}
		if _, err := io.WriteString(w, ` by `); err != nil {
			return err
		}
		if err := renderLink("https://araihu.com/", "Arai Hû", true); err != nil {
			return err
		}
		if _, err := io.WriteString(w, ` · `); err != nil {
			return err
		}
		if err := renderLink(b.publicationArtifactHref(LLMSPath), "llms.txt", false); err != nil {
			return err
		}
		if _, err := io.WriteString(w, ` · `); err != nil {
			return err
		}
		if err := renderLink(b.publicationArtifactHref(SitemapPath), "sitemap.xml", false); err != nil {
			return err
		}
		_, err := io.WriteString(w, `</p>`)
		return err
	})
}

func (b *builder) publicationArtifactHref(filename string) string {
	basePath := normalizedBasePath(b.config.BasePath)
	if basePath == "/" {
		return "/" + strings.TrimPrefix(filename, "/")
	}
	return path.Join(basePath, filename)
}

const componentDocShellNavigationSwap = "outerHTML transition:true swap:160ms settle:240ms"

func (b *builder) decorateComponentDocShellNavigation(root *html.Node, source Source) error {
	current, exists := b.configured[source.Path]
	if !exists {
		return nil
	}
	routes := make(map[string]struct{}, len(b.configPages)*2)
	addRoute := func(value string) {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Path == "" {
			return
		}
		routes[parsed.Path] = struct{}{}
	}
	for _, page := range b.configPages {
		if page.Locale != current.page.Locale {
			continue
		}
		addRoute(b.shellPageHref(page))
		addRoute(relativeAssetPath(path.Dir(current.page.Output), page.Output))
	}
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "div" && attributeValue(node, "id") == "componentdocshell-sidebar-content" {
			setHTMLAttribute(node, "hx-swap-oob", "outerHTML:#componentdocshell-sidebar-content")
		}
		if node.Type == html.ElementNode && node.Data == "a" && !hasClass(node, "component-doc-shell__brand") {
			if index := attributeIndex(node, "href"); index >= 0 {
				value := node.Attr[index].Val
				parsed, err := url.Parse(value)
				if err == nil && parsed.Scheme == "" && parsed.Host == "" && parsed.Path != "" {
					if _, known := routes[parsed.Path]; known {
						setHTMLAttribute(node, "hx-get", value)
						setHTMLAttribute(node, "hx-select", "#main-content")
						setHTMLAttribute(node, "hx-target", "#main-content")
						setHTMLAttribute(node, "hx-swap", componentDocShellNavigationSwap)
						setHTMLAttribute(node, "hx-push-url", "true")
						setHTMLAttribute(node, "data-margo-navigation", "true")
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return nil
}

func setHTMLAttribute(node *html.Node, key, value string) {
	if index := attributeIndex(node, key); index >= 0 {
		node.Attr[index].Val = value
		return
	}
	node.Attr = append(node.Attr, html.Attribute{Key: key, Val: value})
}

func componentDocShellColorScheme(value string) componentdocshell.ColorScheme {
	switch value {
	case "light":
		return componentdocshell.ColorSchemeLight
	case "dark":
		return componentdocshell.ColorSchemeDark
	default:
		return componentdocshell.ColorSchemeSystem
	}
}

func componentDocShellPageID(b *builder, page Page) string {
	if page.Source == b.config.Site.Home && page.Locale == b.config.Locales.Default {
		return "overview"
	}
	value := strings.TrimSuffix(page.Output, path.Ext(page.Output))
	value = strings.NewReplacer("/", "-", "_", "-").Replace(value)
	value = strings.Trim(value, "-")
	if value == "" {
		return "feature"
	}
	return "feature-" + value
}

func (b *builder) shellPageHref(page Page) string {
	basePath := normalizedBasePath(b.config.BasePath)
	route := "/" + page.Output
	if page.Source == b.config.Site.Home && page.Locale == b.config.Locales.Default {
		route = "/"
	}
	if basePath != "/" {
		return strings.TrimSuffix(basePath, "/") + route
	}
	return route
}

func (b *builder) renderComponentDocShellHead(page Page) string {
	var builder strings.Builder
	iconURL, _ := relativeSitePath(path.Dir(page.Output), b.config.Site.Icon)
	if iconURL != "" {
		builder.WriteString(`<link rel="icon" href="` + stdhtml.EscapeString(iconURL) + `">`)
	}
	builder.WriteString(`<link rel="stylesheet" href="/margo-assets/site.css">`)
	builder.WriteString(`<script defer src="/` + stdhtml.EscapeString(pageActionsScriptPath) + `"></script>`)
	for _, css := range b.config.CustomCSS {
		builder.WriteString(`<link rel="stylesheet" href="/` + stdhtml.EscapeString(strings.TrimPrefix(css.CSSURL, "/")) + `">`)
	}
	builder.WriteString(`<script defer src="` + stdhtml.EscapeString(b.shellAssetPrefix+componentDocShellScrollSpyAssetName) + `"></script>`)
	return builder.String()
}

func injectComponentDocShellPageDependencies(document, dependencies []byte, source string) ([]byte, error) {
	if len(dependencies) == 0 {
		return document, nil
	}
	const closingHead = "</head>"
	index := bytes.Index(bytes.ToLower(document), []byte(closingHead))
	if index < 0 {
		return nil, diagnostic(
			"site.html_invalid",
			"componentdocshell output has no closing head element",
			"Keep page dependencies inside the generated document head.",
			source,
		)
	}
	result := make([]byte, 0, len(document)+len(dependencies))
	result = append(result, document[:index]...)
	result = append(result, dependencies...)
	result = append(result, document[index:]...)
	return result, nil
}

func injectComponentDocShellSearchModal(document []byte, config search.Config, ctx context.Context, source string) ([]byte, error) {
	var modal bytes.Buffer
	if err := search.SearchModal(config).Render(ctx, &modal); err != nil {
		return nil, err
	}
	const closingBody = "</body>"
	index := bytes.Index(bytes.ToLower(document), []byte(closingBody))
	if index < 0 {
		return nil, diagnostic(
			"site.html_invalid",
			"componentdocshell output has no closing body element",
			"Keep the Goshtoso search modal inside the generated document body.",
			source,
		)
	}
	result := make([]byte, 0, len(document)+modal.Len())
	result = append(result, document[:index]...)
	result = append(result, modal.Bytes()...)
	result = append(result, document[index:]...)
	return result, nil
}

func decorateComponentDocShellHeadings(root *html.Node) {
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && (node.Data == "h2" || node.Data == "h3") && attributeValue(node, "id") != "" && hasHTMLAncestorClass(node, "margo-showcase-article") {
			setHTMLAttribute(node, "data-toc-heading", "")
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
}

func hasHTMLAncestorClass(node *html.Node, className string) bool {
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if parent.Type != html.ElementNode {
			continue
		}
		for _, class := range strings.Fields(attributeValue(parent, "class")) {
			if class == className {
				return true
			}
		}
	}
	return false
}

func validateConfiguredShellDocument(document []byte, page Page) error {
	root, err := html.Parse(bytes.NewReader(document))
	if err != nil {
		return diagnostic("site.html_invalid", err.Error(), "Report this generated shell defect.", page.Source)
	}
	counts := map[string]int{}
	skip := 0
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			counts[node.Data]++
			if node.Data == "a" && attributeValue(node, "href") == "#main-content" {
				skip++
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	if counts["html"] != 1 || counts["head"] != 1 || counts["body"] != 1 || counts["main"] != 1 || counts["h1"] != 1 || skip != 1 {
		return diagnostic("site.semantic_structure", "componentdocshell output must contain one document, main, h1, and skip link", "Keep the Margo article inside the Goshtoso documentation shell.", page.Source)
	}
	if err := validateRequiredHead(root, page); err != nil {
		return err
	}
	return nil
}
