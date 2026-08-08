(() => {
  "use strict";

  const SVG_NS = "http://www.w3.org/2000/svg";
  const XLINK_NS = "http://www.w3.org/1999/xlink";
  const XMLNS_NS = "http://www.w3.org/2000/xmlns/";
  const PROFILE_FINGERPRINT = "6e4899904bf55acdd2b5c39a290dbac378a7f6fdf8e904b41c38c4d9c3fdda75";
  const PROFILE_DOMAIN = "margo/mermaid-svg-profile/v1\n";
  const ROOT_ID = /^margo-(ri-[0-9a-z]{8,32})-mermaid-[0-9]{8}$/;
  const CSS_TREE_VERSION = "3.1.0";
  const NUMBER = /^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:e[+-]?\d+)?$/i;
  const LENGTH_PARTS = /^([+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:e[+-]?\d+)?)(%|[a-z]+)?$/i;
  const HEX_COLOR = /^#[0-9a-f]{3,4}(?:[0-9a-f]{2}){0,2}$/i;
  const NAMED_COLOR = /^[a-z]+$/i;
  const SAFE_FONT = /^(?:"?trebuchet ms"?|verdana|arial|sans-serif)(?:\s*,\s*(?:"?trebuchet ms"?|verdana|arial|sans-serif))*$/i;
  const PATH_DATA = /^[\s,\.0-9+\-eEaAcChHlLmMqQsStTvVzZ]*$/;
  const POINTS = /^[\s,\.0-9+\-eE]*$/;
  const TRANSFORM = /^(?:(?:matrix|translate|scale|rotate|skewX|skewY)\([\s,\.0-9+\-eE]+\)\s*)+$/;

  const KEYWORDS = Object.freeze({
    "keyword-baseline": new Set(["auto", "middle", "central", "alphabetic", "hanging"]),
    "keyword-cursor": new Set(["auto", "default", "pointer", "crosshair", "move", "text", "not-allowed"]),
    "keyword-font-style": new Set(["normal", "italic", "oblique"]),
    "keyword-linecap": new Set(["butt", "round", "square"]),
    "keyword-linejoin": new Set(["miter", "round", "bevel"]),
    "keyword-shape-rendering": new Set(["auto", "optimizespeed", "crispedges", "geometricprecision"]),
    "keyword-text-align": new Set(["start", "end", "left", "right", "center"]),
    "keyword-text-anchor": new Set(["start", "middle", "end"]),
    "keyword-text-decoration": new Set(["none", "underline", "overline", "line-through"]),
    "keyword-vector-effect": new Set(["none", "non-scaling-stroke"]),
    "keyword-white-space": new Set(["normal", "pre", "nowrap", "pre-wrap", "pre-line"]),
  });

  class SVGValidationError extends Error {
    constructor(code, message) {
      super(message);
      this.name = "SVGValidationError";
      this.code = code;
    }
  }

  function fail(code, message) {
    throw new SVGValidationError(code, message);
  }

  function canonicalJSON(value) {
    if (value === null || typeof value === "boolean" || typeof value === "string") return JSON.stringify(value);
    if (typeof value === "number" && Number.isFinite(value)) return JSON.stringify(value);
    if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(",")}]`;
    if (value && typeof value === "object") {
      return `{${Object.keys(value).sort().map((key) => `${JSON.stringify(key)}:${canonicalJSON(value[key])}`).join(",")}}`;
    }
    fail("mermaid.profile_mismatch", "profile contains a non-canonical JSON value");
  }

  function sha256Hex(source) {
    const bytes = new TextEncoder().encode(source);
    const length = Math.ceil((bytes.length + 9) / 64) * 64;
    const message = new Uint8Array(length);
    message.set(bytes);
    message[bytes.length] = 0x80;
    const view = new DataView(message.buffer);
    const bitLength = bytes.length * 8;
    view.setUint32(length - 8, Math.floor(bitLength / 0x100000000), false);
    view.setUint32(length - 4, bitLength >>> 0, false);
    const constants = [
      0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
      0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
      0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
      0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
      0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
      0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
      0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
      0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
    ];
    const state = [0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19];
    const words = new Uint32Array(64);
    const rotate = (value, bits) => (value >>> bits) | (value << (32 - bits));
    for (let offset = 0; offset < length; offset += 64) {
      for (let index = 0; index < 16; index += 1) words[index] = view.getUint32(offset + index * 4, false);
      for (let index = 16; index < 64; index += 1) {
        const s0 = rotate(words[index - 15], 7) ^ rotate(words[index - 15], 18) ^ (words[index - 15] >>> 3);
        const s1 = rotate(words[index - 2], 17) ^ rotate(words[index - 2], 19) ^ (words[index - 2] >>> 10);
        words[index] = (words[index - 16] + s0 + words[index - 7] + s1) >>> 0;
      }
      let [a, b, c, d, e, f, g, h] = state;
      for (let index = 0; index < 64; index += 1) {
        const sum1 = rotate(e, 6) ^ rotate(e, 11) ^ rotate(e, 25);
        const choice = (e & f) ^ (~e & g);
        const first = (h + sum1 + choice + constants[index] + words[index]) >>> 0;
        const sum0 = rotate(a, 2) ^ rotate(a, 13) ^ rotate(a, 22);
        const majority = (a & b) ^ (a & c) ^ (b & c);
        const second = (sum0 + majority) >>> 0;
        h = g; g = f; f = e; e = (d + first) >>> 0; d = c; c = b; b = a; a = (first + second) >>> 0;
      }
      [a, b, c, d, e, f, g, h].forEach((value, index) => { state[index] = (state[index] + value) >>> 0; });
    }
    return state.map((value) => value.toString(16).padStart(8, "0")).join("");
  }

  function exactSet(value, name) {
    if (!Array.isArray(value) || value.length === 0 || value.some((entry) => typeof entry !== "string" || entry === "")) {
      fail("mermaid.profile_mismatch", `${name} must be a non-empty string array`);
    }
    const result = new Set(value);
    if (result.size !== value.length) fail("mermaid.profile_mismatch", `${name} must contain unique entries`);
    return result;
  }

  function exactLengthUnits(value) {
    if (!Array.isArray(value) || value.length === 0 || value.some((unit) => typeof unit !== "string" || !/^(?:|%|[a-z]+)$/.test(unit))) {
      fail("mermaid.profile_mismatch", "valueGrammarParameters.lengthUnits must be a non-empty canonical unit array");
    }
    if (new Set(value).size !== value.length || value.some((unit, index) => index > 0 && value[index - 1] >= unit)) {
      fail("mermaid.profile_mismatch", "valueGrammarParameters.lengthUnits must be unique and byte-sorted");
    }
    return new Set(value);
  }

  function validateProfile(profile, fingerprint, family) {
    if (fingerprint !== PROFILE_FINGERPRINT || !profile || typeof profile !== "object" || Array.isArray(profile)) {
      fail("mermaid.profile_mismatch", "Mermaid profile identity mismatch");
    }
    if (sha256Hex(PROFILE_DOMAIN + canonicalJSON(profile)) !== fingerprint) {
      fail("mermaid.profile_mismatch", "Mermaid profile bytes do not match the descriptor fingerprint");
    }
    if (profile.schemaVersion !== "margo-mermaid-svg/v1" || profile.normalizationAlgorithm !== "margo-mermaid-svg-normalization/v2") {
      fail("mermaid.profile_mismatch", "unsupported Mermaid SVG profile");
    }
    if (globalThis.__margoCSSTreePackageVersion !== CSS_TREE_VERSION || profile.normalizationReductions?.cssTreeVersion !== CSS_TREE_VERSION) {
      fail("mermaid.profile_mismatch", "css-tree runtime version mismatch");
    }
    const families = new Set(profile.supportedFamilies?.map((entry) => entry.name));
    if (!families.has(family)) fail("mermaid.svg_family_unsupported", "Mermaid family is not profile-listed");
    const limits = profile.limits;
    if (!limits || Object.values(limits).some((value) => !Number.isSafeInteger(value) || value <= 0)) {
      fail("mermaid.profile_mismatch", "profile resource limits are invalid");
    }
    return Object.freeze({
      profile,
      family,
      limits,
      elements: exactSet(profile.allowedElements, "allowedElements"),
      globalAttributes: exactSet(profile.globalAttributes, "globalAttributes"),
      fragmentAttributes: exactSet(profile.idReferenceSites?.fragmentAttributes, "fragmentAttributes"),
      presentationAttributes: exactSet(profile.idReferenceSites?.presentationAttributes, "presentationAttributes"),
      ariaAttributes: exactSet(profile.idReferenceSites?.ariaIdrefAttributes, "ariaIdrefAttributes"),
      selectorKinds: exactSet(profile.selectorGrammar?.selectors, "selectorGrammar.selectors"),
      combinators: exactSet(profile.selectorGrammar?.combinators, "selectorGrammar.combinators"),
      pseudoClasses: exactSet(profile.selectorGrammar?.pseudoClasses, "selectorGrammar.pseudoClasses"),
      lengthUnits: exactLengthUnits(profile.valueGrammarParameters?.lengthUnits),
    });
  }

  function requireCSSParser() {
    const parser = globalThis.csstree;
    if (!parser || typeof parser.parse !== "function" || typeof parser.generate !== "function" || typeof parser.walk !== "function") {
      fail("mermaid.svg_css_forbidden", "css-tree browser parser is required");
    }
    return parser;
  }

  function parseSVG(input, context) {
    if (typeof input !== "string" || input === "") fail("mermaid.svg_parse_failed", "SVG must be a non-empty string");
    const bytes = new TextEncoder().encode(input).length;
    if (bytes > context.limits.maxSvgBytes) fail("mermaid.svg_resource_limit", "SVG byte limit exceeded");
    const document = new DOMParser().parseFromString(input, "image/svg+xml");
    if (document.getElementsByTagName("parsererror").length !== 0) fail("mermaid.svg_parse_failed", "SVG XML parsing failed");
    const root = document.documentElement;
    if (!root || root.namespaceURI !== SVG_NS || root.localName !== "svg") fail("mermaid.svg_namespace_forbidden", "document root is not SVG");
    return { bytes, document, root };
  }

  function validateIDs(root) {
    const rootID = root.getAttribute("id") ?? "";
    if (!ROOT_ID.test(rootID)) fail("mermaid.svg_id_forbidden", "SVG root ID is not normalized");
    const descendantPattern = new RegExp(`^${rootID.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}--id-[0-9]{8}$`);
    const ids = new Set([rootID]);
    for (const element of root.querySelectorAll("[id]")) {
      const id = element.getAttribute("id") ?? "";
      if (!descendantPattern.test(id) || ids.has(id)) fail("mermaid.svg_id_forbidden", "SVG descendant ID is not normalized or unique");
      ids.add(id);
    }
    return { rootID, ids };
  }

  function localReference(value, ids) {
    if (typeof value !== "string" || !value.startsWith("#") || value.length < 2 || !ids.has(value.slice(1))) {
      fail("mermaid.svg_reference_forbidden", "SVG reference is not a resolved local fragment");
    }
  }

  function localURL(value, ids) {
    const match = /^url\((?:"|')?#([^"')\s]+)(?:"|')?\)$/i.exec(value.trim());
    if (!match || !ids.has(match[1])) fail("mermaid.svg_reference_forbidden", "SVG URL is not a resolved local fragment");
  }

  function finiteNumber(value) {
    return NUMBER.test(value.trim()) && Number.isFinite(Number(value));
  }

  function lengthValue(value, units) {
    const match = LENGTH_PARTS.exec(value.trim());
    if (!match || !Number.isFinite(Number(match[1]))) return false;
    return units.has((match[2] ?? "").toLowerCase());
  }

  function colorValue(value) {
    const source = value.trim();
    if (HEX_COLOR.test(source) || NAMED_COLOR.test(source)) return true;
    const match = /^(rgba?|RGBA?)\((.*)\)$/.exec(source);
    if (!match) return false;
    const parts = match[2].split(",").map((part) => part.trim());
    if (parts.length !== (match[1].toLowerCase() === "rgba" ? 4 : 3)) return false;
    return parts.every((part, index) => {
      if (!/^[+-]?(?:\d+(?:\.\d*)?|\.\d+)%?$/.test(part)) return false;
      const numeric = Number.parseFloat(part);
      if (!Number.isFinite(numeric)) return false;
      if (index === 3) return numeric >= 0 && numeric <= 1 && !part.endsWith("%");
      return numeric >= 0 && numeric <= (part.endsWith("%") ? 100 : 255);
    });
  }

  function validateDataPoints(value) {
    if (!/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(value)) {
      fail("mermaid.svg_attribute_forbidden", "data-points is not canonical base64");
    }
    let decoded;
    try {
      decoded = atob(value);
    } catch {
      fail("mermaid.svg_attribute_forbidden", "data-points is not valid base64");
    }
    if (btoa(decoded) !== value || !/^[\x20-\x7e]*$/.test(decoded)) {
      fail("mermaid.svg_attribute_forbidden", "data-points is not canonical ASCII JSON");
    }
    let points;
    try {
      points = JSON.parse(decoded);
    } catch {
      fail("mermaid.svg_attribute_forbidden", "data-points JSON does not parse");
    }
    if (!Array.isArray(points) || points.length === 0 || points.some((point) => {
      if (!point || typeof point !== "object" || Array.isArray(point)) return true;
      if (Object.keys(point).join(",") !== "x,y") return true;
      return !Number.isFinite(point.x) || !Number.isFinite(point.y);
    }) || JSON.stringify(points) !== decoded) {
      fail("mermaid.svg_attribute_forbidden", "data-points is outside the canonical finite-point grammar");
    }
  }

  function validateValue(grammar, value, context, ids, parser) {
    const source = value.trim();
    let ast;
    try {
      ast = parser.parse(source, { context: "value" });
    } catch {
      fail("mermaid.svg_css_value_forbidden", `CSS value does not parse for ${grammar}`);
    }
    parser.walk(ast, (node) => {
      if (node.type === "Url") {
        const generated = parser.generate(node);
        localURL(generated, ids);
      }
      if (node.type === "Function" && node.name !== "rgb" && node.name !== "rgba") {
        fail("mermaid.svg_css_value_forbidden", `CSS function ${node.name} is not allowed`);
      }
    });

    let valid = false;
    switch (grammar) {
      case "color": valid = colorValue(source); break;
      case "paint": valid = source === "none" || source === "transparent" || colorValue(source) || /^url\(/i.test(source); break;
      case "finite-unit-interval": valid = finiteNumber(source) && Number(source) >= 0 && Number(source) <= 1; break;
      case "finite-nonnegative-number": valid = finiteNumber(source) && Number(source) >= 0; break;
      case "font-weight": valid = source === "normal" || source === "bold" || /^(?:[1-9]00)$/.test(source); break;
      case "length": valid = lengthValue(source, context.lengthUnits); break;
      case "length-or-finite-number": valid = lengthValue(source, context.lengthUnits) || finiteNumber(source); break;
      case "length-list-or-none": valid = source === "none" || source.split(/[\s,]+/).filter(Boolean).every((part) => lengthValue(part, context.lengthUnits)); break;
      case "local-fragment-or-none": valid = source === "none" || /^url\(/i.test(source); break;
      case "embedded-font-family": valid = SAFE_FONT.test(source); break;
      default: valid = KEYWORDS[grammar]?.has(source.toLowerCase()) ?? false;
    }
    if (!valid) fail("mermaid.svg_css_value_forbidden", `CSS value ${source} is outside ${grammar}`);
  }

  function validateDeclaration(node, context, ids, parser) {
    const property = node.property.toLowerCase();
    if (property.startsWith("--")) fail("mermaid.svg_css_forbidden", "CSS custom properties are forbidden");
    const grammar = context.profile.cssProperties[property];
    if (!grammar) fail("mermaid.svg_css_forbidden", `CSS property ${property} is not profile-listed`);
    validateValue(grammar, parser.generate(node.value), context, ids, parser);
  }

  function validateSelector(selector, context, root, rootID, ids, parser) {
    const source = parser.generate(selector);
    if (new TextEncoder().encode(source).length > context.limits.maxSelectorBytes) {
      fail("mermaid.svg_resource_limit", "CSS selector byte limit exceeded");
    }
    const nodes = selector.children.toArray();
    if (nodes.length === 0 || nodes[0].type !== "IdSelector" || nodes[0].name !== rootID) {
      fail("mermaid.svg_css_forbidden", "CSS selector is not rooted at the normalized SVG root");
    }
    let rootCount = 0;
    parser.walk(selector, (node) => {
      switch (node.type) {
        case "IdSelector":
          if (node.name === rootID) rootCount += 1;
          else if (!ids.has(node.name)) fail("mermaid.svg_css_forbidden", "CSS ID selector does not name a local normalized ID");
          break;
        case "ClassSelector":
          if (!context.selectorKinds.has("class")) fail("mermaid.svg_css_forbidden", "class selectors are not profile-listed");
          break;
        case "TypeSelector":
          if (!context.selectorKinds.has("type") || !context.elements.has(node.name)) fail("mermaid.svg_css_forbidden", "type selector is not profile-listed");
          break;
        case "Combinator": {
          const kind = node.name === " " ? "descendant" : node.name === ">" ? "child" : "forbidden";
          if (!context.combinators.has(kind)) fail("mermaid.svg_css_forbidden", "CSS combinator is not profile-listed");
          break;
        }
        case "PseudoClassSelector": {
          const kind = node.name === "nth-child" ? "nth-child-integer" : node.name;
          if (!context.pseudoClasses.has(kind)) fail("mermaid.svg_css_forbidden", "CSS pseudo-class is not profile-listed");
          if (node.name === "nth-child" && !/^:nth-child\([1-9][0-9]*\)$/.test(parser.generate(node))) {
            fail("mermaid.svg_css_forbidden", "nth-child requires a positive integer");
          }
          break;
        }
        case "AttributeSelector":
        case "UniversalSelector":
        case "PseudoElementSelector":
          fail("mermaid.svg_css_forbidden", `CSS ${node.type} is forbidden`);
          break;
        default:
          break;
      }
    });
    if (rootCount !== 1) fail("mermaid.svg_css_forbidden", "CSS root selector must appear exactly once");
    try {
      root.ownerDocument.querySelector(source);
    } catch {
      fail("mermaid.svg_css_forbidden", "CSS selector cannot be evaluated");
    }
  }

  function validateDeclarationList(source, context, ids, parser) {
    let ast;
    try {
      ast = parser.parse(source, { context: "declarationList" });
    } catch {
      fail("mermaid.svg_css_forbidden", "inline CSS does not parse");
    }
    for (const node of ast.children.toArray()) {
      if (node.type !== "Declaration") fail("mermaid.svg_css_forbidden", "inline CSS contains a non-declaration node");
      validateDeclaration(node, context, ids, parser);
    }
  }

  function validateStylesheet(source, context, root, rootID, ids, parser, counters) {
    let ast;
    try {
      ast = parser.parse(source, { context: "stylesheet", positions: false });
    } catch {
      fail("mermaid.svg_css_forbidden", "SVG stylesheet does not parse");
    }
    for (const child of ast.children.toArray()) {
      if (child.type !== "Rule" || child.prelude?.type !== "SelectorList" || child.block?.type !== "Block") {
        fail("mermaid.svg_css_forbidden", "CSS at-rules and unknown rule forms are forbidden");
      }
      counters.cssRules += 1;
      if (counters.cssRules > context.limits.maxCssRules) fail("mermaid.svg_resource_limit", "CSS rule limit exceeded");
      for (const selector of child.prelude.children.toArray()) validateSelector(selector, context, root, rootID, ids, parser);
      for (const declaration of child.block.children.toArray()) {
        if (declaration.type !== "Declaration") fail("mermaid.svg_css_forbidden", "CSS block contains a non-declaration node");
        validateDeclaration(declaration, context, ids, parser);
      }
    }
  }

  function validateAttributeValue(element, attribute, context, ids, parser) {
    const name = attribute.name;
    const value = attribute.value;
    if (context.fragmentAttributes.has(name)) localReference(value, ids);
    if (context.ariaAttributes.has(name)) {
      const references = value.trim().split(/\s+/).filter(Boolean);
      if (references.length === 0 || references.some((id) => !ids.has(id))) fail("mermaid.svg_reference_forbidden", "ARIA IDREF is unresolved");
    }
    if (context.presentationAttributes.has(name) && /url\(/i.test(value)) localURL(value, ids);
    if (name === "style") validateDeclarationList(value, context, ids, parser);
    if (context.profile.cssProperties[name] && name !== "style") {
      validateValue(context.profile.cssProperties[name], value, context, ids, parser);
    }
    if (name === "d" && !PATH_DATA.test(value)) fail("mermaid.svg_attribute_forbidden", "d contains invalid path data");
    if (name === "data-points") validateDataPoints(value);
    if (name === "points" && !POINTS.test(value)) fail("mermaid.svg_attribute_forbidden", "points contains invalid coordinates");
    if (/transform$/i.test(name) && !TRANSFORM.test(value)) fail("mermaid.svg_attribute_forbidden", `${name} contains an invalid transform`);
    if (/^(?:x|y|x1|x2|y1|y2|cx|cy|r|rx|ry|dx|dy|width|height|refX|refY|markerWidth|markerHeight|pathLength|textLength|startOffset|stdDeviation|offset)$/i.test(name)) {
      const values = value.trim().split(/[\s,]+/).filter(Boolean);
      if (values.length === 0 || values.some((entry) => !lengthValue(entry, context.lengthUnits))) fail("mermaid.svg_attribute_forbidden", `${name} contains a non-finite length`);
    }
    if (/url\(/i.test(value) && !context.presentationAttributes.has(name) && name !== "style") {
      fail("mermaid.svg_reference_forbidden", "URL-bearing attribute is not a registered reference site");
    }
  }

  function validateStructure(root, context, ids, parser) {
    const elements = [root, ...root.querySelectorAll("*")];
    if (elements.length > context.limits.maxElements) fail("mermaid.svg_resource_limit", "SVG element limit exceeded");
    let attributes = 0;
    for (const element of elements) {
      if (element.namespaceURI !== SVG_NS) fail("mermaid.svg_namespace_forbidden", "non-SVG element namespace is forbidden");
      if (!context.elements.has(element.localName)) fail("mermaid.svg_element_forbidden", `SVG element ${element.localName} is not profile-listed`);
      const elementAttributes = new Set(context.profile.elementAttributes[element.localName] ?? []);
      for (const attribute of element.attributes) {
        attributes += 1;
        if (attributes > context.limits.maxAttributes) fail("mermaid.svg_resource_limit", "SVG attribute limit exceeded");
        if (attribute.name.toLowerCase().startsWith("on")) fail("mermaid.svg_attribute_forbidden", "event handler attributes are forbidden");
        const namespaceAllowed = attribute.namespaceURI === null || attribute.namespaceURI === XLINK_NS || attribute.namespaceURI === XMLNS_NS;
        if (!namespaceAllowed || (attribute.namespaceURI === XLINK_NS && attribute.name !== "xlink:href") || (attribute.namespaceURI === XMLNS_NS && !["xmlns", "xmlns:xlink"].includes(attribute.name))) {
          fail("mermaid.svg_namespace_forbidden", "attribute namespace is forbidden");
        }
        if (!context.globalAttributes.has(attribute.name) && !elementAttributes.has(attribute.name)) {
          fail("mermaid.svg_attribute_forbidden", `SVG attribute ${attribute.name} is not profile-listed for ${element.localName}`);
        }
        validateAttributeValue(element, attribute, context, ids, parser);
      }
    }
    return { elements: elements.length, attributes };
  }

  function validateSVG(input, descriptor) {
    if (!descriptor || typeof descriptor !== "object" || Array.isArray(descriptor)) fail("mermaid.profile_mismatch", "validation descriptor is required");
    const context = validateProfile(descriptor.profile, descriptor.profileFingerprint, descriptor.family);
    const parser = requireCSSParser();
    const parsed = parseSVG(input, context);
    const { rootID, ids } = validateIDs(parsed.root);
    const structure = validateStructure(parsed.root, context, ids, parser);
    const counters = { cssRules: 0 };
    const styles = [...parsed.root.querySelectorAll("style")];
    for (const style of styles) validateStylesheet(style.textContent ?? "", context, parsed.root, rootID, ids, parser, counters);
    const serialized = new XMLSerializer().serializeToString(parsed.root);
    const reparsed = new DOMParser().parseFromString(serialized, "image/svg+xml");
    if (reparsed.getElementsByTagName("parsererror").length !== 0 || serialized !== input) {
      fail("mermaid.svg_noncanonical", "validated SVG is not the canonical normalized serialization");
    }
    return Object.freeze({
      svg: serialized,
      svgBytes: parsed.bytes,
      elements: structure.elements,
      attributes: structure.attributes,
      cssRules: counters.cssRules,
      cssText: styles.map((style) => style.textContent ?? "").join("\n"),
      family: context.family,
      profileFingerprint: descriptor.profileFingerprint,
      canonicalReparse: true,
    });
  }

  globalThis.__margo = Object.freeze({ ...(globalThis.__margo ?? {}), validateSVG });
})();
