  // This source is embedded inside the shared Mermaid executor's closure.
  const mermaidViewers = new WeakMap();
  const expandedMermaidViews = new Set();
  let mermaidPrintViews = [];
  function prepareMermaidPrint() {
    if (mermaidPrintViews.length) return;
    for (const suspend of expandedMermaidViews) mermaidPrintViews.push(suspend());
  }
  function restoreMermaidPrint() {
    const views = mermaidPrintViews;
    mermaidPrintViews = [];
    for (const resume of views) resume();
  }
  window.addEventListener("beforeprint", prepareMermaidPrint);
  window.addEventListener("afterprint", restoreMermaidPrint);
  window.matchMedia("print").addEventListener("change", (event) => {
    if (event.matches) prepareMermaidPrint();
    else restoreMermaidPrint();
  });

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
    const expand = button("Expand", () => openExpanded(expand));
    expand.setAttribute("aria-haspopup", "dialog");
    expand.setAttribute("aria-expanded", "false");
    const fullscreen = button("Fullscreen", toggleFullscreen);
    fullscreen.setAttribute("aria-pressed", "false");
    const close = button("Close expanded view", () => closeExpanded());
    close.hidden = true;
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

    // Like chart controls, move the existing content; never clone the SVG or
    // initialize another renderer. A native dialog also works without Alpine.
    const viewer = document.createElement("div");
    viewer.className = "margo-mermaid__viewer";
    controls.before(viewer);
    viewer.append(controls, canvas, hint);
    const source = node.querySelector(".margo-mermaid__accessible-source");
    if (source) viewer.append(source);
    const dialog = document.createElement("dialog");
    dialog.className = "margo-mermaid__dialog";
    const title = document.createElement("h2");
    title.className = "margo-mermaid__dialog-title";
    title.id = hint.id + "-title";
    title.textContent = node.querySelector("figcaption")?.textContent || "Mermaid diagram";
    dialog.setAttribute("aria-labelledby", title.id);
    dialog.append(title);
    viewer.after(dialog);
    let inlineView;
    let returnFocus;
    let fullscreenOnly = false;
    let fallbackFullscreen = false;
    let pageOverflow;
    const captionID = canvas.getAttribute("aria-labelledby");

    function openExpanded(trigger, full = false) {
      if (dialog.open || !node.isConnected) return;
      inlineView = {scale, left: canvas.scrollLeft, top: canvas.scrollTop};
      returnFocus = trigger;
      fullscreenOnly = full;
      pageOverflow = document.documentElement.style.overflow;
      document.documentElement.style.overflow = "hidden";
      dialog.append(viewer);
      dialog.showModal();
      expandedMermaidViews.add(suspendForPrint);
      expand.hidden = true;
      expand.setAttribute("aria-expanded", "true");
      close.hidden = false;
      canvas.setAttribute("aria-labelledby", title.id);
      scale = 1;
      fit();
      canvas.scrollTo(0, 0);
      close.focus({preventScroll: true});
    }

    function closeExpanded(restoreFocus = true) {
      if (!inlineView) return;
      const saved = inlineView;
      inlineView = null;
      expandedMermaidViews.delete(suspendForPrint);
      if (document.fullscreenElement === viewer) {
        document.exitFullscreen().catch(() => {});
      }
      if (dialog.open) dialog.close();
      dialog.before(viewer);
      document.documentElement.style.overflow = pageOverflow;
      dialog.classList.remove("margo-mermaid__dialog--fullscreen");
      fallbackFullscreen = false;
      fullscreenOnly = false;
      expand.hidden = false;
      expand.setAttribute("aria-expanded", "false");
      close.hidden = true;
      canvas.setAttribute("aria-labelledby", captionID);
      updateFullscreen();
      scale = saved.scale;
      fit();
      canvas.scrollTo(saved.left, saved.top);
      if (restoreFocus && returnFocus?.isConnected) returnFocus.focus({preventScroll: true});
    }

    function suspendForPrint() {
      const saved = {
        scale, left: canvas.scrollLeft, top: canvas.scrollTop,
        trigger: returnFocus, fullscreenOnly,
        full: document.fullscreenElement === viewer || fallbackFullscreen
      };
      closeExpanded(false);
      return () => {
        if (!node.isConnected) return;
        openExpanded(saved.trigger, saved.fullscreenOnly);
        // Restoring native fullscreen would require a fresh user gesture.
        if (saved.full) enterFallback();
        scale = saved.scale;
        fit();
        canvas.scrollTo(saved.left, saved.top);
      };
    }

    function updateFullscreen() {
      const active = document.fullscreenElement === viewer || fallbackFullscreen;
      fullscreen.textContent = active ? "Exit fullscreen" : "Fullscreen";
      fullscreen.setAttribute("aria-pressed", String(active));
    }

    function enterFallback() {
      if (!dialog.open || !node.isConnected) return;
      fallbackFullscreen = true;
      dialog.classList.add("margo-mermaid__dialog--fullscreen");
      updateFullscreen();
      fit();
    }

    function toggleFullscreen() {
      if (document.fullscreenElement === viewer) {
        document.exitFullscreen().catch(() => {});
      } else if (fallbackFullscreen) {
        fallbackFullscreen = false;
        dialog.classList.remove("margo-mermaid__dialog--fullscreen");
        if (fullscreenOnly) closeExpanded();
        else { updateFullscreen(); fit(); }
      } else {
        if (!dialog.open) openExpanded(fullscreen, true);
        // Request fullscreen on the content: dialog elements cannot themselves
        // enter native fullscreen. All actions remain inside the full view.
        if (typeof viewer.requestFullscreen === "function") {
          viewer.requestFullscreen().catch(enterFallback);
        } else enterFallback();
      }
    }

    viewer.addEventListener("fullscreenchange", () => {
      updateFullscreen();
      if (!inlineView) return;
      if (document.fullscreenElement !== viewer && fullscreenOnly && !fallbackFullscreen) closeExpanded();
      else { fit(); fullscreen.focus({preventScroll: true}); }
    });
    dialog.addEventListener("cancel", (event) => {
      event.preventDefault();
      closeExpanded();
    });
    dialog.addEventListener("keydown", (event) => {
      if (event.key !== "Tab") return;
      const stops = [...viewer.querySelectorAll('button:not(:disabled), a[href], [tabindex]:not([tabindex="-1"])')]
        .filter((element) => element.getClientRects().length > 0);
      const first = stops[0];
      const last = stops[stops.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last?.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first?.focus();
      }
    });
    dialog.addEventListener("close", () => {
      if (!dialog.open) closeExpanded();
    });
    dialog.addEventListener("click", (event) => {
      if (event.target !== dialog) return;
      const rect = dialog.getBoundingClientRect();
      if (event.clientX < rect.left || event.clientX > rect.right ||
          event.clientY < rect.top || event.clientY > rect.bottom) closeExpanded();
    });

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
      const maxHeight = dialog.open ? canvas.clientHeight : 560;
      const ratio = Math.min(1, available / width, Math.max(1, maxHeight) / height);
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

    // Track the moved content so modal/fullscreen resizes refit the same SVG.
    let observedWidth = 0;
    let observedHeight = 0;
    let resizeObserver;
    if (typeof ResizeObserver === "function") {
      resizeObserver = new ResizeObserver((entries) => {
        if (!node.isConnected) {
          closeExpanded(false);
          resizeObserver.disconnect();
          return;
        }
        const next = entries[0].contentRect.width;
        const nextHeight = entries[0].contentRect.height;
        if (Math.abs(next - observedWidth) < 0.5 &&
            (!dialog.open || Math.abs(nextHeight - observedHeight) < 0.5)) return;
        observedWidth = next;
        observedHeight = nextHeight;
        fit();
      });
      resizeObserver.observe(viewer);
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
        if (resizeObserver) resizeObserver.observe(viewer);
        fit();
        if (position) canvas.scrollTo(position.left, position.top);
      }
    });
    fit();
  }
