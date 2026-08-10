package deck

import (
	"context"
	"strings"
	"testing"

	"github.com/araihu/margo"
	"github.com/araihu/margo/charts"
)

func TestRenderPreservesPopularImagesAndExtensions(t *testing.T) {
	source := `# Media

![PNG](images/sample.png)
![JPEG](images/sample.jpg)
![WebP](images/sample.webp)
![GIF](images/sample.gif)
![SVG](images/sample.svg)

| Format | Ready |
|---|---|
| PNG | yes |

` + "```go\nfmt.Println(\"deck\")\n```\n\n```mermaid\ngraph TD; A-->B\n```\n" + `---
# Chart

` + "```goshtosochart\n" + `schemaVersion: 1
type: bar
title: Revenue
categories: [Development, Production]
series:
  - name: Revenue
    values: [12, 18]
` + "```\n"
	compiler := margo.New(margo.WithExtension(charts.Extension(charts.WithExternalizedControlRuntime(true))))
	result, err := Render(context.Background(), compiler, RenderInput{Name: "integration.md", Markdown: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	markup := string(result.HTML())
	for _, fragment := range []string{
		`src="images/sample.png"`,
		`src="images/sample.jpg"`,
		`src="images/sample.webp"`,
		`src="images/sample.gif"`,
		`src="images/sample.svg"`,
		`<table`,
		`data-code-block`,
		`margo-mermaid`,
		`<svg`,
		`data-goshtoso-chart-wrapper`,
		`data-margo-requirement="goshtoso-charts.controls"`,
	} {
		if !strings.Contains(markup, fragment) {
			t.Fatalf("deck HTML missing %q", fragment)
		}
	}
	if result.SlideCount() != 2 {
		t.Fatalf("slides = %d", result.SlideCount())
	}
}
