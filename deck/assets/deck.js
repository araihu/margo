(() => {
  "use strict";

  const deck = document.querySelector(".margo-deck");
  if (!deck) return;

  const stage = deck.closest(".margo-deck-stage");

  const slides = [...deck.querySelectorAll(":scope > .margo-deck__slide")];
  if (slides.length === 0) return;

  const previous = document.querySelector("[data-margo-deck-previous]");
  const next = document.querySelector("[data-margo-deck-next]");
  const print = document.querySelector("[data-margo-deck-print]");
  const status = document.querySelector("[data-margo-deck-status]");
  let current = 0;
  const chartPrintReplacements = [];

  const allDeckFontFaces = [
    ["Margo Sans", 400], ["Margo Sans", 600], ["Margo Sans", 700], ["Margo Sans", 800],
    ["Margo Serif", 400], ["Margo Serif", 600], ["Margo Serif", 700],
    ["Margo Mono", 400], ["Margo Mono", 600],
  ];
  const deckFontFaces = document.documentElement.dataset.theme === "minimal"
    ? allDeckFontFaces
    : allDeckFontFaces.filter(([family]) => family !== "Margo Serif");

  const fontFamilyFromRule = (value) => value.trim().replace(/^['"]|['"]$/g, "");
  const fontFaceData = (family, weight) => {
    for (const sheet of [...document.styleSheets]) {
      let rules;
      try { rules = sheet.cssRules ? [...sheet.cssRules] : []; } catch { continue; }
      for (const rule of rules) {
        if (rule.type !== CSSRule.FONT_FACE_RULE) continue;
        if (fontFamilyFromRule(rule.style.getPropertyValue("font-family")) !== family) continue;
        if (rule.style.getPropertyValue("font-weight").trim() !== String(weight)) continue;
        const match = /url\(\s*["']?data:font\/woff2;base64,([A-Za-z0-9+/=]+)["']?\s*\)/.exec(rule.style.getPropertyValue("src"));
        if (!match) return null;
        const encoded = atob(match[1]);
        return Uint8Array.from(encoded, (character) => character.charCodeAt(0));
      }
    }
    return null;
  };

  const hexDigest = async (bytes) => {
    if (!globalThis.crypto?.subtle) return "";
    const digest = await globalThis.crypto.subtle.digest("SHA-256", bytes);
    return [...new Uint8Array(digest)].map((value) => value.toString(16).padStart(2, "0")).join("");
  };

  const getDeckFontEvidence = async () => {
    if (document.fonts?.ready) await document.fonts.ready;
    const checks = [];
    for (const [family, weight] of deckFontFaces) {
      const query = `${weight} 1em "${family}"`;
      const loadedFaces = await document.fonts.load(query, "Margo font check");
      checks.push({family, query, loaded: loadedFaces.length > 0 && document.fonts.check(query)});
    }
    const encoder = new TextEncoder();
    const chunks = [encoder.encode("margo-font-bundle/v1\0")];
    for (const [family, weight] of deckFontFaces) {
      const bytes = fontFaceData(family, weight);
      if (!bytes) return { fontChecks: checks, fontBundleDigest: "" };
      chunks.push(encoder.encode(`${family}\0${weight}\0`));
      const length = new Uint8Array(8);
      new DataView(length.buffer).setBigUint64(0, BigInt(bytes.byteLength), false);
      chunks.push(length, bytes);
    }
    const preimage = new Uint8Array(chunks.reduce((total, chunk) => total + chunk.byteLength, 0));
    let offset = 0;
    for (const chunk of chunks) { preimage.set(chunk, offset); offset += chunk.byteLength; }
    return { fontChecks: checks, fontBundleDigest: await hexDigest(preimage) };
  };

  globalThis.margoGetDeckFontEvidence = getDeckFontEvidence;

  const show = (index, focus) => {
    current = Math.max(0, Math.min(slides.length - 1, index));
    slides.forEach((slide, slideIndex) => {
      const active = slideIndex === current;
      slide.hidden = !active;
      if (active) slide.setAttribute("aria-current", "page");
      else slide.removeAttribute("aria-current");
    });
    if (previous) previous.disabled = current === 0;
    if (next) next.disabled = current === slides.length - 1;
    if (status) {
      const label = status.dataset.margoLabelSlide || "Slide";
      const separator = status.dataset.margoLabelSeparator || "of";
      status.textContent = `${label} ${current + 1} ${separator} ${slides.length}`;
    }
    if (typeof validateScreenLayout === "function") validateScreenLayout();
    if (focus) slides[current].focus({ preventScroll: true });
  };

  const prepareInteractiveCharts = async () => {
    const wrappers = [...document.querySelectorAll('[data-goshtoso-chart-capability="interactive-raster"]')];
    for (const wrapper of wrappers) {
      const figure = wrapper.querySelector(".goshtoso-charts-interactive");
      if (!figure || chartPrintReplacements.some((entry) => entry.original === figure)) continue;
      const ratio = Number(wrapper.dataset.goshtosoChartExportPixelRatio) || 1;
      const surface = getComputedStyle(wrapper).getPropertyValue("--color-chart-surface").trim() || "#ffffff";
      const request = { format: "png", pixelRatio: ratio, backgroundColor: surface, dataURL: "" };
      wrapper.dispatchEvent(new CustomEvent("goshtoso-charts:export-request", { bubbles: true, detail: request }));
      if (typeof request.dataURL !== "string" || !request.dataURL.startsWith("data:image/png")) {
        throw new Error("Interactive chart PNG export is unavailable.");
      }
      const image = new Image();
      image.dataset.margoChartPrintImage = "";
      image.alt = figure.getAttribute("aria-label") || "Chart";
      image.src = request.dataURL;
      await image.decode();
      chartPrintReplacements.push({ original: figure, replacement: image });
      figure.replaceWith(image);
    }
  };

  const preparePrint = async () => {
    slides.forEach((slide) => { slide.hidden = false; });
    document.querySelector("[data-margo-screen-diagnostic]")?.remove();
    await prepareInteractiveCharts();
  };

  // Print styles intentionally keep each slide on its fixed canvas. A chart
  // data table can still be clipped by an intermediate scroll/overflow box,
  // though, which would make an exact-data table silently incomplete in the
  // PDF. Validate every visible chart row after print media is enabled so the
  // Chromium exporter can fail before it publishes a misleading artifact.
  const validatePrintLayout = () => {
    for (let slideIndex = 0; slideIndex < slides.length; slideIndex += 1) {
      const slide = slides[slideIndex];
      const slideRect = slide.getBoundingClientRect();
      const dataBlocks = [...slide.querySelectorAll(".margo-chart-data")].filter((block) => {
        return getComputedStyle(block).display !== "none";
      });
      for (const block of dataBlocks) {
        const rows = [...block.querySelectorAll("[data-margo-chart-row]")];
        for (let rowIndex = 0; rowIndex < rows.length; rowIndex += 1) {
          const row = rows[rowIndex];
          const rowRect = row.getBoundingClientRect();
          if (rowRect.width <= 0 || rowRect.height <= 0) continue;
          const clippingAncestors = [slide];
          for (let ancestor = row.parentElement; ancestor && ancestor !== slide; ancestor = ancestor.parentElement) {
            const style = getComputedStyle(ancestor);
            if (/(hidden|clip|auto|scroll)/.test(`${style.overflowX} ${style.overflowY}`)) clippingAncestors.push(ancestor);
          }
          for (const ancestor of clippingAncestors) {
            const rect = ancestor.getBoundingClientRect();
            if (rowRect.left < rect.left - 1 || rowRect.right > rect.right + 1 ||
                rowRect.top < rect.top - 1 || rowRect.bottom > rect.bottom + 1) {
              return `slide ${slideIndex + 1} chart data row ${rowIndex + 1} of ${rows.length} exceeds the printable canvas; reduce chart data or choose a larger slide size`;
            }
          }
        }
      }

      // Mermaid's normal document treatment permits a locally scrollable
      // canvas. A printed deck has no such escape hatch: the fixed slide
      // canvas would silently crop the diagram. Check the rendered SVG after
      // Mermaid has run, including any clipping wrapper it is nested in.
      const diagrams = [...slide.querySelectorAll(".margo-mermaid__canvas > svg")].filter((diagram) => {
        const style = getComputedStyle(diagram);
        return style.display !== "none" && style.visibility !== "hidden";
      });
      for (let diagramIndex = 0; diagramIndex < diagrams.length; diagramIndex += 1) {
        const diagram = diagrams[diagramIndex];
        const diagramRect = diagram.getBoundingClientRect();
        if (diagramRect.width <= 0 || diagramRect.height <= 0) continue;
        const clippingAncestors = [slide];
        for (let ancestor = diagram.parentElement; ancestor && ancestor !== slide; ancestor = ancestor.parentElement) {
          const style = getComputedStyle(ancestor);
          if (/(hidden|clip|auto|scroll)/.test(`${style.overflowX} ${style.overflowY}`)) clippingAncestors.push(ancestor);
        }
        for (const ancestor of clippingAncestors) {
          const rect = ancestor.getBoundingClientRect();
          if (diagramRect.left < rect.left - 1 || diagramRect.right > rect.right + 1 ||
              diagramRect.top < rect.top - 1 || diagramRect.bottom > rect.bottom + 1) {
            return `slide ${slideIndex + 1} Mermaid diagram ${diagramIndex + 1} of ${diagrams.length} exceeds the printable canvas; simplify the diagram or choose a larger slide size`;
          }
        }
      }
    }
    return "";
  };

  const restoreScreen = () => {
    for (let index = chartPrintReplacements.length - 1; index >= 0; index -= 1) {
      const { original, replacement } = chartPrintReplacements[index];
      if (replacement.isConnected) replacement.replaceWith(original);
    }
    chartPrintReplacements.length = 0;
    show(current, false);
  };

  const resizeStage = () => {
    if (!stage) return;
    const logicalWidth = Number(deck.dataset.margoWidth || 1280);
    const logicalHeight = Number(deck.dataset.margoHeight || 720);
    const availableWidth = Math.max(0, stage.clientWidth - 32);
    const availableHeight = Math.max(0, stage.clientHeight - 64 - 32);
    const scale = Math.min(1.5, availableWidth / logicalWidth, availableHeight / logicalHeight);
    const quantized = Math.max(0.01, Math.round(scale * 64) / 64);
    deck.style.setProperty("--margo-stage-scale", String(quantized));
    // CSS transforms do not change flex layout size. Reserve the scaled
    // canvas height so the control rail follows the visible deck instead of
    // being pushed below a narrow viewport.
    deck.style.marginBlockEnd = `${((quantized - 1) * logicalHeight / 2).toFixed(2)}px`;
  };

  const isVisible = (element) => {
    if (element.hidden) return false;
    const style = getComputedStyle(element);
    return style.display !== "none" && style.visibility !== "hidden";
  };

  const isPresentationChrome = (element) => element.matches(
    ".margo-deck__background, .margo-deck__header, .margo-deck__footer, " +
    ".margo-deck__bottom-chrome, .margo-deck__pagination, .margo-mermaid__accessible-source"
  ) || Boolean(element.closest(
    ".margo-deck__background, .margo-deck__header, .margo-deck__footer, " +
    ".margo-deck__bottom-chrome, .margo-deck__pagination, .margo-mermaid__accessible-source"
  ));

  const isLocalScrollContainer = (element) => {
    const style = getComputedStyle(element);
    return [style.overflowX, style.overflowY].some((value) => value === "auto" || value === "scroll");
  };

  const hasLocalScrollAncestor = (element, slide) => {
    for (let ancestor = element.parentElement; ancestor && ancestor !== slide; ancestor = ancestor.parentElement) {
      if (isLocalScrollContainer(ancestor)) return true;
    }
    return false;
  };

  const isIntrinsicLayoutBox = (element) => element.matches(".margo-layout, .margo-layout__slot") &&
    Boolean(element.closest(".margo-layout"));

  // Narrow fixed-canvas projections can make the layout grid and its local
  // scroll slots bleed a few physical pixels past the clipped slide edge.
  // Keep this exception local to renderer-owned layout boxes: visible
  // descendants outside the slide are still checked below.
  const intrinsicLayoutBleedLimit = 8;
  const containedLayoutBleed = (element, rect, slideRect) => {
    if (!isIntrinsicLayoutBox(element)) return false;
    const bleed = Math.max(
      slideRect.left - rect.left,
      rect.right - slideRect.right,
      slideRect.top - rect.top,
      rect.bottom - slideRect.bottom,
      0,
    );
    return bleed > 0 && bleed <= intrinsicLayoutBleedLimit;
  };

  const exceeds = (inner, outer, tolerance = 1) => inner.left < outer.left - tolerance ||
    inner.right > outer.right + tolerance || inner.top < outer.top - tolerance ||
    inner.bottom > outer.bottom + tolerance;

  const hasVisibleOverflow = (slide) => {
    const slideRect = slide.getBoundingClientRect();
    for (const element of [slide, ...slide.querySelectorAll("*")]) {
      if (element !== slide && (!isVisible(element) || isPresentationChrome(element))) continue;
      if (element !== slide && hasLocalScrollAncestor(element, slide)) continue;
      const rect = element.getBoundingClientRect();
      if (element !== slide && rect.width > 0 && rect.height > 0 && exceeds(rect, slideRect) &&
          !containedLayoutBleed(element, rect, slideRect)) return true;
      // The slide's own scroll dimensions include intrinsic grid tracks and
      // local scroll regions. Those are not visual overflow when their
      // rendered boxes remain inside (or have bounded layout bleed at) the
      // fixed canvas. Horizontal scroll
      // metrics remain useful for unwrapped text, while vertical metrics on
      // visible-flow elements include normal line-box ink in Chromium and
      // would make ordinary headings fail. Clipped descendants still fail.
      if (element !== slide && !isLocalScrollContainer(element)) {
        const style = getComputedStyle(element);
        if (element.scrollWidth > element.clientWidth + 1 ||
            (["hidden", "clip"].includes(style.overflowY) && element.scrollHeight > element.clientHeight + 1)) return true;
      }
    }
    return false;
  };

  const validateScreenLayout = () => {
    const overflowing = slides.filter((slide) => hasVisibleOverflow(slide));
    const failed = overflowing.length > 0;
    deck.dataset.margoScreenRuntime = failed ? "failed" : "ready";
    let diagnostic = document.querySelector("[data-margo-screen-diagnostic]");
    if (failed && !diagnostic) {
      diagnostic = document.createElement("div");
      diagnostic.dataset.margoScreenDiagnostic = "true";
      diagnostic.setAttribute("role", "alert");
      diagnostic.tabIndex = 0;
      diagnostic.textContent = "This deck content exceeds the available slide canvas.";
      stage?.prepend(diagnostic);
    } else if (!failed && diagnostic) {
      diagnostic.remove();
    }
    return !failed;
  };

  resizeStage();
  globalThis.margoValidateDeckScreen = validateScreenLayout;
  validateScreenLayout();
  if (typeof ResizeObserver === "function" && stage) {
    const observer = new ResizeObserver(() => { resizeStage(); validateScreenLayout(); });
    observer.observe(stage);
  } else {
    window.addEventListener("resize", resizeStage);
  }

  globalThis.margoPrepareDeckPrint = preparePrint;
  globalThis.margoValidateDeckPrint = validatePrintLayout;
  globalThis.margoRestoreDeckScreen = restoreScreen;
  window.addEventListener("beforeprint", () => { void preparePrint(); });
  window.addEventListener("afterprint", restoreScreen);
  const printMedia = window.matchMedia("print");
  if (typeof printMedia.addEventListener === "function") {
    printMedia.addEventListener("change", (event) => event.matches ? void preparePrint() : restoreScreen());
  }

  if (previous) previous.addEventListener("click", () => show(current - 1, false));
  if (next) next.addEventListener("click", () => show(current + 1, false));
  if (print) print.addEventListener("click", () => window.print());

  document.addEventListener("keydown", (event) => {
    const target = event.target;
    if (target instanceof HTMLElement && (
      target.isContentEditable ||
      /^(INPUT|SELECT|TEXTAREA)$/.test(target.tagName) ||
      target.closest("button, a, [role='button'], [role='link'], [role='slider'], [role='spinbutton'], [data-goshtoso-chart-capability]")
    )) return;
    let destination = current;
    switch (event.key) {
      case "ArrowLeft": destination = current - 1; break;
      case "ArrowRight": destination = current + 1; break;
      case "Home": destination = 0; break;
      case "End": destination = slides.length - 1; break;
      default: return;
    }
    event.preventDefault();
    show(destination, true);
  });

  show(0, false);
})();
