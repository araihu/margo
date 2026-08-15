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
			Capabilities:      append([]string{"runtime-task", "strict-configuration"}, browserCapabilities...),
		},
		Fences:  []string{"mermaid"},
		Factory: func(RenderContext) (ExtensionSession, error) { return mermaidSession{}, nil },
		compile: compileMermaidNode,
	}
	return WithExtension(registration)(config)
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
    const outputs = [];
    for (let index = 0; index < nodes.length; index += 1) {
      const node = nodes[index];
      const sourceNode = node.querySelector(".margo-mermaid__source code");
      const target = node.querySelector(".margo-mermaid__canvas");
      if (!sourceNode || !target) throw new Error("malformed Mermaid runtime marker");
      engine.initialize({
        startOnLoad: false,
        securityLevel: "strict",
        htmlLabels: false,
        flowchart: {htmlLabels: false},
        look: "classic",
        layout: "dagre",
        deterministicIds: true,
        deterministicIDSeed: "margo-html-" + index
      });
      const rendered = await engine.render("margo-html-" + index, sourceNode.textContent);
      if (!rendered || typeof rendered.svg !== "string" || rendered.svg.length === 0) throw new Error("Mermaid returned no SVG");
      target.innerHTML = rendered.svg;
      const svg = target.querySelector("svg");
      if (svg) {
        svg.setAttribute("aria-hidden", "true");
        svg.setAttribute("focusable", "false");
      }
      const fallback = node.querySelector(".margo-mermaid__source");
      if (fallback) fallback.hidden = true;
      node.dataset.margoRuntimeStatus = "succeeded";
      outputs.push(rendered.svg);
    }
    document.documentElement.dataset.margoRuntimeStatus = "ready";
    return outputs;
  };
  globalThis.margoRuntimeReady = execute().catch((error) => {
    document.documentElement.dataset.margoRuntimeStatus = "failed";
    for (const node of document.querySelectorAll('[data-margo-runtime-task="mermaid"]')) node.dataset.margoRuntimeStatus = "failed";
    throw error;
  });
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
