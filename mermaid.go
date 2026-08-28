package margo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/araihu/margo/assets"
	internalmermaid "github.com/araihu/margo/internal/mermaid"
)

type mermaidCompiledTask struct {
	descriptor internalmermaid.TaskDescriptor
}

func (task mermaidCompiledTask) cloneCompiledExtensionNode() compiledExtensionNode {
	return task
}

type mermaidSession struct{}

func (mermaidSession) Render(context.Context, ExtensionNode, io.Writer) error {
	// M2 freezes task input only. M6 owns browser execution and accepted SVG.
	return nil
}

func installDefaultExtensions(config *compilerConfig) error {
	configurationHash := internalmermaid.StrictConfigurationHash()
	browserCapabilities, err := mermaidBrowserCapabilities()
	if err != nil {
		return err
	}
	registration := ExtensionRegistration{
		Identity: ExtensionIdentity{
			Name:              "mermaid",
			Version:           internalmermaid.RuntimeVersion,
			ConfigurationHash: hex.EncodeToString(configurationHash[:]),
			Capabilities:      append([]string{"runtime-task", "strict-configuration", "NamespacedIDsV1"}, browserCapabilities...),
		},
		Fences:  []string{"mermaid"},
		Factory: func(RenderContext) (ExtensionSession, error) { return mermaidSession{}, nil },
		compile: compileMermaidNode,
	}
	if err := WithExtension(registration)(config); err != nil {
		return err
	}
	if err := WithExtension(jsonSchemaExtensionRegistration())(config); err != nil {
		return err
	}
	return nil
}

var mermaidBrowserCapabilityCache struct {
	sync.Once
	capabilities []string
	err          error
}

func mermaidBrowserCapabilities() ([]string, error) {
	mermaidBrowserCapabilityCache.Do(func() {
		file, err := assets.MuambaOpen("mermaid", "browser-bundle")
		if err != nil {
			mermaidBrowserCapabilityCache.err = fmt.Errorf("mermaid.runtime_unavailable: %w", err)
			return
		}
		defer file.Close()
		bundle, err := io.ReadAll(file)
		if err != nil {
			mermaidBrowserCapabilityCache.err = fmt.Errorf("mermaid.runtime_unavailable: %w", err)
			return
		}
		bundleDigest := sha256.Sum256(bundle)
		executor := []byte(mermaidBrowserExecutor)
		executorDigest := sha256.Sum256(executor)
		runtimeCapability, err := HTMLRequirementCapability(HTMLRequirement{
			ID:       "margo.mermaid.runtime",
			Kind:     HTMLScript,
			LocalURL: "/margo-assets/mermaid/11.16.1/mermaid.min.js",
			Inline: AssetRef{
				Path: "mermaid.min.js", MediaType: "application/javascript",
				SHA256: hex.EncodeToString(bundleDigest[:]), Content: bundle,
			},
		})
		if err != nil {
			mermaidBrowserCapabilityCache.err = err
			return
		}
		executorCapability, err := HTMLRequirementCapability(HTMLRequirement{
			ID:        "margo.mermaid.execute",
			Kind:      HTMLScript,
			LocalURL:  "/margo-assets/runtime/mermaid-run.js",
			LoadAfter: []string{"margo.mermaid.runtime"},
			Inline: AssetRef{
				Path: "mermaid-run.js", MediaType: "application/javascript",
				SHA256: hex.EncodeToString(executorDigest[:]), Content: executor,
			},
		})
		if err != nil {
			mermaidBrowserCapabilityCache.err = err
			return
		}
		mermaidBrowserCapabilityCache.capabilities = []string{runtimeCapability, executorCapability}
	})
	return append([]string(nil), mermaidBrowserCapabilityCache.capabilities...), mermaidBrowserCapabilityCache.err
}

const mermaidBrowserExecutor = `(() => {
  "use strict";

  function materializeTreeViewIcons(svg) {
    // Mermaid strict mode strips <use>; only restore fixed built-in paths.
    const tree = svg.querySelector(".tree-view");
    if (!tree) return;
    const paths = Object.freeze({
      folder: "M10.59 4.59A2 2 0 0 0 9.17 4H4a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.17z",
      file: "M6 2a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8.83a2 2 0 0 0-.59-1.42l-4.82-4.82A2 2 0 0 0 13.17 2H6Zm7.5 1.9l4.6 4.6h-3.6a1 1 0 0 1-1-1V3.9Z"
    });
    for (const group of [...tree.children]) {
      if (group.localName !== "g") continue;
      const label = [...group.children].find((child) => child.localName === "text" && child.classList.contains("treeView-node-label"));
      if (!label || [...group.children].some((child) => child.classList.contains("treeView-node-icon"))) continue;
      const x = Number(label.getAttribute("x"));
      const y = Number(label.getAttribute("y"));
      if (!Number.isFinite(x) || !Number.isFinite(y)) continue;
      const remainder = ((x - 5) % 15 + 15) % 15;
      if (Math.abs(remainder - 3) > 0.5) continue;
      const type = label.classList.contains("treeView-node-dir") ? "folder" : "file";
      const icon = svg.ownerDocument.createElementNS("http://www.w3.org/2000/svg", "path");
      icon.setAttribute("class", "treeView-node-icon");
      icon.setAttribute("d", paths[type]);
      icon.setAttribute("fill", "currentColor");
      if (type === "file") {
        icon.setAttribute("fill-rule", "evenodd");
        icon.setAttribute("clip-rule", "evenodd");
      }
      icon.setAttribute("transform", "translate(" + (x - 18) + " " + (y - 7) + ") scale(" + (14 / 24) + ")");
      group.insertBefore(icon, label);
    }
  }

  function margoMermaidConfiguration() {
    const styles = getComputedStyle(document.documentElement);
    const read = (name) => styles.getPropertyValue(name).trim();
    const canvas = read("--margo-mermaid-canvas");
    const node = read("--margo-mermaid-node");
    const nodeBorder = read("--margo-mermaid-node-border");
    const text = read("--margo-mermaid-text");
    const edge = read("--margo-mermaid-edge");
    const edgeLabel = read("--margo-mermaid-edge-label");
    const edgeLabelBackground = read("--margo-mermaid-edge-label-background");
    if (!canvas || !node || !nodeBorder || !text || !edge || !edgeLabel || !edgeLabelBackground) return {};
    const themeVariables = {
      background: canvas,
      darkMode: document.documentElement.classList.contains("dark"),
      fontFamily: read("--font-body"),
      primaryColor: node,
      primaryTextColor: text,
      primaryBorderColor: nodeBorder,
      secondaryColor: canvas,
      secondaryTextColor: text,
      secondaryBorderColor: nodeBorder,
      tertiaryColor: canvas,
      tertiaryTextColor: text,
      tertiaryBorderColor: nodeBorder,
      textColor: edgeLabel,
      titleColor: text,
      lineColor: edge,
      defaultLinkColor: edge,
      arrowheadColor: edge,
      nodeBkg: node,
      nodeBorder,
      nodeTextColor: text,
      clusterBkg: canvas,
      clusterBorder: nodeBorder,
      edgeLabelBackground,
      labelBackground: edgeLabelBackground
    };
    return {theme: "base", themeVariables};
  }

  function margoMermaidThemeSignature() {
    const styles = getComputedStyle(document.documentElement);
    const read = (name) => styles.getPropertyValue(name).trim();
    const values = [
      document.documentElement.classList.contains("dark") ? "dark" : "light",
      read("--margo-mermaid-canvas"),
      read("--margo-mermaid-node"),
      read("--margo-mermaid-node-border"),
      read("--margo-mermaid-text"),
      read("--margo-mermaid-edge"),
      read("--margo-mermaid-edge-label"),
      read("--margo-mermaid-edge-label-background")
    ];
    return values.slice(1).every(Boolean) ? values.join("|") : "";
  }

  const execute = async () => {
    if (document.readyState === "loading") {
      await new Promise((resolve) => document.addEventListener("DOMContentLoaded", resolve, {once: true}));
    }
    const nodes = [...document.querySelectorAll('[data-margo-runtime-task="mermaid"]')];
    if (nodes.length === 0) {
      document.documentElement.dataset.margoRuntimeStatus = "ready";
      return [];
    }
    const engine = globalThis.mermaid;
    if (!engine || typeof engine.initialize !== "function" || typeof engine.render !== "function") {
      throw new Error("embedded Mermaid runtime is unavailable");
    }
    const mermaidConfiguration = margoMermaidConfiguration();
    const outputs = [];
    for (let index = 0; index < nodes.length; index += 1) {
      const node = nodes[index];
      const sourceNode = node.querySelector(".margo-mermaid__source code");
      const target = node.querySelector(".margo-mermaid__canvas");
      if (!sourceNode || !target) throw new Error("malformed Mermaid runtime marker");
      engine.initialize({
        ...mermaidConfiguration,
        startOnLoad: false,
        securityLevel: "strict",
        htmlLabels: false,
        flowchart: {htmlLabels: false},
        look: "classic",
        layout: "dagre",
        treeView: {showIcons: true},
        deterministicIds: true,
        deterministicIDSeed: "margo-html-" + index
      });
      const rendered = await engine.render("margo-html-" + index, sourceNode.textContent);
      if (!rendered || typeof rendered.svg !== "string" || rendered.svg.length === 0) throw new Error("Mermaid returned no SVG");
      target.innerHTML = rendered.svg;
      const svg = target.querySelector("svg");
      if (svg) {
        materializeTreeViewIcons(svg);
        svg.setAttribute("aria-hidden", "true");
        svg.setAttribute("focusable", "false");
      }
      const fallback = node.querySelector(".margo-mermaid__source");
      if (fallback) fallback.hidden = true;
      node.dataset.margoRuntimeStatus = "succeeded";
      outputs.push(svg ? svg.outerHTML : rendered.svg);
    }
    document.documentElement.dataset.margoRuntimeStatus = "ready";
    return outputs;
  };

  let themeRenderQueue = Promise.resolve();
  function observeMermaidTheme() {
    if (typeof MutationObserver !== "function") return;
    if (globalThis.margoMermaidThemeObserver) globalThis.margoMermaidThemeObserver.disconnect();
    let signature = margoMermaidThemeSignature();
    if (!signature) return;
    const observer = new MutationObserver(() => {
      const next = margoMermaidThemeSignature();
      if (!next || next === signature) return;
      signature = next;
      themeRenderQueue = themeRenderQueue.catch(() => {}).then(() => execute()).catch(() => {
        document.documentElement.dataset.margoRuntimeStatus = "failed";
        for (const node of document.querySelectorAll('[data-margo-runtime-task="mermaid"]')) node.dataset.margoRuntimeStatus = "failed";
      });
      globalThis.margoRuntimeReady = themeRenderQueue;
    });
    observer.observe(document.documentElement, {attributes: true, attributeFilter: ["class", "data-theme"]});
    globalThis.margoMermaidThemeObserver = observer;
  }

  const run = () => {
    const promise = execute().then((outputs) => {
      observeMermaidTheme();
      return outputs;
    }).catch((error) => {
      document.documentElement.dataset.margoRuntimeStatus = "failed";
      for (const node of document.querySelectorAll('[data-margo-runtime-task="mermaid"]')) node.dataset.margoRuntimeStatus = "failed";
      throw error;
    });
    globalThis.margoRuntimeReady = promise;
    return promise;
  };
  globalThis.margoRunMermaid = run;
  globalThis.margoRuntimeReady = run();
})();`

func compileMermaidNode(compileContext extensionCompileContext, node ExtensionNode, ordinal uint32) (ExtensionNode, error) {
	if err := validateMermaidMode(compileContext.normalized, node.Source.Source); err != nil {
		return ExtensionNode{}, err
	}
	descriptor, err := internalmermaid.Compile(node.Payload, ordinal)
	if err != nil {
		line, column := node.Source.Line, node.Source.Column
		var internalDiagnostic *internalmermaid.DiagnosticError
		if errors.As(err, &internalDiagnostic) {
			line += bytes.Count(node.Payload[:internalDiagnostic.Offset], []byte{'\n'})
			lastLine := bytes.LastIndexByte(node.Payload[:internalDiagnostic.Offset], '\n')
			if lastLine >= 0 {
				column = internalDiagnostic.Offset - lastLine
			} else {
				column += internalDiagnostic.Offset
			}
		}
		return ExtensionNode{}, newDiagnosticError(Diagnostic{
			Code:     internalmermaid.ConfigurationForbiddenCode,
			Severity: SeverityError,
			Source:   node.Source.Source,
			Line:     line,
			Column:   column,
			Message:  err.Error(),
		})
	}
	node.compiled = mermaidCompiledTask{descriptor: descriptor}
	return node, nil
}

func validateMermaidMode(sourceNormalization, string) error { return nil }
