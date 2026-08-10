// Package native defines the stable capability boundary for platform-native
// PDF engines. A build that does not include a verified backend reports that
// fact without probing, downloading, or launching anything.
package native

import (
	"context"
	"fmt"

	"github.com/araihu/margo/pdf"
)

// Capability is the machine-readable native engine state for this build and
// host. Compiled and Available are separate so missing runtimes stay honest.
type Capability struct {
	Name      string `json:"name"`
	Compiled  bool   `json:"compiled"`
	Available bool   `json:"available"`
	Code      string `json:"code,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// Probe performs only the build-selected backend's read-only capability check.
func Probe(ctx context.Context) Capability {
	if ctx == nil {
		ctx = context.Background()
	}
	return platformProbe(ctx)
}

// New constructs the build-selected backend or returns a stable typed error.
func New() (pdf.Engine, error) { return platformNew() }

func compiledOut(reason string) Capability {
	return Capability{Name: "native", Code: "pdf.native.compiled_out", Reason: reason}
}

func capabilityError(capability Capability) error {
	code := capability.Code
	if code == "" {
		code = "pdf.native.unavailable"
	}
	reason := capability.Reason
	if reason == "" {
		reason = "native PDF engine is unavailable"
	}
	return fmt.Errorf("%s: %s", code, reason)
}
