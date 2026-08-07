import { expect, test } from "@playwright/test";
import { createRequire } from "node:module";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, "../../..");
const normalizerPath = path.join(root, "assets/runtime/svg-normalize.js");
const mermaidModuleURL = "https://margo.invalid/mermaid/mermaid.esm.min.mjs";
const mermaidAssetRoot = path.join(root, "assets/mermaid/11.16.1");
const fixturesRoot = path.join(root, "testdata/mermaid/normalization");
const positiveRoot = path.join(root, "testdata/mermaid/positive");
const profile = JSON.parse(fs.readFileSync(path.join(root, "profiles/margo-mermaid-svg-v1.json"), "utf8"));
const vectors = JSON.parse(fs.readFileSync(path.join(fixturesRoot, "vectors.json"), "utf8"));
const require = createRequire(import.meta.url);
const cssTreeEntry = require.resolve("css-tree");
const cssTreePath = path.resolve(path.dirname(cssTreeEntry), "../dist/csstree.js");
const cssTreePackageVersion = require("css-tree/package.json").version;
const profileFingerprint = "cd9edc30096cae2622b8e3489361465b6bcba66ad891934353bfdfb0035fff24";

function fixture(name) {
  return fs.readFileSync(path.join(fixturesRoot, name), "utf8");
}

function context(vector) {
  return {
    sourceRootID: vector.sourceRootID,
    renderInstanceID: vector.renderInstanceID ?? "ri-0000000a",
    blockOrdinal: vector.blockOrdinal ?? 0,
    family: vector.family ?? "sequence",
    profile,
    profileFingerprint,
  };
}

async function loadNormalizer(page) {
  await page.goto(pathToFileURL(path.join(root, "test/browser/fixtures/probe.html")).href);
  await page.addScriptTag({ path: cssTreePath });
  await page.evaluate((version) => { globalThis.__margoCSSTreePackageVersion = version; }, cssTreePackageVersion);
  await page.addScriptTag({ path: normalizerPath });
  return page.evaluate(() => Object.keys(globalThis.__margo).sort());
}

async function installMermaidRoute(page) {
  await page.route("https://margo.invalid/mermaid/**", async (route) => {
    const relative = decodeURIComponent(new URL(route.request().url()).pathname).replace(/^\/mermaid\//, "");
    const target = path.resolve(mermaidAssetRoot, relative);
    if (!target.startsWith(`${mermaidAssetRoot}${path.sep}`)) return route.abort("blockedbyclient");
    return route.fulfill({ body: fs.readFileSync(target), contentType: "text/javascript; charset=utf-8" });
  });
}

function pinnedFixtures() {
  return fs.readdirSync(positiveRoot)
    .filter((name) => name.endsWith(".mmd"))
    .sort()
    .map((name) => ({ family: name.split("-")[0], name, source: fs.readFileSync(path.join(positiveRoot, name), "utf8") }));
}

test("@svg-normalize reduces and normalizes all eight pinned Mermaid fixtures", async ({ page }) => {
  const nonLocalRequests = [];
  page.on("request", (request) => {
    if (!request.url().startsWith("file:") && !request.url().startsWith("https://margo.invalid/mermaid/")) nonLocalRequests.push(request.url());
  });
  await installMermaidRoute(page);
  await loadNormalizer(page);
  const fixtures = pinnedFixtures();
  const results = await page.evaluate(async ({ fixtures, moduleURL, profile, profileFingerprint }) => {
    const mermaid = (await import(moduleURL)).default;
    mermaid.initialize({
      securityLevel: "strict",
      startOnLoad: false,
      deterministicIds: true,
      deterministicIDSeed: "margo-m4-normalization-v2",
      htmlLabels: false,
      flowchart: { htmlLabels: false },
      sequence: { htmlLabels: false },
      themeCSS: "",
      look: "classic",
      layout: "dagre",
      fontFamily: "Arial, sans-serif",
    });
    const output = [];
    for (let index = 0; index < fixtures.length; index += 1) {
      const item = fixtures[index];
      const sourceRootID = `msrc-${String(index).padStart(8, "0")}`;
      try {
        const rendered = await mermaid.render(sourceRootID, item.source);
        const normalized = globalThis.__margo.normalizeSVG(rendered.svg, {
          sourceRootID,
          renderInstanceID: "ri-0000000a",
          blockOrdinal: index,
          family: item.family,
          profile,
          profileFingerprint,
        });
        output.push({ name: item.name, algorithm: normalized.algorithm, reductions: normalized.reductions, svg: normalized.svg });
      } catch (error) {
        output.push({ name: item.name, code: error.code ?? error.message, message: error.message });
      }
    }
    return output;
  }, { fixtures, moduleURL: mermaidModuleURL, profile, profileFingerprint });

  expect(results).toHaveLength(8);
  const totals = { deadSelectorBranches: 0, sequenceSelectorRewrites: 0, discardedAtRules: 0, discardedDeclarations: 0 };
  for (const result of results) {
    expect(result.code, `${result.name}: ${result.message ?? ""}`).toBeUndefined();
    expect(result.algorithm, result.name).toBe("margo-mermaid-svg-normalization/v2");
    expect(result.svg, result.name).not.toMatch(/\[id\$=|@(?:-\w+-)?keyframes|\bfilter\s*:/);
    for (const key of Object.keys(totals)) totals[key] += result.reductions[key];
  }
  expect(totals).toEqual({ deadSelectorBranches: 427, sequenceSelectorRewrites: 12, discardedAtRules: 16, discardedDeclarations: 2 });
  expect(nonLocalRequests).toEqual([]);
});

test("@svg-normalize fails when any approved reduction row is removed", async ({ page }) => {
  test.setTimeout(120_000);
  await installMermaidRoute(page);
  await loadNormalizer(page);
  const failures = await page.evaluate(async ({ fixtures, moduleURL, profile, profileFingerprint }) => {
    const mermaid = (await import(moduleURL)).default;
    mermaid.initialize({
      securityLevel: "strict",
      startOnLoad: false,
      deterministicIds: true,
      deterministicIDSeed: "margo-m4-row-mutations-v2",
      htmlLabels: false,
      flowchart: { htmlLabels: false },
      sequence: { htmlLabels: false },
      themeCSS: "",
      look: "classic",
      layout: "dagre",
      fontFamily: "Arial, sans-serif",
    });
    const rendered = [];
    for (let index = 0; index < fixtures.length; index += 1) {
      const fixture = fixtures[index];
      const sourceRootID = `msrc-${String(index).padStart(8, "0")}`;
      rendered.push({ ...fixture, sourceRootID, svg: (await mermaid.render(sourceRootID, fixture.source)).svg, blockOrdinal: index });
    }
    const clone = (value) => JSON.parse(JSON.stringify(value));
    const attempt = (fixture, mutated) => {
      try {
        globalThis.__margo.normalizeSVG(fixture.svg, {
          sourceRootID: fixture.sourceRootID,
          renderInstanceID: "ri-0000000a",
          blockOrdinal: fixture.blockOrdinal,
          family: fixture.family,
          profile: mutated,
          profileFingerprint,
        });
        return "accepted";
      } catch (error) {
        return error.code;
      }
    };
    const failures = [];
    const categories = ["deadSelectorRules", "discardedAtRules", "discardedDeclarations", "sequenceSelectorRewrites"];
    for (const category of categories) {
      const rows = profile.normalizationReductions[category];
      for (let index = 0; index < rows.length; index += 1) {
        const mutated = clone(profile);
        mutated.normalizationReductions[category].splice(index, 1);
        const row = rows[index];
        const candidates = rendered.filter((fixture) => !row.family || fixture.family === row.family);
        if (!candidates.some((fixture) => attempt(fixture, mutated) !== "accepted")) {
          failures.push({ category, index, row });
        }
      }
    }
    return failures;
  }, { fixtures: pinnedFixtures(), moduleURL: mermaidModuleURL, profile, profileFingerprint });
  expect(failures).toEqual([]);
});

test("@svg-normalize rewrites root, descendants, attributes, ARIA, selectors, and CSS URLs", async ({ page }) => {
  const keys = await loadNormalizer(page);
  expect(keys).toContain("normalizeSVG");
  const vector = vectors.positive[0];
  const output = await page.evaluate(({ svg, value }) => globalThis.__margo.normalizeSVG(svg, value), {
    svg: fixture(vector.path),
    value: context(vector),
  });

  expect(output.rootID).toBe(vector.normalizedRootID);
  expect(output.algorithm).toBe("margo-mermaid-svg-normalization/v2");
  expect(output.originalRootID).toBe(vector.sourceRootID);
  expect(output.descendantMap).toHaveLength(vector.descendantCount);
  expect(output.canonicalReparse).toBe(true);
  expect(output.svg).not.toContain(vector.sourceRootID);
  expect(output.svg.match(new RegExp(`#${vector.normalizedRootID} \\.edge-pattern-dashed`, "g"))).toHaveLength(1);
  expect(output.svg.match(new RegExp(`#${vector.normalizedRootID} \\.default`, "g"))).toHaveLength(1);

  const scan = await page.evaluate((serialized) => {
    const document = new DOMParser().parseFromString(serialized, "image/svg+xml");
    const root = document.documentElement;
    const ids = [...root.querySelectorAll("[id]")].map((element) => element.id);
    return {
      rootID: root.id,
      descendantIDs: ids,
      hrefs: [...root.querySelectorAll("[href]")].map((element) => element.getAttribute("href")),
      xlinks: [...root.querySelectorAll("[xlink\\:href]")].map((element) => element.getAttribute("xlink:href")),
      labelledBy: root.querySelector("g").getAttribute("aria-labelledby").split(/\s+/),
      describedBy: root.querySelector("g").getAttribute("aria-describedby").split(/\s+/),
    };
  }, output.svg);
  expect(scan.rootID).toBe(vector.normalizedRootID);
  expect(scan.descendantIDs.every((id) => id.startsWith(`${vector.normalizedRootID}--id-`))).toBe(true);
  expect([...scan.hrefs, ...scan.xlinks].every((value) => value.startsWith(`#${vector.normalizedRootID}--id-`))).toBe(true);
  expect([...scan.labelledBy, ...scan.describedBy].every((value) => value.startsWith(`${vector.normalizedRootID}--id-`))).toBe(true);
});

test("@svg-normalize rejects a profile fingerprint mismatch before returning SVG bytes", async ({ page }) => {
  await loadNormalizer(page);
  const vector = vectors.positive[0];
  const result = await page.evaluate(({ svg, value }) => {
    try {
      globalThis.__margo.normalizeSVG(svg, { ...value, profileFingerprint: "0".repeat(64) });
      return { code: "accepted" };
    } catch (error) {
      return { code: error.code };
    }
  }, { svg: fixture(vector.path), value: context(vector) });
  expect(result).toEqual({ code: "mermaid.profile_mismatch" });

  const parserResult = await page.evaluate(({ svg, value }) => {
    const original = globalThis.__margoCSSTreePackageVersion;
    globalThis.__margoCSSTreePackageVersion = "3.1.1";
    try {
      globalThis.__margo.normalizeSVG(svg, value);
      return { code: "accepted" };
    } catch (error) {
      return { code: error.code };
    } finally {
      globalThis.__margoCSSTreePackageVersion = original;
    }
  }, { svg: fixture(vector.path), value: context(vector) });
  expect(parserResult).toEqual({ code: "mermaid.profile_mismatch" });
});

test("@svg-normalize rejects zero, multiple, and malformed sequence selector rewrites", async ({ page }) => {
  await loadNormalizer(page);
  const cases = [
    { name: "zero carrier", selector: '#msrc-sequence [id$="-arrowhead"] path', body: '<g id="other"><path d="M0 0"/></g>' },
    { name: "multiple carriers", selector: '#msrc-sequence [id$="-arrowhead"] path', body: '<g id="a-arrowhead"><path d="M0 0"/></g><g id="b-arrowhead"><path d="M0 0"/></g>' },
    { name: "wrong operator", selector: '#msrc-sequence [id*="-arrowhead"] path', body: '<g id="a-arrowhead"><path d="M0 0"/></g>' },
    { name: "wrong tail", selector: '#msrc-sequence [id$="-arrowhead"] rect', body: '<g id="a-arrowhead"><rect width="1" height="1"/></g>' },
    { name: "wrong suffix", selector: '#msrc-sequence [id$="-unknown"] path', body: '<g id="a-unknown"><path d="M0 0"/></g>' },
  ];
  const results = await page.evaluate(({ cases, profile, profileFingerprint }) => cases.map((item) => {
    const svg = `<svg xmlns="http://www.w3.org/2000/svg" id="msrc-sequence"><style>${item.selector}{fill:red}</style>${item.body}</svg>`;
    try {
      globalThis.__margo.normalizeSVG(svg, {
        sourceRootID: "msrc-sequence",
        renderInstanceID: "ri-0000000a",
        blockOrdinal: 0,
        family: "sequence",
        profile,
        profileFingerprint,
      });
      return { name: item.name, code: "accepted" };
    } catch (error) {
      return { name: item.name, code: error.code };
    }
  }), { cases, profile, profileFingerprint });
  expect(results).toEqual(cases.map((item) => ({ name: item.name, code: "mermaid.svg_css_sequence_selector_invalid" })));
});

test("@svg-normalize rejects unprofiled CSS reductions before returning accepted bytes", async ({ page }) => {
  await installMermaidRoute(page);
  await loadNormalizer(page);
  const sequence = pinnedFixtures().find((item) => item.name === "sequence-conditional.mmd");
  const results = await page.evaluate(async ({ moduleURL, profile, profileFingerprint, sequence }) => {
    const mermaid = (await import(moduleURL)).default;
    mermaid.initialize({
      securityLevel: "strict",
      startOnLoad: false,
      deterministicIds: true,
      deterministicIDSeed: "margo-m4-negative-v2",
      htmlLabels: false,
      flowchart: { htmlLabels: false },
      sequence: { htmlLabels: false },
      themeCSS: "",
      look: "classic",
      layout: "dagre",
      fontFamily: "Arial, sans-serif",
    });
    const sourceRootID = "msrc-negative";
    const base = (await mermaid.render(sourceRootID, sequence.source)).svg;
    const SVG_NS = "http://www.w3.org/2000/svg";
    const mutate = (operation) => {
      const document = new DOMParser().parseFromString(base, "image/svg+xml");
      const root = document.documentElement;
      const live = root.querySelector("g");
      live.classList.add("margo-live");
      const addStyle = (source) => {
        const style = document.createElementNS(SVG_NS, "style");
        style.textContent = source;
        root.append(style);
      };
      if (operation === "referenced-keyframes") addStyle(`#${sourceRootID} .margo-live{animation-name:dash}`);
      if (operation === "renamed-keyframes") {
        const style = [...root.querySelectorAll("style")].find((item) => item.textContent.includes("@keyframes dash"));
        style.textContent = style.textContent.replace("@keyframes dash", "@keyframes dash-renamed");
      }
      if (operation === "unknown-at-rule") addStyle(`@media print{#${sourceRootID} .margo-live{fill:red}}`);
      if (operation === "unproven-filter") root.querySelector(".labelBox").style.filter = "blur(1px)";
      return new XMLSerializer().serializeToString(root);
    };
    const cases = [
      ["referenced-keyframes", "mermaid.svg_css_at_rule_forbidden"],
      ["renamed-keyframes", "mermaid.svg_css_at_rule_forbidden"],
      ["unknown-at-rule", "mermaid.svg_css_at_rule_forbidden"],
      ["unproven-filter", "mermaid.svg_css_noop_unproven"],
    ];
    const output = cases.map(([name, expected]) => {
      try {
        globalThis.__margo.normalizeSVG(mutate(name), {
          sourceRootID,
          renderInstanceID: "ri-0000000a",
          blockOrdinal: 0,
          family: "sequence",
          profile,
          profileFingerprint,
        });
        return { name, expected, code: "accepted" };
      } catch (error) {
        return { name, expected, code: error.code };
      }
    });
    const unlistedDead = `<svg xmlns="${SVG_NS}" id="msrc-dead"><style>.missing{fill:red}</style><g id="live"/></svg>`;
    try {
      globalThis.__margo.normalizeSVG(unlistedDead, {
        sourceRootID: "msrc-dead",
        renderInstanceID: "ri-0000000a",
        blockOrdinal: 0,
        family: "sequence",
        profile,
        profileFingerprint,
      });
      output.push({ name: "unlisted-dead", expected: "mermaid.svg_css_reduction_unknown", code: "accepted" });
    } catch (error) {
      output.push({ name: "unlisted-dead", expected: "mermaid.svg_css_reduction_unknown", code: error.code });
    }
    const wrongCSSVersion = JSON.parse(JSON.stringify(profile));
    wrongCSSVersion.normalizationReductions.cssTreeVersion = "3.1.1";
    try {
      globalThis.__margo.normalizeSVG(base, {
        sourceRootID,
        renderInstanceID: "ri-0000000a",
        blockOrdinal: 0,
        family: "sequence",
        profile: wrongCSSVersion,
        profileFingerprint,
      });
      output.push({ name: "css-tree-version", expected: "mermaid.profile_mismatch", code: "accepted" });
    } catch (error) {
      output.push({ name: "css-tree-version", expected: "mermaid.profile_mismatch", code: error.code });
    }
    return output;
  }, { moduleURL: mermaidModuleURL, profile, profileFingerprint, sequence });
  for (const result of results) expect(result.code, result.name).toBe(result.expected);
});

test("@svg-normalize rejects duplicates, unresolved/external references, unknown sites, and root mismatch", async ({ page }) => {
  await loadNormalizer(page);
  for (const vector of vectors.negative) {
    const code = await page.evaluate(({ svg, value }) => {
      try {
        globalThis.__margo.normalizeSVG(svg, value);
        return "accepted";
      } catch (error) {
        return error.code;
      }
    }, { svg: fixture(vector.path), value: context(vector) });
    expect(code, vector.path).toBe(vector.errorCode);
  }
});

test("@svg-normalize emits identical canonical bytes for repeated execution", async ({ page }) => {
  await loadNormalizer(page);
  const vector = vectors.positive[0];
  const values = await page.evaluate(({ svg, value }) => [
    globalThis.__margo.normalizeSVG(svg, value).svg,
    globalThis.__margo.normalizeSVG(svg, value).svg,
  ], { svg: fixture(vector.path), value: context(vector) });
  expect(values[0]).toBe(values[1]);
});
