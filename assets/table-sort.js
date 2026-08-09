(() => {
  "use strict";

  const collator = new Intl.Collator(undefined, {
    numeric: true,
    sensitivity: "base",
  });

  const compare = (left, right) => collator.compare(left.trim(), right.trim());

  const apply = (table, tbody, sourceRows, active) => {
    const ordered = sourceRows.slice().sort((left, right) => {
      const sourceOrder =
        Number(left.dataset.margoSourceIndex) -
        Number(right.dataset.margoSourceIndex);
      if (active.state === "source") return sourceOrder;

      const value = compare(
        left.cells[active.column]?.textContent || "",
        right.cells[active.column]?.textContent || "",
      );
      if (value === 0) return sourceOrder;
      return active.state === "descending" ? -value : value;
    });

    ordered.forEach((row) => tbody.append(row));
    Array.from(table.tHead.rows[0].cells).forEach((header) =>
      header.removeAttribute("aria-sort"),
    );
    if (active.state !== "source") {
      table.tHead.rows[0].cells[active.column].setAttribute(
        "aria-sort",
        active.state,
      );
    }
  };

  const installButton = (table, tbody, sourceRows, cell, column, active) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "margo-table-sort-button";
    button.textContent = cell.textContent.trim();
    cell.textContent = "";
    cell.append(button);

    button.addEventListener("click", () => {
      const next =
        active.column !== column || active.state === "source"
          ? "ascending"
          : active.state === "ascending"
            ? "descending"
            : "source";
      active.state = next;
      active.column = next === "source" ? -1 : column;
      apply(table, tbody, sourceRows, active);
      button.focus();
    });
  };

  const initialize = (root) => {
    if (root.dataset.margoTableSortReady === "true") return;
    const table = root.querySelector("table");
    const tbody = table && table.tBodies[0];
    if (!table || !tbody || !table.tHead || !table.tHead.rows[0]) return;

    const sourceRows = Array.from(tbody.rows);
    const active = { state: "source", column: -1 };
    sourceRows.forEach((row, index) => {
      row.dataset.margoSourceIndex = String(index);
    });
    Array.from(table.tHead.rows[0].cells).forEach((cell, column) =>
      installButton(table, tbody, sourceRows, cell, column, active),
    );

    let printState = { state: "source", column: -1 };
    addEventListener("beforeprint", () => {
      printState = { ...active };
      active.state = "source";
      active.column = -1;
      apply(table, tbody, sourceRows, active);
    });
    addEventListener("afterprint", () => {
      active.state = printState.state;
      active.column = printState.column;
      apply(table, tbody, sourceRows, active);
    });

    root.dataset.margoTableSortReady = "true";
  };

  const scan = () =>
    document
      .querySelectorAll('[data-margo-table-sort="natural"]')
      .forEach(initialize);

  document.addEventListener("DOMContentLoaded", scan, { once: true });
  document.addEventListener("htmx:afterSettle", scan);
  if (document.readyState !== "loading") scan();
})();
