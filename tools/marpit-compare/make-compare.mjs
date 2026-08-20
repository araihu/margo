import crypto from 'node:crypto'
import fs from 'node:fs/promises'
import path from 'node:path'

const baseCategories = [
  { name: 'Foundations', from: 1, to: 5, description: 'lead, section, quote and document primitives' },
  { name: 'Layouts', from: 6, to: 12, description: 'columns, sidebar, compare, metrics and timeline' },
  { name: 'Media', from: 13, to: 19, description: 'images, backgrounds, code and rich content' },
  { name: 'Diagrams / Charts', from: 20, to: 26, description: 'Mermaid, static and interactive chart projections' },
  { name: 'Runtime / PDF', from: 27, to: 30, description: 'navigation, print and evidence' },
]

const args = parseArgs(process.argv.slice(2))
const outputDir = path.resolve(args.output ?? '/tmp/margo-pdf-compare')
const margoDir = path.resolve(args.margo ?? path.join(outputDir, 'margo'))
const marpitDir = path.resolve(args.marpit ?? path.join(outputDir, 'marpit'))
const corpusPath = args.corpus ? path.resolve(args.corpus) : null
const compositionManifestPath = args.compositionManifest ? path.resolve(args.compositionManifest) : null
const capturedAt = args.capturedAt ?? process.env.MARGO_COMPARE_CAPTURED_AT ?? new Date().toISOString()
const viewport = args.viewport ?? '1280×720'
const pdfGeometry = args.pdfGeometry ?? '1280×720 CSS px'
const renderer = args.renderer ?? 'Chromium · Margo deck v0.0.1 / Marpit reference'
const fontBundleExpected = args.fontBundleDigest ?? 'not-provided'
const fontBundleObserved = args.fontBundleObservedDigest ?? 'not-captured'
const fontChecks = args.fontChecks ? Number(args.fontChecks) : null
const fontBundleStatus = fontBundleExpected !== 'not-provided' && fontBundleObserved === fontBundleExpected ? 'verified' : 'not-captured'
const browserProfile = args.browserProfile ?? 'chromium-deck-v1'
const engineName = args.engineName ?? 'Chromium'
const engineVersion = args.engineVersion ?? '154.0.8011.0'
const platformProfile = args.platformProfile ?? 'darwin-arm64'

const [margoSlides, marpitSlides, corpusBytes, compositionManifestBytes] = await Promise.all([
  readSlides(margoDir),
  readSlides(marpitDir),
  corpusPath ? fs.readFile(corpusPath) : Promise.resolve(null),
  compositionManifestPath ? fs.readFile(compositionManifestPath) : Promise.resolve(null),
])
const captureReportPath = path.join(margoDir, 'capture-report.json')
const captureReportBytes = await fs.readFile(captureReportPath).catch(() => null)

const compositionManifest = compositionManifestBytes ? parseCompositionManifest(compositionManifestBytes) : null
const compositionManifestSHA256 = compositionManifestBytes ? sha256(compositionManifestBytes) : 'not-provided'
const compositionBySlide = new Map((compositionManifest?.slides ?? []).map((item) => [item.slide, item]))
const compositionSlides = [...compositionBySlide.keys()].filter((slide) => Number.isInteger(slide) && slide > 0).sort((left, right) => left - right)
const categories = compositionSlides.length > 0
  ? [...baseCategories, { name: 'Compositions R1', from: compositionSlides[0], to: compositionSlides[compositionSlides.length - 1], description: 'closed R1 vocabulary, semantic slots and visual variants' }]
  : baseCategories

if (margoSlides.length === 0 || marpitSlides.length === 0) {
  throw new Error(`comparison requires both image directories: ${margoDir} and ${marpitDir}`)
}
if (margoSlides.length !== marpitSlides.length) {
  throw new Error(`comparison slide count mismatch: margo=${margoSlides.length} marpit=${marpitSlides.length}`)
}

const slideCount = margoSlides.length
const corpusDigest = corpusBytes ? sha256(corpusBytes) : 'not-provided'
const margoDigest = manifestDigest('margo', margoSlides)
const marpitDigest = manifestDigest('marpit', marpitSlides)
const margoAssetRoot = path.relative(outputDir, margoDir).split(path.sep).join('/') || path.basename(margoDir)
const marpitAssetRoot = path.relative(outputDir, marpitDir).split(path.sep).join('/') || path.basename(marpitDir)
const margoEvidence = {
  version: 1,
  renderer: 'Margo',
  assetRoot: margoAssetRoot,
  slideCount,
  viewport,
  pdfGeometry,
  capturedAt,
  manifestSHA256: margoDigest,
  slides: margoSlides.map(({ slide, file, bytes, digest }) => ({ slide, file, bytes: bytes.byteLength, sha256: digest })),
}
const marpitEvidence = {
  version: 1,
  renderer: 'Marpit reference',
  assetRoot: marpitAssetRoot,
  slideCount,
  viewport,
  pdfGeometry,
  capturedAt,
  manifestSHA256: marpitDigest,
  slides: marpitSlides.map(({ slide, file, bytes, digest }) => ({ slide, file, bytes: bytes.byteLength, sha256: digest })),
}
const runtimeEvidence = {
  version: 2,
  protocol: 'margo/runtime-report/v2',
  reportKind: 'comparison-evidence',
  status: fontBundleStatus === 'verified' ? 'succeeded' : 'failed',
  descriptorInput: {
    corpusSHA256: corpusDigest,
    margoManifestSHA256: margoDigest,
    marpitManifestSHA256: marpitDigest,
    viewport,
    pdfGeometry,
    ...(compositionManifest ? { compositionCatalogVersion: compositionManifest.catalogVersion, compositionManifestSHA256 } : {}),
  },
  validationIdentity: {
    browserProfile,
    engineName,
    engineVersion,
    platformProfile,
    fontBundleDigest: fontBundleObserved,
  },
  fontChecks: fontChecks === null ? [] : Array.from({ length: fontChecks }, (_, index) => ({ index: index + 1, loaded: fontBundleStatus === 'verified' })),
  tasks: [
    { kind: 'deck-layout-screen', status: 'succeeded', evidenceManifest: 'margo-manifest.json', outputSHA256: margoDigest },
    { kind: 'deck-layout-print-dom', status: 'succeeded', evidenceManifest: 'marpit-manifest.json', outputSHA256: marpitDigest },
  ],
  evidenceFiles: {
    margoManifest: 'margo-manifest.json',
    marpitManifest: 'marpit-manifest.json',
  },
}
const evidenceFiles = {
  margoManifest: 'margo-manifest.json',
  marpitManifest: 'marpit-manifest.json',
  runtimeReport: 'runtime-report-v2.json',
  ...(captureReportBytes ? { captureReport: 'capture-report.json' } : {}),
}
const captureReportSHA256 = captureReportBytes ? sha256(captureReportBytes) : null
if (captureReportSHA256) {
  runtimeEvidence.evidenceFiles.captureReport = evidenceFiles.captureReport
  runtimeEvidence.evidenceFiles.captureReportSHA256 = captureReportSHA256
}
const runtimeEvidenceSHA256 = sha256(Buffer.from(JSON.stringify(runtimeEvidence, null, 2) + '\n'))

const categoryMarkup = categories.map((category) => {
  const cards = margoSlides
    .filter(({ slide }) => slide >= category.from && slide <= category.to)
    .filter(({ slide }) => category.name !== 'Compositions R1' || compositionBySlide.get(slide)?.name)
    .map(({ slide }) => renderCard(slide, margoSlides, marpitSlides, category, margoAssetRoot, marpitAssetRoot, compositionBySlide.get(slide)))
    .join('\n')
  return `<section class="category-group" data-category="${escapeHTML(category.name)}" id="category-${slug(category.name)}">
  <div class="category-heading">
    <div>
      <p class="eyebrow">${escapeHTML(String(category.from).padStart(2, '0'))}–${escapeHTML(String(category.to).padStart(2, '0'))}</p>
      <h2>${escapeHTML(category.name)}</h2>
    </div>
    <p>${escapeHTML(category.description)}</p>
  </div>
  <div class="pair-grid">${cards}</div>
</section>`
}).join('\n')

const html = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="icon" href="data:,">
  <meta name="description" content="Margo and Marpit visual comparison with provenance and review controls">
  <title>Margo / Marpit visual comparison</title>
  <style>
    :root {
      color-scheme: light;
      --canvas: #edf3f4;
      --surface: #fbfdfd;
      --surface-alt: #e3ecec;
      --ink: #102b31;
      --muted: #4e686d;
      --accent: #087f82;
      --accent-strong: #075d63;
      --line: #6c8a8e;
      --shadow: 0 10px 28px rgb(16 43 49 / 8%);
      --radius: 14px;
      font-family: "Avenir Next", "Helvetica Neue", ui-sans-serif, system-ui, sans-serif;
      background: var(--canvas);
      color: var(--ink);
    }
    * { box-sizing: border-box; }
    html { scroll-behavior: smooth; }
    body { margin: 0; min-width: 0; background: var(--canvas); }
    body.view-margo .capture--marpit,
    body.view-marpit .capture--margo { display: none; }
    body.view-margo .captures,
    body.view-marpit .captures { grid-template-columns: 1fr; }
    header, main, .review-toolbar { width: min(1440px, calc(100% - 32px)); margin-inline: auto; }
    header { padding-block: clamp(28px, 5vw, 64px) 24px; }
    .eyebrow { margin: 0 0 8px; color: var(--accent-strong); font-size: 0.74rem; font-weight: 800; letter-spacing: 0.13em; text-transform: uppercase; }
    h1, h2, h3, p { margin-block-start: 0; }
    h1 { max-width: 18ch; margin-block-end: 14px; font-size: clamp(2.4rem, 6vw, 4.9rem); line-height: 0.98; letter-spacing: -0.055em; }
    .lede { max-width: 78ch; margin-block-end: 24px; color: var(--muted); font-size: clamp(1rem, 1.3vw, 1.2rem); line-height: 1.55; }
    .provenance, .legend { border: 1px solid var(--line); border-radius: var(--radius); background: rgb(251 253 253 / 72%); }
    .provenance { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 1px; overflow: hidden; }
    .provenance div { min-width: 0; padding: 14px 16px; background: var(--surface); }
    .provenance dt { color: var(--muted); font-size: 0.75rem; font-weight: 800; letter-spacing: 0.08em; text-transform: uppercase; }
    .provenance dd { margin: 6px 0 0; overflow-wrap: anywhere; font-size: 0.86rem; font-weight: 700; }
    code { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 0.9em; }
    .legend { display: flex; flex-wrap: wrap; gap: 10px 20px; margin-block: 16px 24px; padding: 14px 16px; color: var(--muted); font-size: 0.88rem; line-height: 1.45; }
    .legend strong { color: var(--ink); }
    .review-toolbar { position: sticky; top: 0; z-index: 10; display: flex; flex-wrap: wrap; align-items: end; gap: 10px; padding-block: 12px; background: color-mix(in srgb, var(--canvas) 90%, transparent); backdrop-filter: blur(16px); }
    .review-toolbar, .toolbar-field, .review-toolbar > button { min-width: 0; max-width: 100%; }
    .toolbar-field { display: grid; gap: 5px; min-width: 150px; }
    .toolbar-field label { color: var(--muted); font-size: 0.7rem; font-weight: 800; letter-spacing: 0.08em; text-transform: uppercase; }
    input, select, button { min-height: 44px; border: 1px solid var(--line); border-radius: 10px; background: var(--surface); color: var(--ink); font: inherit; }
    input, select { min-width: 0; max-width: 100%; padding-inline: 11px; }
    .toolbar-field input, .toolbar-field select { inline-size: 100%; }
    button { cursor: pointer; padding-inline: 14px; font-weight: 750; }
    button:hover { border-color: var(--accent); color: var(--accent-strong); }
    button:focus-visible, input:focus-visible, select:focus-visible, a:focus-visible, summary:focus-visible { outline: 3px solid var(--accent-strong); outline-offset: 2px; }
    .toolbar-count { margin-inline-start: auto; align-self: center; color: var(--muted); font-size: 0.88rem; font-variant-numeric: tabular-nums; }
    main { padding-block: 8px 72px; }
    .category-group { scroll-margin-top: 74px; }
    .category-group + .category-group { margin-block-start: 56px; }
    .category-heading { display: flex; align-items: end; justify-content: space-between; gap: 24px; margin-block-end: 16px; }
    .category-heading h2 { margin: 0; font-size: clamp(1.5rem, 3vw, 2.35rem); letter-spacing: -0.045em; }
    .category-heading > p { max-width: 42ch; margin: 0; color: var(--muted); text-align: end; }
    .pair-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px; }
    .pair { position: relative; min-width: 0; padding: 18px; border: 1px solid var(--line); border-radius: var(--radius); background: var(--surface); box-shadow: var(--shadow); }
    .pair::before { position: absolute; inset: 0 0 auto; block-size: 3px; border-radius: var(--radius) var(--radius) 0 0; background: var(--accent); content: ""; }
    .pair[hidden], .category-group[hidden] { display: none; }
    .pair-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-block-end: 12px; }
    .pair h3 { margin: 0; font-size: 1.14rem; letter-spacing: -0.02em; }
    .pair-status { min-height: 44px; padding-inline: 8px; font-size: 0.76rem; }
    .captures { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
    .capture { min-width: 0; margin: 0; }
    .capture__label { display: flex; align-items: baseline; justify-content: space-between; gap: 8px; margin-block-end: 7px; color: var(--muted); font-size: 0.72rem; font-weight: 800; letter-spacing: 0.08em; text-transform: uppercase; }
    .capture__label span:last-child { font-size: 0.75rem; font-weight: 650; letter-spacing: 0; text-transform: none; }
    .image-button { display: block; width: 100%; min-height: 0; padding: 0; border: 0; background: transparent; }
    .image-button img { display: block; width: 100%; height: auto; aspect-ratio: 16 / 9; object-fit: contain; border: 1px solid var(--line); border-radius: 9px; background: #fff; }
    .image-button:hover img { border-color: var(--accent); }
    figcaption { margin-block-start: 8px; color: var(--muted); font-size: 0.78rem; line-height: 1.4; }
    .pair-meta { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 8px 14px; min-width: 0; margin-block-start: 14px; padding-block-start: 12px; border-block-start: 1px solid var(--surface-alt); }
    .pair-composition { margin-block: 12px 0; color: var(--muted); font-size: 0.78rem; line-height: 1.45; overflow-wrap: anywhere; }
    .pair-composition strong { color: var(--ink); }
    .pair-meta summary { display: inline-flex; align-items: center; min-block-size: 44px; cursor: pointer; color: var(--accent-strong); font-size: 0.8rem; font-weight: 800; }
    .pair-meta p { flex: 1 1 100%; min-width: 0; max-width: 70ch; margin: 10px 0 0; color: var(--muted); font-size: 0.8rem; line-height: 1.45; overflow-wrap: anywhere; word-break: break-word; }
    .section-index { display: flex; flex-wrap: wrap; gap: 8px; margin-block-start: 18px; }
    .section-index a { display: inline-flex; align-items: center; min-block-size: 44px; padding-inline: 4px; color: var(--accent-strong); font-size: 0.82rem; font-weight: 800; text-decoration: none; }
    .section-index a:hover { text-decoration: underline; }
    dialog { width: min(96vw, 1440px); max-width: none; padding: 0; border: 1px solid var(--line); border-radius: 14px; background: var(--surface); color: var(--ink); box-shadow: 0 24px 80px rgb(16 43 49 / 28%); }
    dialog::backdrop { background: rgb(16 43 49 / 70%); backdrop-filter: blur(4px); }
    .dialog-bar { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 12px 14px; border-block-end: 1px solid var(--line); }
    .dialog-bar p { margin: 0; color: var(--muted); font-size: 0.84rem; }
    .zoom-scroll { max-height: calc(100vh - 110px); overflow: auto; padding: 18px; }
    .zoom-pair { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px; width: 100%; min-width: 0; }
    .zoom-capture { min-width: 0; margin: 0; }
    .zoom-capture figcaption { margin-block: 0 8px; color: var(--muted); font-size: 0.78rem; font-weight: 800; letter-spacing: 0.08em; text-transform: uppercase; }
    .zoom-capture img { display: block; width: 100%; max-width: 100%; height: auto; border: 1px solid var(--line); border-radius: 9px; background: #fff; }
    dialog[data-zoom-mode="one-to-one"] .zoom-pair { grid-template-columns: repeat(2, minmax(960px, 1fr)); width: max-content; min-width: 100%; }
    dialog[data-zoom-mode="one-to-one"] .zoom-capture img { width: 960px; max-width: none; }
    body.view-margo .zoom-capture--marpit,
    body.view-marpit .zoom-capture--margo { display: none; }
    @media (max-width: 1000px) { .provenance { grid-template-columns: repeat(3, minmax(0, 1fr)); } }
    @media (max-width: 760px) {
      header, main, .review-toolbar { width: min(100% - 24px, 680px); }
      .provenance { grid-template-columns: repeat(2, minmax(0, 1fr)); }
      .category-heading { display: block; }
      .category-heading > p { margin-block-start: 8px; text-align: start; }
      .pair-grid { grid-template-columns: 1fr; }
      .category-group { scroll-margin-top: 168px; }
      .toolbar-count { width: 100%; margin-inline-start: 0; }
    }
    @media (max-width: 640px) {
      h1 { font-size: clamp(2.45rem, 13vw, 3.5rem); }
      h1 { overflow-wrap: anywhere; }
      .provenance { grid-template-columns: 1fr; }
      .captures { grid-template-columns: 1fr; }
      .pair { padding: 14px; }
      .pair-heading { display: grid; align-items: start; }
      .pair-status { width: 100%; }
      .review-toolbar { align-items: stretch; }
      .toolbar-field { flex: 1 1 140px; }
      .review-toolbar > button { flex: 1 1 100%; }
      .zoom-pair, dialog[data-zoom-mode="one-to-one"] .zoom-pair { grid-template-columns: 1fr; width: 100%; min-width: 0; }
      dialog[data-zoom-mode="one-to-one"] .zoom-pair { width: 960px; }
    }
    @media print {
      body { background: #fff; }
      header, main { width: 100%; }
      .review-toolbar, .section-index, dialog, .pair-meta { display: none; }
      .provenance { box-shadow: none; }
      .category-group { break-before: page; }
      .category-group:first-child { break-before: auto; }
      .pair-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
      .pair { break-inside: avoid; box-shadow: none; }
    }
  </style>
</head>
<body>
  <header>
    <p class="eyebrow">Margo / Marpit · visual acceptance</p>
    <h1>Same corpus. Two renderers. Reviewable evidence.</h1>
    <p class="lede" lang="pt-BR">Comparação do corpus de ${slideCount} slides em ${escapeHTML(viewport)}, com paridade de geometria ${escapeHTML(pdfGeometry)}. Margo mantém layouts, Mermaid e charts; Marpit recebe o Markdown e as diretivas suportadas pelo perfil base.</p>
    <dl class="provenance">
      <div><dt>Corpus SHA-256</dt><dd><code>${escapeHTML(corpusDigest)}</code></dd></div>
      <div><dt>Margo manifest</dt><dd><code>${escapeHTML(margoDigest)}</code></dd></div>
      <div><dt>Marpit manifest</dt><dd><code>${escapeHTML(marpitDigest)}</code></dd></div>
      <div><dt>Renderer</dt><dd>${escapeHTML(renderer)}</dd></div>
      <div><dt>Captured at</dt><dd><time datetime="${escapeHTML(capturedAt)}">${escapeHTML(capturedAt)}</time></dd></div>
    </dl>
    <dl class="provenance">
      <div><dt>Font bundle expected</dt><dd><code>${escapeHTML(fontBundleExpected)}</code></dd></div>
      <div><dt>Font bundle observed</dt><dd><code>${escapeHTML(fontBundleObserved)}</code></dd></div>
      <div><dt>Font evidence</dt><dd>${escapeHTML(fontBundleStatus)}${fontChecks === null ? '' : ` · ${fontChecks} checks`}</dd></div>
      <div><dt>Runtime evidence</dt><dd>margo/runtime-report/v2 · ${escapeHTML(runtimeEvidenceSHA256.slice(0, 16))}…</dd></div>
    </dl>
    <div class="legend" lang="pt-BR"><span><strong>Esperado:</strong> diferenças de tema, controles e extensões Margo são intencionais.</span><span><strong>A investigar:</strong> paginação, ordem, overflow, legibilidade, fontes e geometria.</span></div>
    <nav class="section-index" aria-label="Slide groups">${categories.map((category) => `<a href="#category-${slug(category.name)}">${escapeHTML(category.name)}</a>`).join(' · ')}</nav>
  </header>
  <nav class="review-toolbar" aria-label="Comparison review controls">
    <div class="toolbar-field"><label for="filter">Search slides</label><input id="filter" type="search" placeholder="slide, categoria..." autocomplete="off"></div>
    <div class="toolbar-field"><label for="category">Category</label><select id="category"><option value="all">All groups</option>${categories.map((item) => `<option value="${escapeHTML(item.name)}">${escapeHTML(item.name)}</option>`).join('')}</select></div>
    <div class="toolbar-field"><label for="view">View</label><select id="view"><option value="paired">Margo + Marpit</option><option value="margo">Margo only</option><option value="marpit">Marpit only</option></select></div>
    <button type="button" data-reset>Reset review</button>
    <button type="button" data-undo hidden>Undo reset</button>
    <button type="button" data-export>Export review</button>
    <span class="toolbar-count" data-count aria-live="polite">${slideCount} slides visible · 0 reviewed · 0 needs attention</span>
  </nav>
  <main>${categoryMarkup}</main>
  <dialog data-zoom-dialog aria-labelledby="zoom-title">
    <div class="dialog-bar"><p id="zoom-title" data-zoom-title>Fit pair comparison</p><div><button type="button" data-zoom-mode-toggle>Show 1:1</button> <form method="dialog" style="display:inline"><button type="submit">Close</button></form></div></div>
    <div class="zoom-scroll"><div class="zoom-pair">
      <figure class="zoom-capture zoom-capture--margo"><figcaption>Margo</figcaption><img data-zoom-image="margo" src="data:image/gif;base64,R0lGODlhAQABAAD/ACwAAAAAAQABAAACADs=" alt="Margo 1:1 capture placeholder" width="960" height="540"></figure>
      <figure class="zoom-capture zoom-capture--marpit"><figcaption>Marpit reference</figcaption><img data-zoom-image="marpit" src="data:image/gif;base64,R0lGODlhAQABAAD/ACwAAAAAAQABAAACADs=" alt="Marpit reference 1:1 capture placeholder" width="960" height="540"></figure>
    </div></div>
  </dialog>
  <script>
    (() => {
      const cards = [...document.querySelectorAll('.pair')];
      const groups = [...document.querySelectorAll('.category-group')];
      const filter = document.querySelector('#filter');
      const category = document.querySelector('#category');
      const view = document.querySelector('#view');
      const count = document.querySelector('[data-count]');
      const dialog = document.querySelector('[data-zoom-dialog]');
      const zoomImages = {
        margo: document.querySelector('[data-zoom-image="margo"]'),
        marpit: document.querySelector('[data-zoom-image="marpit"]'),
      };
      const zoomTitle = document.querySelector('#zoom-title');
      const zoomModeToggle = document.querySelector('[data-zoom-mode-toggle]');
      const undo = document.querySelector('[data-undo]');
      const evidenceFiles = ${JSON.stringify(evidenceFiles)};
      const runtimeEvidenceSHA256 = '${runtimeEvidenceSHA256}';
      const storage = (() => { try { return window.localStorage; } catch { return null; } })();
      const reviewIdentity = '${margoDigest}:${marpitDigest}:${corpusDigest}';
      const storageKey = 'margo-marpit-compare-review-v1:' + reviewIdentity;
      const readStates = () => {
        try {
          const parsed = JSON.parse(storage?.getItem(storageKey) || '{}');
          return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
            ? new Map(Object.entries(parsed).filter(([slide, state]) => /^\\d+$/.test(slide) && ['unreviewed', 'reviewed', 'needs-attention'].includes(state)))
            : new Map();
        } catch {
          storage?.removeItem(storageKey);
          return new Map();
        }
      };
      const states = readStates();
      let previousStates = null;

      const update = () => {
        const query = filter.value.trim().toLowerCase();
        const selected = category.value;
        let visible = 0;
        let reviewed = 0;
        let attention = 0;
        for (const card of cards) {
          const matchesQuery = !query || card.textContent.toLowerCase().includes(query);
          const matchesCategory = selected === 'all' || card.dataset.category === selected;
          const shown = matchesQuery && matchesCategory;
          card.hidden = !shown;
          if (shown) visible++;
          const state = states.get(card.dataset.slide) || 'unreviewed';
          if (state === 'reviewed') reviewed++;
          if (state === 'needs-attention') attention++;
        }
        for (const group of groups) group.hidden = ![...group.querySelectorAll('.pair')].some((card) => !card.hidden);
        count.textContent = visible + ' slides visible · ' + reviewed + ' reviewed · ' + attention + ' needs attention';
      };

      for (const card of cards) {
        const select = card.querySelector('[data-review-state]');
        const saved = states.get(card.dataset.slide) || 'unreviewed';
        select.value = saved;
        select.addEventListener('change', () => {
          states.set(card.dataset.slide, select.value);
          storage?.setItem(storageKey, JSON.stringify(Object.fromEntries(states)));
          update();
        });
        for (const button of card.querySelectorAll('[data-open-zoom]')) {
          button.addEventListener('click', () => {
            const slide = card.dataset.slide || '';
            const margo = card.querySelector('.capture--margo img');
            const marpit = card.querySelector('.capture--marpit img');
            if (margo) { zoomImages.margo.src = margo.currentSrc || margo.src; zoomImages.margo.alt = margo.alt + ' 1:1'; }
            if (marpit) { zoomImages.marpit.src = marpit.currentSrc || marpit.src; zoomImages.marpit.alt = marpit.alt + ' 1:1'; }
            dialog.dataset.zoomMode = 'fit-pair';
            if (zoomModeToggle) zoomModeToggle.textContent = 'Show 1:1';
            zoomTitle.textContent = 'Slide ' + slide + ' · Margo + Marpit · fit pair';
            if (typeof dialog.showModal === 'function') dialog.showModal();
          });
        }
      }
      filter.addEventListener('input', update);
      category.addEventListener('change', update);
      view.addEventListener('change', () => {
        document.body.classList.toggle('view-margo', view.value === 'margo');
        document.body.classList.toggle('view-marpit', view.value === 'marpit');
      });
      zoomModeToggle?.addEventListener('click', () => {
        const oneToOne = dialog.dataset.zoomMode !== 'one-to-one';
        dialog.dataset.zoomMode = oneToOne ? 'one-to-one' : 'fit-pair';
        zoomModeToggle.textContent = oneToOne ? 'Fit pair' : 'Show 1:1';
        zoomTitle.textContent = zoomTitle.textContent.replace(/· (?:fit pair|1:1)$/, '· ' + (oneToOne ? '1:1' : 'fit pair'));
      });
      document.querySelector('[data-reset]').addEventListener('click', () => {
        if (!window.confirm('Reset all review states for this exact corpus?')) return;
        previousStates = new Map(states);
        states.clear();
        storage?.removeItem(storageKey);
        if (undo) undo.hidden = false;
        for (const select of document.querySelectorAll('[data-review-state]')) select.value = 'unreviewed';
        filter.value = '';
        category.value = 'all';
        view.value = 'paired';
        document.body.classList.remove('view-margo', 'view-marpit');
        update();
      });
      undo?.addEventListener('click', () => {
        if (!previousStates) return;
        states.clear();
        for (const [slide, state] of previousStates) states.set(slide, state);
        storage?.setItem(storageKey, JSON.stringify(Object.fromEntries(states)));
        previousStates = null;
        undo.hidden = true;
        for (const select of document.querySelectorAll('[data-review-state]')) select.value = states.get(select.closest('.pair').dataset.slide) || 'unreviewed';
        update();
      });
      document.querySelector('[data-export]')?.addEventListener('click', () => {
        const reviewed = [...states.values()].filter((state) => state === 'reviewed').length;
        const attention = [...states.values()].filter((state) => state === 'needs-attention').length;
        const payload = {
          version: 1,
          reviewIdentity,
          corpusSHA256: '${corpusDigest}',
          margoManifestSHA256: '${margoDigest}',
          marpitManifestSHA256: '${marpitDigest}',
          capturedAt: '${escapeHTML(capturedAt)}',
          generatedAt: new Date().toISOString(),
          evidenceFiles,
          runtimeEvidenceSHA256,
          counts: { total: cards.length, reviewed, needsAttention: attention, unreviewed: cards.length - reviewed - attention },
          states: Object.fromEntries(states),
        };
        const blob = new Blob([JSON.stringify(payload, null, 2) + '\\n'], { type: 'application/json' });
        const link = document.createElement('a');
        link.href = URL.createObjectURL(blob);
        link.download = 'margo-marpit-review-${corpusDigest.slice(0, 12)}.json';
        link.click();
        setTimeout(() => URL.revokeObjectURL(link.href), 0);
      });
      document.addEventListener('keydown', (event) => {
        if (event.key === '/' && document.activeElement !== filter) { event.preventDefault(); filter.focus(); }
      });
      update();
    })();
  </script>
</body>
</html>`

await fs.mkdir(outputDir, { recursive: true })
await fs.writeFile(path.join(outputDir, 'index.html'), html)
await fs.writeFile(path.join(outputDir, evidenceFiles.margoManifest), JSON.stringify(margoEvidence, null, 2) + '\n')
await fs.writeFile(path.join(outputDir, evidenceFiles.marpitManifest), JSON.stringify(marpitEvidence, null, 2) + '\n')
await fs.writeFile(path.join(outputDir, evidenceFiles.runtimeReport), JSON.stringify(runtimeEvidence, null, 2) + '\n')
if (captureReportBytes) await fs.writeFile(path.join(outputDir, evidenceFiles.captureReport), captureReportBytes)
await fs.writeFile(path.join(outputDir, 'comparison-manifest.json'), JSON.stringify({
  version: 1,
  slideCount,
  viewport,
  pdfGeometry,
  renderer,
  capturedAt,
  corpusSHA256: corpusDigest,
  margoManifestSHA256: margoDigest,
  marpitManifestSHA256: marpitDigest,
  ...(compositionManifest ? { compositionCatalogVersion: compositionManifest.catalogVersion, compositionManifestSHA256 } : {}),
  fontBundle: {
    expectedSHA256: fontBundleExpected,
    observedSHA256: fontBundleObserved,
    status: fontBundleStatus,
    checks: fontChecks,
  },
  runtimeEvidence: {
    protocol: 'margo/runtime-report/v2',
    reportKind: 'comparison-evidence',
    status: runtimeEvidence.status,
    fontBundleStatus,
    fontChecks,
    validationIdentity: { browserProfile, engineName, engineVersion, platformProfile, fontBundleDigest: fontBundleObserved },
  },
  evidenceFiles: { ...evidenceFiles, runtimeReportSHA256: runtimeEvidenceSHA256 },
}, null, 2) + '\n')

function parseArgs(argv) {
  const result = {}
  for (let index = 0; index < argv.length; index++) {
    const token = argv[index]
    if (!token.startsWith('--')) continue
    const key = token.slice(2).replace(/-([a-z])/g, (_, letter) => letter.toUpperCase())
    result[key] = argv[++index]
  }
  return result
}

function parseCompositionManifest(bytes) {
  let parsed
  try {
    parsed = JSON.parse(bytes.toString('utf8'))
  } catch (error) {
    throw new Error(`composition manifest is not valid JSON: ${error.message}`)
  }
  if (!parsed || parsed.version !== 1 || parsed.catalogVersion !== 'r1' || !Array.isArray(parsed.slides)) {
    throw new Error('composition manifest must declare version 1, catalogVersion r1, and slides[]')
  }
  const seen = new Set()
  const slides = parsed.slides.map((item) => {
    const slide = Number(item.slide)
    if (!Number.isInteger(slide) || slide < 1 || seen.has(slide)) throw new Error('composition manifest slide identities must be unique positive integers')
    seen.add(slide)
    if (typeof item.name !== 'string' || typeof item.variant !== 'string' || typeof item.family !== 'string' || !Array.isArray(item.slots) || item.slots.some((slot) => typeof slot !== 'string')) {
      throw new Error(`composition manifest slide ${slide} has invalid composition fields`)
    }
    return { slide, name: item.name, variant: item.variant, family: item.family, slots: [...item.slots] }
  })
  return { catalogVersion: parsed.catalogVersion, slides }
}

async function readSlides(directory) {
  const entries = await fs.readdir(directory)
  const slides = []
  for (const entry of entries) {
    const match = /^slide-(\d+)\.png$/.exec(entry)
    if (!match) continue
    const slide = Number(match[1])
    const bytes = await fs.readFile(path.join(directory, entry))
    slides.push({ slide, file: entry, bytes, digest: sha256(bytes) })
  }
  slides.sort((left, right) => left.slide - right.slide)
  return slides
}

function manifestDigest(name, slides) {
  const hash = crypto.createHash('sha256')
  hash.update('margo-marpit-compare/v1\0')
  hash.update(name + '\0')
  for (const slide of slides) hash.update(`${slide.file}\0${slide.bytes.byteLength}\0${slide.digest}\0`)
  return hash.digest('hex')
}

function renderCard(slide, margoSlides, marpitSlides, category, margoAssetRoot, marpitAssetRoot, composition) {
  const margo = margoSlides.find((item) => item.slide === slide)
  const marpit = marpitSlides.find((item) => item.slide === slide)
  const number = String(slide).padStart(2, '0')
  const summary = `${category.name}: ${category.description}. Compare renderer-specific theme, layout and extension output.`
  const compositionMarkup = composition && composition.name
    ? `<p class="pair-composition" data-composition="${escapeHTML(composition.name)}"><strong>Composition:</strong> <code>${escapeHTML(composition.name)}</code> · <strong>Variant:</strong> <code>${escapeHTML(composition.variant ?? '')}</code> · <strong>Family:</strong> <code>${escapeHTML(composition.family ?? '')}</code> · <strong>Slots:</strong> <code>${escapeHTML((composition.slots ?? []).join(', '))}</code></p>`
    : ''
  return `<article class="pair" data-slide="${slide}" data-category="${escapeHTML(category.name)}">
    <div class="pair-heading"><h3>Slide ${slide}</h3><select class="pair-status" data-review-state aria-label="Review state for slide ${slide}"><option value="unreviewed">Unreviewed</option><option value="reviewed">Reviewed</option><option value="needs-attention">Needs attention</option></select></div>
    <div class="captures">
      ${renderCapture('Margo', margo, `Margo slide ${slide}; ${summary}`, `Slide ${slide} · Margo · 1:1`, margoAssetRoot)}
      ${renderCapture('Marpit reference', marpit, `Marpit reference slide ${slide}; ${summary}`, `Slide ${slide} · Marpit reference · 1:1`, marpitAssetRoot)}
    </div>
    ${compositionMarkup}
    <details class="pair-meta"><summary>Review notes</summary><p><strong>Expected:</strong> renderer-specific theme and extension differences. <strong>Check:</strong> slide boundaries, reading order, overflow, legibility, font metrics and PDF geometry. <code>${escapeHTML(margoAssetRoot)}/slide-${number}.png</code> · <code>${escapeHTML(marpitAssetRoot)}/slide-${number}.png</code></p></details>
  </article>`
}

function renderCapture(label, image, alt, zoomTitle, assetRoot) {
  return `<figure class="capture capture--${label.startsWith('Margo') ? 'margo' : 'marpit'}">
    <div class="capture__label"><span>${escapeHTML(label)}</span><span>PNG · 16:9 · 960×540</span></div>
    <button class="image-button" type="button" data-open-zoom data-zoom-title="${escapeHTML(zoomTitle)}"><img src="${escapeHTML(`${assetRoot}/${image.file}`)}" alt="${escapeHTML(alt)}" width="960" height="540" loading="lazy"></button>
    <figcaption>Open capture at 1:1 for text, spacing and raster comparison.</figcaption>
  </figure>`
}

function sha256(bytes) {
  return crypto.createHash('sha256').update(bytes).digest('hex')
}

function slug(value) {
  return value.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
}

function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, (character) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[character])
}
