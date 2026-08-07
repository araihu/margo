function clamp(value) {
  return Math.max(0, Math.min(1, value));
}

function parseAlpha(value) {
  if (value == null || value === "") return 1;
  const parsed = Number.parseFloat(value);
  return value.endsWith("%") ? parsed / 100 : parsed;
}

function parseRGBChannel(value) {
  const parsed = Number.parseFloat(value);
  return value.endsWith("%") ? parsed / 100 : parsed / 255;
}

function parseColor(value) {
  const normalized = value.trim().toLowerCase();
  if (normalized === "transparent") return [0, 0, 0, 0];
  if (normalized.startsWith("#")) {
    const hex = normalized.slice(1);
    const expanded = hex.length === 3 || hex.length === 4
      ? [...hex].map((part) => `${part}${part}`).join("")
      : hex;
    if (![6, 8].includes(expanded.length)) return null;
    return [
      Number.parseInt(expanded.slice(0, 2), 16) / 255,
      Number.parseInt(expanded.slice(2, 4), 16) / 255,
      Number.parseInt(expanded.slice(4, 6), 16) / 255,
      expanded.length === 8 ? Number.parseInt(expanded.slice(6, 8), 16) / 255 : 1,
    ];
  }
  const rgb = normalized.match(/^rgba?\((.*)\)$/);
  if (rgb) {
    const parts = rgb[1].replace(/\//g, " ").split(/[\s,]+/).filter(Boolean);
    if (parts.length < 3) return null;
    return [parseRGBChannel(parts[0]), parseRGBChannel(parts[1]), parseRGBChannel(parts[2]), parseAlpha(parts[3])];
  }
  const oklch = normalized.match(/^oklch\((.*)\)$/);
  if (oklch) {
    const parts = oklch[1].replace(/\//g, " / ").split(/[\s]+/).filter(Boolean);
    const slash = parts.indexOf("/");
    const channels = slash === -1 ? parts : parts.slice(0, slash);
    if (channels.length < 3) return null;
    let lightness = Number.parseFloat(channels[0]);
    if (channels[0].endsWith("%") || lightness > 1) lightness /= 100;
    let chroma = Number.parseFloat(channels[1]);
    if (channels[1].endsWith("%")) chroma /= 100;
    const hue = Number.parseFloat(channels[2]) * Math.PI / 180;
    const alpha = slash === -1 ? 1 : parseAlpha(parts[slash + 1]);
    const a = chroma * Math.cos(hue);
    const b = chroma * Math.sin(hue);
    const l = (lightness + 0.3963377774 * a + 0.2158037573 * b) ** 3;
    const m = (lightness - 0.1055613458 * a - 0.0638541728 * b) ** 3;
    const s = (lightness - 0.0894841775 * a - 1.291485548 * b) ** 3;
    const linear = [
      4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s,
      -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s,
      -0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s,
    ];
    return [...linear.map((channel) => channel <= 0.0031308
      ? 12.92 * channel
      : 1.055 * (channel ** (1 / 2.4)) - 0.055), alpha];
  }
  return null;
}

function composite(foreground, background, alpha = foreground[3]) {
  const opacity = clamp(alpha);
  return [
    foreground[0] * opacity + background[0] * (1 - opacity),
    foreground[1] * opacity + background[1] * (1 - opacity),
    foreground[2] * opacity + background[2] * (1 - opacity),
    1,
  ];
}

function luminance(color) {
  return color.slice(0, 3).map((channel) => channel <= 0.03928
    ? channel / 12.92
    : ((channel + 0.055) / 1.055) ** 2.4).reduce((sum, channel, index) => sum + channel * [0.2126, 0.7152, 0.0722][index], 0);
}

function ratio(foreground, background) {
  const light = Math.max(luminance(foreground), luminance(background));
  const dark = Math.min(luminance(foreground), luminance(background));
  return (light + 0.05) / (dark + 0.05);
}

export async function auditTextContrast(page) {
  return page.evaluate(({ parseColorSource }) => {
    const { parseColor, composite, ratio } = new Function(parseColorSource)();
    const body = document.body;
    const rootBackground = parseColor(getComputedStyle(document.documentElement).backgroundColor) ?? [1, 1, 1, 1];
    const elements = [];
    const walker = document.createTreeWalker(body, NodeFilter.SHOW_TEXT);
    let node;
    while ((node = walker.nextNode())) {
      if (!node.textContent?.trim()) continue;
      const element = node.parentElement;
      if (!element || ["SCRIPT", "STYLE", "NOSCRIPT"].includes(element.tagName) || element.closest("svg")) continue;
      const style = getComputedStyle(element);
      if (style.display === "none" || style.visibility === "hidden" || Number.parseFloat(style.opacity) === 0) continue;
      if (!element.getClientRects().length || element.closest('[aria-hidden="true"]')) continue;
      elements.push({ element, text: node.textContent.trim().replace(/\s+/g, " ") });
    }

    const failures = [];
    for (const { element, text } of elements) {
      const chain = [];
      for (let current = element; current && current !== document.documentElement; current = current.parentElement) chain.unshift(current);
      let background = rootBackground;
      let inheritedOpacity = 1;
      for (const current of chain) {
        const style = getComputedStyle(current);
        const layer = parseColor(style.backgroundColor);
        inheritedOpacity *= Number.parseFloat(style.opacity) || 1;
        if (layer) background = composite(layer, background, layer[3] * (Number.parseFloat(style.opacity) || 1));
      }
      const style = getComputedStyle(element);
      const foreground = parseColor(style.color);
      if (!foreground) continue;
      const effectiveForeground = composite(foreground, background, foreground[3] * inheritedOpacity);
      const actualRatio = ratio(effectiveForeground, background);
      const fontSize = Number.parseFloat(style.fontSize) || 16;
      const large = fontSize >= 24 || (fontSize >= 18.66 && Number.parseInt(style.fontWeight, 10) >= 700);
      const required = large ? 3 : 4.5;
      if (actualRatio + 0.01 < required) {
        failures.push({
          selector: `${element.tagName.toLowerCase()}${element.id ? `#${element.id}` : ""}${[...element.classList].slice(0, 3).map((name) => `.${name}`).join("")}`,
          text: text.slice(0, 120),
          foreground: style.color,
          background: getComputedStyle(element).backgroundColor,
          ratio: Number(actualRatio.toFixed(2)),
          required,
        });
      }
    }
    return { checked: elements.length, failures };
  }, {
    parseColorSource: [
      `const clamp = ${clamp.toString()};`,
      `const parseAlpha = ${parseAlpha.toString()};`,
      `const parseRGBChannel = ${parseRGBChannel.toString()};`,
      `const parseColor = ${parseColor.toString()};`,
      `const composite = ${composite.toString()};`,
      `const luminance = ${luminance.toString()};`,
      `const ratio = ${ratio.toString()};`,
      "return { parseColor, composite, ratio };",
    ].join("\n"),
  });
}

export { ratio };
