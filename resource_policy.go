package margo

import "fmt"

// MaxDocumentBytes bounds source bytes before any renderer or extension runs.
const MaxDocumentBytes int64 = 16 << 20

// ResourceLimits contains the host's document resource ceilings.
type ResourceLimits struct {
	DocumentBytes int64
}

// ValidateResourceSize applies a positive configured limit without allowing
// integer wraparound or an accidental unlimited zero value.
func ValidateResourceSize(size int64, limits ResourceLimits) error {
	if size < 0 || limits.DocumentBytes < 1 {
		return fmt.Errorf("policy.resource.invalid: document byte limit must be positive")
	}
	if size > limits.DocumentBytes {
		return fmt.Errorf("policy.resource.document_too_large: %d > %d", size, limits.DocumentBytes)
	}
	return nil
}
