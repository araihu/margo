(() => {
  "use strict";

  const SVG_NS = "http://www.w3.org/2000/svg";
  const INSTANCE_PATTERN = /^ri-[0-9a-z]{8,32}$/;
  const MAX_ORDINAL = 99999999;
  const ALLOWED_PSEUDO_CLASSES = new Set(["first-child", "last-child", "nth-child"]);

  class SVGNormalizationError extends Error {
    constructor(code, message) {
      super(message);
      this.name = "SVGNormalizationError";
      this.code = code;
    }
  }

  function fail(code, message) {
    throw new SVGNormalizationError(code, message);
  }

  function requireCSSParser() {
    const parser = globalThis.csstree;
    if (!parser || typeof parser.parse !== "function" || typeof parser.generate !== "function" || typeof parser.walk !== "function") {
      fail("svg.normalize.css_parser_missing", "css-tree browser parser is required");
    }
    return parser;
  }

  function decimal8(value) {
    if (!Number.isSafeInteger(value) || value < 0 || value > MAX_ORDINAL) {
      fail("svg.normalize.resource_limit", "ordinal is outside the eight-digit range");
    }
    return String(value).padStart(8, "0");
  }

  function validateStringArray(value, name) {
    if (!Array.isArray(value) || value.length === 0 || value.some((entry) => typeof entry !== "string" || entry === "")) {
      fail("svg.normalize.context_invalid", `${name} must be a non-empty string array`);
    }
    return new Set(value);
  }

  function validateContext(value) {
    if (!value || typeof value !== "object" || Array.isArray(value)) {
      fail("svg.normalize.context_invalid", "normalization context is required");
    }
    if (typeof value.sourceRootID !== "string" || value.sourceRootID === "" || !INSTANCE_PATTERN.test(value.renderInstanceID)) {
      fail("svg.normalize.context_invalid", "source root or render instance is invalid");
    }
    const registry = value.referenceRegistry;
    if (!registry || typeof registry !== "object" || Array.isArray(registry)) {
      fail("svg.normalize.context_invalid", "reference registry is required");
    }
    const css = validateStringArray(registry.css, "referenceRegistry.css");
    if (!css.has("id-selector") || !css.has("url-local-fragment")) {
      fail("svg.normalize.context_invalid", "CSS reference registry is incomplete");
    }
    return Object.freeze({
      sourceRootID: value.sourceRootID,
      renderInstanceID: value.renderInstanceID,
      blockOrdinal: value.blockOrdinal,
      fragmentAttributes: validateStringArray(registry.fragmentAttributes, "referenceRegistry.fragmentAttributes"),
      presentationAttributes: validateStringArray(registry.presentationAttributes, "referenceRegistry.presentationAttributes"),
      ariaAttributes: validateStringArray(registry.ariaIdrefAttributes, "referenceRegistry.ariaIdrefAttributes"),
    });
  }

  function parseSVG(input) {
    if (typeof input !== "string" || input === "") {
      fail("svg.normalize.input_invalid", "SVG source must be a non-empty string");
    }
    const document = new DOMParser().parseFromString(input, "image/svg+xml");
    if (document.getElementsByTagName("parsererror").length !== 0) {
      fail("svg.normalize.parse_failed", "SVG XML parsing failed");
    }
    const root = document.documentElement;
    if (!root || root.namespaceURI !== SVG_NS || root.localName !== "svg") {
      fail("svg.normalize.parse_failed", "document root is not SVG");
    }
    return { document, root };
  }

  function collectAndRewriteIDs(root, context, rootID) {
    const originalRootID = root.getAttribute("id");
    if (originalRootID !== context.sourceRootID) {
      fail("svg.normalize.root_id_mismatch", "SVG root does not match SourceRootID");
    }
    const rootMap = new Map([[originalRootID, rootID]]);
    const descendantMap = new Map();
    const descendantRows = [];
    const descendants = [...root.querySelectorAll("[id]")];
    for (let ordinal = 0; ordinal < descendants.length; ordinal += 1) {
      const sourceID = descendants[ordinal].getAttribute("id");
      if (!sourceID || rootMap.has(sourceID) || descendantMap.has(sourceID)) {
        fail("svg.normalize.id_duplicate", "duplicate SVG source ID");
      }
      const targetID = `${rootID}--id-${decimal8(ordinal)}`;
      descendantMap.set(sourceID, targetID);
      descendantRows.push(Object.freeze([sourceID, targetID]));
    }
    root.setAttribute("id", rootID);
    descendants.forEach((element, ordinal) => element.setAttribute("id", descendantRows[ordinal][1]));
    return {
      originalRootID,
      rootMap,
      descendantMap,
      descendantRows,
      references: new Map([...rootMap, ...descendantMap]),
    };
  }

  function rewriteReference(source, references) {
    const target = references.get(source);
    if (!target) {
      fail("svg.normalize.reference_unresolved", `unresolved SVG reference: ${source}`);
    }
    return target;
  }

  function rewriteFragment(value, references) {
    if (typeof value !== "string" || !value.startsWith("#") || value.length === 1 || /\s/.test(value)) {
      fail("svg.normalize.reference_external", "fragment reference must be same-SVG and local");
    }
    return `#${rewriteReference(value.slice(1), references)}`;
  }

  function rewriteURLNode(node, references) {
    if (typeof node.value !== "string" || !node.value.startsWith("#") || node.value.length === 1) {
      fail("svg.normalize.reference_external", "CSS URL must be a same-SVG local fragment");
    }
    node.value = `#${rewriteReference(node.value.slice(1), references)}`;
  }

  function parseCSS(source, context) {
    const parser = requireCSSParser();
    try {
      return parser.parse(source, { context, positions: false, parseValue: true, parseCustomProperty: false });
    } catch (error) {
      fail("svg.normalize.css_parse_failed", "CSS parsing failed");
    }
  }

  function verifyCSSOM(source, declarationList = false) {
    try {
      if (declarationList) {
        const probe = document.createElement("div");
        probe.setAttribute("style", source);
        void probe.style.length;
      } else {
        const sheet = new CSSStyleSheet();
        sheet.replaceSync(source);
        void sheet.cssRules.length;
      }
    } catch (error) {
      fail("svg.normalize.css_parse_failed", "browser CSSOM rejected CSS");
    }
  }

  function rewriteURLs(ast, references) {
    const parser = requireCSSParser();
    parser.walk(ast, {
      visit: "Url",
      enter(node) {
        rewriteURLNode(node, references);
      },
    });
  }

  function validateSelectorNode(node) {
    if (["TypeSelector", "ClassSelector", "IdSelector"].includes(node.type)) return;
    if (node.type === "Combinator" && (node.name === " " || node.name === ">")) return;
    if (node.type === "PseudoClassSelector" && ALLOWED_PSEUDO_CLASSES.has(node.name)) {
      if (node.name !== "nth-child") return;
      const generated = requireCSSParser().generate(node);
      if (/^:nth-child\([1-9][0-9]*\)$/.test(generated)) return;
    }
    fail("svg.normalize.selector_forbidden", `forbidden selector node: ${node.type}`);
  }

  function normalizeSelector(selector, root, rootID, references) {
    const parser = requireCSSParser();
    const children = [...selector.children];
    for (const node of children) {
      validateSelectorNode(node);
      if (node.type === "IdSelector") node.name = rewriteReference(node.name, references);
    }
    const rootPositions = children.flatMap((node, index) => node.type === "IdSelector" && node.name === rootID ? [index] : []);
    if (rootPositions.length > 1 || (rootPositions.length === 1 && rootPositions[0] !== 0)) {
      fail("svg.normalize.selector_root_position", "normalized root selector is not the first selector token");
    }
    const branch = parser.generate(selector);
    let matchesRoot = false;
    let matchesDescendants = false;
    try {
      matchesRoot = root.matches(branch);
      matchesDescendants = root.querySelector(branch) !== null;
    } catch (error) {
      fail("svg.normalize.css_parse_failed", "normalized selector is invalid");
    }
    if (rootPositions.length === 1) {
      return matchesRoot || matchesDescendants ? [branch] : [];
    }
    const normalized = [];
    if (matchesRoot) normalized.push(`#${CSS.escape(rootID)}`);
    if (matchesDescendants) normalized.push(`#${CSS.escape(rootID)} ${branch}`);
    return normalized;
  }

  function normalizeStylesheet(source, root, rootID, references) {
    verifyCSSOM(source);
    const parser = requireCSSParser();
    const sheet = parseCSS(source, "stylesheet");
    const output = [];
    for (const node of sheet.children) {
      if (node.type !== "Rule" || node.prelude?.type !== "SelectorList" || node.block?.type !== "Block") {
        fail("svg.normalize.selector_forbidden", "only style rules are supported");
      }
      rewriteURLs(node.block, references);
      const branches = [];
      for (const selector of node.prelude.children) {
        branches.push(...normalizeSelector(selector, root, rootID, references));
      }
      if (branches.length !== 0) output.push(`${branches.join(",")}${parser.generate(node.block)}`);
    }
    const normalized = output.join("");
    verifyCSSOM(normalized);
    return normalized;
  }

  function normalizeDeclarationList(source, references) {
    verifyCSSOM(source, true);
    const parser = requireCSSParser();
    const declarations = parseCSS(source, "declarationList");
    rewriteURLs(declarations, references);
    const normalized = parser.generate(declarations);
    verifyCSSOM(normalized, true);
    return normalized;
  }

  function valueContainsURL(value) {
    let found = false;
    try {
      const ast = parseCSS(value, "value");
      requireCSSParser().walk(ast, {
        visit: "Url",
        enter() {
          found = true;
        },
      });
    } catch (error) {
      return false;
    }
    return found;
  }

  function rejectUnknownReferenceSite(attribute, references, context) {
    const value = attribute.value;
    if (attribute.name === "style" || context.fragmentAttributes.has(attribute.name) || context.presentationAttributes.has(attribute.name) || context.ariaAttributes.has(attribute.name)) return;
    if (attribute.localName === "href" || valueContainsURL(value)) {
      fail("svg.normalize.reference_site_unknown", `unknown reference attribute: ${attribute.name}`);
    }
    const tokens = value.trim().split(/\s+/).filter(Boolean);
    if (attribute.name.startsWith("aria-") && tokens.some((token) => references.has(token))) {
      fail("svg.normalize.reference_site_unknown", `unknown IDREF attribute: ${attribute.name}`);
    }
  }

  function rewriteReferenceSites(root, rootID, references, context) {
    for (const element of [root, ...root.querySelectorAll("*")]) {
      for (const attribute of [...element.attributes]) {
        rejectUnknownReferenceSite(attribute, references, context);
        if (context.fragmentAttributes.has(attribute.name)) {
          attribute.value = rewriteFragment(attribute.value, references);
        } else if (context.ariaAttributes.has(attribute.name)) {
          const tokens = attribute.value.trim().split(/\s+/).filter(Boolean);
          if (tokens.length === 0) fail("svg.normalize.reference_unresolved", "ARIA IDREF list is empty");
          attribute.value = tokens.map((token) => rewriteReference(token, references)).join(" ");
        } else if (attribute.name === "style") {
          attribute.value = normalizeDeclarationList(attribute.value, references);
        } else if (context.presentationAttributes.has(attribute.name) && valueContainsURL(attribute.value)) {
          const value = parseCSS(attribute.value, "value");
          rewriteURLs(value, references);
          attribute.value = requireCSSParser().generate(value);
        }
      }
      if (element.localName === "style") {
        element.textContent = normalizeStylesheet(element.textContent ?? "", root, rootID, references);
      }
    }
  }

  function scanNormalized(root, rootID, originalIDs, context) {
    const ids = new Set([rootID]);
    for (const element of root.querySelectorAll("[id]")) {
      const id = element.getAttribute("id");
      if (!id.startsWith(`${rootID}--id-`) || ids.has(id) || originalIDs.has(id)) {
        fail("svg.normalize.canonical_scan_failed", "normalized descendant ID is invalid");
      }
      ids.add(id);
    }
    function assertTarget(target) {
      if (!ids.has(target)) fail("svg.normalize.canonical_scan_failed", "normalized reference is unresolved");
      if (originalIDs.has(target)) fail("svg.normalize.source_id_survived", "source ID survived normalization");
    }
    for (const element of [root, ...root.querySelectorAll("*")]) {
      for (const attribute of [...element.attributes]) {
        if (context.fragmentAttributes.has(attribute.name)) {
          if (!attribute.value.startsWith("#")) fail("svg.normalize.canonical_scan_failed", "external fragment survived normalization");
          assertTarget(attribute.value.slice(1));
        } else if (context.ariaAttributes.has(attribute.name)) {
          attribute.value.trim().split(/\s+/).filter(Boolean).forEach(assertTarget);
        } else if (context.presentationAttributes.has(attribute.name) && valueContainsURL(attribute.value)) {
          const value = parseCSS(attribute.value, "value");
          requireCSSParser().walk(value, {
            visit: "Url",
            enter(node) {
              if (!node.value.startsWith("#")) fail("svg.normalize.canonical_scan_failed", "external presentation URL survived normalization");
              assertTarget(node.value.slice(1));
            },
          });
        }
      }
      const cssSources = [];
      if (element.hasAttribute("style")) cssSources.push([element.getAttribute("style"), "declarationList"]);
      if (element.localName === "style") cssSources.push([element.textContent ?? "", "stylesheet"]);
      for (const [source, cssContext] of cssSources) {
        const ast = parseCSS(source, cssContext);
        requireCSSParser().walk(ast, (node) => {
          if (node.type === "IdSelector") {
            assertTarget(node.name);
            if (originalIDs.has(node.name)) fail("svg.normalize.source_id_survived", "source ID survived in CSS selector");
          } else if (node.type === "Url") {
            if (!node.value.startsWith("#")) fail("svg.normalize.canonical_scan_failed", "external CSS URL survived normalization");
            assertTarget(node.value.slice(1));
          }
        });
      }
    }
  }

  function canonicalSerialize(root) {
    const serialized = new XMLSerializer().serializeToString(root);
    const reparsed = parseSVG(serialized);
    const repeated = new XMLSerializer().serializeToString(reparsed.root);
    if (serialized !== repeated || reparsed.document.documentElement.namespaceURI !== SVG_NS) {
      fail("svg.normalize.canonical_reparse_failed", "SVG serialization is not stable after reparse");
    }
    return { serialized, reparsed };
  }

  function normalizeSVG(input, rawContext) {
    const context = validateContext(rawContext);
    const rootID = `margo-${context.renderInstanceID}-mermaid-${decimal8(context.blockOrdinal)}`;
    const { root } = parseSVG(input);
    const ids = collectAndRewriteIDs(root, context, rootID);
    rewriteReferenceSites(root, rootID, ids.references, context);
    const canonical = canonicalSerialize(root);
    scanNormalized(canonical.reparsed.root, rootID, new Set(ids.references.keys()), context);
    return Object.freeze({
      algorithm: "margo-mermaid-svg-normalization/v1",
      originalRootID: ids.originalRootID,
      rootID,
      rootMap: Object.freeze([[ids.originalRootID, rootID]]),
      descendantMap: Object.freeze(ids.descendantRows),
      canonicalReparse: true,
      svg: canonical.serialized,
    });
  }

  const api = Object.freeze({ normalizeSVG });
  globalThis.__margo = Object.freeze({ ...(globalThis.__margo ?? {}), ...api });
})();
