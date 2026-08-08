import * as csstree from "css-tree";

const svgElements = new Set(["svg", "g", "path", "rect", "text"]);
const cssProperties = new Set(["color", "fill", "font-family", "font-size", "stroke"]);

export async function probeDOM(page, svgSource, cssSource) {
  const dom = await page.evaluate(({ svgSource, cssSource }) => {
    const documentNode = new DOMParser().parseFromString(svgSource, "image/svg+xml");
    if (documentNode.querySelector("parsererror")) {
      throw new Error("margo.harness_svg_malformed");
    }
    const names = [...documentNode.querySelectorAll("*")].map((node) => node.localName);
    const serialized = new XMLSerializer().serializeToString(documentNode.documentElement);
    const style = document.createElement("style");
    style.textContent = cssSource;
    document.head.append(style);
    const cssRuleCount = style.sheet?.cssRules.length ?? 0;
    const probe = document.createElement("div");
    probe.className = "probe";
    document.body.append(probe);
    const computedColor = getComputedStyle(probe).color;
    return { names, serialized, cssRuleCount, computedColor };
  }, { svgSource, cssSource });

  for (const name of dom.names) {
    if (!svgElements.has(name)) {
      throw new Error(`margo.harness_svg_element_unknown:${name}`);
    }
  }

  const ast = csstree.parse(cssSource, { context: "stylesheet", positions: false });
  csstree.walk(ast, (node) => {
    if (node.type === "Atrule") {
      throw new Error(`margo.harness_css_at_rule_forbidden:${node.name}`);
    }
    if (node.type === "Declaration" && !cssProperties.has(node.property)) {
      throw new Error(`margo.harness_css_property_unknown:${node.property}`);
    }
  });
  return dom;
}
