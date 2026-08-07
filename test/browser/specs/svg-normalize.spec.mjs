import { expect, test } from "@playwright/test";
import { createRequire } from "node:module";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, "../../..");
const normalizerPath = path.join(root, "assets/runtime/svg-normalize.js");
const fixturesRoot = path.join(root, "testdata/mermaid/normalization");
const profile = JSON.parse(fs.readFileSync(path.join(root, "profiles/margo-mermaid-svg-v1.json"), "utf8"));
const vectors = JSON.parse(fs.readFileSync(path.join(fixturesRoot, "vectors.json"), "utf8"));
const require = createRequire(import.meta.url);
const cssTreeEntry = require.resolve("css-tree");
const cssTreePath = path.resolve(path.dirname(cssTreeEntry), "../dist/csstree.js");

function fixture(name) {
  return fs.readFileSync(path.join(fixturesRoot, name), "utf8");
}

function context(vector) {
  return {
    sourceRootID: vector.sourceRootID,
    renderInstanceID: vector.renderInstanceID ?? "ri-0000000a",
    blockOrdinal: vector.blockOrdinal ?? 0,
    referenceRegistry: profile.idReferenceSites,
  };
}

async function loadNormalizer(page) {
  await page.goto("about:blank");
  await page.addScriptTag({ path: cssTreePath });
  await page.addScriptTag({ path: normalizerPath });
  return page.evaluate(() => Object.keys(globalThis.__margo).sort());
}

test("@svg-normalize rewrites root, descendants, attributes, ARIA, selectors, and CSS URLs", async ({ page }) => {
  const keys = await loadNormalizer(page);
  expect(keys).toContain("normalizeSVG");
  const vector = vectors.positive[0];
  const output = await page.evaluate(({ svg, value }) => globalThis.__margo.normalizeSVG(svg, value), {
    svg: fixture(vector.path),
    value: context(vector),
  });

  expect(output.rootID).toBe(vector.normalizedRootID);
  expect(output.originalRootID).toBe(vector.sourceRootID);
  expect(output.descendantMap).toHaveLength(vector.descendantCount);
  expect(output.canonicalReparse).toBe(true);
  expect(output.svg).not.toContain(vector.sourceRootID);
  expect(output.svg).not.toContain(".unused");
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
