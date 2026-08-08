package margo

import (
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestBrandValidationMatrix(t *testing.T) {
	good := Brand{
		Header:    templ.Raw("<header>Trusted</header>"),
		Footer:    templ.Raw("<footer>Trusted</footer>"),
		Watermark: "Internal",
		Stamps:    []string{"v0.0.1", "preview"},
		Tokens:    map[DocumentToken]string{TokenPageBackground: "#ffffff"},
	}
	if err := good.Validate(); err != nil {
		t.Fatalf("valid brand rejected: %v", err)
	}
	bad := good
	bad.Watermark = strings.Repeat("x", maxBrandWatermarkBytes+1)
	if err := bad.Validate(); err == nil {
		t.Fatal("oversized watermark unexpectedly accepted")
	}
	bad = good
	bad.Stamps = []string{"unsafe\nvalue"}
	if err := bad.Validate(); err == nil {
		t.Fatal("unsafe stamp unexpectedly accepted")
	}
}
