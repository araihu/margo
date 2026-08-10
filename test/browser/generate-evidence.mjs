#!/usr/bin/env node

import { createHash } from "node:crypto";
import { constants } from "node:fs";
import { access, mkdtemp, mkdir, readFile, rename, rm, writeFile } from "node:fs/promises";
import { createRequire } from "node:module";
import path from "node:path";
import process from "node:process";
import { fileURLToPath, pathToFileURL } from "node:url";
import { chromium } from "playwright";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = path.resolve(SCRIPT_DIR, "../..");
const MERMAID_MODULE_URL = "https://margo.invalid/mermaid/mermaid.esm.min.mjs";
const PROFILE_FINGERPRINT = "bfe4c79b9ccb911c2511c5d24fe14458d148cd64e4bcd5faab97acc84b6cfd1a";
const LOCAL_PROTOCOLS = new Set(["file:", "data:", "blob:", "about:"]);

function usage(message = "") {
  const prefix = message ? `${message}\n\n` : "";
  return `${prefix}Usage: node generate-evidence.mjs --html ABSOLUTE_PATH --evidence ABSOLUTE_PATH [options]\n\n` +
    "Options:\n" +
    "  --root ABSOLUTE_DIR       repository root (default: inferred)\n" +
    "  --mode light|dark         expected document mode (default: light)\n" +
    "  --evidence ABSOLUTE_PATH  write browser evidence atomically\n";
}

export function parseArgs(argv) {
  const options = { root: DEFAULT_ROOT, mode: "light", evidence: "" };
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === "--help" || argument === "-h") return { ...options, help: true };
    if (!["--root", "--html", "--evidence", "--mode"].includes(argument)) {
      throw new Error(`margo.browser_evidence_argument_unknown:${argument}`);
    }
    const value = argv[index + 1];
    if (!value || value.startsWith("--")) throw new Error(`margo.browser_evidence_argument_missing:${argument}`);
    options[argument.slice(2)] = value;
    index += 1;
  }
  if (!options.html) throw new Error("margo.browser_evidence_html_required");
  if (!options.evidence) throw new Error("margo.browser_evidence_output_required");
  if (!["light", "dark"].includes(options.mode)) throw new Error(`margo.browser_evidence_mode_invalid:${options.mode}`);
  for (const key of ["root", "html", "evidence"]) {
    if (!path.isAbsolute(options[key])) throw new Error(`margo.browser_evidence_absolute_required:${key}`);
  }
  return options;
}

function digest(bytes) {
  return createHash("sha256").update(bytes).digest("hex");
}

async function requireFile(filePath, diagnostic) {
  try {
    await access(filePath, constants.F_OK | constants.R_OK);
  } catch {
    throw new Error(`${diagnostic}:${filePath}`);
  }
}

async function writeAtomic(filePath, content) {
  await mkdir(path.dirname(filePath), { recursive: true });
  const temporaryDir = await mkdtemp(path.join(path.dirname(filePath), ".margo-browser-evidence-"));
  const temporaryPath = path.join(temporaryDir, path.basename(filePath));
  try {
    await writeFile(temporaryPath, content, { mode: 0o600 });
    await rename(temporaryPath, filePath);
  } finally {
    await rm(temporaryDir, { recursive: true, force: true });
  }
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

function packagePaths() {
  const require = createRequire(import.meta.url);
  const cssTreeEntry = require.resolve("css-tree");
  return {
    cssTree: path.resolve(path.dirname(cssTreeEntry), "../dist/csstree.js"),
    cssTreeVersion: require("css-tree/package.json").version,
  };
}

function isLocalResource(url) {
  try {
    return LOCAL_PROTOCOLS.has(new URL(url).protocol);
  } catch {
    return false;
  }
}

async function installMermaidRoute(page, mermaidAssetRoot, blockedRequests) {
  await page.route("**/*", async (route) => {
    const url = route.request().url();
    if (url.startsWith("https://margo.invalid/mermaid/")) {
      const relative = decodeURIComponent(new URL(url).pathname).replace(/^\/mermaid\//, "");
      const target = path.resolve(mermaidAssetRoot, relative);
      if (!target.startsWith(`${mermaidAssetRoot}${path.sep}`)) {
        blockedRequests.push(url);
        await route.abort("blockedbyclient");
        return;
      }
      try {
        await requireFile(target, "margo.browser_evidence_mermaid_asset_missing");
      } catch {
        blockedRequests.push(url);
        await route.abort("blockedbyclient");
        return;
      }
      await route.fulfill({ body: await readFile(target), contentType: "text/javascript; charset=utf-8" });
      return;
    }
    if (isLocalResource(url)) {
      await route.continue();
      return;
    }
    blockedRequests.push(url);
    await route.abort("blockedbyclient");
  });
}

export async function run(options, environment = process.env) {
  const root = path.resolve(options.root);
  const htmlPath = path.resolve(options.html);
  const evidencePath = path.resolve(options.evidence);
  const executablePath = environment.MARGO_CHROMIUM_EXECUTABLE ?? "";
  if (!path.isAbsolute(executablePath)) throw new Error("margo.browser_evidence_chromium_absolute_required");
  await requireFile(htmlPath, "margo.browser_evidence_html_missing");
  await requireFile(executablePath, "margo.browser_evidence_chromium_missing");
  const paths = packagePaths();
  const runtimePaths = {
    mermaid: path.join(root, "assets/runtime/mermaid.js"),
    normalizer: path.join(root, "assets/runtime/svg-normalize.js"),
    validator: path.join(root, "assets/runtime/svg-validate.js"),
    mermaidAssetRoot: path.join(root, "assets/mermaid/11.16.1"),
    profile: path.join(root, "profiles/margo-mermaid-svg-v1.json"),
  };
  await Promise.all([
    requireFile(paths.cssTree, "margo.browser_evidence_css_tree_missing"),
    requireFile(runtimePaths.mermaid, "margo.browser_evidence_runtime_missing"),
    requireFile(runtimePaths.normalizer, "margo.browser_evidence_normalizer_missing"),
    requireFile(runtimePaths.validator, "margo.browser_evidence_validator_missing"),
    requireFile(runtimePaths.profile, "margo.browser_evidence_profile_missing"),
  ]);
  const [htmlBytes, profile] = await Promise.all([
    readFile(htmlPath),
    readFile(runtimePaths.profile).then((bytes) => JSON.parse(bytes)),
  ]);
  const requests = [];
  const blockedRequests = [];
  const consoleErrors = [];
  const pageErrors = [];
  const browser = await chromium.launch({ executablePath, headless: true });
  try {
    const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
    const page = await context.newPage();
    page.on("request", (request) => requests.push(request.url()));
    page.on("console", (message) => { if (message.type() === "error") consoleErrors.push(message.text()); });
    page.on("pageerror", (error) => pageErrors.push(error.message));
    await installMermaidRoute(page, runtimePaths.mermaidAssetRoot, blockedRequests);
    await page.goto(pathToFileURL(htmlPath).href, { waitUntil: "load" });
    await page.addScriptTag({ path: paths.cssTree });
    await page.evaluate((version) => { globalThis.__margoCSSTreePackageVersion = version; }, paths.cssTreeVersion);
    await page.addScriptTag({ path: runtimePaths.normalizer });
    await page.addScriptTag({ path: runtimePaths.validator });
    await page.addScriptTag({ path: runtimePaths.mermaid });
    await page.evaluate(async () => {
      await document.fonts.ready;
      await Promise.all([...document.images].map((image) => image.decode().catch(() => undefined)));
      await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)));
    });
    const rendered = await page.evaluate(async ({ moduleURL, queueConfig, profileValue, profileFingerprintValue }) => {
      const mermaid = (await import(moduleURL)).default;
      const queue = globalThis.MargoMermaidQueue.create({
        mermaid,
        baseConfig: queueConfig,
        normalizeSVG: globalThis.__margo.normalizeSVG,
        validateSVG: globalThis.__margo.validateSVG,
        profile: profileValue,
        profileFingerprint: profileFingerprintValue,
      });
      const figures = [...document.querySelectorAll('[data-margo-runtime-task="mermaid"]')];
      const rows = [];
      for (const [index, figure] of figures.entries()) {
        const source = figure.querySelector(".margo-mermaid__source pre")?.textContent ?? "";
        const family = /^\s*sequenceDiagram\b/m.test(source) ? "sequence" : "flowchart";
        const output = await queue.run({
          instanceID: "ri-00000000",
          blockOrdinal: Number(figure.dataset.margoRuntimeTaskOrdinal ?? index),
          source,
          family,
          target: figure.querySelector(".margo-mermaid__canvas"),
        });
        rows.push({ family, output });
      }
      return rows;
    }, {
      moduleURL: MERMAID_MODULE_URL,
      queueConfig: baseConfig(),
      profileValue: profile,
      profileFingerprintValue: PROFILE_FINGERPRINT,
    });
    const screenContract = await page.evaluate(() => ({
      colorMode: document.documentElement.dataset.colorMode,
      darkClass: document.documentElement.classList.contains("dark"),
      figures: document.querySelectorAll('[data-margo-runtime-task="mermaid"]').length,
      renderedFigures: document.querySelectorAll('[data-margo-runtime-task="mermaid"] svg').length,
      openSources: document.querySelectorAll('[data-margo-runtime-task="mermaid"] details[open]').length,
    }));
    if (screenContract.colorMode !== options.mode || screenContract.darkClass !== (options.mode === "dark") || screenContract.figures !== 3 || screenContract.renderedFigures !== 3 || screenContract.openSources !== 0) {
      throw new Error(`margo.browser_evidence_screen_contract:${JSON.stringify(screenContract)}`);
    }
    if (rendered.some((entry) => entry.output.status !== "succeeded")) {
      throw new Error(`margo.browser_evidence_mermaid_failed:${JSON.stringify(rendered.map((entry) => entry.output))}`);
    }
    await page.emulateMedia({ media: "print" });
    const printState = await page.evaluate(() => {
      window.margoPreparePrintTOC();
      const style = (selector) => {
        const computed = getComputedStyle(document.querySelector(selector));
        return {
          display: computed.display,
          background: computed.backgroundColor,
          color: computed.color,
          borderStart: computed.borderBlockStartColor,
          borderEnd: computed.borderBlockEndColor,
        };
      };
      const documentBackground = getComputedStyle(document.querySelector(".goshtoso-document")).backgroundColor;
      return {
        bodyBackground: getComputedStyle(document.body).backgroundColor,
        documentBackground,
        header: style(".goshtoso-document__header"),
        footer: style(".goshtoso-document__footer"),
        tocBackground: getComputedStyle(document.querySelector(".goshtoso-document__toc")).backgroundColor,
        tocColumns: getComputedStyle(document.querySelector(".goshtoso-document__toc ol")).columnCount,
        openSources: document.querySelectorAll('[data-margo-runtime-task="mermaid"] details[open]').length,
        breakMarkers: document.querySelectorAll('[data-margo-print-break-before="page"]').length,
      };
    });
    if (printState.openSources !== 3 || printState.tocColumns !== "1" || printState.header.display !== "flex" || printState.footer.display !== "flex") {
      throw new Error(`margo.browser_evidence_print_contract:${JSON.stringify(printState)}`);
    }
    if (printState.header.background !== printState.documentBackground || printState.footer.background !== printState.documentBackground || printState.tocBackground !== printState.documentBackground) {
      throw new Error(`margo.browser_evidence_print_surface_mismatch:${JSON.stringify(printState)}`);
    }
    if (blockedRequests.length > 0 || consoleErrors.length > 0 || pageErrors.length > 0) {
      throw new Error(`margo.browser_evidence_runtime_failure:${JSON.stringify({ blockedRequests, consoleErrors, pageErrors })}`);
    }
    const evidence = {
      schema: "margo/optimistic-browser-evidence/v4",
      variant: options.mode,
      browser: { executablePath, version: await browser.version(), node: process.version },
      source: { path: htmlPath, bytes: htmlBytes.byteLength, sha256: digest(htmlBytes) },
      mermaid: rendered.map((entry) => ({
        family: entry.family,
        inputSHA256: entry.output.inputSHA256,
        outputSHA256: entry.output.outputSHA256,
        outputBytes: entry.output.outputBytes,
        status: entry.output.status,
        errorCode: entry.output.errorCode,
      })),
      screenContract,
      printState,
      requests,
      blockedRequests,
      consoleErrors,
      pageErrors,
    };
    await writeAtomic(evidencePath, `${JSON.stringify(evidence, null, 2)}\n`);
    return evidence;
  } finally {
    await browser.close();
  }
}

export async function main(argv = process.argv.slice(2), environment = process.env) {
  let options;
  try {
    options = parseArgs(argv);
  } catch (error) {
    process.stderr.write(`${usage(error.message)}\n`);
    return 2;
  }
  if (options.help) {
    process.stdout.write(usage());
    return 0;
  }
  try {
    const evidence = await run(options, environment);
    process.stdout.write(`${JSON.stringify(evidence)}\n`);
    return 0;
  } catch (error) {
    process.stderr.write(`margo.browser_evidence_error:${error instanceof Error ? error.message : String(error)}\n`);
    return 1;
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  process.exitCode = await main();
}
