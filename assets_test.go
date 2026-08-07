package margo

import (
	"bytes"
	"os"
	"regexp"
	"testing"
)

func TestLibraryCSSNeverTargetsHostRoot(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(css, []byte("@layer reset, goshtoso, document, brand, overrides")) {
		t.Fatalf("document stylesheet is missing the locked layer order")
	}
	if regexp.MustCompile(`(^|[,{])\s*(html|body|:root)\b`).Match(css) {
		t.Fatalf("document stylesheet targets a host root")
	}
	if !bytes.Contains(css, []byte(".goshtoso-document")) {
		t.Fatalf("document stylesheet is not scoped to .goshtoso-document")
	}
}

func TestAssetOverrideRejectsInvalidPathWithoutFallback(t *testing.T) {
	result := mustRenderSource(t, "# override")
	_, err := RenderStandalone(result, WithAssetOverride("document.css", AssetRef{Path: "../outside.css"}))
	if err == nil {
		t.Fatal("invalid asset override unexpectedly succeeded")
	}
	if got := err.Error(); got == "" {
		t.Fatal("invalid asset override returned an empty diagnostic")
	}
}
