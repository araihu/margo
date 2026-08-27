import { expect, test } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import path from "node:path";

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, "../../..");

async function loadDeck(page) {
  const [fixture, deckCSS, documentCSS, deckJavaScript] = await Promise.all([
    readFile(path.join(root, "test/browser/fixtures/deck-screen.html"), "utf8"),
    readFile(path.join(root, "deck/assets/deck.css"), "utf8"),
    readFile(path.join(root, "assets/document.css"), "utf8"),
    readFile(path.join(root, "deck/assets/deck.js"), "utf8"),
  ]);
  await page.setViewportSize({width: 390, height: 900});
  await page.setContent(fixture);
  await page.addStyleTag({content: deckCSS});
  await page.addStyleTag({content: documentCSS});
  await page.addScriptTag({content: deckJavaScript});
  await page.keyboard.press("ArrowRight");
  await page.waitForTimeout(50);
}

async function screenState(page) {
  return page.evaluate(() => {
    const slide = document.querySelector("#sidebar");
    const cue = slide.querySelector(".margo-mermaid__overflow-cue");
    return {
      validation: globalThis.margoValidateDeckScreen(),
      slide: {clientHeight: slide.clientHeight, scrollHeight: slide.scrollHeight},
      cue: getComputedStyle(cue).display,
      diagnostic: Boolean(document.querySelector("[data-margo-screen-diagnostic]")),
    };
  });
}

test("narrow sidebar Mermaid overflow cue does not fail a contained slide", async ({page}) => {
  await loadDeck(page);
  const state = await screenState(page);

  // The intrinsic grid track is taller than the fixed canvas because the
  // narrow-screen cue is visible, but its rendered boxes remain contained.
  expect(state.slide.scrollHeight).toBeGreaterThan(state.slide.clientHeight);
  expect(state.cue).toBe("block");
  expect(state.validation).toBe(true);
  expect(state.diagnostic).toBe(false);
});

test("screen validator tolerates only the real layout track bleed", async ({page}) => {
  await loadDeck(page);
  const state = await page.evaluate(() => {
    const slide = document.querySelector("#sidebar");
    const layout = slide.querySelector(".margo-layout");
    // The real 15-slide deck's narrow sidebar track ends 5.78px below the
    // clipped slide edge. Reproduce that intrinsic layout bleed without
    // moving its visible content outside the canvas.
    layout.style.transform = "translateY(65px)";
    const slideRect = slide.getBoundingClientRect();
    const layoutRect = layout.getBoundingClientRect();
    return {
      validation: globalThis.margoValidateDeckScreen(),
      bleed: layoutRect.bottom - slideRect.bottom,
      layoutBottom: layoutRect.bottom,
      slideBottom: slideRect.bottom,
      oversizedValidation: (() => {
        layout.style.transform = "translateY(100px)";
        return globalThis.margoValidateDeckScreen();
      })(),
    };
  });

  expect(state.bleed).toBeGreaterThan(5);
  expect(state.bleed).toBeLessThan(8);
  expect(state.layoutBottom).toBeGreaterThan(state.slideBottom);
  expect(state.validation).toBe(true);
  expect(state.oversizedValidation).toBe(false);
});

test("screen validator still rejects a visible descendant outside the slide", async ({page}) => {
  await loadDeck(page);
  await page.evaluate(() => {
    const overflow = document.createElement("div");
    overflow.id = "real-overflow";
    overflow.textContent = "Visible overflow";
    overflow.style.cssText = "position:absolute;inset-block-start:700px;inset-inline-start:0;width:100px;height:50px;background:red";
    // Keep the overflow in the non-scrolling layout wrapper. The validator
    // must not let the renderer-owned layout exception hide it.
    document.querySelector("#sidebar .margo-layout").append(overflow);
  });

  const state = await screenState(page);
  expect(state.validation).toBe(false);
  expect(state.diagnostic).toBe(true);
});
