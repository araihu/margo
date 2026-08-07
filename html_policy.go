package margo

import (
	"fmt"

	"github.com/araihu/margo/internal/htmlpolicy"
)

// ValidateHTML validates a fragment against the versioned margo-html-v1
// allowlist. It deliberately returns an error instead of rewriting unsafe
// markup so callers cannot mistake a partially sanitized tree for accepted
// document content.
func ValidateHTML(fragment string) error {
	if err := htmlpolicy.Validate([]byte(fragment)); err != nil {
		return fmt.Errorf("policy.html.invalid: %w", err)
	}
	return nil
}
