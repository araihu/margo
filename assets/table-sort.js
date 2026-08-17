(() => {
  "use strict";

  const collator = new Intl.Collator(undefined, {
    numeric: true,
    sensitivity: "base",
  });

  const compare = (left, right) => collator.compare(left.trim(), right.trim());

  const sortHeaderClass =
    "margo-table-sort-button p-4 cursor-pointer select-none transition-colors";
  const activeSortHeaderClasses = ["margo-table-sort-active"];

  const sortIconPath = (state) => {
    if (state === "ascending") {
      return "M10 17a.75.75 0 01-.75-.75V5.612L5.29 9.77a.75.75 0 01-1.08-1.04l5.25-5.5a.75.75 0 011.08 0l5.25 5.5a.75.75 0 11-1.08 1.04l-3.96-4.158V16.25A.75.75 0 0110 17z";
    }
    if (state === "descending") {
      return "M10 3a.75.75 0 01.75.75v10.638l3.96-4.158a.75.75 0 111.08 1.04l-5.25 5.5a.75.75 0 01-1.08 0l-5.25-5.5a.75.75 0 111.08-1.04l3.96 4.158V3.75A.75.75 0 0110 3z";
    }
    return "M10 3a.75.75 0 01.53.22l3.25 3.25a.75.75 0 01-1.06 1.06L10 4.81 7.28 7.53a.75.75 0 01-1.06-1.06l3.25-3.25A.75.75 0 0110 3zm-3.72 9.47a.75.75 0 011.06 0L10 15.19l2.72-2.72a.75.75 0 111.06 1.06l-3.25 3.25a.75.75 0 01-1.06 0l-3.25-3.25a.75.75 0 010-1.06z";
  };

  const renderHeader = (cell, state) => {
    const label = cell.dataset.margoSortLabel || cell.textContent.trim();
    const wrapper = document.createElement("div");
    wrapper.className = "flex items-center gap-1";
    const text = document.createElement("span");
    text.textContent = label;
    const icon = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    icon.setAttribute("aria-hidden", "true");
    icon.setAttribute("viewBox", "0 0 20 20");
    icon.setAttribute("fill", "currentColor");
    icon.setAttribute("class", "size-4");
    if (state === "source") icon.classList.add("opacity-40");
    const path = document.createElementNS("http://www.w3.org/2000/svg", "path");
    path.setAttribute("fill-rule", "evenodd");
    path.setAttribute("d", sortIconPath(state));
    icon.append(path);
    wrapper.append(text, icon);
    cell.replaceChildren(wrapper);
  };

  const applyHeaderState = (table, active) => {
    Array.from(table.tHead.rows[0].cells).forEach((header, column) => {
      const state = active.column === column ? active.state : "source";
      activeSortHeaderClasses.forEach((className) =>
        header.classList.toggle(className, state !== "source"),
      );
      header.removeAttribute("aria-sort");
      if (state !== "source") header.setAttribute("aria-sort", state);
      renderHeader(header, state);
    });
  };

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
    applyHeaderState(table, active);
  };

  const installHeader = (table, tbody, sourceRows, cell, column, active) => {
    cell.className = sortHeaderClass;
    cell.tabIndex = 0;
    cell.dataset.margoSortLabel = cell.textContent.trim();
    cell.setAttribute("aria-label", `Sort by ${cell.dataset.margoSortLabel}`);

    const activate = () => {
      const next =
        active.column !== column || active.state === "source"
          ? "ascending"
          : active.state === "ascending"
            ? "descending"
            : "source";
      active.state = next;
      active.column = next === "source" ? -1 : column;
      apply(table, tbody, sourceRows, active);
      cell.focus();
    };

    cell.addEventListener("click", activate);
    cell.addEventListener("keydown", (event) => {
      if (event.key !== "Enter" && event.key !== " ") return;
      event.preventDefault();
      activate();
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
      installHeader(table, tbody, sourceRows, cell, column, active),
    );
    applyHeaderState(table, active);

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
