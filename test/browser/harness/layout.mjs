export async function auditDocumentLayout(page) {
  return page.evaluate(() => {
    const selectorFor = (element) => {
      if (element.id) return `${element.tagName.toLowerCase()}#${element.id}`;
      const classes = [...element.classList].slice(0, 3).map((name) => `.${name}`).join("");
      return `${element.tagName.toLowerCase()}${classes}`;
    };
    const visible = (element) => {
      const style = getComputedStyle(element);
      return style.display !== "none" && style.visibility !== "hidden" &&
        Number.parseFloat(style.opacity) !== 0 && element.getClientRects().length > 0;
    };
    const tolerance = 2;
    const failures = [];
    const root = document.querySelector(".goshtoso-document");
    if (!root) {
      return { checked: 0, failures: [{ rule: "document.root", selector: "body", detail: "missing .goshtoso-document" }] };
    }

    const rootRect = root.getBoundingClientRect();
    let checked = 0;
    const fail = (rule, element, detail) => failures.push({ rule, selector: selectorFor(element), detail });
    const inspect = (element, rule) => {
      if (!visible(element)) return;
      checked += 1;
      const rect = element.getBoundingClientRect();
      if (rect.width <= 0 || rect.height <= 0) {
        fail(`${rule}.empty`, element, `rect=${rect.width}x${rect.height}`);
        return;
      }
      if (rect.left < rootRect.left - tolerance || rect.right > rootRect.right + tolerance) {
        fail(`${rule}.horizontal_bounds`, element, `left=${rect.left} right=${rect.right} root=${rootRect.left}..${rootRect.right}`);
      }
      const style = getComputedStyle(element);
      const clippedX = ["hidden", "clip"].includes(style.overflowX) && element.scrollWidth > element.clientWidth + tolerance;
      const clippedY = ["hidden", "clip"].includes(style.overflowY) && element.scrollHeight > element.clientHeight + tolerance;
      if (clippedX || clippedY) {
        fail(`${rule}.clipping`, element, `overflow=${style.overflowX}/${style.overflowY} scroll=${element.scrollWidth}x${element.scrollHeight} client=${element.clientWidth}x${element.clientHeight}`);
      }
    };

    for (const selector of [
      ".goshtoso-document__toc",
      ".goshtoso-document > .margo-document figure",
      ".goshtoso-document > .margo-document blockquote",
      ".goshtoso-document > .margo-document pre",
      ".goshtoso-document > .margo-document [data-code-block]",
      ".goshtoso-document > .margo-document [data-table-client-sort=\"true\"]",
      ".goshtoso-document > .margo-document .margo-mermaid",
    ]) {
      for (const element of root.querySelectorAll(selector)) inspect(element, "block");
    }

    for (const table of root.querySelectorAll("table")) {
      if (!visible(table)) continue;
      inspect(table, "table");
      const tableRect = table.getBoundingClientRect();
      for (const row of table.querySelectorAll("tr")) {
        if (!visible(row)) continue;
        const rowRect = row.getBoundingClientRect();
        if (rowRect.top < tableRect.top - tolerance || rowRect.bottom > tableRect.bottom + tolerance) {
          fail("table.row_bounds", row, `row=${rowRect.top}..${rowRect.bottom} table=${tableRect.top}..${tableRect.bottom}`);
        }
        if (rowRect.height <= 0) fail("table.row_empty", row, `height=${rowRect.height}`);
      }
      for (const cell of table.querySelectorAll("th, td")) {
        if (!visible(cell)) continue;
        const cellRect = cell.getBoundingClientRect();
        if (cellRect.left < tableRect.left - tolerance || cellRect.right > tableRect.right + tolerance) {
          fail("table.cell_bounds", cell, `cell=${cellRect.left}..${cellRect.right} table=${tableRect.left}..${tableRect.right}`);
        }
      }
    }

    const documentElement = document.documentElement;
    if (documentElement.scrollWidth > window.innerWidth + tolerance) {
      fail("document.horizontal_overflow", documentElement, `scrollWidth=${documentElement.scrollWidth} viewport=${window.innerWidth}`);
    }
    return { checked, failures };
  });
}
