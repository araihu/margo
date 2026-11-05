(() => {
  "use strict";
  if (globalThis.margoRunMath) return;
  const results = new WeakMap();
  const encoder = new TextEncoder();
  let fontPromise;
  let fontFailure = false;
  const fail = (code, message) => Object.assign(new Error(message), {code});
  const bytes = value => encoder.encode(value).byteLength;
  const source = node => node.querySelector(".margo-math__source");
  async function fonts() {
    if (!document.fonts) throw fail("math.font_unavailable", "Font loading is unavailable.");
    const sheet = [...document.styleSheets].find(sheet => sheet.ownerNode?.dataset.margoRequirement === "margo.math.styles");
    const rules = [...(sheet?.cssRules || [])].filter(rule => rule.type === CSSRule.FONT_FACE_RULE);
    // Load our exact sources even when a host already declares other KaTeX faces.
    const faces = rules.map(rule => new FontFace(rule.style.fontFamily, rule.style.src, {style:rule.style.fontStyle,weight:rule.style.fontWeight}));
    if (faces.length !== 20) throw fail("math.font_unavailable", "The bundled math font faces are missing.");
    try { await Promise.all(faces.map(face => face.load())); }
    catch { throw fail("math.font_unavailable", "A bundled math font did not load."); }
    for (const face of faces) document.fonts.add(face);
    await document.fonts.ready;
    const checks = faces.map(face => ({family: face.family.replace(/["']/g, ""), query: `${face.style} ${face.weight} 16px ${face.family}`, loaded: face.status === "loaded"}));
    if (checks.some(check => !check.loaded)) throw fail("math.font_unavailable", "A bundled math font did not load.");
    return checks;
  }
  function overflow(node) {
    const overflowing = node.scrollWidth > node.clientWidth + 1;
    if (node.dataset.margoMathMode === "display" && overflowing) {
      node.tabIndex = 0;
      node.setAttribute("role", "region");
      node.setAttribute("aria-label", "Mathematical expression; scroll horizontally");
    } else {
      node.removeAttribute("tabindex"); node.removeAttribute("role"); node.removeAttribute("aria-label");
    }
  }
  async function execute() {
    if (fontFailure) { fontPromise = undefined; fontFailure = false; }
    const nodes = [...document.querySelectorAll('[data-margo-runtime-task="math"]')];
    if (!nodes.length) return [];
    const totalPayload = nodes.reduce((total, node) => total + bytes(source(node)?.textContent || ""), 0);
    let totalOutput = 0;
    const content = [...document.querySelectorAll(".margo-document")];
    const initialBytes = content.reduce((total, node) => {
      const clone = node.cloneNode(true);
      for (const output of clone.querySelectorAll(".margo-math__output, .margo-math__error")) output.replaceChildren();
      return total + bytes(clone.outerHTML);
    }, 0);
    const ceiling = Math.min(8 << 20, ...nodes.map(node => Number(node.dataset.margoOutputLimit) || 0));
    for (const [index, node] of nodes.entries()) {
      const sourceNode = source(node), target = node.querySelector(".margo-math__output"), errorNode = node.querySelector(".margo-math__error");
      if (!sourceNode || !target || !errorNode) throw fail("math.runtime_unavailable", "Malformed math marker.");
      const prior = results.get(node);
      if (prior?.status === "succeeded" && prior.inputSHA256 === node.dataset.margoMathInput && nodes.length <= 1024 && totalPayload <= (1 << 20) && totalOutput + bytes(prior.output) <= ceiling && initialBytes + totalOutput + bytes(prior.output) <= Number(node.dataset.margoOutputLimit)) {
        totalOutput += bytes(prior.output); continue;
      }
      node.dataset.margoRuntimeStatus = "running";
      const result = {elementID: node.id, inputSHA256: node.dataset.margoMathInput, source:{source:node.dataset.margoSource,line:Number(node.dataset.margoLine),column:Number(node.dataset.margoColumn)}, output: "", status: "failed", errorCode: "", message: ""};
      try {
        if (!globalThis.katex?.renderToString) throw fail("math.runtime_unavailable", "The bundled math engine is unavailable.");
        if (nodes.length > 1024 || totalPayload > (1 << 20)) throw fail("math.document_limit_exceeded", "The publication contains too much math.");
        if (bytes(sourceNode.textContent) > (16 << 10)) throw fail("math.expression_too_large", "The equation is too large.");
        let denied = false;
        const output = globalThis.katex.renderToString(sourceNode.textContent, {
          displayMode: node.dataset.margoMathMode === "display", output: "htmlAndMathml",
          throwOnError: true, strict: "error", maxExpand: 1000, maxSize: 20,
          macros: Object.create(null), trust: () => { denied = true; return false; },
        });
        if (denied) throw fail("math.command_forbidden", "The expression attempted a forbidden command.");
        totalOutput += bytes(output);
        if (bytes(output) > (256 << 10) || totalOutput > ceiling || initialBytes + totalOutput > Number(node.dataset.margoOutputLimit)) throw fail("math.output_limit_exceeded", "The typeset math exceeds the output limit.");
        const detached = document.createElement("span");
        // Only the pinned engine's output enters this sink. Source/errors use textContent.
        detached.innerHTML = output;
        if (!detached.querySelector(".katex-mathml math") || !detached.querySelector(".katex-html")) throw fail("math.expression_invalid", "The math engine returned incomplete output.");
        if (!fontPromise) fontPromise = fonts().catch(error => { fontFailure = true; throw error; });
        await fontPromise;
        target.replaceChildren(...detached.childNodes);
        sourceNode.hidden = true; errorNode.hidden = true;
        result.output = output; result.status = "succeeded";
        node.dataset.margoRuntimeStatus = "succeeded";
        overflow(node);
      } catch (error) {
        result.errorCode = typeof error.code === "string" ? error.code : "math.expression_invalid";
        result.message = error.message || "Unable to render this expression.";
        target.replaceChildren(); sourceNode.hidden = false;
        errorNode.textContent = `${result.errorCode}: ${result.message}`;
        errorNode.hidden = false;
        node.dataset.margoRuntimeStatus = "failed";
      }
      results.set(node, result);
      if (index % 16 === 15) await new Promise(resolve => setTimeout(resolve, 0));
    }
    const output = nodes.map(node => results.get(node));
    globalThis.margoMathFontChecks = fontPromise ? await fontPromise.catch(() => []) : [];
    if (output.some(result => result.status !== "succeeded")) throw fail("math.expression_invalid", "One or more equations could not be rendered.");
    return output;
  }
  globalThis.margoMathResults = () => [...document.querySelectorAll('[data-margo-runtime-task="math"]')].map(node => results.get(node));
  globalThis.margoGetMathFontEvidence = async () => {
    const checks = fontPromise ? await fontPromise : await fonts();
    if (!globalThis.crypto?.subtle) throw fail("math.runtime_unavailable", "Font identity hashing is unavailable.");
    const sheet = [...document.styleSheets].find(sheet => sheet.ownerNode?.dataset.margoRequirement === "margo.math.styles");
    const chunks = [encoder.encode("margo-math-fonts/v1\0")];
    let count = 0;
    for (const rule of [...(sheet?.cssRules || [])]) {
      if (rule.type !== CSSRule.FONT_FACE_RULE) continue;
      const family = rule.style.getPropertyValue("font-family").trim().replace(/["']/g, "");
      if (!family.startsWith("KaTeX_")) continue;
      const style = rule.style.getPropertyValue("font-style").trim();
      const weight = rule.style.getPropertyValue("font-weight").trim();
      const match = /url\(["']?data:font\/woff2;base64,([A-Za-z0-9+/=]+)["']?\)/.exec(rule.style.getPropertyValue("src"));
      if (!match) throw fail("math.font_unavailable", "Math fonts are not bundled.");
      const data = Uint8Array.from(atob(match[1]), c => c.charCodeAt(0));
      const length = new Uint8Array(8); new DataView(length.buffer).setBigUint64(0, BigInt(data.length), false);
      chunks.push(encoder.encode(`${family}\0${style}\0${weight}\0`), length, data); count++;
    }
    if (count !== 20) throw fail("math.font_unavailable", "Math font identity is incomplete.");
    const input = new Uint8Array(chunks.reduce((size, chunk) => size + chunk.length, 0));
    let offset = 0; for (const chunk of chunks) { input.set(chunk, offset); offset += chunk.length; }
    const digest = [...new Uint8Array(await crypto.subtle.digest("SHA-256", input))].map(value => value.toString(16).padStart(2, "0")).join("");
    return {fontChecks: checks, fontBundleDigest: digest};
  };
  globalThis.margoRunMath = () => globalThis.margoRuntimeCoordinator.run("math");
  globalThis.margoValidateMathPrint = (width, height) => {
    const bounded = [...document.querySelectorAll(".margo-document")].filter(node => !node.closest(".margo-deck") && node.getBoundingClientRect().width > width).map(node => ({node,maxWidth:node.style.maxWidth}));
    for (const item of bounded) item.node.style.maxWidth = `${width}px`;
    try {
    for (const node of document.querySelectorAll('[data-margo-runtime-task="math"]')) {
      const output = node.querySelector(".margo-math__output");
      if (node.dataset.margoRuntimeStatus !== "succeeded") return "math.expression_invalid";
      const parts = [...output.querySelectorAll(".katex-base, .katex-tag")].map(part => part.getBoundingClientRect());
      const intrinsicWidth = parts.length ? Math.max(...parts.map(part=>part.right)) - Math.min(...parts.map(part=>part.left)) : output.getBoundingClientRect().width;
      const intrinsicHeight = parts.length ? Math.max(...parts.map(part=>part.bottom)) - Math.min(...parts.map(part=>part.top)) : 0;
      const renderedHeight = Math.max(intrinsicHeight, output.getBoundingClientRect().height, output.querySelector(".katex-display")?.getBoundingClientRect().height || 0);
      const bounds = parts.length ? {left:Math.min(...parts.map(part=>part.left)),right:Math.max(...parts.map(part=>part.right)),top:Math.min(...parts.map(part=>part.top)),bottom:Math.max(...parts.map(part=>part.bottom))} : output.getBoundingClientRect();
      let clipped = false;
      for (let ancestor = node.parentElement; ancestor; ancestor = ancestor.parentElement) {
        const rect = ancestor.getBoundingClientRect(), style = getComputedStyle(ancestor);
        const slide = ancestor.classList.contains("margo-deck__slide");
        const clipX = slide || /^(hidden|clip|auto|scroll)$/.test(style.overflowX);
        const clipY = slide || /^(hidden|clip|auto|scroll)$/.test(style.overflowY);
        if ((clipX && (bounds.left < rect.left-1 || bounds.right > rect.right+1)) ||
            (clipY && (bounds.top < rect.top-1 || bounds.bottom > rect.bottom+1))) clipped = true;
      }
      if (clipped || intrinsicWidth > Math.min(node.clientWidth || width, width) + 1 || renderedHeight > height + 1) {
        globalThis.margoMathPrintDiagnostic = {source:node.dataset.margoSource,line:Number(node.dataset.margoLine),column:Number(node.dataset.margoColumn),width:intrinsicWidth,height:renderedHeight,availableWidth:Math.min(node.clientWidth||width,width),availableHeight:height};
        return "math.print_overflow";
      }
    }
    return "";
    } finally { for (const item of bounded) item.node.style.maxWidth = item.maxWidth; }
  };
  globalThis.margoRuntimeCoordinator.register("math", execute);
  if (typeof ResizeObserver === "function") new ResizeObserver(() => {
    for (const node of document.querySelectorAll(".margo-math--display")) overflow(node);
  }).observe(document.documentElement);
})();
