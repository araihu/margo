(() => {
  "use strict";

  const deck = document.querySelector(".margo-deck");
  if (!deck) return;

  const slides = [...deck.querySelectorAll(":scope > .margo-deck__slide")];
  if (slides.length === 0) return;

  const previous = document.querySelector("[data-margo-deck-previous]");
  const next = document.querySelector("[data-margo-deck-next]");
  const print = document.querySelector("[data-margo-deck-print]");
  const status = document.querySelector("[data-margo-deck-status]");
  let current = 0;

  const show = (index, focus) => {
    current = Math.max(0, Math.min(slides.length - 1, index));
    slides.forEach((slide, slideIndex) => {
      const active = slideIndex === current;
      slide.hidden = !active;
      if (active) slide.setAttribute("aria-current", "page");
      else slide.removeAttribute("aria-current");
    });
    if (previous) previous.disabled = current === 0;
    if (next) next.disabled = current === slides.length - 1;
    if (status) status.textContent = `Slide ${current + 1} of ${slides.length}`;
    if (focus) slides[current].focus({ preventScroll: true });
  };

  const preparePrint = () => {
    slides.forEach((slide) => { slide.hidden = false; });
  };

  const restoreScreen = () => show(current, false);

  globalThis.margoPrepareDeckPrint = preparePrint;
  globalThis.margoRestoreDeckScreen = restoreScreen;
  window.addEventListener("beforeprint", preparePrint);
  window.addEventListener("afterprint", restoreScreen);
  const printMedia = window.matchMedia("print");
  if (typeof printMedia.addEventListener === "function") {
    printMedia.addEventListener("change", (event) => event.matches ? preparePrint() : restoreScreen());
  }

  if (previous) previous.addEventListener("click", () => show(current - 1, true));
  if (next) next.addEventListener("click", () => show(current + 1, true));
  if (print) print.addEventListener("click", () => window.print());

  document.addEventListener("keydown", (event) => {
    const target = event.target;
    if (target instanceof HTMLElement && (target.isContentEditable || /^(INPUT|SELECT|TEXTAREA)$/.test(target.tagName))) return;
    let destination = current;
    switch (event.key) {
      case "ArrowLeft": destination = current - 1; break;
      case "ArrowRight": destination = current + 1; break;
      case "Home": destination = 0; break;
      case "End": destination = slides.length - 1; break;
      default: return;
    }
    event.preventDefault();
    show(destination, true);
  });

  show(0, false);
})();
