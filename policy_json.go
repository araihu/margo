package margo

import (
	"encoding/json"
	"fmt"
)

const MaxPolicyBytes = 64 << 10

// ParsePolicyJSON validates canonical v1 JSON before applying documented
// defaults and semantic origin normalization.
func ParsePolicyJSON(input []byte) (Policy, error) {
	if len(input) == 0 || len(input) > MaxPolicyBytes {
		return Policy{}, fmt.Errorf("policy.input_invalid: policy must contain 1 to %d bytes", MaxPolicyBytes)
	}
	if _, err := validateJSONSchema(SchemaPolicy, input); err != nil {
		return Policy{}, fmt.Errorf("policy.schema_invalid: %w", err)
	}
	type wirePolicy struct {
		SchemaVersion string        `json:"schemaVersion"`
		RawHTML       RawHTMLMode   `json:"rawHTML"`
		InputBytes    *int64        `json:"inputBytes"`
		OutputBytes   *int64        `json:"outputBytes"`
		Iframe        *IframePolicy `json:"iframe"`
	}
	var wire wirePolicy
	if err := json.Unmarshal(input, &wire); err != nil {
		return Policy{}, fmt.Errorf("policy.input_invalid: %w", err)
	}
	policy := DefaultPolicy()
	policy.SchemaVersion = wire.SchemaVersion
	if wire.RawHTML != "" {
		policy.RawHTML = wire.RawHTML
	}
	if wire.InputBytes != nil {
		policy.InputBytes = *wire.InputBytes
	}
	if wire.OutputBytes != nil {
		policy.OutputBytes = *wire.OutputBytes
	}
	if wire.Iframe != nil {
		normalized, err := normalizeIframePolicy(*wire.Iframe)
		if err != nil {
			return Policy{}, fmt.Errorf("policy.iframe_invalid: %w", err)
		}
		policy.Iframe = &normalized
	}
	return policy, nil
}
