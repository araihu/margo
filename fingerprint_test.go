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
