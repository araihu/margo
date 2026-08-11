export function fixture(documentCSS, standaloneCSS, mode) {
  const dark = mode === "dark";
  return `<!doctype html>
<html class="${dark ? "dark" : ""}" data-color-mode="${mode}">
  <head>
    <style>
      :root {
        --color-neutral-100: oklch(97% 0 0);
        --color-neutral-300: oklch(87% 0 0);
        --color-neutral-500: oklch(55.6% 0 0);
        --color-neutral-600: oklch(43.9% 0 0);
        --color-neutral-700: oklch(37.1% 0 0);
        --color-neutral-800: oklch(26.9% 0 0);
        --color-neutral-900: oklch(20.5% 0 0);
        --color-neutral-950: oklch(14.5% 0 0);
        --color-surface: var(--color-neutral-100);
        --color-surface-alt: oklch(98.5% 0 0);
        --color-on-surface: var(--color-neutral-600);
        --color-on-surface-strong: var(--color-neutral-800);
        --color-outline: var(--color-neutral-300);
        --color-primary: oklch(45% 0.16 250);
        --color-surface-dark: var(--color-neutral-800);
        --color-surface-dark-alt: var(--color-neutral-900);
        --color-on-surface-dark: var(--color-neutral-300);
        --color-on-surface-dark-strong: var(--color-neutral-100);
        --color-outline-dark: var(--color-neutral-700);
        --color-primary-dark: oklch(82% 0.12 250);
        --document-page-background: var(--color-surface);
        --document-content-width: 72rem;
        --document-font-body: Arial, sans-serif;
        --document-font-heading: Arial, sans-serif;
        --document-line-height: 1.5;
        --spacing: 0.25rem;
        --radius-radius: 0.375rem;
        --text-xs: 0.75rem;
        --text-sm: 0.875rem;
        --text-base: 1rem;
        --text-lg: 1.125rem;
        --text-xl: 1.25rem;
        --text-2xl: 1.5rem;
        --text-4xl: 2.25rem;
        --text-xs--line-height: 1;
        --text-sm--line-height: 1.25;
        --text-base--line-height: 1.5;
        --text-lg--line-height: 1.5;
        --text-xl--line-height: 1.4;
        --text-2xl--line-height: 1.3;
        --text-4xl--line-height: 1.1;
        --font-weight-medium: 500;
        --font-weight-semibold: 600;
        --font-weight-bold: 700;
      }
    </style>
    <style>${documentCSS}</style>
    <style>${standaloneCSS}</style>
  </head>
  <body>
    <div class="goshtoso-document">
      <header class="goshtoso-document__header"><strong>Margo</strong><span>Markdown for Goshtoso</span></header>
      <aside class="goshtoso-document__stamps" aria-label="Document status"><span class="goshtoso-document__stamp">dark review</span></aside>
      <nav class="goshtoso-document__toc" aria-label="Table of contents"><p class="goshtoso-document__toc-title">Contents</p><a href="#audit">Contrast audit</a></nav>
      <article class="margo-document">
        <h1 id="audit">Contrast audit</h1>
        <p>Visible text, <a href="#audit">links</a>, tables, code, and source disclosures must remain readable.</p>
        <blockquote>Boundary evidence is part of the human artifact.</blockquote>
        <table><thead><tr><th>Surface</th><th>Result</th></tr></thead><tbody><tr><td>Document</td><td>Pass</td></tr></tbody></table>
        <pre><code>GOWORK=off GOFLAGS=-mod=readonly go test ./...</code></pre>
        <figure class="margo-mermaid">
          <div class="margo-mermaid__canvas" role="img" aria-label="Mermaid diagram"></div>
          <details open class="margo-mermaid__source"><summary>Mermaid source</summary><pre><code>flowchart LR
source --> dark</code></pre></details>
        </figure>
      </article>
      <footer class="goshtoso-document__footer"><span>Human review</span><span>PDF</span></footer>
      <span class="goshtoso-document__watermark" aria-hidden="true">Margo benchmark</span>
    </div>
  </body>
</html>`;
}
