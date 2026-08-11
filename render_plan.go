package margo

import (
	"bytes"
	"context"
	"fmt"

	"github.com/yuin/goldmark/ast"
	tableast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

type renderPlan struct {
	compilerFingerprint CompilerConfigFingerprint
	documentFingerprint DocumentFingerprint
	effectivePolicy     EffectivePolicy
	registrations       []ExtensionRegistration
	nodes               []plannedExtensionNode
	htmlRequirements    HTMLRequirements
}

const extensionSlotAttribute = "margo-extension-slot"

type plannedExtensionNode struct {
	ExtensionNode
	registrationIndex int
	slot              uint32
}

func (n plannedExtensionNode) clone() plannedExtensionNode {
	n.ExtensionNode = n.ExtensionNode.clone()
	return n
}

func extensionSlot(node *ast.FencedCodeBlock) (uint32, bool) {
	value, ok := node.AttributeString(extensionSlotAttribute)
	slot, valid := value.(uint32)
	return slot, ok && valid
}

func (p renderPlan) clone() renderPlan {
	registrations := p.registrations
	nodes := p.nodes
	p.registrations = make([]ExtensionRegistration, len(registrations))
	for i, registration := range registrations {
		p.registrations[i] = cloneRegistration(registration)
	}
	p.nodes = make([]plannedExtensionNode, len(nodes))
	for i, node := range nodes {
		p.nodes[i] = node.clone()
	}
	p.htmlRequirements = HTMLRequirements{requirements: p.htmlRequirements.List()}
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
		fenceOwners := make(map[string]int)
		for registrationIndex, registration := range plan.registrations {
			for _, fence := range registration.Fences {
				fenceOwners[fence] = registrationIndex
			}
		}
		extensionOrdinals := make(map[int]uint32)
		usedRegistrations := make(map[int]struct{})
		requirementCandidates, err := coreHTMLRequirements(false)
		if err != nil {
			return renderPlan{}, err
		}
		hasTable := false
		compileContext := extensionCompileContext{normalized: normalized}
		var missing *Diagnostic
		walkErr := ast.Walk(parsed.root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			if _, table := node.(*tableast.Table); table {
				hasTable = true
				return ast.WalkContinue, nil
			}
			if node.Kind() != ast.KindFencedCodeBlock {
				return ast.WalkContinue, nil
			}
			fenced := node.(*ast.FencedCodeBlock)
			body := parsed.frontmatter.body
			fence := string(fenced.Language(body))
			if fence == "" {
				return ast.WalkContinue, nil
			}
			if fence == "trusted-embed" {
				line := lineAtOffset(source.Content, parsed.frontmatter.bodyOffset+segmentAtStart(fenced.Lines()))
				return ast.WalkStop, diagnosticAt("source.trusted_embed_removed", source.Name, "/embed", "trusted-embed was removed; use a standard HTML <iframe src=\"https://…\" title=\"…\" width=\"640\" height=\"360\"></iframe> authorized by the host policy", line, 1)
			}
			registrationIndex, registered := fenceOwners[fence]
			if !registered {
				if fence != "goshtosochart" {
					return ast.WalkContinue, nil
				}
				missing = &Diagnostic{Code: "extension.missing_integration", Severity: SeverityError, Source: source.Name, Message: "goshtosochart requires the charts extension"}
				return ast.WalkStop, nil
			}
			segment := fenced.Lines()
			payload := append([]byte(nil), segment.Value(body)...)
			extensionNode := ExtensionNode{
				Fence:   fence,
				Payload: payload,
				Source:  SourcePosition{Source: source.Name, Line: lineAtOffset(source.Content, parsed.frontmatter.bodyOffset+segmentAtStart(segment)), Column: 1},
			}
			registration := plan.registrations[registrationIndex]
			if _, used := usedRegistrations[registrationIndex]; !used {
				requirements, err := extensionRegistrationHTMLRequirements(registration)
				if err != nil {
					return ast.WalkStop, err
				}
				requirementCandidates = append(requirementCandidates, requirements...)
				usedRegistrations[registrationIndex] = struct{}{}
			}
			if registration.compile != nil {
				ordinal := extensionOrdinals[registrationIndex]
				compiled, err := registration.compile(compileContext, extensionNode.clone(), ordinal)
				if err != nil {
					return ast.WalkStop, err
				}
				extensionNode = compiled
				extensionOrdinals[registrationIndex] = ordinal + 1
			}
			slot := uint32(len(plan.nodes))
			fenced.SetAttributeString(extensionSlotAttribute, slot)
			plan.nodes = append(plan.nodes, plannedExtensionNode{
				ExtensionNode:     extensionNode,
				registrationIndex: registrationIndex,
				slot:              slot,
			})
			return ast.WalkContinue, nil
		})
		if walkErr != nil {
			return renderPlan{}, walkErr
		}
		if missing != nil {
			return renderPlan{}, &DiagnosticError{Diagnostics: []Diagnostic{*missing}}
		}
		if hasTable {
			coreWithTable, err := coreHTMLRequirements(true)
			if err != nil {
				return renderPlan{}, err
			}
			requirementCandidates = append(requirementCandidates, coreWithTable[len(coreWithTable)-1])
		}
		requirements, err := mergeHTMLRequirements(requirementCandidates)
		if err != nil {
			return renderPlan{}, err
		}
		plan.htmlRequirements = requirements
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
func executeRenderPlanSlots(ctx context.Context, plan renderPlan) ([][]byte, error) {
	slots := make([][]byte, len(plan.nodes))
	renderContext := RenderContext{EffectivePolicy: plan.effectivePolicy}
	for registrationIndex, registration := range plan.registrations {
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
		for _, planned := range plan.nodes {
			if planned.registrationIndex != registrationIndex {
				continue
			}
			if int(planned.slot) >= len(slots) {
				return nil, fmt.Errorf("extension.slot_invalid: %d", planned.slot)
			}
			var buffer bytes.Buffer
			if err := session.Render(ctx, planned.ExtensionNode.clone(), &buffer); err != nil {
				return nil, err
			}
			slots[planned.slot] = append([]byte(nil), buffer.Bytes()...)
		}
	}
	return slots, nil
}

func executeRenderPlan(ctx context.Context, plan renderPlan) ([]byte, error) {
	slots, err := executeRenderPlanSlots(ctx, plan)
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	for _, slot := range slots {
		if _, err := buffer.Write(slot); err != nil {
			return nil, err
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

func lineAtOffset(source []byte, offset int) int {
	if offset < 0 || offset > len(source) {
		return 1
	}
	return 1 + bytes.Count(source[:offset], []byte{'\n'})
}
