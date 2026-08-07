package margo

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
)

// RawHTMLMode is the versioned raw-HTML capability vocabulary.
type RawHTMLMode string

const (
	RawHTMLDeny      RawHTMLMode = "deny"
	RawHTMLSanitized RawHTMLMode = "sanitized"
)

const (
	// MinOutputBytes and MaxOutputBytes are the immutable root output policy
	// bounds. They are deliberately int64 so optional modules can copy the value
	// without narrowing it before their own checked arithmetic.
	MinOutputBytes int64 = 1
	MaxOutputBytes int64 = 64 << 20
)

// Policy describes a host capability ceiling. A zero Policy is not a valid
// explicit host policy; callers that do not provide one receive the built-in
// deny/MaxOutputBytes ceiling.
type Policy struct {
	RawHTML     RawHTMLMode `json:"rawHTML"`
	OutputBytes int64       `json:"outputBytes"`
}

// EffectivePolicy is the immutable intersection stored on a compiled
// Document. It is a value, not a pointer, so renderers cannot mutate the
// compiler's decision after Compile.
type EffectivePolicy struct {
	RawHTML     RawHTMLMode `json:"rawHTML"`
	OutputBytes int64       `json:"outputBytes"`
}

// WithHostPolicy supplies the host ceiling. Validation happens at Compile so
// an invalid value produces a stable diagnostic rather than a construction
// panic.
func WithHostPolicy(policy Policy) Option {
	return func(config *compilerConfig) error {
		config.values["hostPolicy"] = policy
		config.values["hostPolicySet"] = true
		return nil
	}
}

func defaultEvaluatePolicy(config compilerConfig, normalized sourceNormalization) (EffectivePolicy, error) {
	host := Policy{RawHTML: RawHTMLDeny, OutputBytes: MaxOutputBytes}
	if value, ok := config.values["hostPolicy"]; ok {
		candidate, valid := value.(Policy)
		if !valid {
			return EffectivePolicy{}, policyDiagnostic("policy.host.invalid", "host policy has the wrong type")
		}
		host = candidate
	}
	if host.RawHTML == "" {
		host.RawHTML = RawHTMLDeny
	}
	if err := validateRawHTMLMode(host.RawHTML); err != nil {
		return EffectivePolicy{}, err
	}
	if host.OutputBytes < MinOutputBytes || host.OutputBytes > MaxOutputBytes {
		return EffectivePolicy{}, policyDiagnostic("policy.output_bytes_invalid", "output byte limit must be between 1 and 67108864")
	}

	effective := EffectivePolicy{RawHTML: RawHTMLDeny, OutputBytes: host.OutputBytes}
	declaredMode, declared := declaredRawHTML(normalized)
	if declared {
		if err := validateRawHTMLMode(declaredMode); err != nil {
			return EffectivePolicy{}, err
		}
		if declaredMode == RawHTMLSanitized && host.RawHTML == RawHTMLDeny {
			return EffectivePolicy{}, policyDiagnostic("policy.raw_html.mismatch", "document requests sanitized raw HTML above the host deny ceiling")
		}
		if declaredMode == RawHTMLSanitized {
			effective.RawHTML = RawHTMLSanitized
		}
	}

	if normalized.sourceBytes > MaxDocumentBytes {
		return EffectivePolicy{}, policyDiagnostic("policy.resource.document_too_large", "document exceeds the maximum byte limit")
	}
	if hasRawHTML(normalized) {
		if !declared {
			return EffectivePolicy{}, policyDiagnostic("policy.raw_html.undeclared", "raw HTML requires an explicit sanitized declaration")
		}
		if declaredMode == RawHTMLDeny {
			return EffectivePolicy{}, policyDiagnostic("policy.raw_html.denied", "raw HTML is denied by the document policy")
		}
	}
	return effective, nil
}

func validateRawHTMLMode(mode RawHTMLMode) error {
	if mode != RawHTMLDeny && mode != RawHTMLSanitized {
		return policyDiagnostic("policy.raw_html.invalid", fmt.Sprintf("unsupported raw HTML mode %q", mode))
	}
	return nil
}

func policyDiagnostic(code, message string) error {
	return newDiagnosticError(Diagnostic{Code: code, Severity: SeverityError, Message: message})
}

func declaredRawHTML(normalized sourceNormalization) (RawHTMLMode, bool) {
	parsed, ok := normalized.parsed.(normalizedMarkdown)
	if !ok || parsed.frontmatter.goshtoso == nil {
		return "", false
	}
	security, ok := parsed.frontmatter.goshtoso["security"].(map[string]any)
	if !ok {
		return "", false
	}
	value, ok := security["rawHTML"].(string)
	if !ok {
		return "", false
	}
	return RawHTMLMode(value), true
}

func hasRawHTML(normalized sourceNormalization) bool {
	parsed, ok := normalized.parsed.(normalizedMarkdown)
	if !ok || parsed.root == nil {
		return false
	}
	found := false
	_ = ast.Walk(parsed.root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && node.Kind() == ast.KindRawHTML {
			found = true
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return found
}
