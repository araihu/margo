import { expect, test } from "@playwright/test";
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { auditTextContrast } from "../harness/contrast.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");
const origin = "http://margo.test";
let workspace;
const lifecycle = `stateDiagram-v2
    [*] --> Pending: request accepted
    Pending --> Installing: dependencies ready
    Installing --> Ready: current revision converged
    Installing --> Failed: attempts exhausted
    Failed --> Pending: explicit retry
    Ready --> Updating: new request accepted
    Updating --> Ready: current revision converged
    Updating --> Degraded: partial failure
    Degraded --> Updating: correction or retry
    Ready --> Suspended: suspension confirmed
    Suspended --> Updating: resume
    Ready --> Degraded: dependency unavailable
    Pending --> Deleting: removal requested
    Installing --> Deleting: removal requested
    Failed --> Deleting: removal requested
    Ready --> Deleting: removal requested
    Updating --> Deleting: removal requested
    Suspended --> Deleting: removal requested
    Degraded --> Deleting: removal requested
    Deleting --> Blocked: cleanup pending
    Blocked --> Deleting: blocker resolved or retry
    Deleting --> Deleted: inventory closed and status confirmed
    Deleted --> [*]`;
const diagram = "```mermaid\n" + lifecycle + "\n```\n";

test.beforeAll(() => {
  workspace = fs.mkdtempSync(path.join(os.tmpdir(), "margo-mermaid-viewer-"));
  const binary = path.join(workspace, process.platform === "win32" ? "margo.exe" : "margo");
  execFileSync("go", ["build", "-o", binary, "./cmd/margo"], {cwd: root, env: {...process.env, GOWORK: "off"}});
  fs.mkdirSync(path.join(workspace, "docs"));
  fs.mkdirSync(path.join(workspace, "assets"));
  fs.copyFileSync(path.join(root, "assets/logo.svg"), path.join(workspace, "assets/logo.svg"));
  fs.copyFileSync(path.join(root, "assets/social/margo-social-v2.jpg"), path.join(workspace, "assets/social.jpg"));
  fs.mkdirSync(path.join(workspace, "themes"));
  for (const name of ["margo.css", "margo.tokens.yaml"]) {
    fs.copyFileSync(path.join(root, "themes", name), path.join(workspace, "themes", name));
  }
  const source = "# Lifecycle\n\n" + diagram + "\n## Independent diagram\n\n" + diagram +
    "\n## Reading after the diagrams\n\n" + "Content below the diagram.\n\n".repeat(30);
  fs.writeFileSync(path.join(workspace, "docs/index.md"), source);
  fs.writeFileSync(path.join(workspace, "docs/guide.md"), "# Guide\n\n" + diagram);
  fs.writeFileSync(path.join(workspace, "site.yaml"), `version: 1
source: docs
assets: local
site:
  name: Mermaid viewer
  description: Mermaid interaction browser fixture.
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Mermaid viewer browser fixture.
shell:
  builtin: componentdocshell
locales:
  default: en
  supported: [en]
theme:
  builtin: false
  name: margo
  allow_switch_theme: true
  color_mode: light
themes:
  - name: margo
    css_url: themes/margo.css
    token_catalog: themes/margo.tokens.yaml
`);
  execFileSync(binary, ["site", path.join(workspace, "site.yaml"), "--output-dir", path.join(workspace, "site")]);
  execFileSync(binary, ["html", path.join(workspace, "docs/index.md"), "--output", path.join(workspace, "standalone.html")]);
  fs.writeFileSync(path.join(workspace, "deck.md"), "# Diagram\n\n```mermaid\nflowchart LR\n  A --> B\n```\n");
  execFileSync(binary, ["deck", path.join(workspace, "deck.md"), "--output", path.join(workspace, "deck.html")]);
});

test.afterAll(() => fs.rmSync(workspace, {recursive: true, force: true}));

async function routeArtifacts(page) {
  await page.route(`${origin}/**`, async (route) => {
    const pathname = decodeURIComponent(new URL(route.request().url()).pathname);
    const relative = pathname === "/" ? "index.html" : pathname.slice(1);
    const file = ["/standalone.html", "/deck.html"].includes(pathname) ? path.join(workspace, relative) : path.join(workspace, "site", relative);
    if (!file.startsWith(workspace + path.sep) || !fs.existsSync(file)) return route.fulfill({status: 404, body: "Missing artifact"});
    const ext = path.extname(file);
    const types = {".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".json": "application/json", ".svg": "image/svg+xml"};
    return route.fulfill({body: fs.readFileSync(file), contentType: types[ext] || "application/octet-stream"});
  });
}

async function open(page, url = "/standalone.html") {
  await routeArtifacts(page);
  await page.goto(origin + url);
  await expect(page.locator(".margo-mermaid__controls")).toHaveCount(2);
  await expect(page.locator(".margo-mermaid__controls output").first()).toHaveText("100%");
}

const figure = (page, index = 0) => page.locator("figure.margo-mermaid").nth(index);
const canvas = (page) => figure(page).locator(".margo-mermaid__canvas");
const zoomText = (page, index = 0) => figure(page, index).locator("output");
async function zoomIn(page, count = 3) {
  for (let n = 0; n < count; n++) await figure(page).getByRole("button", {name: "Zoom in", exact: true}).click();
}

test("@mermaid-viewer independent controls, keyboard, pointer pan and static print", async ({page}, testInfo) => {
  await page.setViewportSize({width: 984, height: 1109});
  await open(page);
  const initial = await canvas(page).locator("svg").boundingBox();
  await zoomIn(page);
  await expect(zoomText(page)).toHaveText("195%");
  await expect(zoomText(page, 1)).toHaveText("100%");
  const enlarged = await canvas(page).locator("svg").boundingBox();
  expect(enlarged.width).toBeGreaterThan(initial.width * 1.9);
  expect(enlarged.height).toBeGreaterThan(initial.height * 1.9);
  await page.screenshot({path: testInfo.outputPath("mermaid-zoom.png")});

  await page.keyboard.press("Tab");
  await canvas(page).focus();
  await expect(canvas(page)).toBeFocused();
  expect(await canvas(page).evaluate((element) => getComputedStyle(element).outlineStyle)).toBe("solid");
  await canvas(page).evaluate((element) => element.scrollTo(0, 0));
  await page.keyboard.press("ArrowDown");
  await expect.poll(() => canvas(page).evaluate((element) => element.scrollTop)).toBeGreaterThan(0);
  await page.keyboard.press("ArrowRight");
  await expect.poll(() => canvas(page).evaluate((element) => element.scrollLeft)).toBeGreaterThan(0);
  await page.keyboard.press("+");
  await expect(zoomText(page)).toHaveText("244%");
  await page.keyboard.press("-");
  await expect(zoomText(page)).toHaveText("195%");
  const area = await canvas(page).boundingBox();
  await canvas(page).evaluate((element) => {
    element.addEventListener("auxclick", (event) => {
      if (event.button === 1) window.viewerMiddleDefaultPrevented = event.defaultPrevented;
    });
  });
  for (const button of ["left", "middle"]) {
    await canvas(page).evaluate((element) => element.scrollTo(0, 0));
    await page.mouse.move(area.x + area.width / 2, area.y + area.height * 0.7);
    await page.mouse.down({button});
    await page.mouse.move(area.x + area.width / 2 - 80, area.y + area.height * 0.7 - 100, {steps: 5});
    await page.mouse.up({button});
    const panned = await canvas(page).evaluate((element) => ({left: element.scrollLeft, top: element.scrollTop}));
    expect(panned.top).toBeGreaterThan(80);
    expect(panned.left).toBeGreaterThan(60);
    await page.mouse.move(area.x + area.width / 2 - 100, area.y + area.height * 0.7 - 120);
    expect(await canvas(page).evaluate((element) => ({left: element.scrollLeft, top: element.scrollTop}))).toEqual(panned);
  }
  expect(await page.evaluate(() => window.viewerMiddleDefaultPrevented)).toBe(true);
  const beforeWheel = await canvas(page).evaluate((element) => element.scrollTop);
  await page.mouse.wheel(0, 60);
  await expect.poll(() => canvas(page).evaluate((element) => element.scrollTop)).toBeGreaterThan(beforeWheel);
  await page.keyboard.press("0");
  await expect(zoomText(page)).toHaveText("100%");
  expect(await canvas(page).evaluate((element) => element.scrollTop)).toBe(0);
  await expect(figure(page).locator(".margo-mermaid__accessible-source")).toContainText("stateDiagram-v2");

  await zoomIn(page, 4);
  await canvas(page).evaluate((element) => element.scrollTo(70, 140));
  await page.emulateMedia({media: "print"});
  await expect(figure(page).locator(".margo-mermaid__controls")).toBeHidden();
  await expect(figure(page).locator(".margo-mermaid__hint")).toHaveCount(0);
  const printed = await canvas(page).evaluate((element) => {
    const svg = element.querySelector("svg").getBoundingClientRect();
    const viewport = element.getBoundingClientRect();
    return {svg: {top: svg.top, bottom: svg.bottom, width: svg.width}, viewport: {top: viewport.top, bottom: viewport.bottom, width: viewport.width}};
  });
  expect(printed.svg.top).toBeGreaterThanOrEqual(printed.viewport.top - 1);
  expect(printed.svg.bottom).toBeLessThanOrEqual(printed.viewport.bottom + 1);
  expect(printed.svg.width).toBeLessThanOrEqual(printed.viewport.width + 1);
  const pdf = await page.pdf({path: testInfo.outputPath("mermaid-static.pdf")});
  expect(pdf.subarray(0, 5).toString()).toBe("%PDF-");
  await page.emulateMedia({media: "screen"});
  await expect(zoomText(page)).toHaveText("244%");
});

test("@mermaid-viewer HTMX navigation and theme rerenders keep one usable control group", async ({page}, testInfo) => {
  await open(page, "/");
  await zoomIn(page);
  await canvas(page).evaluate((element) => element.scrollTo(60, 90));
  const lightSVG = await canvas(page).locator("svg").evaluate((element) => element.outerHTML);
  await page.locator("#componentdocshell-dark-mode").click();
  await expect(zoomText(page)).toHaveText("195%");
  await expect.poll(() => canvas(page).locator("svg").evaluate((element, previous) => element.outerHTML !== previous, lightSVG)).toBe(true);
  expect(await canvas(page).evaluate((element) => ({left: element.scrollLeft, top: element.scrollTop}))).toEqual({left: 60, top: 90});
  const contrast = await auditTextContrast(page);
  expect(contrast.checked).toBeGreaterThan(8);
  expect(contrast.failures.filter(({text}) =>
    ["Zoom in", "Zoom out", "Fit diagram", "195%"].includes(text) || text.startsWith("Use + /"))).toEqual([]);
  await expect(figure(page).locator(".margo-mermaid__controls")).toHaveCount(1);
  await figure(page).getByRole("button", {name: "Zoom in", exact: true}).click();
  await expect(zoomText(page)).toHaveText("244%");
  await figure(page).getByRole("button", {name: "Expand", exact: true}).click();
  const expandedContrast = await auditTextContrast(page);
  expect(expandedContrast.failures.filter(({text}) =>
    ["Zoom in", "Zoom out", "Fit diagram", "Fullscreen", "Close expanded view", "100%"].includes(text) || text.startsWith("Use + /"))).toEqual([]);
  await page.screenshot({path: testInfo.outputPath("mermaid-expanded-dark.png")});
  await page.keyboard.press("Escape");
  await expect(zoomText(page)).toHaveText("244%");
  await page.evaluate(() => { window.viewerNavigationSentinel = true; });
  await page.locator('a[href="/guide.html"][hx-get]').first().click();
  await expect(page.locator("#main-content h1")).toHaveText("Guide");
  await expect(page.locator(".margo-mermaid__controls")).toHaveCount(1);
  await zoomIn(page, 1);
  await expect(zoomText(page)).toHaveText("125%");
  expect(await page.evaluate(() => window.viewerNavigationSentinel)).toBe(true);
  await page.goBack();
  await expect(page.locator("#main-content h1")).toHaveText("Lifecycle");
  await expect(page.locator(".margo-mermaid__controls")).toHaveCount(2);
  await figure(page).getByRole("button", {name: "Fit diagram", exact: true}).click();
  await expect(zoomText(page)).toHaveText("100%");
});

test("@mermaid-viewer bounded zoom and refitting after resize", async ({page}) => {
  await open(page);
  await canvas(page).focus();
  for (let step = 0; step < 12; step++) await page.keyboard.press("+");
  await expect(zoomText(page)).toHaveText("800%");
  await expect(figure(page).getByRole("button", {name: "Zoom in", exact: true})).toBeDisabled();
  for (let step = 0; step < 20; step++) await page.keyboard.press("-");
  await expect(zoomText(page)).toHaveText("25%");
  await expect(figure(page).getByRole("button", {name: "Zoom out", exact: true})).toBeDisabled();
  await page.setViewportSize({width: 390, height: 844});
  await figure(page).getByRole("button", {name: "Fit diagram", exact: true}).click();
  await expect(zoomText(page)).toHaveText("100%");
  await expect.poll(() => canvas(page).evaluate((element) => element.scrollWidth <= element.clientWidth + 1 && element.scrollHeight <= element.clientHeight + 1)).toBe(true);
  await canvas(page).focus();
  await page.keyboard.press("+");
  await page.keyboard.press("Home");
  await expect(zoomText(page)).toHaveText("100%");
});

test("@mermaid-viewer preserves presentation geometry", async ({page}) => {
  await routeArtifacts(page);
  await page.goto(origin + "/deck.html");
  await expect(page.locator(".margo-mermaid__canvas svg")).toBeVisible();
  await expect(page.locator(".margo-mermaid__controls")).toHaveCount(0);
  await expect(page.locator(".margo-mermaid__stage")).toHaveCount(0);
  expect(await page.evaluate(() => globalThis.margoValidateDeckScreen())).toBe(true);
});

test("@mermaid-viewer touch pans locally and outside gestures scroll the page", async ({browser}) => {
  const context = await browser.newContext({viewport: {width: 390, height: 844}, hasTouch: true, isMobile: true});
  const page = await context.newPage();
  try {
    await open(page);
    await zoomIn(page, 4);
    await canvas(page).scrollIntoViewIfNeeded();
    await canvas(page).evaluate((element) => element.scrollTo(0, 0));
    const client = await context.newCDPSession(page);
    const swipe = async (x, y) => {
      await client.send("Input.dispatchTouchEvent", {type: "touchStart", touchPoints: [{x, y}]});
      for (let step = 1; step <= 8; step++) {
        await client.send("Input.dispatchTouchEvent", {type: "touchMove", touchPoints: [{x, y: y - step * 15}]});
      }
      await client.send("Input.dispatchTouchEvent", {type: "touchEnd", touchPoints: []});
    };
    const area = await canvas(page).boundingBox();
    const before = await page.evaluate(() => window.scrollY);
    await swipe(area.x + area.width / 2, area.y + Math.min(area.height - 10, 200));
    await expect.poll(() => canvas(page).evaluate((element) => element.scrollTop)).toBeGreaterThan(30);
    expect(Math.abs(await page.evaluate(() => window.scrollY) - before)).toBeLessThan(5);
    await swipe(5, 600);
    await expect.poll(() => page.evaluate(() => window.scrollY)).toBeGreaterThan(before + 30);
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390);
  } finally {
    await context.close();
  }
});

test("@mermaid-viewer source fallback works without JavaScript", async ({browser}) => {
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();
  try {
    await routeArtifacts(page);
    await page.goto(origin + "/standalone.html");
    await expect(page.locator(".margo-mermaid__controls")).toHaveCount(0);
    const source = figure(page).locator(".margo-mermaid__source");
    await expect(source).toBeVisible();
    await source.locator("summary").click();
    await expect(source.locator("code")).toBeVisible();
    await expect(source.locator("code")).toContainText("stateDiagram-v2");
  } finally {
    await context.close();
  }
});

test("@mermaid-viewer modal keeps the SVG, traps focus and restores the inline view", async ({page}, testInfo) => {
  await page.setViewportSize({width: 1440, height: 900});
  await open(page);
  await zoomIn(page);
  await canvas(page).evaluate((element) => {
    element.scrollTo(60, 90);
    element.querySelector("svg").viewerIdentity = "original";
  });
  const expand = figure(page).getByRole("button", {name: "Expand", exact: true});
  await expand.click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();
  await expect(dialog).toHaveAccessibleName(/Mermaid diagram for Lifecycle/);
  const closeBounds = await dialog.getByRole("button", {name: "Close expanded view", exact: true}).boundingBox();
  const dialogBounds = await dialog.boundingBox();
  expect(closeBounds.x + closeBounds.width).toBeGreaterThan(dialogBounds.x + dialogBounds.width - 30);
  expect(closeBounds.y).toBeLessThan(dialogBounds.y + 30);
  const fullscreen = dialog.getByRole("button", {name: "Fullscreen", exact: true});
  await fullscreen.hover();
  expect(await fullscreen.evaluate((element) => getComputedStyle(element, "::after").content)).toBe('"Fullscreen"');
  expect(await fullscreen.evaluate((element) => getComputedStyle(element, "::after").visibility)).toBe("visible");
  await page.mouse.move(0, 0);
  await expect(zoomText(page)).toHaveText("100%");
  expect(await canvas(page).locator("svg").evaluate((element) => element.viewerIdentity)).toBe("original");
  await expect(figure(page).locator(".margo-mermaid__canvas svg")).toHaveCount(1);
  expect(await page.evaluate(() => document.documentElement.style.overflow)).toBe("hidden");
  await dialog.getByRole("button", {name: "Zoom out", exact: true}).focus();
  await page.keyboard.press("Shift+Tab");
  const closeButton = dialog.getByRole("button", {name: "Close expanded view", exact: true});
  await expect(closeButton).toBeFocused();
  expect(await closeButton.evaluate((element) => getComputedStyle(element, "::after").visibility)).toBe("visible");
  await zoomIn(page, 2);
  await expect(zoomText(page)).toHaveText("156%");
  await canvas(page).evaluate((element) => element.scrollTo(0, 0));
  const expandedArea = await canvas(page).boundingBox();
  await page.mouse.move(expandedArea.x + expandedArea.width / 2, expandedArea.y + expandedArea.height / 2);
  await page.mouse.down({button: "middle"});
  await page.mouse.move(expandedArea.x + expandedArea.width / 2 - 40, expandedArea.y + expandedArea.height / 2 - 50, {steps: 5});
  await page.mouse.up({button: "middle"});
  expect(await canvas(page).evaluate((element) => element.scrollTop)).toBeGreaterThan(40);
  await page.screenshot({path: testInfo.outputPath("mermaid-expanded.png")});
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
  await expect(expand).toBeFocused();
  await expect(zoomText(page)).toHaveText("195%");
  expect(await canvas(page).evaluate((element) => ({left: element.scrollLeft, top: element.scrollTop}))).toEqual({left: 60, top: 90});
  expect(await page.evaluate(() => document.documentElement.style.overflow)).toBe("");
  await expand.click();
  await page.mouse.click(2, 2);
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(expand).toBeFocused();
});

test("@mermaid-viewer double-click expands fitted and zoomed diagrams and restores focus", async ({page}) => {
  await open(page);
  for (const steps of [0, 3]) {
    await zoomIn(page, steps);
    const inlineZoom = await zoomText(page).textContent();
    await canvas(page).evaluate((element) => element.scrollTo(60, 90));
    const inlinePan = await canvas(page).evaluate((element) => ({left: element.scrollLeft, top: element.scrollTop}));
    await canvas(page).dblclick();
    await expect(page.getByRole("dialog")).toHaveCount(1);
    await expect(page.getByRole("dialog")).toBeVisible();
    await expect(zoomText(page)).toHaveText("100%");
    await zoomIn(page, 1);
    await canvas(page).dblclick();
    await expect(page.getByRole("dialog")).toHaveCount(1);
    await expect(zoomText(page)).toHaveText("125%");
    await page.keyboard.press("Escape");
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(canvas(page)).toBeFocused();
    await expect(zoomText(page)).toHaveText(inlineZoom);
    expect(await canvas(page).evaluate((element) => ({left: element.scrollLeft, top: element.scrollTop}))).toEqual(inlinePan);
  }
});

test("@mermaid-viewer Ctrl+wheel zooms around the pointer and plain wheel still scrolls", async ({page}) => {
  await page.setViewportSize({width: 984, height: 1109});
  await open(page);
  await zoomIn(page, 2);
  await canvas(page).scrollIntoViewIfNeeded();
  const bounds = await canvas(page).boundingBox();
  const pointer = {x: bounds.x + bounds.width * 0.6, y: bounds.y + bounds.height * 0.6};
  const before = await canvas(page).locator("svg").boundingBox();
  const point = {x: (pointer.x - before.x) / before.width, y: (pointer.y - before.y) / before.height};
  const pageMetrics = await page.evaluate(() => ({width: innerWidth, ratio: devicePixelRatio}));
  await page.mouse.move(pointer.x, pointer.y);
  await page.keyboard.down("Control");
  try {
    await page.mouse.wheel(0, -120);
    await expect.poll(() => zoomText(page).textContent()).not.toBe("156%");
  } finally {
    await page.keyboard.up("Control");
  }
  const after = await canvas(page).locator("svg").boundingBox();
  expect(after.width).toBeGreaterThan(before.width);
  expect(Math.abs(after.x + point.x * after.width - pointer.x)).toBeLessThan(2);
  expect(Math.abs(after.y + point.y * after.height - pointer.y)).toBeLessThan(2);
  expect(await page.evaluate(() => ({width: innerWidth, ratio: devicePixelRatio}))).toEqual(pageMetrics);
  await expect(zoomText(page, 1)).toHaveText("100%");
  const zoomed = await zoomText(page).textContent();
  await canvas(page).evaluate((element) => element.scrollTo(0, 0));
  await page.mouse.wheel(0, 90);
  await expect.poll(() => canvas(page).evaluate((element) => element.scrollTop)).toBeGreaterThan(0);
  await expect(zoomText(page)).toHaveText(zoomed);
  await page.keyboard.down("Control");
  try {
    await page.mouse.wheel(0, 120);
    await expect.poll(() => zoomText(page).textContent()).not.toBe(zoomed);
  } finally {
    await page.keyboard.up("Control");
  }
});

test("@mermaid-viewer Goshtoso help replaces the footer and dismisses before the modal", async ({page}, testInfo) => {
  await open(page);
  expect(await page.evaluate(() => typeof window.Alpine)).toBe("undefined");
  await expect(page.locator(".margo-mermaid__hint")).toHaveCount(0);
  const help = figure(page).getByRole("button", {name: "Diagram help", exact: true});
  const panel = figure(page).getByRole("tooltip");
  await expect(help).toHaveAttribute("aria-expanded", "false");
  await help.focus();
  await page.keyboard.press("Enter");
  await expect(panel).toBeVisible();
  await expect(panel).toContainText("Ctrl + wheel");
  await expect(panel).toContainText("middle-button drag");
  expect(await help.evaluate((element) => getComputedStyle(element, "::after").content)).toBe("none");
  await expect(help).toHaveAttribute("aria-controls", await panel.getAttribute("id"));
  await canvas(page).click();
  await expect(panel).toBeHidden();
  await figure(page).getByRole("button", {name: "Expand", exact: true}).click();
  await help.click();
  await expect(panel).toBeVisible();
  await page.screenshot({path: testInfo.outputPath("mermaid-help.png")});
  await page.keyboard.press("Escape");
  await expect(panel).toBeHidden();
  await expect(help).toBeFocused();
  await expect(page.getByRole("dialog")).toBeVisible();
  await help.click();
  await expect(panel).toBeVisible();
  await page.emulateMedia({media: "print"});
  await expect(panel).toBeHidden();
  await page.emulateMedia({media: "screen"});
  await figure(page).getByRole("button", {name: "Close expanded view", exact: true}).click();
  await help.click();
  await expect(panel).toBeVisible();
  await figure(page).evaluate((element) => element.remove());
  await expect(page.locator("[data-tooltip-portal-open]")).toHaveCount(0);
  await figure(page).getByRole("button", {name: "Diagram help", exact: true}).click();
  await expect(figure(page).getByRole("tooltip")).toBeVisible();
});

test("@mermaid-viewer native fullscreen exits to its originating view", async ({page}) => {
  await open(page);
  const viewer = figure(page).locator(".margo-mermaid__viewer");
  const fullscreen = figure(page).getByRole("button", {name: "Fullscreen", exact: true});
  await fullscreen.click();
  await expect.poll(() => viewer.evaluate((element) => document.fullscreenElement === element)).toBe(true);
  await expect(figure(page).getByRole("button", {name: "Exit fullscreen", exact: true})).toHaveAttribute("aria-pressed", "true");
  await zoomIn(page, 1);
  await expect(zoomText(page)).toHaveText("125%");
  await page.evaluate(() => document.exitFullscreen());
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(fullscreen).toBeFocused();
  await expect(zoomText(page)).toHaveText("100%");

  await figure(page).getByRole("button", {name: "Expand", exact: true}).click();
  await fullscreen.click();
  await expect.poll(() => viewer.evaluate((element) => document.fullscreenElement === element)).toBe(true);
  await figure(page).getByRole("button", {name: "Exit fullscreen", exact: true}).click();
  await expect.poll(() => page.evaluate(() => document.fullscreenElement === null)).toBe(true);
  await expect(page.getByRole("dialog")).toBeVisible();
  await figure(page).getByRole("button", {name: "Close expanded view", exact: true}).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
});

test("@mermaid-viewer fullscreen fallback fits narrow screens and restores focus", async ({page}) => {
  await page.setViewportSize({width: 390, height: 844});
  await open(page);
  for (const denied of [false, true]) {
    await page.evaluate((reject) => {
      Element.prototype.requestFullscreen = reject ? () => Promise.reject(new Error("Fullscreen denied")) : undefined;
    }, denied);
    const fullscreen = figure(page).getByRole("button", {name: "Fullscreen", exact: true});
    await fullscreen.click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toHaveClass(/margo-mermaid__dialog--fullscreen/);
    const bounds = await dialog.boundingBox();
    expect(bounds.width).toBeCloseTo(390, 0);
    expect(bounds.height).toBeCloseTo(844, 0);
    expect(await dialog.evaluate((element) => element.scrollWidth <= element.clientWidth + 1)).toBe(true);
    await expect(figure(page).getByRole("button", {name: "Exit fullscreen", exact: true})).toHaveAttribute("aria-pressed", "true");
    await canvas(page).focus();
    await page.keyboard.press("+");
    await expect(zoomText(page)).toHaveText("125%");
    await page.keyboard.press("Escape");
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(fullscreen).toBeFocused();
    expect(await page.evaluate(() => document.documentElement.style.overflow)).toBe("");
  }
});

test("@mermaid-viewer an expanded diagram prints whole and detachment unlocks the page", async ({page}, testInfo) => {
  await open(page, "/");
  for (const action of ["Expand", "Fullscreen"]) {
    await figure(page).getByRole("button", {name: action, exact: true}).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    if (action === "Fullscreen") await expect.poll(() => page.evaluate(() => !!document.fullscreenElement)).toBe(true);
    await zoomIn(page, 4);
    await page.emulateMedia({media: "print"});
    await expect.poll(() => page.evaluate(() => document.fullscreenElement === null)).toBe(true);
    await expect(figure(page).locator(".margo-mermaid__controls")).toBeHidden();
    const printBounds = await canvas(page).evaluate((element) => {
      const viewport = element.getBoundingClientRect();
      const figure = element.closest("figure").getBoundingClientRect();
      const svg = element.querySelector("svg").getBoundingClientRect();
      return svg.width <= viewport.width + 1 && svg.top >= viewport.top - 1 && svg.bottom <= viewport.bottom + 1 &&
        svg.top >= figure.top && svg.left >= figure.left && svg.right <= figure.right;
    });
    expect(printBounds).toBe(true);
    await page.pdf({path: testInfo.outputPath("mermaid-" + action.toLowerCase() + "-static.pdf")});
    await page.emulateMedia({media: "screen"});
    await expect(page.getByRole("dialog")).toBeVisible();
    await expect(zoomText(page)).toHaveText("244%");
    await figure(page).getByRole("button", {name: "Close expanded view", exact: true}).click();
  }
  await figure(page).getByRole("button", {name: "Expand", exact: true}).click();
  await figure(page).evaluate((element) => element.remove());
  await expect.poll(() => page.evaluate(() => document.documentElement.style.overflow)).toBe("");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await figure(page).getByRole("button", {name: "Expand", exact: true}).click();
  await expect(page.getByRole("dialog")).toBeVisible();
});
