package margo

import (
	"context"
	"testing"
)

func TestCompilerConfigFingerprintDomainSeparation(t *testing.T) {
	a := New()
	b := New()
	if a.fingerprint != b.fingerprint {
		t.Fatalf("same config fingerprints differ: %x != %x", a.fingerprint, b.fingerprint)
	}
	if a.fingerprint == (CompilerConfigFingerprint{}) {
		t.Fatal("zero compiler fingerprint")
	}
}

func TestDocumentBindsCompilerFingerprint(t *testing.T) {
	c := New()
	doc, err := c.Compile(context.Background(), Source{Name: "x.md", Content: []byte("# one")})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if doc.compilerFingerprint != c.fingerprint {
		t.Fatalf("document fingerprint binding differs")
	}
}

func TestArtifactFingerprintExcludesExecutionID(t *testing.T) {
	a := terminalReport("exec-a")
	b := terminalReport("exec-b")
	if artifactFingerprint(a) != artifactFingerprint(b) {
		t.Fatal("execution ID changed artifact fingerprint")
	}
	b.Layout.ScrollWidth++
	if artifactFingerprint(a) == artifactFingerprint(b) {
		t.Fatal("terminal layout mutation did not change artifact fingerprint")
	}
}

func TestArtifactDigestUsesExactBytes(t *testing.T) {
	a := ArtifactDigestOf([]byte("one"))
	b := ArtifactDigestOf([]byte("two"))
	if a == b {
		t.Fatal("different emitted bytes share an artifact digest")
	}
}

func terminalReport(executionID string) TerminalReport {
	return TerminalReport{
		ExecutionID:      executionID,
		Document:         DocumentFingerprint{1},
		RenderInstanceID: "ri-00000000",
		Kind:             "html",
		Serializer:       "margo/html/v1",
		Engine:           "none",
		TerminalStatus:   "complete",
		Layout:           LayoutMetrics{ScrollWidth: 800, ScrollHeight: 600},
		TaskInputHashes:  []string{"input"},
		TaskOutputHashes: []string{"output"},
		FontChecks:       []string{"font-ok"},
		BlockedRequests:  []string(nil),
	}
}
