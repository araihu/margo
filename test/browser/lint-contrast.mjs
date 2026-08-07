import { createHash } from "node:crypto";
import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import { pathToFileURL } from "node:url";
import { chromium } from "playwright";
import { auditTextContrast } from "./harness/contrast.mjs";

const MODES = ["light", "dark"];
const EXIT_USAGE = 2;

function usage(message) {
  const prefix = message ? `${message}\n\n` : "";
  return `${prefix}Usage: node lint-contrast.mjs --html ABSOLUTE_PATH [options]\n\n` +
    "Options:\n" +
    "  --mode light|dark|both   Color modes to audit (default: both)\n" +
    "  --format json|text       Report format (default: json)\n" +
    "  --output ABSOLUTE_PATH   Write report to a file instead of stdout\n";
}

export function parseArgs(argv) {
  const options = { mode: "both", format: "json", output: "" };
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    const nextValue = () => {
      const value = argv[++index];
      if (value === undefined || value.startsWith("--")) {
        throw new Error(`margo.contrast_lint_value_required:${argument}`);
      }
      return value;
    };
    switch (argument) {
      case "--html":
        options.html = nextValue();
        break;
      case "--mode":
        options.mode = nextValue();
        break;
      case "--format":
        options.format = nextValue();
        break;
      case "--output":
        options.output = nextValue();
        break;
      case "--help":
      case "-h":
        options.help = true;
        break;
      default:
        throw new Error(`margo.contrast_lint_unknown_argument:${argument}`);
    }
  }
  if (options.help) return options;
  if (!options.html) throw new Error("margo.contrast_lint_html_required");
  if (!path.isAbsolute(options.html)) throw new Error("margo.contrast_lint_html_absolute_required");
  if (!options.output && options.output !== "") throw new Error("margo.contrast_lint_output_required");
  if (options.output && !path.isAbsolute(options.output)) throw new Error("margo.contrast_lint_output_absolute_required");
  if (!["light", "dark", "both"].includes(options.mode)) throw new Error(`margo.contrast_lint_mode_invalid:${options.mode}`);
  if (!["json", "text"].includes(options.format)) throw new Error(`margo.contrast_lint_format_invalid:${options.format}`);
  return options;
}

function selectedModes(mode) {
  return mode === "both" ? MODES : [mode];
}

function digest(bytes) {
  return createHash("sha256").update(bytes).digest("hex");
}

function normalizeBlockedResources(resources) {
  const unique = new Map();
  for (const { method, resourceType, url } of resources) {
    const key = `${method}\u0000${url}`;
    if (!unique.has(key)) unique.set(key, { method, resourceType, url });
  }
  return [...unique.values()]
    .sort((left, right) => JSON.stringify(left).localeCompare(JSON.stringify(right)));
}

function summarizeFailure(failure) {
  return `${failure.selector} ratio=${failure.ratio} required=${failure.required} text=${JSON.stringify(failure.text)}`;
}

export function formatText(report) {
  const lines = [
    `margo contrast lint: ${report.status.toUpperCase()}`,
    `source.sha256 ${report.source.sha256}`,
    `source.bytes ${report.source.bytes}`,
    `media ${report.media}`,
    `rules.normal ${report.rules.normalTextMinimumRatio}:1`,
    `rules.large ${report.rules.largeTextMinimumRatio}:1`,
    `rules.network ${report.rules.network}`,
  ];
  for (const mode of report.modes) {
    lines.push(`${mode.mode} checked=${mode.checked} failures=${mode.failures.length}`);
    for (const failure of mode.failures) lines.push(`  ${summarizeFailure(failure)}`);
  }
  lines.push(`network.blocked ${report.network.blocked.length}`);
  for (const resource of report.network.blocked) {
    lines.push(`  ${resource.method} ${resource.resourceType} ${resource.url}`);
  }
  return `${lines.join("\n")}\n`;
}

export async function lintHTML({ html, executablePath, mode = "both" }) {
  if (!executablePath || !path.isAbsolute(executablePath)) {
    throw new Error("margo.contrast_lint_chromium_absolute_required");
  }
  const bytes = await readFile(html);
  const blocked = [];
  const browser = await chromium.launch({ executablePath, headless: true });
  try {
    const context = await browser.newContext({ javaScriptEnabled: false });
    await context.route("**/*", async (route) => {
      const request = route.request();
      blocked.push({ method: request.method(), resourceType: request.resourceType(), url: request.url() });
      await route.abort("blockedbyclient");
    });
    const modes = [];
    for (const selectedMode of selectedModes(mode)) {
      const page = await context.newPage();
      await page.emulateMedia({ media: "print" });
      await page.setContent(bytes.toString("utf8"), { waitUntil: "load" });
      const declaredResources = await page.evaluate(() => {
        const resources = [];
        for (const element of document.querySelectorAll("img[src],script[src],link[href],iframe[src],video[src],audio[src],source[src],object[data]")) {
          const attribute = element.hasAttribute("src") ? "src" : element.hasAttribute("href") ? "href" : "data";
          const value = element.getAttribute(attribute)?.trim() ?? "";
          if (!value || /^(data|blob|about|javascript):/i.test(value) || value.startsWith("#")) continue;
          resources.push({ method: "GET", resourceType: element.tagName.toLowerCase(), url: value });
        }
        return resources;
      });
      blocked.push(...declaredResources);
      await page.evaluate((colorMode) => {
        const root = document.documentElement;
        root.classList.toggle("dark", colorMode === "dark");
        root.dataset.colorMode = colorMode;
      }, selectedMode);
      const audit = await auditTextContrast(page);
      modes.push({ mode: selectedMode, checked: audit.checked, failures: audit.failures });
      await page.close();
    }
    const normalizedBlocked = normalizeBlockedResources(blocked);
    const failures = modes.flatMap((result) => result.failures);
    return {
      schemaVersion: "margo/contrast-lint/v1",
      source: { bytes: bytes.byteLength, sha256: digest(bytes) },
      media: "print",
      rules: { normalTextMinimumRatio: 4.5, largeTextMinimumRatio: 3, network: "deny" },
      modes,
      network: { blocked: normalizedBlocked },
      status: failures.length === 0 && normalizedBlocked.length === 0 ? "pass" : "fail",
    };
  } finally {
    await browser.close();
  }
}

async function emit(report, format, output) {
  const content = format === "json"
    ? `${JSON.stringify(report, null, 2)}\n`
    : formatText(report);
  if (output) {
    await writeFile(output, content, "utf8");
  } else {
    process.stdout.write(content);
  }
}

export async function main(argv = process.argv.slice(2), environment = process.env) {
  let options;
  try {
    options = parseArgs(argv);
  } catch (error) {
    process.stderr.write(`${usage(error.message)}\n`);
    return EXIT_USAGE;
  }
  if (options.help) {
    process.stdout.write(usage());
    return 0;
  }
  try {
    const report = await lintHTML({
      html: options.html,
      executablePath: environment.MARGO_CHROMIUM_EXECUTABLE,
      mode: options.mode,
    });
    await emit(report, options.format, options.output);
    return report.status === "pass" ? 0 : 1;
  } catch (error) {
    process.stderr.write(`margo.contrast_lint_error:${error.message}\n`);
    return EXIT_USAGE;
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  process.exitCode = await main();
}
