package margo

import (
	"fmt"
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
	SchemaVersion string        `json:"schemaVersion,omitempty"`
	RawHTML       RawHTMLMode   `json:"rawHTML"`
	InputBytes    int64         `json:"inputBytes"`
	OutputBytes   int64         `json:"outputBytes"`
	Iframe        *IframePolicy `json:"iframe,omitempty"`
}

// EffectivePolicy is the immutable intersection stored on a compiled
// Document. It is a value, not a pointer, so renderers cannot mutate the
// compiler's decision after Compile.
type EffectivePolicy struct {
	RawHTML         RawHTMLMode   `json:"rawHTML"`
	InputBytes      int64         `json:"inputBytes"`
	OutputBytes     int64         `json:"outputBytes"`
	Iframe          *IframePolicy `json:"iframe,omitempty"`
	AllowUnsafeHTML bool          `json:"allowUnsafeHTML,omitempty"`
}

// DefaultPolicy returns the least-authoritative host policy and documented
// resource defaults for this Margo version.
func DefaultPolicy() Policy {
	return Policy{
		SchemaVersion: "margo-policy/v1",
		RawHTML:       RawHTMLDeny, InputBytes: MaxDocumentBytes, OutputBytes: MaxOutputBytes,
	}
}

func configuredInputLimit(config compilerConfig) (int64, error) {
	limit := int64(MaxDocumentBytes)
	value, ok := config.values["hostPolicy"]
	if !ok {
		return limit, nil
	}
	policy, ok := value.(Policy)
	if !ok {
		return 0, policyDiagnostic("policy.host.invalid", "host policy has the wrong type")
	}
	if policy.InputBytes != 0 {
		limit = policy.InputBytes
	}
	if limit < 1 || limit > MaxDocumentBytes {
		return 0, policyDiagnostic("policy.input_bytes_invalid", "input byte limit must be between 1 and 16777216")
	}
	return limit, nil
}

// WithHostPolicy supplies the host ceiling. Validation happens at Compile so
// an invalid value produces a stable diagnostic rather than a construction
// panic.
func WithHostPolicy(policy Policy) Option {
	frozen := clonePolicy(policy)
	return func(config *compilerConfig) error {
		config.values["hostPolicy"] = clonePolicy(frozen)
		config.values["hostPolicySet"] = true
		return nil
	}
}

// WithUnsafeHTML opts a compiler into passing through document-authored HTML,
// including arbitrary iframe markup. The option is intentionally separate
// from Policy so a project cannot accidentally persist this capability in a
// reusable policy file; callers must make the decision at compiler setup.
func WithUnsafeHTML() Option {
	return func(config *compilerConfig) error {
		config.values["allowUnsafeHTML"] = true
		return nil
	}
}

func defaultEvaluatePolicy(config compilerConfig, normalized sourceNormalization) (EffectivePolicy, error) {
	host := DefaultPolicy()
	if value, ok := config.values["hostPolicy"]; ok {
		candidate, valid := value.(Policy)
		if !valid {
			return EffectivePolicy{}, policyDiagnostic("policy.host.invalid", "host policy has the wrong type")
		}
		host = clonePolicy(candidate)
	}
	if host.SchemaVersion == "" {
		host.SchemaVersion = "margo-policy/v1"
	}
	if host.SchemaVersion != "margo-policy/v1" {
		return EffectivePolicy{}, policyDiagnostic("policy.schema_version_invalid", "schemaVersion must be margo-policy/v1")
	}
	if host.RawHTML == "" {
		host.RawHTML = RawHTMLDeny
	}
	if err := validateRawHTMLMode(host.RawHTML); err != nil {
		return EffectivePolicy{}, err
	}
	if host.InputBytes == 0 {
		host.InputBytes = MaxDocumentBytes
	}
	if host.InputBytes < 1 || host.InputBytes > MaxDocumentBytes {
		return EffectivePolicy{}, policyDiagnostic("policy.input_bytes_invalid", "input byte limit must be between 1 and 16777216")
	}
	if host.OutputBytes < MinOutputBytes || host.OutputBytes > MaxOutputBytes {
		return EffectivePolicy{}, policyDiagnostic("policy.output_bytes_invalid", "output byte limit must be between 1 and 67108864")
	}

	if host.Iframe != nil {
		normalized, err := normalizeIframePolicy(*host.Iframe)
		if err != nil {
			return EffectivePolicy{}, policyDiagnostic("policy.iframe_invalid", err.Error())
		}
		host.Iframe = &normalized
	}

	allowUnsafeHTML, _ := config.values["allowUnsafeHTML"].(bool)
	effective := EffectivePolicy{RawHTML: host.RawHTML, InputBytes: host.InputBytes, OutputBytes: host.OutputBytes, Iframe: cloneIframePolicy(host.Iframe), AllowUnsafeHTML: allowUnsafeHTML}

	if normalized.sourceBytes > effective.InputBytes {
		return EffectivePolicy{}, policyDiagnostic("policy.resource.document_too_large", "document exceeds the maximum byte limit")
	}
	rawHTML, err := inspectSourceHTML(normalized, effective.Iframe, effective.AllowUnsafeHTML)
	if err != nil {
		return EffectivePolicy{}, err
	}
	if rawHTML {
		if effective.RawHTML == RawHTMLDeny && !effective.AllowUnsafeHTML {
			return EffectivePolicy{}, policyDiagnostic("policy.raw_html.denied", "raw HTML is denied by the host policy")
		}
	}
	if !normalized.skipRemoteImages {
		if err := rejectRemoteImages(normalized); err != nil {
			return EffectivePolicy{}, err
		}
	}
	return effective, nil
}

func clonePolicy(policy Policy) Policy {
	policy.Iframe = cloneIframePolicy(policy.Iframe)
	return policy
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
