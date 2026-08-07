package margo

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io"

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
	registration := ExtensionRegistration{
		Identity: ExtensionIdentity{
			Name:              "mermaid",
			Version:           internalmermaid.RuntimeVersion,
			ConfigurationHash: hex.EncodeToString(configurationHash[:]),
			Capabilities:      []string{"runtime-task", "strict-configuration"},
		},
		Fences:  []string{"mermaid"},
		Factory: func(RenderContext) (ExtensionSession, error) { return mermaidSession{}, nil },
		compile: compileMermaidNode,
	}
	return WithExtension(registration)(config)
}

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

func validateMermaidMode(normalized sourceNormalization, source string) error {
	parsed, ok := normalized.parsed.(normalizedMarkdown)
	if !ok || parsed.frontmatter.goshtoso == nil {
		return nil
	}
	security, ok := parsed.frontmatter.goshtoso["security"].(map[string]any)
	if !ok {
		return nil
	}
	value, configured := security["mermaid"]
	if !configured {
		return nil
	}
	mode, ok := value.(string)
	if !ok || mode != "strict" {
		return diagnosticAt(internalmermaid.ConfigurationForbiddenCode, source, "/goshtoso/security/mermaid", "Mermaid mode must be the literal strict value", 1, 1)
	}
	return nil
}
