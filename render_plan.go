package margo

import (
	"bytes"
	"context"
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type renderPlan struct {
	compilerFingerprint CompilerConfigFingerprint
	documentFingerprint DocumentFingerprint
	effectivePolicy     EffectivePolicy
	registrations       []ExtensionRegistration
	nodes               []ExtensionNode
}

func (p renderPlan) clone() renderPlan {
	registrations := p.registrations
	nodes := p.nodes
	p.registrations = make([]ExtensionRegistration, len(registrations))
	for i, registration := range registrations {
		p.registrations[i] = cloneRegistration(registration)
	}
	p.nodes = make([]ExtensionNode, len(nodes))
	for i, node := range nodes {
		p.nodes[i] = node.clone()
	}
	return p
}

func buildRenderPlan(source Source, normalized sourceNormalization, registry extensionRegistry, compilerFingerprint CompilerConfigFingerprint, documentFingerprint DocumentFingerprint, effectivePolicy EffectivePolicy) (renderPlan, error) {
	plan := renderPlan{
		compilerFingerprint: compilerFingerprint,
		documentFingerprint: documentFingerprint,
		effectivePolicy:     effectivePolicy,
		registrations:       registry.clone().registrations,
	}
	if parsed, ok := normalized.parsed.(normalizedMarkdown); ok && parsed.root != nil {
		fenceOwners := make(map[string]struct{})
		for _, registration := range plan.registrations {
			for _, fence := range registration.Fences {
				fenceOwners[fence] = struct{}{}
			}
		}
		var missing *Diagnostic
		_ = ast.Walk(parsed.root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering || node.Kind() != ast.KindFencedCodeBlock {
				return ast.WalkContinue, nil
			}
			fenced := node.(*ast.FencedCodeBlock)
			fence := string(fenced.Language(source.Content))
			if fence == "" {
				return ast.WalkContinue, nil
			}
			if _, registered := fenceOwners[fence]; !registered {
				if fence != "goshtosochart" {
					return ast.WalkContinue, nil
				}
				missing = &Diagnostic{Code: "extension.missing_integration", Severity: SeverityError, Source: source.Name, Message: "goshtosochart requires the charts extension"}
				return ast.WalkStop, nil
			}
			segment := fenced.Lines()
			payload := append([]byte(nil), segment.Value(source.Content)...)
			plan.nodes = append(plan.nodes, ExtensionNode{
				Fence:   fence,
				Payload: payload,
				Source:  SourcePosition{Source: source.Name, Line: lineAtOffset(source.Content, segmentAtStart(segment)), Column: 1},
			})
			return ast.WalkContinue, nil
		})
		if missing != nil {
			return renderPlan{}, &DiagnosticError{Diagnostics: []Diagnostic{*missing}}
		}
	}
	return plan, nil
}

func segmentAtStart(segments *text.Segments) int {
	if segments == nil || segments.Len() == 0 {
		return 0
	}
	return segments.At(0).Start
}

// executeRenderPlan creates all sessions only after the binding check and
// spools extension bytes privately so a failed session cannot partially write
// a caller's component.
func executeRenderPlan(ctx context.Context, plan renderPlan) ([]byte, error) {
	var buffer bytes.Buffer
	renderContext := RenderContext{EffectivePolicy: plan.effectivePolicy}
	for _, registration := range plan.registrations {
		if err := contextError(ctx); err != nil {
			return nil, err
		}
		session, err := callExtensionFactory(registration.Factory, renderContext)
		if err != nil {
			return nil, err
		}
		if session == nil {
			return nil, fmt.Errorf("extension.session_invalid: %s returned nil", registration.Identity.Name)
		}
		for _, node := range plan.nodes {
			if !ownsFence(registration, node.Fence) {
				continue
			}
			if err := session.Render(ctx, node.clone(), &buffer); err != nil {
				return nil, err
			}
		}
	}
	return buffer.Bytes(), nil
}

func callExtensionFactory(factory ExtensionFactory, context RenderContext) (session ExtensionSession, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("extension.factory_panic: %v", recovered)
		}
	}()
	return factory(context)
}

func ownsFence(registration ExtensionRegistration, fence string) bool {
	for _, owned := range registration.Fences {
		if owned == fence {
			return true
		}
	}
	return false
}

func lineAtOffset(source []byte, offset int) int {
	if offset < 0 || offset > len(source) {
		return 1
	}
	return 1 + bytes.Count(source[:offset], []byte{'\n'})
}
