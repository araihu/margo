  // This source is embedded inside the shared Mermaid executor's closure.
  /* Arrow icons follow Goshtoso Charts' Heroicons controls.
   * MIT License — Copyright (c) Tailwind Labs, Inc.
   * Permission is hereby granted, free of charge, to any person obtaining a copy
   * of this software and associated documentation files (the "Software"), to deal
   * in the Software without restriction, including without limitation the rights
   * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
   * copies of the Software, and to permit persons to whom the Software is
   * furnished to do so, subject to the following conditions:
   * The above copyright notice and this permission notice shall be included in all
   * copies or substantial portions of the Software.
   * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
   * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
   * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
   * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
   * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
   * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
   * SOFTWARE.
   */
  const mermaidControlIcons = Object.freeze({
    plus: "M12 5v14M5 12h14",
    minus: "M5 12h14",
    fit: "M8 4H4v4m12-4h4v4M4 16v4h4m12-4v4h-4M9 12h6m-3-3v6",
    expand: "M14 4h6v6m0-6-9 9M9 4H6a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-3",
    fullscreen: "M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15",
    collapse: "M9 9V4.5M9 9H4.5M9 9 3.75 3.75M9 15v4.5M9 15H4.5M9 15l-5.25 5.25M15 9h4.5M15 9V4.5M15 9l5.25-5.25M15 15h4.5M15 15v4.5m0-4.5 5.25 5.25",
    close: "m6 6 12 12M6 18 18 6",
    info: "M12 11v6m0-10h.01M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0"
  });
  function setMermaidControlIcon(button, icon, label) {
    button.setAttribute("aria-label", label);
    const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    svg.setAttribute("viewBox", "0 0 24 24");
    svg.setAttribute("aria-hidden", "true");
    svg.setAttribute("focusable", "false");
    svg.setAttribute("fill", "none");
    svg.setAttribute("stroke", "currentColor");
    svg.setAttribute("stroke-width", "1.75");
    svg.setAttribute("stroke-linecap", "round");
    svg.setAttribute("stroke-linejoin", "round");
    const path = document.createElementNS(svg.namespaceURI, "path");
    path.setAttribute("d", mermaidControlIcons[icon]);
    svg.append(path);
    button.replaceChildren(svg);
  }
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
    const button = (label, icon, action) => {
      const element = document.createElement("button");
      element.type = "button";
      element.className = "margo-mermaid__icon-button";
      setMermaidControlIcon(element, icon, label);
      element.addEventListener("click", action);
      controls.append(element);
      return element;
    };
    const zoomIn = button("Zoom in", "plus", () => zoom(scale * 1.25));
    const zoomOut = button("Zoom out", "minus", () => zoom(scale / 1.25));
    const fitButton = button("Fit diagram", "fit", () => zoom(1, true));
    const expand = button("Expand", "expand", () => openExpanded(expand));
    expand.setAttribute("aria-haspopup", "dialog");
    expand.setAttribute("aria-expanded", "false");
    const fullscreen = button("Fullscreen", "fullscreen", toggleFullscreen);
    fullscreen.setAttribute("aria-pressed", "false");
    const close = button("Close expanded view", "close", () => closeExpanded());
    close.classList.add("margo-mermaid__close");
    close.hidden = true;
    const status = document.createElement("output");
    status.setAttribute("aria-live", "polite");
    status.setAttribute("aria-atomic", "true");
    status.setAttribute("aria-label", "Diagram zoom");
    const zoomControls = document.createElement("div");
    zoomControls.className = "margo-mermaid__zoom-controls";
    zoomControls.setAttribute("role", "group");
    zoomControls.setAttribute("aria-label", "Diagram zoom controls");
    zoomControls.append(zoomOut, status, zoomIn);
    const viewControls = document.createElement("div");
    viewControls.className = "margo-mermaid__view-controls";
    viewControls.setAttribute("role", "group");
    viewControls.setAttribute("aria-label", "Diagram view controls");
    viewControls.append(fitButton, expand, fullscreen);
    controls.replaceChildren(zoomControls, viewControls);
    canvas.before(controls);

    const controlsID = canvas.getAttribute("aria-describedby") + "-controls";
    const helpRoot = node.querySelector("template[data-margo-mermaid-help]")?.content.firstElementChild?.cloneNode(true);
    const helpButton = helpRoot?.querySelector("button");
    const helpPanel = helpRoot?.querySelector('[role="tooltip"]');
    const helpTrigger = helpRoot?.querySelector("[data-tooltip-content-id]");
    let helpOpen = false;
    let helpInitialized = false;
    let helpListeners;
    if (helpRoot) {
      // Adapt the rendered Goshtoso Help to the shared standalone runtime.
      // Template content stays inert until prepared, avoiding a second Alpine
      // controller when a configured site also owns Goshtoso interactions.
      for (const element of [helpRoot, ...helpRoot.querySelectorAll("*")]) {
        for (const attribute of [...element.attributes]) {
          if (attribute.name.startsWith("x-")) element.removeAttribute(attribute.name);
        }
      }
      helpRoot.classList.add("margo-mermaid__help");
      helpRoot.dataset.tooltipPosition = "bottom";
      helpButton.className = "margo-mermaid__icon-button";
      setMermaidControlIcon(helpButton, "info", "Diagram help");
      helpButton.setAttribute("aria-expanded", "false");
      helpButton.setAttribute("aria-controls", helpPanel.id);
      helpButton.setAttribute("aria-describedby", helpPanel.id);
      helpPanel.hidden = true;
      helpButton.addEventListener("click", () => setHelpOpen(!helpOpen));
      viewControls.append(helpRoot);
    }

    function setHelpOpen(open) {
      if (!helpRoot) return;
      if (!helpInitialized) {
        window.goshtosoInitTooltipTrigger(helpTrigger, false);
        window.goshtosoInitTooltipPortal(helpRoot);
        helpInitialized = true;
      }
      helpOpen = open;
      helpListeners?.abort();
      window.goshtosoSetTooltipExpanded(helpTrigger, open);
      helpPanel.hidden = !open;
      helpPanel.style.opacity = open ? "1" : "";
      window.goshtosoSetTooltipPortal(helpRoot, open);
      if (!open) return;
      helpListeners = new AbortController();
      const options = {capture: true, signal: helpListeners.signal};
      document.addEventListener("pointerdown", (event) => {
        if (!helpRoot.contains(event.target)) setHelpOpen(false);
      }, options);
      document.addEventListener("keydown", (event) => {
        if (event.key !== "Escape") return;
        event.preventDefault();
        event.stopPropagation();
        setHelpOpen(false);
        helpButton.focus({preventScroll: true});
      }, options);
      window.addEventListener("beforeprint", () => setHelpOpen(false), {signal: helpListeners.signal});
      window.matchMedia("print").addEventListener("change", (event) => {
        if (event.matches) setHelpOpen(false);
      }, {signal: helpListeners.signal});
    }

    // Like chart controls, move the existing content; never clone the SVG or
    // initialize another renderer. A native dialog also works without Alpine.
    const viewer = document.createElement("div");
    viewer.className = "margo-mermaid__viewer";
    controls.before(viewer);
    viewer.append(controls, canvas, close);
    const source = node.querySelector(".margo-mermaid__accessible-source");
    if (source) viewer.append(source);
    const dialog = document.createElement("dialog");
    dialog.className = "margo-mermaid__dialog";
    const title = document.createElement("h2");
    title.className = "margo-mermaid__dialog-title";
    title.id = controlsID + "-title";
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
      setHelpOpen(false);
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
      setHelpOpen(false);
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
      setMermaidControlIcon(fullscreen, active ? "collapse" : "fullscreen", active ? "Exit fullscreen" : "Fullscreen");
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
      setHelpOpen(false);
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
      if (helpOpen) setHelpOpen(false);
      else closeExpanded();
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

    function zoom(next, reset = false, anchor) {
      const viewport = canvas.getBoundingClientRect();
      const before = stage.getBoundingClientRect();
      const centerX = anchor?.clientX ?? viewport.left + canvas.clientLeft + canvas.clientWidth / 2;
      const centerY = anchor?.clientY ?? viewport.top + canvas.clientTop + canvas.clientHeight / 2;
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

    canvas.addEventListener("wheel", (event) => {
      if (!event.ctrlKey || event.altKey || event.metaKey || event.deltaY === 0) return;
      event.preventDefault();
      const unit = event.deltaMode === 1 ? 16 : event.deltaMode === 2 ? canvas.clientHeight : 1;
      const delta = Math.max(-240, Math.min(240, event.deltaY * unit));
      zoom(scale * Math.exp(-delta * 0.002), false, event);
    }, {passive: false});
    canvas.addEventListener("dblclick", (event) => {
      if (event.button !== 0 || dialog.open || event.target.closest("a, button")) return;
      event.preventDefault();
      openExpanded(canvas);
    });
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
    // Pointer capture is only needed for primary/middle-button mouse dragging
    // and pen dragging. Cancel middle-click defaults so native autoscroll does
    // not compete with the diagram's pan interaction.
    const preventMiddleDefault = (event) => {
      if (event.button === 1 && scale > 1 && !event.target.closest("a, button")) event.preventDefault();
    };
    canvas.addEventListener("mousedown", preventMiddleDefault);
    canvas.addEventListener("auxclick", preventMiddleDefault);
    canvas.addEventListener("pointerdown", (event) => {
      if (event.pointerType === "touch" || (event.button !== 0 && event.button !== 1) || scale <= 1 ||
          event.target.closest("a, button")) return;
      canvas.focus({preventScroll: true});
      drag = {id: event.pointerId, buttons: event.button === 1 ? 4 : 1,
        x: event.clientX, y: event.clientY, left: canvas.scrollLeft, top: canvas.scrollTop};
      canvas.setPointerCapture(event.pointerId);
      canvas.classList.add("margo-mermaid__viewport--dragging");
      event.preventDefault();
    });
    canvas.addEventListener("pointermove", (event) => {
      if (!drag || drag.id !== event.pointerId) return;
      if (!(event.buttons & drag.buttons)) {
        endDrag();
        return;
      }
      canvas.scrollTo(drag.left + drag.x - event.clientX, drag.top + drag.y - event.clientY);
    });
    const endDrag = () => {
      const pointerID = drag?.id;
      drag = null;
      canvas.classList.remove("margo-mermaid__viewport--dragging");
      if (pointerID !== undefined && canvas.hasPointerCapture(pointerID)) canvas.releasePointerCapture(pointerID);
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
