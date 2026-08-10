import { expect, test } from "@playwright/test";
import { createRequire } from "node:module";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, "../../..");
const normalizerPath = path.join(root, "assets/runtime/svg-normalize.js");
const validatorPath = path.join(root, "assets/runtime/svg-validate.js");
const mermaidModuleURL = "https://margo.invalid/mermaid/mermaid.esm.min.mjs";
const mermaidAssetRoot = path.join(root, "assets/mermaid/11.16.1");
const positiveRoot = path.join(root, "testdata/mermaid/positive");
const negativeRoot = path.join(root, "testdata/mermaid/negative");
const profile = JSON.parse(fs.readFileSync(path.join(root, "profiles/margo-mermaid-svg-v1.json"), "utf8"));
const negativeVectors = JSON.parse(fs.readFileSync(path.join(negativeRoot, "vectors.json"), "utf8"));
const profileFingerprint = "bfe4c79b9ccb911c2511c5d24fe14458d148cd64e4bcd5faab97acc84b6cfd1a";
const require = createRequire(import.meta.url);
const cssTreeEntry = require.resolve("css-tree");
const cssTreePath = path.resolve(path.dirname(cssTreeEntry), "../dist/csstree.js");
const cssTreePackageVersion = require("css-tree/package.json").version;

function pinnedFixtures() {
  return fs.readdirSync(positiveRoot)
    .filter((name) => name.endsWith(".mmd"))
    .sort()
    .map((name) => ({ family: name.split("-")[0], name, source: fs.readFileSync(path.join(positiveRoot, name), "utf8") }));
}

async function installMermaidRoute(page) {
  await page.route("https://margo.invalid/mermaid/**", async (route) => {
    const relative = decodeURIComponent(new URL(route.request().url()).pathname).replace(/^\/mermaid\//, "");
    const target = path.resolve(mermaidAssetRoot, relative);
    if (!target.startsWith(`${mermaidAssetRoot}${path.sep}`)) return route.abort("blockedbyclient");
    return route.fulfill({ body: fs.readFileSync(target), contentType: "text/javascript; charset=utf-8" });
  });
}

async function loadRuntime(page) {
  const scriptErrors = [];
  page.on("pageerror", (error) => scriptErrors.push(error.message));
  await page.goto(pathToFileURL(path.join(root, "test/browser/fixtures/probe.html")).href);
  await page.addScriptTag({ path: cssTreePath });
  await page.evaluate((version) => { globalThis.__margoCSSTreePackageVersion = version; }, cssTreePackageVersion);
  await page.addScriptTag({ path: normalizerPath });
  await page.addScriptTag({ path: validatorPath });
  const ready = await page.evaluate(() => typeof globalThis.__margo?.validateSVG === "function");
  if (!ready) throw new Error(`svg validator failed to initialize: ${scriptErrors.join("; ")}`);
}

test("@svg-validate accepts every pinned normalized fixture and preserves audited flowchart properties", async ({ page }) => {
  const nonLocalRequests = [];
  page.on("request", (request) => {
    if (!request.url().startsWith("file:") && !request.url().startsWith("https://margo.invalid/mermaid/")) nonLocalRequests.push(request.url());
  });
  await installMermaidRoute(page);
  await loadRuntime(page);
  const results = await page.evaluate(async ({ fixtures, moduleURL, profile, profileFingerprint }) => {
    const mermaid = (await import(moduleURL)).default;
    mermaid.initialize({
      securityLevel: "strict",
      startOnLoad: false,
      deterministicIds: true,
      deterministicIDSeed: "margo-m5-profile",
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
      const observedProperties = new Set();
      const observedDeclarations = [];
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
        const document = new DOMParser().parseFromString(normalized.svg, "image/svg+xml");
        const collect = (source, context) => {
          const ast = globalThis.csstree.parse(source, { context });
          globalThis.csstree.walk(ast, (node) => {
            if (node.type === "Declaration") {
              observedProperties.add(node.property.toLowerCase());
              observedDeclarations.push(globalThis.csstree.generate(node));
            }
          });
        };
        for (const element of document.querySelectorAll("[style]")) collect(element.getAttribute("style"), "declarationList");
        for (const style of document.querySelectorAll("style")) collect(style.textContent ?? "", "stylesheet");
        const unlistedProperties = [...observedProperties].filter((property) => !profile.cssProperties[property]).sort();
        const validated = globalThis.__margo.validateSVG(normalized.svg, {
          family: item.family,
          profile,
          profileFingerprint,
        });
        output.push({ name: item.name, unlistedProperties, ...validated });
      } catch (error) {
        output.push({
          name: item.name,
          code: error.code ?? "unknown",
          message: error.message,
          unlistedDeclarations: observedDeclarations.filter((row) => !profile.cssProperties[row.slice(0, row.indexOf(":"))]),
          unlistedProperties: [...observedProperties].filter((property) => !profile.cssProperties[property]).sort(),
        });
      }
    }
    return output;
  }, { fixtures: pinnedFixtures(), moduleURL: mermaidModuleURL, profile, profileFingerprint });

  expect(results).toHaveLength(8);
  expect(results.filter((result) => result.code).map(({ name, code, message, unlistedDeclarations, unlistedProperties }) => ({ name, code, message, unlistedDeclarations, unlistedProperties }))).toEqual([]);
  for (const result of results) {
    expect(result.profileFingerprint, result.name).toBe(profileFingerprint);
    expect(result.canonicalReparse, result.name).toBe(true);
    expect(result.svgBytes, result.name).toBeGreaterThan(0);
  }
  const flowchartCSS = results.filter((result) => result.name.startsWith("flowchart-")).map((result) => result.cssText).join("\n");
  expect(flowchartCSS).toContain("background-color");
  expect(flowchartCSS).toContain("text-align");
  expect(flowchartCSS).toContain("cursor");
  expect(nonLocalRequests).toEqual([]);
});

test("@svg-validate rejects the closed negative corpus before insertion", async ({ page }) => {
  await loadRuntime(page);
  const vectors = negativeVectors.map((vector) => ({ ...vector, svg: fs.readFileSync(path.join(negativeRoot, vector.path), "utf8") }));
  const results = await page.evaluate(({ vectors, profile, profileFingerprint }) => {
    let insertions = 0;
    return vectors.map((vector) => {
      try {
        globalThis.__margo.validateSVG(vector.svg, { family: vector.family, profile, profileFingerprint });
        insertions += 1;
        return { name: vector.name, code: "accepted", insertions };
      } catch (error) {
        return { name: vector.name, code: error.code, insertions };
      }
    });
  }, { vectors, profile, profileFingerprint });

  expect(results).toHaveLength(negativeVectors.length);
  for (const result of results) {
    const vector = negativeVectors.find((entry) => entry.name === result.name);
    expect(result.code, result.name).toBe(vector.code);
    expect(result.insertions, result.name).toBe(0);
  }
});

test("@svg-validate rejects profile and family mismatches plus every real resource overflow", async ({ page }) => {
  await loadRuntime(page);
  const rootID = "margo-ri-0000000a-mermaid-00000000";
  const valid = `<svg xmlns="http://www.w3.org/2000/svg" id="${rootID}"><g id="${rootID}--id-00000000"><rect width="1" height="1"/></g></svg>`;
  const result = await page.evaluate(({ valid, rootID, profile, profileFingerprint }) => {
    const attempt = (svg, context) => {
      try {
        globalThis.__margo.validateSVG(svg, context);
        return "accepted";
      } catch (error) {
        return error.code;
      }
    };
    const smallBytes = structuredClone(profile);
    smallBytes.limits.maxSvgBytes = 16;
    const smallElements = structuredClone(profile);
    smallElements.limits.maxElements = 1;
    const smallAttributes = structuredClone(profile);
    smallAttributes.limits.maxAttributes = 1;
    const oversizedBytes = `${valid}${" ".repeat(profile.limits.maxSvgBytes)}`;
    const oversizedElements = `<svg xmlns="http://www.w3.org/2000/svg" id="${rootID}">${"<g/>".repeat(profile.limits.maxElements)}</svg>`;
    const attributeChunk = '<g aria-label="x" aria-roledescription="x" class="x" height="1" name="x" preserveAspectRatio="x" role="x" tabindex="0" viewBox="0 0 1 1" width="1" x="0" y="0" data-et="x" data-id="x" data-look="x" data-type="x" transform="translate(0)"/>';
    const oversizedAttributes = `<svg xmlns="http://www.w3.org/2000/svg" id="${rootID}">${attributeChunk.repeat(Math.ceil(profile.limits.maxAttributes / 17) + 1)}</svg>`;
    const oversizedRules = `<svg xmlns="http://www.w3.org/2000/svg" id="${rootID}"><style>${`#${rootID}{opacity:1}`.repeat(profile.limits.maxCssRules + 1)}</style></svg>`;
    const oversizedSelector = `<svg xmlns="http://www.w3.org/2000/svg" id="${rootID}"><style>#${rootID}${" .a".repeat(Math.ceil(profile.limits.maxSelectorBytes / 3) + 1)}{opacity:1}</style></svg>`;
    return {
      fingerprint: attempt(valid, { family: "flowchart", profile, profileFingerprint: "0".repeat(64) }),
      computedFingerprint: attempt(valid, { family: "flowchart", profile: { ...profile, mermaidVersion: "11.16.2" }, profileFingerprint }),
      family: attempt(valid, { family: "stateDiagram", profile, profileFingerprint }),
      bytes: attempt(valid, { family: "flowchart", profile: smallBytes, profileFingerprint }),
      elements: attempt(valid, { family: "flowchart", profile: smallElements, profileFingerprint }),
      attributes: attempt(valid, { family: "flowchart", profile: smallAttributes, profileFingerprint }),
      realBytes: attempt(oversizedBytes, { family: "flowchart", profile, profileFingerprint }),
      realElements: attempt(oversizedElements, { family: "flowchart", profile, profileFingerprint }),
      realAttributes: attempt(oversizedAttributes, { family: "flowchart", profile, profileFingerprint }),
      realRules: attempt(oversizedRules, { family: "flowchart", profile, profileFingerprint }),
      realSelector: attempt(oversizedSelector, { family: "flowchart", profile, profileFingerprint }),
    };
  }, { valid, rootID, profile, profileFingerprint });

  expect(result).toEqual({
    fingerprint: "mermaid.profile_mismatch",
    computedFingerprint: "mermaid.profile_mismatch",
    family: "mermaid.svg_family_unsupported",
    bytes: "mermaid.profile_mismatch",
    elements: "mermaid.profile_mismatch",
    attributes: "mermaid.profile_mismatch",
    realBytes: "mermaid.svg_resource_limit",
    realElements: "mermaid.svg_resource_limit",
    realAttributes: "mermaid.svg_resource_limit",
    realRules: "mermaid.svg_resource_limit",
    realSelector: "mermaid.svg_resource_limit",
  });
});
