#!/usr/bin/env node

import { createHash } from "node:crypto";
import { constants } from "node:fs";
import { access, mkdtemp, mkdir, readFile, rename, rm, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import { chromium } from "playwright-core";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = path.resolve(SCRIPT_DIR, "../..");

function usage() {
  return [
    "usage: print-pdf.mjs --html ABSOLUTE_HTML --output ABSOLUTE_PDF [options]",
    "",
    "options:",
    "  --root ABSOLUTE_DIR       repository root (default: inferred)",
    "  --mode light|dark         expected document mode (default: light)",
    "  --evidence ABSOLUTE_JSON  write deterministic print evidence",
  ].join("\n");
}

function parseArgs(argv) {
  const values = { root: DEFAULT_ROOT, mode: "light", evidence: "" };
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === "--help") {
      console.log(usage());
      process.exit(0);
    }
    if (!["--root", "--html", "--output", "--mode", "--evidence"].includes(argument)) {
      throw new Error(`margo.pdf_print_argument_unknown:${argument}`);
    }
    const value = argv[index + 1];
    if (!value || value.startsWith("--")) throw new Error(`margo.pdf_print_argument_missing:${argument}`);
    values[argument.slice(2)] = value;
    index += 1;
  }
  if (!values.html || !values.output) throw new Error("margo.pdf_print_html_output_required");
  if (!["light", "dark"].includes(values.mode)) throw new Error(`margo.pdf_print_mode_invalid:${values.mode}`);
  for (const key of ["root", "html", "output", "evidence"]) {
    if (values[key] && !path.isAbsolute(values[key])) throw new Error(`margo.pdf_print_absolute_required:${key}`);
  }
  return values;
}

async function requireFile(filePath, diagnostic) {
  try {
    await access(filePath, constants.F_OK | constants.R_OK);
  } catch {
    throw new Error(`${diagnostic}:${filePath}`);
  }
}

async function writeAtomic(filePath, bytes) {
  await mkdir(path.dirname(filePath), { recursive: true });
  const temporaryDir = await mkdtemp(path.join(path.dirname(filePath), ".margo-pdf-"));
  const temporaryPath = path.join(temporaryDir, path.basename(filePath));
  try {
    await writeFile(temporaryPath, bytes, { mode: 0o600 });
    await rename(temporaryPath, filePath);
  } finally {
    await rm(temporaryDir, { recursive: true, force: true });
  }
}

function sha256(bytes) {
  return createHash("sha256").update(bytes).digest("hex");
}

async function collectPrintContract(page) {
  return page.evaluate(() => {
    const tables = [...document.querySelectorAll('.margo-document [data-table-client-sort="true"] table')].map((table, index) => {
      const body = table.tBodies[0];
      const rows = body ? [...body.rows].map((row) => [...row.cells].map((cell) => cell.textContent.trim()).join(" | ")) : [];
      return {
        index,
        rowCount: rows.length,
        rows,
        headerDisplay: table.tHead ? getComputedStyle(table.tHead).display : "missing",
        bodyDisplay: body ? getComputedStyle(body).display : "missing",
        rowBreakInside: body?.rows[0] ? getComputedStyle(body.rows[0]).breakInside : "missing",
      };
    });
    const rejectionRows = [
      "invalid-data-points | mermaid.svg_attribute_forbidden",
      "invalid-length-unit | mermaid.svg_css_value_forbidden",
      "unrooted-id | mermaid.svg_id_forbidden",
    ];
    const rejectionTable = tables.find((table) => rejectionRows.every((expected) => table.rows.includes(expected)));
    if (!rejectionTable) throw new Error("margo.pdf_print_table_sentinel_missing");
    if (rejectionTable.headerDisplay !== "table-header-group" || rejectionTable.bodyDisplay !== "table-row-group") {
      throw new Error("margo.pdf_print_table_group_display_invalid");
    }
    if (!rejectionTable.rowBreakInside.toLowerCase().includes("avoid")) {
      throw new Error("margo.pdf_print_table_row_break_invalid");
    }
    return {
      tableCount: tables.length,
      tableRowCounts: tables.map(({ rowCount }) => rowCount),
      rejectionTableIndex: rejectionTable.index,
      rejectionTableRowCount: rejectionTable.rowCount,
      rejectionRows,
      tocColumns: document.querySelector(".goshtoso-document__toc")?.dataset.margoTocColumns ?? null,
      breakMarkers: document.querySelectorAll('[data-margo-print-break-before="page"]').length,
    };
  });
}

function headerTemplate({ logo, foreground }) {
  return `<div style="box-sizing:border-box;width:100%;margin:0 18mm;padding:0 0 2mm;border-bottom:1px solid ${foreground};display:flex;align-items:center;justify-content:space-between;font:8px Arial,sans-serif;color:${foreground}"><span style="display:flex;align-items:center;gap:5px"><img src="${logo}" style="width:9px;height:9px"><strong>Margo</strong><span>· full feature benchmark</span></span><span>Markdown for Goshtoso</span></div>`;
}

function footerTemplate({ foreground }) {
  return `<div style="box-sizing:border-box;width:100%;margin:0 18mm;padding:2mm 0 0;border-top:1px solid ${foreground};display:flex;align-items:center;justify-content:space-between;font:7px Arial,sans-serif;color:${foreground}"><span>margo-full-feature-set.md · modern</span><span>OPTIMISTIC</span><span>Page <span class="pageNumber"></span> / <span class="totalPages"></span></span></div>`;
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  const htmlPath = path.resolve(options.html);
  const outputPath = path.resolve(options.output);
  const root = path.resolve(options.root);
  const executablePath = process.env.MARGO_CHROMIUM_EXECUTABLE;
  if (!executablePath || !path.isAbsolute(executablePath)) throw new Error("margo.pdf_print_chromium_path_required");
  await requireFile(htmlPath, "margo.pdf_print_html_missing");
  await requireFile(executablePath, "margo.pdf_print_chromium_missing");
  const logoBytes = await readFile(path.join(root, "assets/logo.svg"));
  const logo = `data:image/svg+xml;base64,${logoBytes.toString("base64")}`;
  const foreground = options.mode === "dark" ? "#d1d5db" : "#374151";
  const blockedRequests = [];
  const consoleErrors = [];
  const browser = await chromium.launch({ executablePath, headless: true });
  try {
    const context = await browser.newContext({ viewport: { width: 794, height: 1123 } });
    await context.route("**/*", async (route) => {
      const url = route.request().url();
      if (["file:", "data:", "about:"].includes(new URL(url).protocol)) {
        await route.continue();
        return;
      }
      blockedRequests.push(url);
      await route.abort("blockedbyclient");
    });
    const page = await context.newPage();
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(error.message));
    await page.goto(pathToFileURL(htmlPath).href, { waitUntil: "load" });
    await page.evaluate(async () => {
      await document.fonts.ready;
      await Promise.all([...document.images].map((image) => image.decode().catch(() => undefined)));
      await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)));
    });
    await page.emulateMedia({ media: "print" });
    await page.evaluate((mode) => {
      const dark = document.documentElement.classList.contains("dark");
      if (dark !== (mode === "dark")) throw new Error(`margo.pdf_print_mode_mismatch:${mode}`);
      if (typeof window.margoPreparePrintTOC !== "function") throw new Error("margo.print_pagination_missing");
      window.margoPreparePrintTOC();
    }, options.mode);
    const printContract = await collectPrintContract(page);
    const pdfBytes = await page.pdf({
      format: "A4",
      printBackground: true,
      preferCSSPageSize: true,
      displayHeaderFooter: true,
      headerTemplate: headerTemplate({ logo, foreground }),
      footerTemplate: footerTemplate({ foreground }),
    });
    if (blockedRequests.length > 0) throw new Error(`margo.pdf_print_network_blocked:${blockedRequests.join(",")}`);
    if (consoleErrors.length > 0) throw new Error(`margo.pdf_print_console_error:${consoleErrors.join(" | ")}`);
    await writeAtomic(outputPath, pdfBytes);
    const evidence = {
      schemaVersion: "margo/pdf-print/v2",
      htmlPath,
      pdfPath: outputPath,
      mode: options.mode,
      viewport: { width: 794, height: 1123 },
      pageFormat: "A4",
      preparedPrint: true,
      network: { blockedRequests },
      consoleErrors,
      printContract,
      bytes: pdfBytes.byteLength,
      sha256: sha256(pdfBytes),
    };
    if (options.evidence) await writeAtomic(path.resolve(options.evidence), `${JSON.stringify(evidence)}\n`);
    console.log(JSON.stringify(evidence));
  } finally {
    await browser.close();
  }
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : String(error));
  process.exitCode = 1;
});
