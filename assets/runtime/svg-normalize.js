(() => {
  "use strict";

  const SVG_NS = "http://www.w3.org/2000/svg";
  const INSTANCE_PATTERN = /^ri-[0-9a-z]{8,32}$/;
  const MAX_ORDINAL = 99999999;
  const PROFILE_FINGERPRINT = "6e4899904bf55acdd2b5c39a290dbac378a7f6fdf8e904b41c38c4d9c3fdda75";
  const NORMALIZATION_ALGORITHM = "margo-mermaid-svg-normalization/v2";
  const CSS_TREE_VERSION = "3.1.0";
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

  function validateStringArray(value, name) {
    if (!Array.isArray(value) || value.length === 0 || value.some((entry) => typeof entry !== "string" || entry === "")) {
      fail("svg.normalize.context_invalid", `${name} must be a non-empty string array`);
    }
    return new Set(value);
  }

  function canonicalJSON(value) {
    if (value === null || typeof value === "boolean" || typeof value === "string") return JSON.stringify(value);
    if (typeof value === "number" && Number.isFinite(value)) return JSON.stringify(value);
    if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(",")}]`;
    if (value && typeof value === "object") {
      return `{${Object.keys(value).sort().map((key) => `${JSON.stringify(key)}:${canonicalJSON(value[key])}`).join(",")}}`;
    }
    fail("svg.normalize.context_invalid", "profile contains a non-canonical JSON value");
  }

  function validateReductionRows(value, name) {
    if (!Array.isArray(value) || value.length === 0) {
      fail("svg.normalize.context_invalid", `${name} must be a non-empty array`);
    }
    const rows = value.map(canonicalJSON);
    if (new Set(rows).size !== rows.length) {
      fail("svg.normalize.context_invalid", `${name} must be unique`);
    }
    return value;
  }

  function validateProfile(value, fingerprint) {
    if (fingerprint !== PROFILE_FINGERPRINT) {
      fail("mermaid.profile_mismatch", "Mermaid profile fingerprint mismatch");
    }
    if (!value || typeof value !== "object" || Array.isArray(value) || value.normalizationAlgorithm !== NORMALIZATION_ALGORITHM) {
      fail("mermaid.profile_mismatch", "Mermaid normalization profile is invalid");
    }
    const reductions = value.normalizationReductions;
    const keys = reductions && typeof reductions === "object" && !Array.isArray(reductions) ? Object.keys(reductions).sort() : [];
    const wantedKeys = ["algorithm", "cssTreeVersion", "deadSelectorRules", "discardedAtRules", "discardedDeclarations", "sequenceSelectorRewrites"].sort();
    if (canonicalJSON(keys) !== canonicalJSON(wantedKeys) || reductions.algorithm !== NORMALIZATION_ALGORITHM || reductions.cssTreeVersion !== CSS_TREE_VERSION) {
      fail("mermaid.profile_mismatch", "Mermaid reduction profile is invalid");
    }
    if (globalThis.__margoCSSTreePackageVersion !== reductions.cssTreeVersion) {
      fail("mermaid.profile_mismatch", "css-tree runtime version mismatch");
    }
    validateReductionRows(reductions.deadSelectorRules, "deadSelectorRules");
    validateReductionRows(reductions.discardedAtRules, "discardedAtRules");
    validateReductionRows(reductions.discardedDeclarations, "discardedDeclarations");
    if (!Array.isArray(reductions.sequenceSelectorRewrites) || reductions.sequenceSelectorRewrites.length !== 3) {
      fail("mermaid.profile_mismatch", "sequenceSelectorRewrites is invalid");
    }
    return { profile: value, reductions };
  }

  function validateContext(value) {
    if (!value || typeof value !== "object" || Array.isArray(value)) {
      fail("svg.normalize.context_invalid", "normalization context is required");
    }
    if (typeof value.sourceRootID !== "string" || value.sourceRootID === "" || !INSTANCE_PATTERN.test(value.renderInstanceID)) {
      fail("svg.normalize.context_invalid", "source root or render instance is invalid");
    }
    if (value.family !== "flowchart" && value.family !== "sequence") {
      fail("svg.normalize.context_invalid", "supported Mermaid family is required");
    }
    const { profile, reductions } = validateProfile(value.profile, value.profileFingerprint);
    const registry = profile.idReferenceSites;
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
      family: value.family,
      profile,
      reductions,
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

  function collectIDMaps(root, context, rootID) {
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
    return {
      originalRootID,
      rootMap,
      descendantMap,
      descendantRows,
      references: new Map([...rootMap, ...descendantMap]),
    };
  }

  function applyIDMaps(root, ids) {
    root.setAttribute("id", ids.rootMap.get(ids.originalRootID));
    [...root.querySelectorAll("[id]")].forEach((element) => {
      element.setAttribute("id", ids.descendantMap.get(element.getAttribute("id")));
    });
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

  function canonicalReductionSource(node, ids) {
    const parser = requireCSSParser();
    if (typeof parser.clone !== "function") {
      fail("svg.normalize.css_parser_missing", "css-tree clone support is required");
    }
    const clone = parser.clone(node);
    const reductionIDs = new Map([[ids.originalRootID, "margo-reduction-root"]]);
    ids.descendantRows.forEach(([sourceID], ordinal) => {
      reductionIDs.set(sourceID, `margo-reduction-id-${decimal8(ordinal)}`);
    });
    parser.walk(clone, (current) => {
      if (current.type === "IdSelector") {
        const target = reductionIDs.get(current.name);
        if (!target) fail("mermaid.svg_css_reduction_unknown", `unmapped CSS ID selector: ${current.name}`);
        current.name = target;
      } else if (current.type === "Url" && typeof current.value === "string" && current.value.startsWith("#")) {
        const target = reductionIDs.get(current.value.slice(1));
        if (!target) fail("mermaid.svg_css_reduction_unknown", `unmapped CSS URL: ${current.value}`);
        current.value = `#${target}`;
      }
    });
    return parser.generate(clone);
  }

  function cssNodeSHA256(node, ids) {
    return sha256Hex(`margo/mermaid-css-node/v1\n${canonicalReductionSource(node, ids)}`);
  }

  function rowSet(rows) {
    return new Set(rows.map(canonicalJSON));
  }

  function hasAttributeSelector(selector) {
    let found = false;
    requireCSSParser().walk(selector, (node) => {
      if (node.type === "AttributeSelector") found = true;
    });
    return found;
  }

  function querySelectorCount(root, selector) {
    try {
      return root.ownerDocument.querySelectorAll(selector).length;
    } catch (error) {
      fail("svg.normalize.css_parse_failed", `selector evaluation failed: ${selector}`);
    }
  }

  function rewriteSequenceSelector(selector, source, root, ids, context) {
    if (context.family !== "sequence") {
      fail("mermaid.svg_css_sequence_selector_invalid", "sequence selector appeared outside sequence family");
    }
    const parser = requireCSSParser();
    const row = context.reductions.sequenceSelectorRewrites.find((candidate) => {
      const expected = `#${CSS.escape(ids.originalRootID)} [id$=${JSON.stringify(candidate.suffix)}]${candidate.tail ? ` ${candidate.tail}` : ""}`;
      return source === expected;
    });
    if (!row || !["sequence-arrowhead", "sequence-crosshead", "sequence-number"].includes(row.pattern)) {
      fail("mermaid.svg_css_sequence_selector_invalid", `unprofiled sequence selector: ${source}`);
    }
    const carriers = [...root.ownerDocument.querySelectorAll(`[id$=${JSON.stringify(row.suffix)}]`)];
    if (carriers.length !== 1) {
      fail("mermaid.svg_css_sequence_selector_invalid", "sequence selector must have exactly one carrier");
    }
    const carrier = carriers[0];
    const sourceID = carrier.getAttribute("id");
    const targetID = ids.descendantMap.get(sourceID);
    const targets = row.tail ? [...carrier.querySelectorAll(row.tail)] : [carrier];
    if (!targetID || targets.length !== 1) {
      fail("mermaid.svg_css_sequence_selector_invalid", "sequence selector must have one mapped target");
    }
    const rewritten = `#${CSS.escape(targetID)}${row.tail ? ` ${row.tail}` : ""}`;
    parseCSS(rewritten, "selector");
    return rewritten;
  }

  function isProfiledSequenceSelector(source, ids, context) {
    return context.reductions.sequenceSelectorRewrites.some((candidate) => {
      const expected = `#${CSS.escape(ids.originalRootID)} [id$=${JSON.stringify(candidate.suffix)}]${candidate.tail ? ` ${candidate.tail}` : ""}`;
      return source === expected;
    });
  }

  function declarationsReferenceAnimation(records, name) {
    const parser = requireCSSParser();
    for (const record of records) {
      for (const declaration of record.declarations) {
        if (declaration.property !== "animation" && declaration.property !== "animation-name") continue;
        let referenced = false;
        parser.walk(declaration.value, (node) => {
          if (node.type === "Identifier" && node.name === name) referenced = true;
        });
        if (referenced) return true;
      }
    }
    return false;
  }

  function renderReducedSheet(sheet, skippedDeclaration = null) {
    const parser = requireCSSParser();
    return sheet.records.map((record) => {
      if (record.kind === "atrule") return record.discarded ? "" : parser.generate(record.node);
      if (record.branches.length === 0) return "";
      const declarations = record.declarations
        .filter((declaration) => declaration !== skippedDeclaration)
        .map((declaration) => parser.generate(declaration));
      return `${record.branches.join(",")}{${declarations.join(";")}}`;
    }).join("");
  }

  function computedFilterValues(root, selector, sheets, skippedDeclaration = null) {
    const host = document.createElement("div");
    host.setAttribute("aria-hidden", "true");
    host.style.cssText = "position:fixed;left:-100000px;top:0;visibility:hidden;pointer-events:none";
    const clone = document.importNode(root, true);
    const cloneStyles = [...clone.querySelectorAll("style")];
    if (cloneStyles.length !== sheets.length) {
      fail("mermaid.svg_css_noop_unproven", "stylesheet clone count changed");
    }
    cloneStyles.forEach((element, index) => {
      element.textContent = renderReducedSheet(sheets[index], skippedDeclaration);
    });
    host.append(clone);
    document.body.append(host);
    try {
      const matches = [...host.querySelectorAll(selector)];
      return matches.map((element) => getComputedStyle(element).filter);
    } finally {
      host.remove();
    }
  }

  function reduceStylesheets(root, ids, context) {
    const parser = requireCSSParser();
    const deadRows = rowSet(context.reductions.deadSelectorRules);
    const atRuleRows = rowSet(context.reductions.discardedAtRules);
    const declarationRows = rowSet(context.reductions.discardedDeclarations);
    const report = { deadSelectorBranches: 0, sequenceSelectorRewrites: 0, discardedAtRules: 0, discardedDeclarations: 0 };
    const styleElements = [...root.querySelectorAll("style")];
    const sheets = styleElements.map((element) => {
      const ast = parseCSS(element.textContent ?? "", "stylesheet");
      const records = [];
      for (const node of ast.children) {
        if (node.type === "Atrule") {
          if (node.name !== "keyframes") {
            fail("mermaid.svg_css_at_rule_forbidden", `forbidden at-rule: ${node.name}`);
          }
          parser.walk(node, {
            visit: "Rule",
            enter(nestedRule) {
              if (nestedRule.prelude?.type !== "SelectorList" || nestedRule.block?.type !== "Block") return;
              for (const selector of nestedRule.prelude.children) {
                const source = parser.generate(selector);
                if (querySelectorCount(root, source) !== 0) {
                  fail("mermaid.svg_css_reduction_unknown", `live nested selector: ${source}`);
                }
                const row = {
                  family: context.family,
                  selectorSHA256: cssNodeSHA256(selector, ids),
                  declarationsSHA256: cssNodeSHA256(nestedRule.block, ids),
                };
                if (!deadRows.has(canonicalJSON(row))) {
                  fail("mermaid.svg_css_reduction_unknown", `unprofiled nested selector: ${source}`);
                }
                report.deadSelectorBranches += 1;
              }
            },
          });
          records.push({ kind: "atrule", node, discarded: false });
          continue;
        }
        if (node.type !== "Rule" || node.prelude?.type !== "SelectorList" || node.block?.type !== "Block") {
          fail("mermaid.svg_css_at_rule_forbidden", "unsupported CSS node before normalization");
        }
        const branches = [];
        const sourceBranches = [];
        for (const selector of node.prelude.children) {
          const source = parser.generate(selector);
          if (hasAttributeSelector(selector) && isProfiledSequenceSelector(source, ids, context)) {
            sourceBranches.push(source);
            branches.push(rewriteSequenceSelector(selector, source, root, ids, context));
            report.sequenceSelectorRewrites += 1;
            continue;
          }
          const count = querySelectorCount(root, source);
          if (count === 0) {
            const row = {
              family: context.family,
              selectorSHA256: cssNodeSHA256(selector, ids),
              declarationsSHA256: cssNodeSHA256(node.block, ids),
            };
            if (!deadRows.has(canonicalJSON(row))) {
              fail("mermaid.svg_css_reduction_unknown", `unprofiled dead selector: ${source}`);
            }
            report.deadSelectorBranches += 1;
            continue;
          }
          sourceBranches.push(source);
          if (hasAttributeSelector(selector)) {
            branches.push(rewriteSequenceSelector(selector, source, root, ids, context));
            report.sequenceSelectorRewrites += 1;
          } else {
            branches.push(source);
          }
        }
        records.push({
          kind: "rule",
          node,
          branches,
          sourceBranches,
          declarations: [...node.block.children],
        });
      }
      return { records };
    });
    const ruleRecords = sheets.flatMap((sheet) => sheet.records.filter((record) => record.kind === "rule" && record.branches.length !== 0));
    for (const sheet of sheets) {
      for (const record of sheet.records) {
        if (record.kind !== "atrule") continue;
        if (record.node.name !== "keyframes") {
          fail("mermaid.svg_css_at_rule_forbidden", `forbidden at-rule: ${record.node.name}`);
        }
        const name = parser.generate(record.node.prelude);
        const row = { family: context.family, name, nodeSHA256: cssNodeSHA256(record.node, ids) };
        if (!atRuleRows.has(canonicalJSON(row)) || declarationsReferenceAnimation(ruleRecords, name)) {
          fail("mermaid.svg_css_at_rule_forbidden", `unprofiled or referenced keyframes: ${name}`);
        }
        record.discarded = true;
        report.discardedAtRules += 1;
      }
    }
    for (const record of ruleRecords) {
      for (const declaration of [...record.declarations]) {
        if (declaration.property !== "filter") continue;
        const value = parser.generate(declaration.value);
        const sourceSelector = `#${CSS.escape(ids.originalRootID)} .labelBox`;
        const row = {
          family: context.family,
          selector: ".labelBox",
          property: "filter",
          value,
          nodeSHA256: cssNodeSHA256(declaration, ids),
        };
        if (value !== "none" || !record.sourceBranches.includes(sourceSelector) || !declarationRows.has(canonicalJSON(row))) {
          fail("mermaid.svg_css_noop_unproven", "unprofiled filter declaration");
        }
        const before = computedFilterValues(root, sourceSelector, sheets);
        const after = computedFilterValues(root, sourceSelector, sheets, declaration);
        if (before.length === 0 || before.some((item) => item !== "none") || after.length !== before.length || after.some((item) => item !== "none")) {
          fail("mermaid.svg_css_noop_unproven", "filter:none computed no-op was not proven");
        }
        record.declarations = record.declarations.filter((item) => item !== declaration);
        report.discardedDeclarations += 1;
      }
    }
    styleElements.forEach((element, index) => {
      element.textContent = renderReducedSheet(sheets[index]);
    });
    return Object.freeze(report);
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
      if (node.type === "IdSelector") {
        if (references.has(node.name)) {
          node.name = rewriteReference(node.name, references);
        } else if (![...references.values()].includes(node.name)) {
          fail("svg.normalize.reference_unresolved", `unresolved SVG reference: ${node.name}`);
        }
      }
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
    const ids = collectIDMaps(root, context, rootID);
    const reductions = reduceStylesheets(root, ids, context);
    applyIDMaps(root, ids);
    rewriteReferenceSites(root, rootID, ids.references, context);
    const canonical = canonicalSerialize(root);
    scanNormalized(canonical.reparsed.root, rootID, new Set(ids.references.keys()), context);
    return Object.freeze({
      algorithm: NORMALIZATION_ALGORITHM,
      originalRootID: ids.originalRootID,
      rootID,
      rootMap: Object.freeze([[ids.originalRootID, rootID]]),
      descendantMap: Object.freeze(ids.descendantRows),
      reductions,
      canonicalReparse: true,
      svg: canonical.serialized,
    });
  }

  const api = Object.freeze({ normalizeSVG });
  globalThis.__margo = Object.freeze({ ...(globalThis.__margo ?? {}), ...api });
})();
