import { expect, test } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import path from "node:path";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");

async function stylesheet(name) {
  return readFile(path.join(root, "assets", name), "utf8");
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
      tocBreakInside: getComputedStyle(documentRoot.querySelector(".goshtoso-document__toc")).breakInside,
    };
  });

  expect(result.headings).toEqual(["avoid-page"]);
  expect(result.protectedBlocks).toHaveLength(7);
  expect(result.protectedBlocks.every((block) => block.breakInside === "avoid-page")).toBe(true);
  expect(result.tocBreakInside).toBe("auto");
});
