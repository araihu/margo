(() => {
  "use strict";

  const blockSelector = "[data-margo-code-copy]";
  const buttonSelector = "[data-margo-code-copy-button]";

  const fallbackCopy = (text) => {
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.setAttribute("readonly", "");
    textarea.style.position = "fixed";
    textarea.style.inset = "0 auto auto 0";
    textarea.style.inlineSize = "1px";
    textarea.style.blockSize = "1px";
    textarea.style.opacity = "0";
    document.body.appendChild(textarea);
    textarea.select();
    let copied = false;
    try {
      copied = document.execCommand("copy");
    } catch (_) {
      copied = false;
    }
    textarea.remove();
    return copied;
  };

  const copyText = (text) => {
    let clipboard = null;
    try {
      clipboard = navigator.clipboard;
    } catch (_) {
      clipboard = null;
    }
    if (clipboard && typeof clipboard.writeText === "function") {
      return Promise.resolve().then(() => clipboard.writeText(text)).catch(() => {
        if (fallbackCopy(text)) return undefined;
        throw new Error("clipboard unavailable");
      });
    }
    if (fallbackCopy(text)) return Promise.resolve();
    return Promise.reject(new Error("clipboard unavailable"));
  };

  const updateState = (block, copied, message) => {
    const button = block.querySelector(buttonSelector);
    if (!button) return;
    const label = button.querySelector("[data-margo-code-copy-label]");
    const copyIcon = button.querySelector('[x-show="!copied"]');
    const copiedIcon = button.querySelector('[x-show="copied"]');
    if (label) {
      label.textContent = message || (copied ? "Copied!" : "Copy");
      label.setAttribute("aria-live", "polite");
    }
    if (!button.dataset.margoCodeCopyDefaultLabel) {
      button.dataset.margoCodeCopyDefaultLabel = button.getAttribute("aria-label") || "Copy code";
    }
    button.setAttribute("aria-label", copied ? "Code copied" : button.dataset.margoCodeCopyDefaultLabel);
    if (copyIcon) {
      copyIcon.hidden = copied;
      copyIcon.removeAttribute("x-show");
    }
    if (copiedIcon) {
      copiedIcon.hidden = !copied;
      copiedIcon.removeAttribute("x-show");
      copiedIcon.removeAttribute("x-cloak");
    }
  };

  const bind = (block) => {
    const button = block.querySelector(buttonSelector);
    if (!button || button.dataset.margoCodeCopyBound === "true") return;
    button.dataset.margoCodeCopyBound = "true";
    const code = block.querySelector("pre code, code");
    if (!code) return;
    block.removeAttribute("x-data");
    button.removeAttribute("@click");
    const label = button.querySelector("[data-margo-code-copy-label]");
    if (label) label.removeAttribute("x-text");
    updateState(block, false);
    button.addEventListener("click", (event) => {
      event.preventDefault();
      if (button.dataset.margoCodeCopyPending === "true") return;
      button.dataset.margoCodeCopyPending = "true";
      copyText(code.textContent || "").then(() => {
        updateState(block, true);
        window.setTimeout(() => updateState(block, false), 2000);
      }).catch(() => {
        updateState(block, false, "Copy failed");
        window.setTimeout(() => updateState(block, false), 2000);
      }).finally(() => {
        button.dataset.margoCodeCopyPending = "false";
      });
    });
  };

  const initialize = () => {
    document.querySelectorAll(blockSelector).forEach(bind);
  };
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initialize, { once: true });
  } else {
    initialize();
  }
})();
