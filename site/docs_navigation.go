package site

const docsNavigationScriptPath = "margo-assets/docs-navigation.js"

// The public shell owns drawer focus, TOC rebuilding, and scrolling. Margo
// supplies complete route regions and mirrors its optional-sidebar setting.
const docsNavigationScript = `(function () {
  "use strict";

  var pendingDependencies = new Map();
  var runtimeQueue = Promise.resolve();
  // Replace route metadata only; scripts, styles and shell state stay installed.
  var pageMetadataSelector = 'title, meta[name="description"], meta[name="author"], meta[name="keywords"], meta[name="robots"], meta[property^="og:"], meta[property^="article:"], meta[name^="twitter:"], link[rel="canonical"], link[rel="alternate"][hreflang], script[type="application/ld+json"]';

  function loadDependency(dependency) {
    var id = dependency.getAttribute("data-margo-requirement");
    if (pendingDependencies.has(id)) return pendingDependencies.get(id);
    var installed = Array.from(document.head.querySelectorAll("[data-margo-requirement]"));
    if (installed.some(function (node) { return node.getAttribute("data-margo-requirement") === id; })) return Promise.resolve();
    var node = document.createElement(dependency.localName);
    Array.from(dependency.attributes).forEach(function (attribute) { node.setAttribute(attribute.name, attribute.value); });
    node.textContent = dependency.textContent;
    var promise = new Promise(function (resolve, reject) {
      if (node.src || node.href) {
        node.onload = resolve;
        node.onerror = function () { node.remove(); reject(new Error("Page dependency unavailable: " + id)); };
      }
      document.head.appendChild(node);
      if (!node.src && !node.href) resolve();
    });
    pendingDependencies.set(id, promise);
    promise.catch(function () { pendingDependencies.delete(id); });
    return promise;
  }

  async function initializePage(main, dependencies) {
    if (!main.isConnected) return;
    if (window.goshtosoDependencies) await window.goshtosoDependencies.ready;
    // Full documents carry page-specific runtimes in their head. HTMX's main
    // selection omits it, so install missing requirements in declared order.
    for (var dependency of dependencies) await loadDependency(dependency);
    if (window.margoRuntimeReady) await window.margoRuntimeReady;
    if (!main.isConnected) return;
    var mermaid = main.querySelector('[data-margo-runtime-task="mermaid"]');
    if (mermaid && mermaid.dataset.margoRuntimeStatus !== "succeeded" && window.margoRunMermaid) await window.margoRunMermaid();
  }

  document.addEventListener("htmx:before:swap", function (event) {
    var target = event.detail && event.detail.ctx && event.detail.ctx.target;
    if (!target || (target.id !== "main-content" && target !== document.body)) return;
    // HTMX parses responses without scripting. Keep noscript fallbacks inert
    // when their containing navigation region is inserted into the live page.
    event.detail.tasks.forEach(function (task) {
      task.fragment.querySelectorAll("noscript").forEach(function (node) { node.remove(); });
    });
    var response = new DOMParser().parseFromString(event.detail.ctx.text, "text/html");
    event.detail.ctx.margoPageURL = response.querySelector("#main-content")?.dataset.margoPageUrl;
    event.detail.ctx.margoDependencies = Array.from(response.head.querySelectorAll("[data-margo-requirement]"));
    event.detail.ctx.margoPageMetadata = Array.from(response.head.querySelectorAll(pageMetadataSelector));
  });

  document.addEventListener("htmx:after:swap", function (event) {
    var target = event.detail && event.detail.ctx && event.detail.ctx.target;
    if (!target || (target.id !== "main-content" && target !== document.body)) return;
    // HTMX assigns ctx.title after this event; route metadata owns it here.
    event.detail.ctx.title = "";
    var main = document.getElementById("main-content");
    // Ignore a late response whose route has already left the visible main.
    if (!main || (event.detail.ctx.margoPageURL && main.dataset.margoPageUrl !== event.detail.ctx.margoPageURL)) return;
    if (event.detail.ctx.margoPageMetadata?.length) {
      document.head.querySelectorAll(pageMetadataSelector).forEach(function (node) { node.remove(); });
      event.detail.ctx.margoPageMetadata.forEach(function (node) { document.head.appendChild(document.importNode(node, true)); });
    }
    if (main.dataset.margoSidebar === "false") document.body.dataset.margoSidebar = "false";
    else delete document.body.dataset.margoSidebar;
    // Navigation API traversal may reset focus after the shell's synchronous
    // swap handler. Restore it once that navigation has finished its turn.
    requestAnimationFrame(function () {
      if (main.isConnected && window.componentDocShell) window.componentDocShell.focusMain();
    });
    var dependencies = event.detail.ctx.margoDependencies || [];
    runtimeQueue = runtimeQueue.catch(function () {}).then(function () {
      return initializePage(main, dependencies);
    }).catch(function (error) {
      if (!main.isConnected) return;
      document.documentElement.dataset.margoRuntimeStatus = "failed";
      console.error("Margo page initialization failed", error);
    });
  });

})();`
