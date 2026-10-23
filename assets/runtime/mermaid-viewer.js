  // This source is embedded inside the shared Mermaid executor's closure.
  const mermaidViewers = new WeakMap();

  function attachMermaidViewer(node, svg, position) {
    // Presentation slides have fixed geometry and their own keyboard controls.
    if (node.closest(".margo-deck")) return;
    const previous = mermaidViewers.get(node);
    if (previous) {
      previous.replace(svg, position);
      return;
    }
    const canvas = node.querySelector(".margo-mermaid__canvas");
    const box = svg.viewBox.baseVal;
    if (!canvas || !box || box.width <= 0 || box.height <= 0) return;
    let width = box.width;
    let height = box.height;
    let scale = 1;
    let fittedWidth = 0;
    let fittedHeight = 0;
    let drag = null;

    const stage = document.createElement("div");
    stage.className = "margo-mermaid__stage";
    canvas.replaceChildren(stage);
    stage.append(svg);
    canvas.classList.add("margo-mermaid__viewport");
    const controls = document.createElement("div");
    controls.className = "margo-mermaid__controls";
    controls.setAttribute("role", "group");
    controls.setAttribute("aria-label", "Diagram controls");
    // Associate each group with the same caption as its diagram.
    if (canvas.hasAttribute("aria-labelledby")) {
      controls.setAttribute("aria-describedby", canvas.getAttribute("aria-labelledby"));
    }
    const button = (label, action) => {
      const element = document.createElement("button");
      element.type = "button";
      element.textContent = label;
      element.addEventListener("click", action);
      controls.append(element);
      return element;
    };
    const zoomOut = button("Zoom out", () => zoom(scale / 1.25));
    const zoomIn = button("Zoom in", () => zoom(scale * 1.25));
    button("Fit diagram", () => zoom(1, true));
    const status = document.createElement("output");
    status.setAttribute("aria-live", "polite");
    status.setAttribute("aria-atomic", "true");
    status.setAttribute("aria-label", "Diagram zoom");
    controls.append(status);
    canvas.before(controls);

    const hint = document.createElement("p");
    hint.className = "margo-mermaid__hint";
    hint.textContent = "Use + / − to zoom, 0 to fit, and arrow keys or drag to pan. On touch screens, swipe to pan a zoomed diagram.";
    hint.id = canvas.getAttribute("aria-describedby") + "-controls";
    canvas.setAttribute("aria-describedby", (canvas.getAttribute("aria-describedby") || "") + " " + hint.id);
    canvas.after(hint);

    function paint() {
      stage.style.width = fittedWidth * scale + "px";
      stage.style.height = fittedHeight * scale + "px";
      canvas.style.height = fittedHeight + "px";
      canvas.classList.toggle("margo-mermaid__viewport--zoomed", scale > 1);
      status.value = Math.round(scale * 100) + "%";
      zoomOut.disabled = scale <= 0.25;
      zoomIn.disabled = scale >= 8;
    }

    function zoom(next, reset = false) {
      const viewport = canvas.getBoundingClientRect();
      const before = stage.getBoundingClientRect();
      const centerX = viewport.left + canvas.clientLeft + canvas.clientWidth / 2;
      const centerY = viewport.top + canvas.clientTop + canvas.clientHeight / 2;
      const x = before.width ? (centerX - before.left) / before.width : 0.5;
      const y = before.height ? (centerY - before.top) / before.height : 0.5;
      scale = Math.max(0.25, Math.min(8, next));
      paint();
      if (reset) {
        canvas.scrollTo(0, 0);
      } else {
        const after = stage.getBoundingClientRect();
        canvas.scrollBy(after.left + x * after.width - centerX, after.top + y * after.height - centerY);
      }
    }

    function fit() {
      // Read the content width independently of scrollbar appearance.
      const available = canvas.getBoundingClientRect().width;
      if (available <= 0) return;
      const ratio = Math.min(1, available / width, 560 / height);
      fittedWidth = width * ratio;
      fittedHeight = Math.ceil(height * ratio);
      paint();
    }

    canvas.addEventListener("keydown", (event) => {
      if (event.target !== canvas || event.ctrlKey || event.metaKey || event.altKey) return;
      if (event.key === "+" || event.key === "=") zoom(scale * 1.25);
      else if (event.key === "-" || event.key === "−") zoom(scale / 1.25);
      else if (event.key === "0" || event.key === "Home") zoom(1, true);
      else {
        const steps = {ArrowLeft: [-48, 0], ArrowRight: [48, 0], ArrowUp: [0, -48], ArrowDown: [0, 48]};
        const step = steps[event.key];
        if (!step || scale <= 1) return;
        canvas.scrollBy(...step);
      }
      event.preventDefault();
    });

    // Touch uses the browser's native overflow panning and scroll chaining.
    // Pointer capture is only needed for mouse/pen dragging.
    canvas.addEventListener("pointerdown", (event) => {
      if (event.pointerType === "touch" || event.button !== 0 || scale <= 1 ||
          event.target.closest("a, button")) return;
      canvas.focus({preventScroll: true});
      drag = {id: event.pointerId, x: event.clientX, y: event.clientY, left: canvas.scrollLeft, top: canvas.scrollTop};
      canvas.setPointerCapture(event.pointerId);
      canvas.classList.add("margo-mermaid__viewport--dragging");
      event.preventDefault();
    });
    canvas.addEventListener("pointermove", (event) => {
      if (!drag || drag.id !== event.pointerId) return;
      canvas.scrollTo(drag.left + drag.x - event.clientX, drag.top + drag.y - event.clientY);
    });
    const endDrag = () => {
      drag = null;
      canvas.classList.remove("margo-mermaid__viewport--dragging");
    };
    canvas.addEventListener("pointerup", endDrag);
    canvas.addEventListener("pointercancel", endDrag);
    canvas.addEventListener("lostpointercapture", endDrag);

    // Observe the figure's width so zoom never changes the observed geometry.
    let observedWidth = 0;
    if (typeof ResizeObserver === "function") {
      const observer = new ResizeObserver((entries) => {
        if (!node.isConnected) return;
        const next = entries[0].contentRect.width;
        if (Math.abs(next - observedWidth) < 0.5) return;
        observedWidth = next;
        fit();
      });
      observer.observe(node);
      // The observer and listeners belong to the figure, not to the document.
      // A removed HTMX page must not be retained by a global resize listener.
    }
    mermaidViewers.set(node, {
      position() {
        return {left: canvas.scrollLeft, top: canvas.scrollTop};
      },
      replace(next, position) {
        width = next.viewBox.baseVal.width;
        height = next.viewBox.baseVal.height;
        stage.replaceChildren(next);
        canvas.replaceChildren(stage);
        fit();
        if (position) canvas.scrollTo(position.left, position.top);
      }
    });
    fit();
  }
