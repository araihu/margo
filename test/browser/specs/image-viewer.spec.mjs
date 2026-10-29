import {expect, test} from "@playwright/test";
import {moveTouches, pinch} from "../harness/touch.mjs";
import {execFileSync} from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import {fileURLToPath} from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");
const origin = "http://margo.test";
const longCaption = "Layouts de apresentação — imagem com zoom e pan";
let workspace;

test.beforeAll(async ({browser}) => {
  workspace = fs.mkdtempSync(path.join(os.tmpdir(), "margo-image-viewer-"));
  const binary = path.join(workspace, "margo");
  execFileSync("go", ["build", "-o", binary, "./cmd/margo"], {cwd: root, env: {...process.env, GOWORK: "off"}});
  fs.mkdirSync(path.join(workspace, "docs"));
  // A raster dashboard large enough that 800% is insufficient on mobile.
  const fixture = await browser.newPage({viewport: {width: 4800, height: 3000}});
  await fixture.setContent(`<style>body{margin:0;background:#f8fafc;color:#102238;font:32px sans-serif;padding:40px}main{display:grid;grid-template-columns:repeat(8,1fr);gap:24px}section{background:white;border:2px solid #64748b;padding:24px;height:210px}h1{font-size:56px}</style><h1>Platform dashboard — deployment health</h1><main>${Array.from({length: 80}, (_, i) => `<section><h2>Service ${i + 1}</h2><p>Requests: ${10000 + i} · Latency: ${20 + i}ms</p><p>Healthy · Revision 2026.10.28</p></section>`).join("")}</main>`);
  await fixture.screenshot({path: path.join(workspace, "docs/dashboard.png")});
  await fixture.close();
  const article = `---
margo:
  imageViewer: true
---
# Dashboard

[Home](index.md)

${"Introduction to this dashboard.\n\n".repeat(8)}
![Service health details](dashboard.png "Platform dashboard")

![Independent dashboard](dashboard.png "Second dashboard")

![Logo](dashboard.png)

Inline ![Icon](dashboard.png "Icon") text.

[![Thumbnail](dashboard.png "Thumbnail")](dashboard.png)

![Unavailable screenshot](missing.png "Missing image")

${"Reading after the screenshots.\n\n".repeat(40)}
`;
  fs.writeFileSync(path.join(workspace, "docs/guide.md"), article);
  fs.writeFileSync(path.join(workspace, "docs/index.md"), "# Home\n\n[Dashboard](guide.md)\n");
  fs.mkdirSync(path.join(workspace, "assets"));
  fs.copyFileSync(path.join(root, "assets/logo.svg"), path.join(workspace, "assets/logo.svg"));
  fs.copyFileSync(path.join(root, "assets/social/margo-social-v2.jpg"), path.join(workspace, "assets/social.jpg"));
  fs.mkdirSync(path.join(workspace, "themes"));
  for (const name of ["margo.css", "margo.tokens.yaml"]) fs.copyFileSync(path.join(root, "themes", name), path.join(workspace, "themes", name));
  fs.writeFileSync(path.join(workspace, "site.yaml"), `version: 1
source: docs
assets: local
site:
  name: Image viewer
  description: Image viewer browser fixture.
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Image viewer fixture.
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
  // Missing images remain useful fallbacks in HTML; the site validates local
  // asset existence, so supply the URL but return a browser-level failure.
  fs.copyFileSync(path.join(workspace, "docs/dashboard.png"), path.join(workspace, "docs/missing.png"));
  execFileSync(binary, ["site", path.join(workspace, "site.yaml"), "--output-dir", path.join(workspace, "site")]);
  fs.writeFileSync(path.join(workspace, "docs/standalone.md"), article.replace('![Unavailable screenshot](missing.png "Missing image")', ""));
  execFileSync(binary, ["html", path.join(workspace, "docs/standalone.md"), "--output", path.join(workspace, "inline.html")]);
  const longArticle = fs.readFileSync(path.join(workspace, "docs/standalone.md"), "utf8").replace('"Platform dashboard"', `"${longCaption}"`);
  fs.writeFileSync(path.join(workspace, "docs/long-caption.md"), longArticle);
  execFileSync(binary, ["html", path.join(workspace, "docs/long-caption.md"), "--output", path.join(workspace, "long-caption.html")]);
});

test.afterAll(() => fs.rmSync(workspace, {recursive: true, force: true}));

async function routeArtifacts(page) {
  await page.route(`${origin}/**`, (route) => {
    const pathname = decodeURIComponent(new URL(route.request().url()).pathname);
    if (pathname.endsWith("missing.png")) return route.fulfill({status: 404, body: "Missing image"});
    let file;
    if (pathname === "/inline.html") file = path.join(workspace, "inline.html");
    else {
      let relative = pathname === "/" ? "index.html" : pathname.slice(1);
      if (pathname !== "/" && pathname.endsWith("/")) relative = path.join(relative, "index.html");
      file = path.join(workspace, "site", relative);
      if (!fs.existsSync(file)) file = path.join(workspace, relative);
    }
    if (!file.startsWith(workspace + path.sep) || !fs.existsSync(file)) return route.fulfill({status: 404, body: "Missing artifact"});
    const types = {".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".png": "image/png", ".svg": "image/svg+xml"};
    return route.fulfill({body: fs.readFileSync(file), contentType: types[path.extname(file)] || "application/octet-stream"});
  });
}

const figure = (page, index = 0) => page.locator("figure.margo-image").nth(index);
const canvas = (page) => figure(page).locator(".margo-image__canvas");
async function open(page, url = "/inline.html") {
  await routeArtifacts(page);
  await page.goto(origin + url);
  await expect(page.locator(".margo-viewer__controls")).toHaveCount(2);
}

for (const assets of ["inline", "local"]) {
  test(`@image-viewer ${assets} assets, independent zoom, native size, keyboard and pointer pan`, async ({page}, testInfo) => {
    await page.setViewportSize({width: 390, height: 844});
    await open(page, assets === "inline" ? "/inline.html" : "/guide.html");
    expect(await page.evaluate(() => typeof window.mermaid)).toBe("undefined");
    await expect(figure(page).locator("img")).toHaveAttribute("alt", "Service health details");
    await expect(figure(page, 3).locator(".margo-viewer__controls")).toHaveCount(0);
    const actual = figure(page).getByRole("button", {name: "Actual size", exact: true});
    await actual.click();
    const size = await canvas(page).locator("img").boundingBox();
    expect(size.width).toBeCloseTo(4800, 0);
    expect(size.height).toBeCloseTo(3000, 0);
    await expect(figure(page, 1).locator("output")).toHaveText("100%");
    await page.keyboard.press("Tab");
    await canvas(page).focus();
    expect(await canvas(page).evaluate((el) => getComputedStyle(el).outlineStyle)).toBe("solid");
    await canvas(page).evaluate((el) => el.scrollTo(0, 0));
    await page.keyboard.press("ArrowDown");
    await page.keyboard.press("ArrowRight");
    expect(await canvas(page).evaluate((el) => [el.scrollLeft, el.scrollTop])).toEqual([48, 48]);
    const box = await canvas(page).boundingBox();
    await page.mouse.move(box.x + 170, box.y + 120);
    await page.mouse.down();
    await page.mouse.move(box.x + 90, box.y + 50, {steps: 4});
    await page.mouse.up();
    expect(await canvas(page).evaluate((el) => el.scrollLeft)).toBeGreaterThan(100);
    await page.keyboard.press("0");
    await expect(figure(page).locator("output")).toHaveText("100%");
    for (const dark of [false, true]) {
      await page.evaluate((value) => document.documentElement.classList.toggle("dark", value), dark);
      await page.screenshot({path: testInfo.outputPath(`${assets}-${dark ? "dark" : "light"}-mobile.png`)});
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    }
  });
}

test("@image-viewer expanded view restores focus, image state and reading position; print stays static", async ({page}, testInfo) => {
  await open(page);
  await figure(page).getByRole("button", {name: "Actual size", exact: true}).click();
  await canvas(page).evaluate((el) => el.scrollTo(80, 120));
  const expand = figure(page).getByRole("button", {name: "Expand", exact: true});
  await expand.scrollIntoViewIfNeeded();
  const readingPosition = await page.evaluate(() => scrollY);
  const zoom = await figure(page).locator("output").textContent();
  await expand.click();
  await expect(page.getByRole("dialog")).toBeVisible();
  await expect(page.getByRole("dialog")).toContainText("Platform dashboard");
  await expect(page.getByRole("dialog").locator("img")).toHaveAttribute("alt", "Service health details");
  await page.getByRole("dialog").getByRole("button", {name: "Actual size", exact: true}).click();
  await page.emulateMedia({media: "print"});
  await expect(page.getByRole("dialog")).toBeHidden();
  await expect(figure(page).locator(".margo-viewer__controls")).toBeHidden();
  const printed = await canvas(page).evaluate((el) => {
    const image = el.querySelector("img").getBoundingClientRect();
    const bounds = el.getBoundingClientRect();
    return image.width <= bounds.width + 1 && image.top >= bounds.top - 1 && image.bottom <= bounds.bottom + 1;
  });
  expect(printed).toBe(true);
  await page.pdf({path: testInfo.outputPath("image-static.pdf")});
  await page.emulateMedia({media: "screen"});
  await expect(page.getByRole("dialog")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(expand).toBeFocused();
  await expect(figure(page).locator("output")).toHaveText(zoom);
  expect(await canvas(page).evaluate((el) => [el.scrollLeft, el.scrollTop])).toEqual([80, 120]);
  expect(await page.evaluate(() => scrollY)).toBeCloseTo(readingPosition, 0);
});

test("@image-viewer fullscreen fallback and detached modal release page scroll", async ({page}) => {
  await open(page);
  await page.evaluate(() => { Element.prototype.requestFullscreen = undefined; });
  const fullscreen = figure(page).getByRole("button", {name: "Fullscreen", exact: true});
  await fullscreen.click();
  await expect(page.getByRole("dialog")).toHaveClass(/margo-viewer__dialog--fullscreen/);
  await page.keyboard.press("Escape");
  await expect(fullscreen).toBeFocused();
  await fullscreen.click();
  await figure(page).evaluate((el) => el.remove());
  await expect.poll(() => page.evaluate(() => document.documentElement.style.overflow)).toBe("");
  await expect(page.getByRole("dialog")).toHaveCount(0);
});

test("@image-viewer native fullscreen keeps alt text and caption and returns focus", async ({page}) => {
  await open(page);
  const fullscreen = figure(page).getByRole("button", {name: "Fullscreen", exact: true});
  await fullscreen.click();
  await expect.poll(() => page.evaluate(() => document.fullscreenElement?.classList.contains("margo-viewer__viewer"))).toBe(true);
  await expect(page.locator(":fullscreen h2")).toHaveText("Platform dashboard");
  await expect(page.locator(":fullscreen img")).toHaveAttribute("alt", "Service health details");
  await page.evaluate(() => document.exitFullscreen());
  await expect(page.getByRole("dialog")).toBeHidden();
  await expect(fullscreen).toBeFocused();
});

test("@image-viewer HTMX loads from a plain page, survives history and initializes once", async ({page}) => {
  await routeArtifacts(page);
  await page.goto(origin);
  await page.evaluate(() => { window.imageViewerNavigationMarker = 1; });
  await page.locator("#main-content").getByRole("link", {name: "Dashboard", exact: true}).click();
  await expect(page.locator(".margo-viewer__controls")).toHaveCount(2);
  expect(await page.evaluate(() => window.imageViewerNavigationMarker)).toBe(1);
  await page.evaluate(() => document.dispatchEvent(new CustomEvent("htmx:after:settle")));
  await expect(page.locator(".margo-viewer__controls")).toHaveCount(2);
  await figure(page).getByRole("button", {name: "Expand", exact: true}).click();
  await page.goBack();
  await expect(page.locator(".margo-viewer__controls")).toHaveCount(0);
  expect(await page.evaluate(() => document.documentElement.style.overflow)).toBe("");
  await page.goForward();
  await expect(page.locator(".margo-viewer__controls")).toHaveCount(2);
  await figure(page).getByRole("button", {name: "Zoom in", exact: true}).click();
  await expect(figure(page).locator("output")).toHaveText("125%");
});

test("@image-viewer images and captions remain available without JavaScript", async ({browser}) => {
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();
  await routeArtifacts(page);
  await page.goto(origin + "/inline.html");
  await expect(figure(page).locator("img")).toBeVisible();
  await expect(figure(page).locator("figcaption")).toHaveText("Platform dashboard");
  await expect(page.locator(".margo-viewer__controls")).toHaveCount(0);
  expect(await figure(page).locator("img").evaluate((el) => el.naturalWidth)).toBe(4800);
  await context.close();
});

test("@image-viewer touch pans within a zoomed image and outside gestures scroll normally", async ({browser}) => {
  const context = await browser.newContext({viewport: {width: 390, height: 844}, isMobile: true, hasTouch: true});
  const page = await context.newPage();
  await open(page);
  await figure(page).getByRole("button", {name: "Expand", exact: true}).tap();
  await figure(page).getByRole("button", {name: "Actual size", exact: true}).tap();
  await canvas(page).evaluate((el) => el.scrollTo(0, 0));
  await canvas(page).scrollIntoViewIfNeeded();
  const box = await canvas(page).boundingBox();
  const session = await context.newCDPSession(page);
  const swipe = async (x, y) => {
    const from = [{id: 1, x, y}];
    await session.send("Input.dispatchTouchEvent", {type: "touchStart", touchPoints: from});
    await moveTouches(page, session, from, [{id: 1, x: x - 60, y: y - 90}]);
    await session.send("Input.dispatchTouchEvent", {type: "touchEnd", touchPoints: []});
  };
  await swipe(box.x + box.width * 0.7, box.y + box.height * 0.8);
  await expect.poll(() => canvas(page).evaluate((el) => el.scrollLeft + el.scrollTop)).toBeGreaterThan(0);
  await page.getByRole("button", {name: "Close expanded view", exact: true}).tap();
  await expect(page.getByRole("dialog")).toBeHidden();
  await expect.poll(() => page.evaluate(() => document.documentElement.style.overflow)).toBe("");
  await page.evaluate(() => scrollTo(0, 0));
  const before = await page.evaluate(() => scrollY);
  await swipe(20, 250);
  await expect.poll(() => page.evaluate(() => scrollY)).toBeGreaterThan(before);
  await context.close();
});

test("@image-viewer expanded pinch shares zoom controls, preserves the focal point and hands off to pan", async ({browser}) => {
  const context = await browser.newContext({viewport: {width: 390, height: 844}, isMobile: true, hasTouch: true});
  const page = await context.newPage();
  await open(page);
  await figure(page).getByRole("button", {name: "Expand", exact: true}).tap();
  expect(await canvas(page).evaluate((el) => getComputedStyle(el).touchAction)).toBe("none");
  const pageScale = await page.evaluate(() => visualViewport.scale);
  const session = await context.newCDPSession(page);
  const image = canvas(page).locator("img");
  const fitted = await image.boundingBox();
  await pinch(page, session, {x: fitted.x + fitted.width / 2, y: fitted.y + fitted.height / 2}, 3);
  expect((await image.boundingBox()).width / fitted.width).toBeCloseTo(3, 1);
  await expect(figure(page).locator("output")).toHaveText("300%");
  await figure(page).getByRole("button", {name: "Zoom in", exact: true}).tap();
  await expect(figure(page).locator("output")).toHaveText("375%");
  expect(await page.evaluate(() => visualViewport.scale)).toBeCloseTo(pageScale, 4);

  await figure(page).getByRole("button", {name: "Actual size", exact: true}).tap();
  await canvas(page).evaluate((el) => el.scrollTo(1000, 1000));
  const box = await canvas(page).boundingBox();
  const center = {x: box.x + box.width / 2, y: box.y + box.height / 2};
  const before = await image.boundingBox();
  const point = {x: (center.x - before.x) / before.width, y: (center.y - before.y) / before.height};
  const offset = {x: -12, y: -20};
  const fingers = await pinch(page, session, center, 0.7, offset, false);
  const after = await image.boundingBox();
  expect(after.width / before.width).toBeCloseTo(0.7, 1);
  expect((center.x + offset.x - after.x) / after.width).toBeCloseTo(point.x, 2);
  expect((center.y + offset.y - after.y) / after.height).toBeCloseTo(point.y, 2);

  // End the second contact while keeping the first on the surface.
  await session.send("Input.dispatchTouchEvent", {type: "touchEnd", touchPoints: [fingers[1]]});
  const start = await canvas(page).evaluate((el) => [el.scrollLeft, el.scrollTop]);
  await moveTouches(page, session, [fingers[0]], [{...fingers[0], x: fingers[0].x - 35, y: fingers[0].y - 40}]);
  const panned = await canvas(page).evaluate((el) => [el.scrollLeft, el.scrollTop]);
  expect(panned[0] - start[0]).toBeCloseTo(35, 0);
  expect(panned[1] - start[1]).toBeCloseTo(40, 0);
  await session.send("Input.dispatchTouchEvent", {type: "touchCancel", touchPoints: []});
  const canceled = await canvas(page).evaluate((el) => [el.scrollLeft, el.scrollTop]);
  const fresh = [{id: 3, x: center.x, y: center.y}];
  await session.send("Input.dispatchTouchEvent", {type: "touchStart", touchPoints: fresh});
  await moveTouches(page, session, fresh, [{id: 3, x: center.x - 25, y: center.y - 25}]);
  // A fresh contact pans independently after cancellation; it has no stale anchor.
  await session.send("Input.dispatchTouchEvent", {type: "touchEnd", touchPoints: []});
  const restarted = await canvas(page).evaluate((el) => [el.scrollLeft, el.scrollTop]);
  expect(restarted[0] - canceled[0]).toBeCloseTo(25, 0);
  expect(restarted[1] - canceled[1]).toBeCloseTo(25, 0);
  expect(await page.evaluate(() => visualViewport.scale)).toBeCloseTo(pageScale, 4);
  await page.getByRole("button", {name: "Close expanded view", exact: true}).tap();
  await expect(figure(page).locator("output")).toHaveText("100%");
  expect(await canvas(page).evaluate((el) => getComputedStyle(el).touchAction)).toBe("auto");
  await context.close();
});

test("@image-viewer inline touch keeps native gestures and double taps do not expand", async ({browser}) => {
  const context = await browser.newContext({viewport: {width: 390, height: 844}, isMobile: true, hasTouch: true});
  const page = await context.newPage();
  await open(page);
  await canvas(page).evaluate((el) => {
    window.inlineTouchDefaults = [];
    for (const name of ["pointerdown", "pointermove"]) el.addEventListener(name, (event) => {
      if (event.pointerType === "touch") window.inlineTouchDefaults.push(event.defaultPrevented);
    });
  });
  await canvas(page).scrollIntoViewIfNeeded();
  const box = await canvas(page).boundingBox();
  const session = await context.newCDPSession(page);
  const center = {x: box.x + box.width / 2, y: box.y + box.height / 2};
  await page.touchscreen.tap(center.x, center.y);
  await page.touchscreen.tap(center.x, center.y);
  await expect(page.getByRole("dialog")).toBeHidden();
  await pinch(page, session, center, 2);
  // Desktop headless Chromium does not perform native page pinch even on a
  // plain control page. Check that the viewer leaves that gesture uncanceled;
  // native page scaling itself requires a physical mobile browser check.
  expect(await canvas(page).evaluate((el) => getComputedStyle(el).touchAction)).toBe("auto");
  const canceled = await page.evaluate(() => window.inlineTouchDefaults);
  expect(canceled.length).toBeGreaterThan(0);
  expect(canceled.every((value) => value === false)).toBe(true);
  await expect(figure(page).locator("output")).toHaveText("100%");
  await expect(page.getByRole("dialog")).toBeHidden();
  await context.close();
});

test("@image-viewer rapid control taps zoom only the image in inline and expanded views", async ({browser}) => {
  const context = await browser.newContext({viewport: {width: 390, height: 844}, isMobile: true, hasTouch: true});
  const page = await context.newPage();
  await open(page);
  const scale = await page.evaluate(() => visualViewport.scale);
  const zoomIn = figure(page).getByRole("button", {name: "Zoom in", exact: true});
  for (const expanded of [false, true]) {
    if (expanded) await figure(page).getByRole("button", {name: "Expand", exact: true}).tap();
    expect(await zoomIn.evaluate((el) => getComputedStyle(el).touchAction)).toBe("manipulation");
    await zoomIn.tap();
    await zoomIn.tap();
    await expect(figure(page).locator("output")).toHaveText("156%");
    expect(await page.evaluate(() => visualViewport.scale)).toBeCloseTo(scale, 4);
    if (!expanded) await expect(page.getByRole("dialog")).toBeHidden();
  }
  await context.close();
});

test("@image-viewer expanded and fullscreen fallback fit an already zoomed page", async ({browser}) => {
  const context = await browser.newContext({viewport: {width: 390, height: 844}, isMobile: true, hasTouch: true});
  const page = await context.newPage();
  await open(page);
  const session = await context.newCDPSession(page);
  await session.send("Emulation.setPageScaleFactor", {pageScaleFactor: 2});
  const expand = figure(page).getByRole("button", {name: "Expand", exact: true});
  await expand.focus();
  await page.keyboard.press("Enter");
  const fits = () => page.evaluate(() => {
    const bounds = document.querySelector("dialog[open]").getBoundingClientRect();
    const viewport = visualViewport;
    const canvas = document.querySelector("dialog[open] .margo-viewer__viewport").getBoundingClientRect();
    return bounds.left >= viewport.offsetLeft - 1 && bounds.top >= viewport.offsetTop - 1 &&
      bounds.right <= viewport.offsetLeft + viewport.width + 1 &&
      bounds.bottom <= viewport.offsetTop + viewport.height + 1 &&
      canvas.height >= Math.min(64, viewport.height / 4) && canvas.width > 0;
  });
  await expect.poll(fits).toBe(true);
  expect(await page.evaluate(() => visualViewport.scale)).toBeCloseTo(2, 3);
  await page.emulateMedia({media: "print"});
  await page.emulateMedia({media: "screen"});
  await expect(page.getByRole("dialog")).toBeVisible();
  await expect.poll(fits).toBe(true);
  await page.setViewportSize({width: 844, height: 390});
  await expect.poll(fits).toBe(true);
  await page.evaluate(() => { Element.prototype.requestFullscreen = undefined; });
  await figure(page).getByRole("button", {name: "Fullscreen", exact: true}).focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("dialog")).toHaveClass(/margo-viewer__dialog--fullscreen/);
  await expect.poll(fits).toBe(true);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toBeHidden();
  expect(await page.evaluate(() => visualViewport.scale)).toBeCloseTo(2, 3);
  await context.close();
});

for (const scenario of [{width: 320, height: 568, scale: 2}, {width: 390, height: 844, scale: 3}]) {
  test(`@image-viewer caption and controls remain reachable at ${scenario.width}px and ${scenario.scale * 100}% page zoom`, async ({browser}, testInfo) => {
    const context = await browser.newContext({viewport: {width: scenario.width, height: scenario.height}, isMobile: true, hasTouch: true});
    const page = await context.newPage();
    await open(page, "/long-caption.html");
    const session = await context.newCDPSession(page);
    await session.send("Emulation.setPageScaleFactor", {pageScaleFactor: scenario.scale});
    await figure(page).getByRole("button", {name: "Expand", exact: true}).focus();
    await page.keyboard.press("Enter");
    const dialog = page.getByRole("dialog");
    const header = dialog.getByRole("region", {name: "Image caption and controls"});
    await expect(header).toBeVisible();
    await expect(header).toHaveAttribute("tabindex", "0");
    await expect(dialog.locator("h2")).toHaveText(longCaption);
    const usefulImage = () => canvas(page).evaluate((el) => {
      const bounds = el.getBoundingClientRect();
      const viewport = visualViewport;
      const image = el.querySelector("img").getBoundingClientRect();
      return bounds.width > 0 && bounds.height >= Math.min(64, viewport.height / 4) &&
        bounds.top >= viewport.offsetTop - 1 && bounds.bottom <= viewport.offsetTop + viewport.height + 1 &&
        image.width > 0 && image.height > 0;
    });
    await expect.poll(usefulImage).toBe(true);

    // Keyboard focus must reveal each control inside the bounded scroll area.
    // toBeVisible alone would accept controls clipped by an ancestor.
    for (const label of ["Zoom out", "Zoom in", "Actual size", "Fit image", "Fullscreen"]) {
      const control = header.getByRole("button", {name: label, exact: true});
      await control.focus();
      await expect.poll(() => control.evaluate((el) => {
        const box = el.getBoundingClientRect();
        const clip = el.closest(".margo-viewer__header").getBoundingClientRect();
        const hit = document.elementFromPoint(box.left + box.width / 2, box.top + box.height / 2);
        return box.left >= clip.left - 1 && box.right <= clip.right + 1 &&
          box.top >= clip.top - 1 && box.bottom <= clip.bottom + 1 &&
          !!hit && (hit === el || el.contains(hit));
      })).toBe(true);
      await expect.poll(usefulImage).toBe(true);
    }
    await header.getByRole("button", {name: "Zoom in", exact: true}).focus();
    await page.keyboard.press("Enter");
    await expect(figure(page).locator("output")).toHaveText("125%");
    await header.focus();
    await header.evaluate((el) => el.scrollTo(0, el.scrollHeight));
    await expect.poll(() => header.locator("h2").evaluate((el) => {
      const text = el.getBoundingClientRect();
      const clip = el.parentElement.getBoundingClientRect();
      return Math.min(text.bottom, clip.bottom) > Math.max(text.top, clip.top);
    })).toBe(true);
    await expect.poll(usefulImage).toBe(true);
    await page.screenshot({path: testInfo.outputPath("compact-viewer.png")});
    const close = dialog.getByRole("button", {name: "Close expanded view", exact: true});
    await expect.poll(() => close.evaluate((el) => {
      const box = el.getBoundingClientRect();
      const viewport = visualViewport;
      const hit = document.elementFromPoint(box.left + box.width / 2, box.top + box.height / 2);
      return box.left >= viewport.offsetLeft - 1 && box.right <= viewport.offsetLeft + viewport.width + 1 &&
        box.top >= viewport.offsetTop - 1 && box.bottom <= viewport.offsetTop + viewport.height + 1 &&
        !!hit && (hit === el || el.contains(hit));
    })).toBe(true);
    await close.focus();
    await page.keyboard.press("Enter");
    await expect(dialog).toBeHidden();
    expect(await page.evaluate(() => visualViewport.scale)).toBeCloseTo(scenario.scale, 3);
    await context.close();
  });
}
