import { expect, test } from "@playwright/test";
import { createRequire } from "node:module";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, "../../..");
const runtimePath = path.join(root, "assets/runtime/mermaid.js");
const normalizerPath = path.join(root, "assets/runtime/svg-normalize.js");
const validatorPath = path.join(root, "assets/runtime/svg-validate.js");
const mermaidAssetRoot = path.join(root, "assets/mermaid/11.16.1");
const mermaidModuleURL = "https://margo.invalid/mermaid/mermaid.esm.min.mjs";
const profile = JSON.parse(fs.readFileSync(path.join(root, "profiles/margo-mermaid-svg-v1.json"), "utf8"));
const profileFingerprint = "bfe4c79b9ccb911c2511c5d24fe14458d148cd64e4bcd5faab97acc84b6cfd1a";
const flowchartSource = fs.readFileSync(path.join(root, "testdata/mermaid/positive/flowchart-basic.mmd"), "utf8");
const require = createRequire(import.meta.url);
const cssTreeEntry = require.resolve("css-tree");
const cssTreePath = path.resolve(path.dirname(cssTreeEntry), "../dist/csstree.js");
const cssTreePackageVersion = require("css-tree/package.json").version;

async function loadQueue(page) {
  await page.goto(pathToFileURL(path.join(root, "test/browser/fixtures/probe.html")).href);
  await page.addScriptTag({ path: runtimePath });
  return page.evaluate(() => Object.keys(globalThis.MargoMermaidQueue ?? {}).sort());
}

async function installMermaidRoute(page) {
  await page.route("https://margo.invalid/mermaid/**", async (route) => {
    const relative = decodeURIComponent(new URL(route.request().url()).pathname).replace(/^\/mermaid\//, "");
    const target = path.resolve(mermaidAssetRoot, relative);
    if (!target.startsWith(`${mermaidAssetRoot}${path.sep}`)) return route.abort("blockedbyclient");
    return route.fulfill({ body: fs.readFileSync(target), contentType: "text/javascript; charset=utf-8" });
  });
}

function baseConfig() {
  return {
    securityLevel: "strict",
    startOnLoad: false,
    deterministicIds: true,
    deterministicIDSeed: "margo-queue-base",
    htmlLabels: false,
    flowchart: { htmlLabels: false },
    sequence: { htmlLabels: false },
    themeCSS: "",
    look: "classic",
    layout: "dagre",
    fontFamily: "Arial, sans-serif",
  };
}

test("@mermaid-queue serializes initialize and render across instances", async ({ page }) => {
  const keys = await loadQueue(page);
  expect(keys).toEqual(["create"]);

  const result = await page.evaluate(async (config) => {
    const callOrder = [];
    const fakeMermaid = {
      initialize(value) {
        callOrder.push(`init:${value.deterministicIDSeed}`);
      },
      async render(rootID, source) {
        callOrder.push(`render:${rootID}`);
        await new Promise((resolve) => setTimeout(resolve, 5));
        return { diagramType: "flowchart", svg: `<svg xmlns="http://www.w3.org/2000/svg" id="${rootID}"><text>${source}</text></svg>`, bindFunctions() {} };
      },
    };
    const queueOptions = {
      mermaid: fakeMermaid,
      baseConfig: config,
      normalizeSVG(svg, context) {
        return { ...context, svg };
      },
      validateSVG() {
        return { ok: true };
      },
    };
    const queueA = globalThis.MargoMermaidQueue.create(queueOptions);
    const queueB = globalThis.MargoMermaidQueue.create(queueOptions);
    const targets = [document.createElement("div"), document.createElement("div")];
    const reports = await Promise.all([
      queueA.run({ instanceID: "ri-0000000a", blockOrdinal: 0, source: "a", family: "flowchart", target: targets[0] }),
      queueB.run({ instanceID: "ri-0000000b", blockOrdinal: 1, source: "b", family: "flowchart", target: targets[1] }),
    ]);
    return { callOrder, reports, html: targets.map((target) => target.innerHTML) };
  }, baseConfig());

  expect(result.callOrder).toEqual([
    "init:msrc-cf9444274153c4758611c7d4a9b8c205d8210a3a02391fcb680cbabed893940a",
    "render:msrc-cf9444274153c4758611c7d4a9b8c205d8210a3a02391fcb680cbabed893940a",
    "init:msrc-7eeb65df6ec2df61462c95003a25e47b9a2e4a9e21a8a6f627c7deaa3e091fba",
    "render:msrc-7eeb65df6ec2df61462c95003a25e47b9a2e4a9e21a8a6f627c7deaa3e091fba",
  ]);
  expect(result.reports.map((entry) => entry.status)).toEqual(["succeeded", "succeeded"]);
  for (const entry of result.reports) {
    expect(entry.report).toEqual({
      id: entry.id,
      kind: "mermaid",
      inputSHA256: entry.inputSHA256,
      outputSHA256: entry.outputSHA256,
      outputBytes: entry.outputBytes,
      status: "succeeded",
      errorCode: "",
    });
    expect(entry.outputSHA256).toMatch(/^[0-9a-f]{64}$/);
    expect(entry.outputBytes).toBe(new TextEncoder().encode(entry.svg).byteLength);
  }
  expect(result.html.every((html) => html.startsWith("<svg"))).toBe(true);
});

test("@mermaid-queue isolates task failures and never binds functions", async ({ page }) => {
  await loadQueue(page);
  const result = await page.evaluate(async (config) => {
    let bindCalls = 0;
    const fakeMermaid = {
      initialize() {},
      async render(rootID, source) {
        if (source === "bad") throw Object.assign(new Error("synthetic render failure"), { code: "mermaid.render_failed" });
        return {
          diagramType: "flowchart",
          svg: `<svg xmlns="http://www.w3.org/2000/svg" id="${rootID}"><text>${source}</text></svg>`,
          bindFunctions() { bindCalls += 1; },
        };
      },
    };
    const targets = [document.createElement("div"), document.createElement("div")];
    const queue = globalThis.MargoMermaidQueue.create({
      mermaid: fakeMermaid,
      baseConfig: config,
      normalizeSVG(svg, context) { return { ...context, svg }; },
      validateSVG() { return { ok: true }; },
    });
    const failed = await queue.run({ instanceID: "ri-0000000a", blockOrdinal: 0, source: "bad", family: "flowchart", target: targets[0] });
    const succeeded = await queue.run({ instanceID: "ri-0000000b", blockOrdinal: 1, source: "good", family: "flowchart", target: targets[1] });
    return { failed, succeeded, bindCalls, failedHTML: targets[0].innerHTML, succeededHTML: targets[1].innerHTML };
  }, baseConfig());

  expect(result.failed).toMatchObject({ status: "failed", errorCode: "mermaid.render_failed", outputBytes: 0, outputSHA256: "", inserted: false });
  expect(result.failed.report).toEqual({
    id: result.failed.id,
    kind: "mermaid",
    inputSHA256: result.failed.inputSHA256,
    outputSHA256: "",
    outputBytes: 0,
    status: "failed",
    errorCode: "mermaid.render_failed",
  });
  expect(result.succeeded).toMatchObject({ status: "succeeded", errorCode: "", outputBytes: expect.any(Number), inserted: true });
  expect(result.bindCalls).toBe(0);
  expect(result.failedHTML).toBe("");
  expect(result.succeededHTML).toMatch(/^<svg/);
});

test("@mermaid-queue freezes config and derives deterministic source roots", async ({ page }) => {
  await loadQueue(page);
  const result = await page.evaluate(async (config) => {
    const observed = [];
    const fakeMermaid = {
      initialize(value) {
        observed.push({ value, frozen: Object.isFrozen(value), nestedFrozen: Object.isFrozen(value.flowchart) });
        value.flowchart.htmlLabels = true;
      },
      async render(rootID) {
        observed.push({ rootID });
        return { svg: `<svg xmlns="http://www.w3.org/2000/svg" id="${rootID}"/>` };
      },
    };
    const queue = globalThis.MargoMermaidQueue.create({
      mermaid: fakeMermaid,
      baseConfig: config,
      normalizeSVG(svg, context) { return { ...context, svg }; },
      validateSVG() { return { ok: true }; },
    });
    const first = await queue.run({ instanceID: "ri-0000000a", blockOrdinal: 7, source: "a", family: "flowchart" });
    const second = await queue.run({ instanceID: "ri-0000000a", blockOrdinal: 7, source: "a", family: "flowchart" });
    return { observed, first, second, config };
  }, baseConfig());

  expect(result.observed[0]).toMatchObject({ frozen: true, nestedFrozen: true });
  expect(result.observed[1]).toEqual({ rootID: "msrc-df2e9e0532d803350883da82c96147f96db9345d772da34c177ae041d3ad94bd" });
  expect(result.observed[2]).toMatchObject({ frozen: true, nestedFrozen: true });
  expect(result.observed[3]).toEqual({ rootID: "msrc-df2e9e0532d803350883da82c96147f96db9345d772da34c177ae041d3ad94bd" });
  expect(result.first.sourceRootID).toBe("msrc-df2e9e0532d803350883da82c96147f96db9345d772da34c177ae041d3ad94bd");
  expect(result.second.sourceRootID).toBe(result.first.sourceRootID);
  expect(result.config.deterministicIDSeed).toBe("margo-queue-base");
});

test("@mermaid-queue executes one real embedded Mermaid task through normalize, validate, and insertion", async ({ page }) => {
  const nonLocalRequests = [];
  page.on("request", (request) => {
    if (!request.url().startsWith("file:") && !request.url().startsWith("https://margo.invalid/mermaid/")) nonLocalRequests.push(request.url());
  });
  await installMermaidRoute(page);
  await page.goto(pathToFileURL(path.join(root, "test/browser/fixtures/probe.html")).href);
  await page.addScriptTag({ path: cssTreePath });
  await page.evaluate((version) => { globalThis.__margoCSSTreePackageVersion = version; }, cssTreePackageVersion);
  await page.addScriptTag({ path: normalizerPath });
  await page.addScriptTag({ path: validatorPath });
  await page.addScriptTag({ path: runtimePath });

  const result = await page.evaluate(async ({ moduleURL, source, config, profileValue, profileFingerprintValue }) => {
    const mermaid = (await import(moduleURL)).default;
    const target = document.createElement("div");
    const queue = globalThis.MargoMermaidQueue.create({
      mermaid,
      baseConfig: config,
      normalizeSVG: globalThis.__margo.normalizeSVG,
      validateSVG: globalThis.__margo.validateSVG,
      profile: profileValue,
      profileFingerprint: profileFingerprintValue,
    });
    const output = await queue.run({ instanceID: "ri-0000000a", blockOrdinal: 0, source, family: "flowchart", target });
    return { output, html: target.innerHTML };
  }, {
    moduleURL: mermaidModuleURL,
    source: flowchartSource,
    config: baseConfig(),
    profileValue: profile,
    profileFingerprintValue: profileFingerprint,
  });

  expect(result.output.status).toBe("succeeded");
  expect(result.output.errorCode).toBe("");
  expect(result.output.sourceRootID).toMatch(/^msrc-[0-9a-f]{64}$/);
  expect(result.output.outputSHA256).toMatch(/^[0-9a-f]{64}$/);
  expect(result.output.outputBytes).toBeGreaterThan(0);
  expect(result.output.svg).toContain(`<svg`);
  expect(result.html).toContain(`<svg`);
  expect(nonLocalRequests).toEqual([]);
});
