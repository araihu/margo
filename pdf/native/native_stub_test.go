//go:build !cgo || (!darwin && !windows && !linux) || (linux && !margo_webkitgtk)

package native

import (
	"context"
	"strings"
	"testing"
)

func TestPortableBuildReportsNativeCompiledOut(t *testing.T) {
	capability := Probe(context.Background())
	if capability.Compiled {
		t.Fatal("portable build claims native engine")
	}
	if capability.Available {
		t.Fatal("portable build claims native runtime availability")
	}
	if capability.Code != "pdf.native.compiled_out" {
		t.Fatalf("code = %q", capability.Code)
	}
	if _, err := New(); diagnosticCode(err) != "pdf.native.compiled_out" {
		t.Fatalf("New() error = %v", err)
	}
}

func diagnosticCode(err error) string {
	if err == nil {
		return ""
	}
	if index := strings.IndexByte(err.Error(), ':'); index >= 0 {
		return err.Error()[:index]
	}
	return err.Error()
}
