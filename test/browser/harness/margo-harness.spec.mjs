import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { expect, test } from "@playwright/test";
import { probeDOM } from "./probe.mjs";

const here = dirname(fileURLToPath(import.meta.url));
const fixtures = join(here, "..", "fixtures");

async function fixture(name) {
  return readFile(join(fixtures, name), "utf8");
}

test("@margo-harness DOM XML CSS and network probe", async ({ page }) => {
  const requests = [];
  await page.route("**/*", async (route) => {
    requests.push(route.request().url());
    await route.abort("blockedbyclient");
  });
  await page.setContent(await fixture("probe.html"));
  const result = await probeDOM(page, await fixture("valid.svg"), await fixture("valid.css"));
  expect(result.names).toEqual(["svg", "g", "rect", "text"]);
  expect(result.serialized).toContain("Probe");
  expect(result.cssRuleCount).toBe(1);
  expect(result.computedColor).toBe("rgb(17, 34, 51)");
  expect(requests).toEqual([]);
});

test("@margo-harness malformed structure and CSS fail closed", async ({ page }) => {
  await page.setContent(await fixture("probe.html"));
  await expect(probeDOM(page, await fixture("malformed.svg"), await fixture("valid.css")))
    .rejects.toThrow("margo.harness_svg_malformed");
  await expect(probeDOM(page, await fixture("unknown.svg"), await fixture("valid.css")))
    .rejects.toThrow("margo.harness_svg_element_unknown:script");
  await expect(probeDOM(page, await fixture("valid.svg"), await fixture("invalid.css")))
    .rejects.toThrow("margo.harness_css_at_rule_forbidden:import");
  await expect(probeDOM(page, await fixture("valid.svg"), await fixture("unknown.css")))
    .rejects.toThrow("margo.harness_css_property_unknown:position");
});
