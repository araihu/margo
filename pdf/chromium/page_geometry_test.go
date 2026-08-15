package chromium

import (
	"strings"
	"testing"

	"github.com/araihu/margo/pdf"
)

func TestInjectPageGeometryUsesOneCanonicalCSSAuthority(t *testing.T) {
	for _, test := range []struct {
		name     string
		page     pdf.PageConfig
		want     string
		wantWide string
	}{
		{name: "A4 portrait margins", page: pdf.PageConfig{Size: pdf.PageA4, Orientation: pdf.Portrait, Margins: pdf.Margins{Top: 10, Right: 11, Bottom: 12, Left: 13}}, want: `@page { size: A4 portrait; margin: 10mm 11mm 12mm 13mm; }`, wantWide: `@page margo-diagram-landscape { size: A4 landscape; margin: 10mm 11mm 12mm 13mm; }`},
		{name: "Letter landscape", page: pdf.PageConfig{Size: pdf.PageLetter, Orientation: pdf.Landscape, Margins: pdf.Margins{Top: 1.5, Right: 2.5, Bottom: 3.5, Left: 4.5}}, want: `@page { size: Letter landscape; margin: 1.5mm 2.5mm 3.5mm 4.5mm; }`, wantWide: `@page margo-diagram-landscape { size: Letter landscape; margin: 1.5mm 2.5mm 3.5mm 4.5mm; }`},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := injectPageGeometry([]byte("<!doctype html><html><head><script>const marker = '</head>';</script><title>x</title></head><body></body></html>"), test.page)
			if err != nil {
				t.Fatal(err)
			}
			markup := string(result)
			if !strings.Contains(markup, test.want) || !strings.Contains(markup, test.wantWide) || strings.Count(markup, "@page") != 2 {
				t.Fatalf("geometry = %s", markup)
			}
			if strings.Index(markup, `const marker = '</head>'`) > strings.Index(markup, `data-margo-page-geometry`) {
				t.Fatalf("geometry was injected inside embedded script: %s", markup)
			}
		})
	}
}
