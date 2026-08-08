import { expect, test } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import path from "node:path";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");

async function stylesheet(name) {
  return readFile(path.join(root, "assets", name), "utf8");
}

async function printPaginationScript() {
  const source = await readFile(path.join(root, "standalone.go"), "utf8");
  const match = source.match(/const standalonePrintPaginationScript = `([\s\S]*?)`/);
  if (!match) throw new Error("standalone print pagination script is missing");
  return match[1];
}

test("@pagination keeps hard-to-read document blocks together in print", async ({ page }) => {
  const [documentCSS, standaloneCSS] = await Promise.all([
    stylesheet("document.css"),
    stylesheet("standalone.css"),
  ]);
  await page.setContent(`<!doctype html>
    <html><head><style>${documentCSS}</style><style>${standaloneCSS}</style></head>
    <body><div class="goshtoso-document">
      <nav class="goshtoso-document__toc"><ol><li>TOC remains independently fragmentable</li></ol></nav>
      <article class="margo-document">
        <h2>Heading stays with the following block</h2>
        <ul><li>list item</li><li>another list item</li></ul>
        <blockquote>quote boundary</blockquote>
        <dl><dt>term</dt><dd>definition</dd></dl>
        <details open><summary>disclosure</summary><p>details content</p></details>
        <div data-table-client-sort="true"><table><thead><tr><th>Column</th></tr></thead><tbody><tr><td>value</td></tr></tbody></table></div>
        <div x-data><div class="codeblock"><pre><code>literal</code></pre></div></div>
        <figure class="margo-mermaid"><div class="margo-mermaid__canvas"></div><details open class="margo-mermaid__source"><summary>Mermaid source</summary><pre><code>flowchart LR
source --> target</code></pre></details></figure>
      </article>
    </div></body></html>`);
  await page.emulateMedia({ media: "print" });

  const result = await page.locator(".goshtoso-document").evaluate((documentRoot) => {
    const article = documentRoot.querySelector("article.margo-document");
    const protectedBlocks = [
      ...article.querySelectorAll(
        ":scope > :is(ul, ol, blockquote, dl, details, figure, table, img, pre), " +
          ':scope > [data-table-client-sort="true"], ' +
          ":scope > [data-code-block], " +
          ":scope > div:has(> .codeblock), " +
          ":scope > .margo-mermaid",
      ),
    ];
    const headings = [...article.querySelectorAll(":scope > :is(h1, h2, h3, h4, h5, h6)")];
    return {
      headings: headings.map((element) => getComputedStyle(element).breakAfter),
      protectedBlocks: protectedBlocks.map((element) => ({
        tag: element.tagName,
        className: element.className,
        breakInside: getComputedStyle(element).breakInside,
      })),
      tocBreakAfter: getComputedStyle(documentRoot.querySelector(".goshtoso-document__toc")).breakAfter,
      tocColumnCount: getComputedStyle(documentRoot.querySelector(".goshtoso-document__toc ol")).columnCount,
      tocBreakInside: getComputedStyle(documentRoot.querySelector(".goshtoso-document__toc")).breakInside,
    };
  });

  expect(result.headings).toEqual(["avoid-page"]);
  expect(result.protectedBlocks).toHaveLength(7);
  expect(result.protectedBlocks.every((block) => block.breakInside === "avoid-page")).toBe(true);
  expect(result.tocBreakAfter).toBe("page");
  expect(result.tocColumnCount).toBe("1");
  expect(result.tocBreakInside).toBe("auto");
});

test("@pagination keeps a short TOC in one column", async ({ page }) => {
  const [documentCSS, standaloneCSS, script] = await Promise.all([
    stylesheet("document.css"),
    stylesheet("standalone.css"),
    printPaginationScript(),
  ]);
  await page.setViewportSize({ width: 794, height: 1123 });
  await page.setContent(`<!doctype html>
    <html><head><style>${documentCSS}</style><style>${standaloneCSS}</style></head>
    <body><div class="goshtoso-document">
      <nav class="goshtoso-document__toc"><p class="goshtoso-document__toc-title">Contents</p><ol><li>One</li><li>Two</li></ol></nav>
    </div>${script}</body></html>`);
  await page.emulateMedia({ media: "print" });
  await page.evaluate(() => window.margoPreparePrintTOC());
  await expect(page.locator(".goshtoso-document__toc")).toHaveAttribute("data-margo-toc-columns", "1");
  await expect(page.locator(".goshtoso-document__toc ol")).toHaveCSS("column-count", "1");
});

test("@pagination keeps long table cells inside the printable content width", async ({ page }) => {
  const [documentCSS, standaloneCSS] = await Promise.all([
    stylesheet("document.css"),
    stylesheet("standalone.css"),
  ]);
  const tokenCSS = `:root {
    --color-surface: #ffffff;
    --color-surface-alt: #fafafa;
    --color-on-surface: #4b5563;
    --color-on-surface-strong: #1f2937;
    --color-outline: #d1d5db;
    --color-primary: #1d4ed8;
    --radius-radius: 0.25rem;
    --spacing: 0.25rem;
    --text-sm: 0.875rem;
    --document-page-background: var(--color-surface);
  }`;
  const longValue = "margo-table-cell-overflow-".repeat(12);
  await page.setViewportSize({ width: 794, height: 1123 });
  await page.setContent(`<!doctype html>
    <html><head><style>${tokenCSS}</style><style>${documentCSS}</style><style>${standaloneCSS}</style>
    <style>
      .margo-document { max-width: 520px; margin: 0; }
      .margo-document table { width: 100%; }
      .whitespace-nowrap { white-space: nowrap; }
    </style></head>
    <body><div class="goshtoso-document">
      <article class="margo-document">
        <h2>Tables</h2>
        <div data-table-client-sort="true">
          <div class="overflow-x-auto overflow-y-clip w-full rounded-radius border border-outline margo-table">
            <table class="w-full text-left text-sm text-on-surface">
              <thead><tr><th scope="col">Name</th><th scope="col">Evidence</th></tr></thead>
              <tbody><tr><td>stable</td><td class="whitespace-nowrap">${longValue}</td></tr></tbody>
            </table>
          </div>
        </div>
      </article>
    </div></body></html>`);
  await page.emulateMedia({ media: "print" });

  const result = await page.locator(".margo-document").evaluate((article) => {
    const wrapper = article.querySelector("[data-table-client-sort='true'] > div");
    const table = wrapper.querySelector("table");
    const articleRect = article.getBoundingClientRect();
    const wrapperRect = wrapper.getBoundingClientRect();
    const tableRect = table.getBoundingClientRect();
    return {
      articleLeft: articleRect.left,
      articleRight: articleRect.right,
      wrapperLeft: wrapperRect.left,
      wrapperRight: wrapperRect.right,
      tableLeft: tableRect.left,
      tableRight: tableRect.right,
      scrollWidth: wrapper.scrollWidth,
      clientWidth: wrapper.clientWidth,
    };
  });

  expect(result.wrapperLeft).toBeGreaterThanOrEqual(result.articleLeft - 0.5);
  expect(result.wrapperRight).toBeLessThanOrEqual(result.articleRight + 0.5);
  expect(result.tableLeft).toBeGreaterThanOrEqual(result.wrapperLeft - 0.5);
  expect(result.tableRight).toBeLessThanOrEqual(result.wrapperRight + 0.5);
  expect(result.scrollWidth).toBeLessThanOrEqual(result.clientWidth + 1);
});

test("@pagination expands Mermaid source only for print and restores screen state", async ({ page }) => {
  const script = await printPaginationScript();
  await page.setContent(`<!doctype html>
    <html><head></head><body><div class="goshtoso-document">
      <nav class="goshtoso-document__toc"><ol><li>One</li></ol></nav>
      <article class="margo-document"><figure class="margo-mermaid">
        <details class="margo-mermaid__source"><summary>Mermaid source</summary><pre><code>flowchart LR
source --> target</code></pre></details>
      </figure></article>
    </div>${script}</body></html>`);

  await expect(page.locator(".margo-mermaid__source")).not.toHaveAttribute("open", "");
  await page.evaluate(() => window.margoPreparePrintTOC());
  await expect(page.locator(".margo-mermaid__source")).toHaveJSProperty("open", true);
  await page.evaluate(() => window.margoRestorePrintState());
  await expect(page.locator(".margo-mermaid__source")).toHaveJSProperty("open", false);
});

test("@pagination moves a protected block that crosses a print page boundary", async ({ page }) => {
  const script = await printPaginationScript();
  await page.setViewportSize({ width: 794, height: 1123 });
  await page.setContent(`<!doctype html>
    <html><head><style>
      .margo-document { margin: 0; }
      .spacer { block-size: 900px; }
      .margo-document ul { block-size: 300px; margin: 0; padding: 0; }
    </style></head><body><div class="goshtoso-document">
      <nav class="goshtoso-document__toc"><ol><li>One</li></ol></nav>
      <article class="margo-document"><div class="spacer"></div><ul id="crossing"><li>protected list</li></ul></article>
    </div>${script}</body></html>`);
  await page.emulateMedia({ media: "print" });
  await page.evaluate(() => window.margoPreparePrintTOC());
  await expect(page.locator("#crossing")).toHaveAttribute("data-margo-print-break-before", "page");
  await expect(page.locator("#crossing")).toHaveCSS("break-before", "page");
  await page.evaluate(() => window.margoRestorePrintState());
  await expect(page.locator("#crossing")).not.toHaveAttribute("data-margo-print-break-before", "page");
});

test("@pagination keeps a heading with a protected block that moves pages", async ({ page }) => {
  const script = await printPaginationScript();
  await page.setViewportSize({ width: 794, height: 1123 });
  await page.setContent(`<!doctype html>
    <html><head><style>
      .margo-document { margin: 0; }
      .spacer { block-size: 900px; }
      .margo-document ol { block-size: 300px; margin: 0; padding: 0; }
    </style></head><body><div class="goshtoso-document">
      <article class="margo-document">
        <div class="spacer"></div>
        <h2 id="orphan-heading">Ordered, restarted, and mixed lists</h2>
        <ol id="ordered"><li>protected ordered item</li></ol>
      </article>
    </div>${script}</body></html>`);
  await page.emulateMedia({ media: "print" });
  await page.evaluate(() => window.margoPreparePrintTOC());
  await expect(page.locator("#orphan-heading")).toHaveAttribute("data-margo-print-break-before", "page");
  await expect(page.locator("#orphan-heading")).toHaveCSS("break-before", "page");
  await expect(page.locator("#ordered")).not.toHaveAttribute("data-margo-print-break-before", "page");
  const pages = await page.locator("#orphan-heading, #ordered").evaluateAll((elements) => {
    const pageHeight = Math.max(1, window.innerHeight);
    return elements.map((element) => Math.floor((element.getBoundingClientRect().top + window.scrollY) / pageHeight));
  });
  expect(pages[0]).toBe(pages[1]);
  await page.evaluate(() => window.margoRestorePrintState());
  await expect(page.locator("#orphan-heading")).not.toHaveAttribute("data-margo-print-break-before", "page");
  await expect(page.locator("#ordered")).not.toHaveAttribute("data-margo-print-break-before", "page");
});

test("@pagination lets an oversized table flow while keeping rows together", async ({ page }) => {
  const [documentCSS, standaloneCSS, script] = await Promise.all([
    stylesheet("document.css"),
    stylesheet("standalone.css"),
    printPaginationScript(),
  ]);
  await page.setViewportSize({ width: 794, height: 1123 });
  await page.setContent(`<!doctype html>
    <html><head><style>${documentCSS}</style><style>${standaloneCSS}</style><style>
      .margo-document { margin: 0; }
      #oversized { block-size: 1200px; }
      #oversized table { block-size: 1200px; }
      #oversized tr { block-size: 600px; }
      .overflow-x-auto { overflow-x: auto; }
      .overflow-y-clip { overflow-y: clip; }
    </style></head><body><div class="goshtoso-document">
      <article class="margo-document">
        <h3>Edge cases first</h3>
        <p>Lead context must not strand an almost empty page before a large table.</p>
        <div id="oversized" data-table-client-sort="true">
          <div class="overflow-x-auto overflow-y-clip">
            <table><tbody><tr><td>first row</td></tr><tr><td>second row</td></tr></tbody></table>
          </div>
        </div>
      </article>
    </div>${script}</body></html>`);
  await page.emulateMedia({ media: "print" });
  await page.evaluate(() => window.margoPreparePrintTOC());

  await expect(page.locator("#oversized")).toHaveAttribute("data-margo-print-oversized", "true");
  await expect(page.locator("#oversized")).toHaveCSS("break-inside", "auto");
  await expect(page.locator("#oversized > div")).toHaveCSS("overflow", "visible");
  await expect(page.locator("#oversized tr")).toHaveCount(2);
  const rowBreaks = await page.locator("#oversized tr").evaluateAll((rows) =>
    rows.map((row) => getComputedStyle(row).breakInside),
  );
  expect(rowBreaks).toEqual(["avoid-page", "avoid-page"]);

  await page.evaluate(() => window.margoRestorePrintState());
  await expect(page.locator("#oversized")).not.toHaveAttribute("data-margo-print-oversized", "true");
});

test("@pagination falls back to two columns only when a TOC is too tall", async ({ page }) => {
  const [documentCSS, standaloneCSS, script] = await Promise.all([
    stylesheet("document.css"),
    stylesheet("standalone.css"),
    printPaginationScript(),
  ]);
  const entries = Array.from({ length: 120 }, (_, index) => `<li>Long entry ${index + 1}</li>`).join("");
  await page.setViewportSize({ width: 794, height: 1123 });
  await page.setContent(`<!doctype html>
    <html><head><style>${documentCSS}</style><style>${standaloneCSS}</style></head>
    <body><div class="goshtoso-document">
      <nav class="goshtoso-document__toc"><p class="goshtoso-document__toc-title">Contents</p><ol>${entries}</ol></nav>
    </div>${script}</body></html>`);
  await page.emulateMedia({ media: "print" });
  await page.evaluate(() => window.margoPreparePrintTOC());
  await expect(page.locator(".goshtoso-document__toc")).toHaveAttribute("data-margo-toc-columns", "2");
  await expect(page.locator(".goshtoso-document__toc ol")).toHaveCSS("column-count", "auto");
  await expect(page.locator(".goshtoso-document__toc ol")).toHaveCSS("column-width", "192px");
});

test("@pagination keeps print margins, document, and chrome on one surface", async ({ page }) => {
  const [documentCSS, standaloneCSS] = await Promise.all([
    stylesheet("document.css"),
    stylesheet("standalone.css"),
  ]);
  const tokenCSS = `:root {
    --color-surface: #ffffff;
    --color-surface-alt: #fafafa;
    --color-on-surface: #4b5563;
    --color-on-surface-strong: #1f2937;
    --color-outline: #d1d5db;
    --color-surface-dark: #171717;
    --color-surface-dark-alt: #262626;
    --color-on-surface-dark: #d1d5db;
    --color-on-surface-dark-strong: #f5f5f5;
    --color-outline-dark: #525252;
    --document-page-background: var(--color-surface);
  }`;
  for (const mode of ["light", "dark"]) {
    await page.setContent(`<!doctype html>
      <html class="${mode === "dark" ? "dark" : ""}" data-color-mode="${mode}">
        <head><style>${tokenCSS}</style><style>${documentCSS}</style><style>${standaloneCSS}</style></head>
        <body><div class="goshtoso-document">
          <header class="goshtoso-document__header">Header</header>
          <article class="margo-document"><h1>Surface</h1><p>Print chrome contract.</p></article>
          <footer class="goshtoso-document__footer">Footer</footer>
        </div></body>
      </html>`);
    await page.emulateMedia({ media: "print" });
    const result = await page.evaluate(() => {
      const styles = (selector) => {
        const element = document.querySelector(selector);
        const computed = getComputedStyle(element);
        return {
          background: computed.backgroundColor,
          display: computed.display,
          pageBackground: computed.getPropertyValue("--margo-print-page-background").trim(),
          chromeBackground: computed.getPropertyValue("--margo-print-chrome-background").trim(),
          chromeOutline: computed.getPropertyValue("--margo-print-chrome-outline").trim(),
        };
      };
      return {
        html: styles("html"),
        body: styles("body"),
        document: styles(".goshtoso-document"),
        header: styles(".goshtoso-document__header"),
        footer: styles(".goshtoso-document__footer"),
      };
    });
    expect(result.html.background, mode).toBe(result.body.background);
    expect(result.body.background, mode).toBe(result.document.background);
    expect(result.document.pageBackground, mode).toBeTruthy();
    expect(result.document.chromeBackground, mode).toBe(result.document.pageBackground);
    expect(result.document.chromeOutline, mode).toBeTruthy();
    for (const chrome of [result.header, result.footer]) {
      expect(chrome.display, mode).toBe("flex");
      expect(chrome.background, mode).toBe(result.document.background);
      expect(chrome.pageBackground, mode).toBe(result.document.pageBackground);
      expect(chrome.chromeBackground, mode).toBe(result.document.pageBackground);
    }
  }
});

test("@shell keeps the TOC visually inside the page surface", async ({ page }) => {
  const [documentCSS, standaloneCSS] = await Promise.all([
    stylesheet("document.css"),
    stylesheet("standalone.css"),
  ]);
  const tokenCSS = `:root {
    --color-surface: #ffffff;
    --color-surface-alt: #fafafa;
    --color-on-surface: #4b5563;
    --color-on-surface-strong: #1f2937;
    --color-outline: #d1d5db;
    --color-surface-dark: #171717;
    --color-surface-dark-alt: #262626;
    --color-on-surface-dark: #d1d5db;
    --color-on-surface-dark-strong: #f5f5f5;
    --color-outline-dark: #525252;
    --document-page-background: var(--color-surface);
  }`;
  for (const mode of ["light", "dark"]) {
    await page.setContent(`<!doctype html>
      <html class="${mode === "dark" ? "dark" : ""}">
        <head><style>${tokenCSS}</style><style>${documentCSS}</style><style>${standaloneCSS}</style></head>
        <body><div class="goshtoso-document">
          <nav class="goshtoso-document__toc"><p class="goshtoso-document__toc-title">Contents</p><ol><li>One</li></ol></nav>
          <article class="margo-document"><h1>Surface</h1></article>
        </div></body>
      </html>`);
    const result = await page.evaluate(() => {
      const toc = document.querySelector(".goshtoso-document__toc");
      const documentRoot = document.querySelector(".goshtoso-document");
      const tocStyle = getComputedStyle(toc);
      return {
        tocBackground: tocStyle.backgroundColor,
        documentBackground: getComputedStyle(documentRoot).backgroundColor,
        borderWidths: [tocStyle.borderTopWidth, tocStyle.borderRightWidth, tocStyle.borderBottomWidth, tocStyle.borderLeftWidth],
        borderRadii: [tocStyle.borderTopLeftRadius, tocStyle.borderTopRightRadius, tocStyle.borderBottomRightRadius, tocStyle.borderBottomLeftRadius],
      };
    });
    expect(result.tocBackground, mode).toBe(result.documentBackground);
    expect(result.borderWidths, mode).toEqual(["0px", "0px", "0px", "0px"]);
    expect(result.borderRadii, mode).toEqual(["0px", "0px", "0px", "0px"]);
  }
});

test("@shell uses dark page tokens for frame, chrome, and stamps", async ({ page }) => {
  const [documentCSS, standaloneCSS] = await Promise.all([
    stylesheet("document.css"),
    stylesheet("standalone.css"),
  ]);
  const tokenCSS = `:root {
    --color-surface: #ffffff;
    --color-surface-alt: #fafafa;
    --color-on-surface: #4b5563;
    --color-on-surface-strong: #1f2937;
    --color-outline: #d1d5db;
    --color-surface-dark: #171717;
    --color-surface-dark-alt: #262626;
    --color-on-surface-dark: #d1d5db;
    --color-on-surface-dark-strong: #f5f5f5;
    --color-outline-dark: #525252;
    --document-page-background: var(--color-surface);
  }`;
  await page.setContent(`<!doctype html>
      <html class="dark" data-color-mode="dark">
      <head><style>${tokenCSS}</style><style>${documentCSS}</style><style>${standaloneCSS}</style></head>
      <body><div class="goshtoso-document">
        <header class="goshtoso-document__header">Header</header>
        <aside class="goshtoso-document__stamps"><span class="goshtoso-document__stamp">Dark</span></aside>
        <article class="margo-document"><h1>Surface</h1></article>
        <footer class="goshtoso-document__footer">Footer</footer>
      </div></body>
      </html>`);
  await page.emulateMedia({ media: "print" });
  const result = await page.evaluate(() => {
    const style = (selector) => {
      const computed = getComputedStyle(document.querySelector(selector));
      return {
        background: computed.backgroundColor,
        color: computed.color,
        display: computed.display,
        borderBlockStart: computed.borderBlockStartColor,
        borderBlockEnd: computed.borderBlockEndColor,
      };
    };
    return {
      htmlBackground: getComputedStyle(document.documentElement).backgroundColor,
      bodyBackground: getComputedStyle(document.body).backgroundColor,
      documentBackground: getComputedStyle(document.querySelector(".goshtoso-document")).backgroundColor,
      header: style(".goshtoso-document__header"),
      footer: style(".goshtoso-document__footer"),
      stamp: style(".goshtoso-document__stamp"),
    };
  });
  expect(result.htmlBackground).toBe("rgb(23, 23, 23)");
  expect(result.bodyBackground).toBe(result.documentBackground);
  expect(result.documentBackground).toBe("rgb(23, 23, 23)");
  for (const chrome of [result.header, result.footer]) {
    expect(chrome.display).toBe("flex");
    expect(chrome.background).toBe("rgb(23, 23, 23)");
    expect(chrome.color).toBe("rgb(209, 213, 219)");
    expect(chrome.borderBlockStart).toBe("rgb(82, 82, 82)");
    expect(chrome.borderBlockEnd).toBe("rgb(82, 82, 82)");
  }
  expect(result.stamp.color).toBe("rgb(209, 213, 219)");
  expect(result.stamp.background).toBe("rgb(38, 38, 38)");
  expect(result.stamp.borderBlockStart).toBe("rgb(82, 82, 82)");
  expect(result.stamp.borderBlockEnd).toBe("rgb(82, 82, 82)");
});
