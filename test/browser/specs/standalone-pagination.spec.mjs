import { expect, test } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const repositoryRoot = path.resolve(here, "../../..");
const standaloneSource = fs.readFileSync(path.join(repositoryRoot, "standalone.go"), "utf8");
const scriptMatch = standaloneSource.match(/const standalonePrintPreparationScript = `([\s\S]*?)`/);
if (!scriptMatch) throw new Error("standalone print preparation script not found");
const printPreparationScript = scriptMatch[1];

test("print preparation contains no page predictor", async () => {
  for (const forbidden of [
    "window.innerHeight",
    "data-margo-print-break-before",
    "markCrossPageBlocks",
    "prepareNestedHeadingGroups",
    "getBoundingClientRect",
  ]) {
    expect(printPreparationScript).not.toContain(forbidden);
  }
});

test("print preparation projects controls and restores screen state", async ({ page }) => {
  await page.setContent(`<!doctype html><html><body>
    <div class="goshtoso-document"><article class="margo-document">
      <p><strong id="strong">Strong <a id="link" href="https://example.com">link</a></strong> and <em>emphasis</em>.</p>
      <button class="margo-table-sort-button">Column</button>
      <input id="task" type="checkbox" checked>
      <details class="margo-mermaid__source"><summary>Source</summary><pre>graph TD</pre></details>
    </article></div>
    ${printPreparationScript}
  </body></html>`);

  await page.evaluate(() => globalThis.margoPreparePrint());
  await expect(page.locator("button")).toHaveCount(0);
  await expect(page.locator("input")).toHaveCount(0);
  await expect(page.locator("[data-margo-print-static]")).toHaveCount(4);
  await expect(page.locator("#link")).toHaveCount(1);
  await expect(page.locator(".margo-mermaid__source")).toHaveAttribute("open", "");

  await page.evaluate(() => globalThis.margoRestorePrintState());
  await expect(page.locator("button")).toHaveCount(1);
  await expect(page.locator("input")).toHaveCount(1);
  await expect(page.locator("[data-margo-print-static]")).toHaveCount(0);
  await expect(page.locator("strong #link")).toHaveCount(1);
  await expect(page.locator(".margo-mermaid__source")).not.toHaveAttribute("open", "");
});

test("print CSS delegates fragmentation to Chromium", async () => {
  const css = fs.readFileSync(path.join(repositoryRoot, "assets/document.css"), "utf8");
  const standaloneCSS = fs.readFileSync(path.join(repositoryRoot, "assets/standalone.css"), "utf8");
  expect(css).not.toContain("data-margo-print-break-before");
  expect(css).not.toContain("data-margo-print-heading-group");
  expect(css).toContain("display: table-header-group");
  expect(css).toContain("break-inside: auto");
  expect(standaloneCSS).not.toContain("size: A4 portrait");
});
