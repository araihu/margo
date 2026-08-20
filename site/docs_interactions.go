package site

const searchInteractionsScriptPath = "margo-assets/search-interactions.js"

// searchInteractionsScript supplements the public Goshtoso search component with
// docs-owned combobox state. Goshtoso continues to own rendering,
// filtering, navigation, and dialog focus trapping; this script only mirrors
// the state into ARIA and restores the invoking trigger after close.
const searchInteractionsScript = `(function () {
  "use strict";

  var states = new WeakMap();

  function visible(element) {
    if (!element || element.hidden) return false;
    var style = window.getComputedStyle(element);
    var rect = element.getBoundingClientRect();
    return style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && rect.height > 0;
  }

  function visibleResults(listbox) {
    return Array.prototype.filter.call(listbox.querySelectorAll('[role="option"]'), visible);
  }

  function open(modal) {
    return visible(modal) && !modal.hasAttribute("hidden");
  }

  function queueSync(state) {
    if (state.frame) return;
    state.frame = window.requestAnimationFrame(function () {
      state.frame = 0;
      sync(state);
    });
  }

  function sync(state) {
    var modal = state.modal;
    var currentlyOpen = open(modal);
    if (state.wasOpen && !currentlyOpen) restoreFocusWhenClosed(state);
    state.wasOpen = currentlyOpen;
    var input = modal.querySelector('[role="combobox"]');
    var listbox = input && document.getElementById(input.getAttribute("aria-controls"));
    var status = modal.querySelector("[data-margo-search-status]");
    if (!input || !listbox) return;
    var query = (input.value || "").trim();
    var allResults = Array.prototype.slice.call(listbox.querySelectorAll('[role="option"]'));
    var results = visibleResults(listbox);
    if (results.length === 0) {
      state.activeIndex = 0;
    } else if (state.activeIndex >= results.length) {
      state.activeIndex = results.length - 1;
    }
    var active = query && results.length > 0 ? results[state.activeIndex] : null;
    input.setAttribute("aria-expanded", query.length > 0 ? "true" : "false");
    if (active && active.id) input.setAttribute("aria-activedescendant", active.id);
    else input.removeAttribute("aria-activedescendant");
    allResults.forEach(function (result) {
      var index = results.indexOf(result);
      result.setAttribute("aria-selected", active === result && index === state.activeIndex ? "true" : "false");
    });
    if (status) {
      if (!query) status.textContent = "";
      else if (results.length === 0) status.textContent = "No matching pages.";
      else status.textContent = results.length + (results.length === 1 ? " result available." : " results available.");
    }
  }

  function focusTrigger(state) {
    if (state.trigger && typeof state.trigger.focus === "function") state.trigger.focus();
  }

  function restoreFocusWhenClosed(state) {
    var token = ++state.restoreToken;
    var attempts = 0;
    var closedAttempts = 0;
    var retry = function () {
      if (token !== state.restoreToken) return;
      if (!open(state.modal)) {
        focusTrigger(state);
        if (closedAttempts++ < 8) window.setTimeout(retry, 32);
        return;
      }
      if (attempts++ < 75) window.setTimeout(retry, 16);
    };
    retry();
  }

  function init(modal) {
    if (states.has(modal)) return;
    var id = modal.getAttribute("data-search-id");
    var field = document.querySelector('[data-search-field][data-search-id="' + id + '"]');
    var trigger = field && field.querySelector("button");
    var state = { modal: modal, trigger: trigger, activeIndex: 0, frame: 0, wasOpen: false, restoreToken: 0 };
    states.set(modal, state);
    if (trigger) {
      trigger.addEventListener("click", function () {
        state.trigger = trigger;
        state.restoreToken++;
        state.activeIndex = 0;
        queueSync(state);
      });
    }
    window.addEventListener("goshtoso-search-close", function (event) {
      if (!event.detail || event.detail.id !== id) return;
      state.trigger = trigger;
      restoreFocusWhenClosed(state);
    });
    modal.addEventListener("input", function (event) {
      if (!event.target.matches('[role="combobox"]')) return;
      state.activeIndex = 0;
      queueSync(state);
    });
    modal.addEventListener("keydown", function (event) {
      if (!event.target.matches('[role="combobox"]')) return;
      if (event.key === "ArrowDown" || event.key === "ArrowUp") {
        window.setTimeout(function () {
          var input = event.target;
          var listbox = document.getElementById(input.getAttribute("aria-controls"));
          var results = listbox ? visibleResults(listbox) : [];
          if (!results.length) return;
          state.activeIndex = (state.activeIndex + (event.key === "ArrowDown" ? 1 : -1) + results.length) % results.length;
          queueSync(state);
        }, 0);
      }
      if (event.key === "Escape") {
        state.trigger = trigger;
        window.setTimeout(function () {
          if (!open(modal)) restoreFocusWhenClosed(state);
          queueSync(state);
        }, 0);
      }
    }, true);
    modal.addEventListener("mouseover", function (event) {
      var result = event.target.closest && event.target.closest('[role="option"]');
      if (!result || !modal.contains(result)) return;
      var input = modal.querySelector('[role="combobox"]');
      var listbox = input && document.getElementById(input.getAttribute("aria-controls"));
      var results = listbox ? visibleResults(listbox) : [];
      var index = results.indexOf(result);
      if (index >= 0) {
        state.activeIndex = index;
        queueSync(state);
      }
    });
    modal.addEventListener("click", function (event) {
      var clear = event.target.closest && event.target.closest("[data-margo-search-clear]");
      if (!clear) return;
      event.preventDefault();
      var input = modal.querySelector('[role="combobox"]');
      if (!input) return;
      input.value = "";
      input.dispatchEvent(new Event("input", { bubbles: true }));
      input.focus();
      queueSync(state);
    });
    document.addEventListener("keydown", function (event) {
      if (event.key !== "Escape" || !open(modal)) return;
      state.trigger = trigger;
      window.setTimeout(function () {
        if (!open(modal)) restoreFocusWhenClosed(state);
      }, 0);
    }, true);
    if (window.MutationObserver) {
      var observer = new MutationObserver(function () { queueSync(state); });
      observer.observe(modal, { attributes: true, attributeFilter: ["style", "class", "hidden"], childList: true, subtree: true });
      state.observer = observer;
    }
    queueSync(state);
  }

  function scan() {
    document.querySelectorAll('[data-search-modal][data-margo-search-a11y="true"]').forEach(init);
  }

  document.addEventListener("DOMContentLoaded", scan);
  document.addEventListener("htmx:afterSwap", scan);
  document.addEventListener("htmx:afterSettle", scan);
  if (document.readyState !== "loading") scan();
})();
`
