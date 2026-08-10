//go:build (!darwin && !windows && !linux) || ((darwin || linux) && !cgo) || (linux && cgo && !margo_webkitgtk)

package native

import (
	"context"

	"github.com/araihu/margo/pdf"
)

func platformProbe(context.Context) Capability {
	return compiledOut("native PDF backend is not compiled in this portable build")
}

func platformNew() (pdf.Engine, error) {
	capability := platformProbe(context.Background())
	return nil, capabilityError(capability)
}
