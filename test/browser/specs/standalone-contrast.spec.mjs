import { expect, test } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import path from "node:path";
import { auditTextContrast } from "../harness/contrast.mjs";
import { fixture } from "../harness/contrast-fixture.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");

async function stylesheet(name) {
  return readFile(path.join(root, "assets", name), "utf8");
}

test("@contrast light and dark standalone text meet WCAG AA", async ({ page }) => {
  const [documentCSS, standaloneCSS] = await Promise.all([
    stylesheet("document.css"),
    stylesheet("standalone.css"),
  ]);

  for (const mode of ["light", "dark"]) {
    await page.setContent(fixture(documentCSS, standaloneCSS, mode));
    const result = await auditTextContrast(page);
    expect(result.checked, `${mode} text nodes audited`).toBeGreaterThan(15);
    expect(result.failures, `${mode} contrast failures: ${JSON.stringify(result.failures)}`).toEqual([]);
  }
});
