//go:build (darwin && cgo) || windows || (linux && cgo && margo_webkitgtk)

package native

import (
	"context"

	"github.com/araihu/margo/pdf"
)

// These platform slots deliberately remain compiled out until their official
// API bridges and matching runner evidence land together.
func platformProbe(context.Context) Capability {
	return compiledOut("native PDF backend has no verified platform runner evidence in this build")
}

func platformNew() (pdf.Engine, error) {
	capability := platformProbe(context.Background())
	return nil, capabilityError(capability)
}
