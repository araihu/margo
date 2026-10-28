package margo

import (
	"context"
	"strings"
	"testing"
)

const imageViewerFrontmatter = "---\nmargo:\n  imageViewer: true\n---\n"

func TestImageViewerSelectsOnlyStandaloneCaptionedImages(t *testing.T) {
	source := imageViewerFrontmatter + `# Images

![Dashboard details](dashboard.png "Dashboard caption")

![Second dashboard](second.png "Second caption")

![Logo](logo.svg)

Inline ![Icon](icon.svg "Icon title") text.

[![Thumbnail](thumb.png "Linked title")](full.png)
`
	result := mustRenderSource(t, source)
	markup := renderComponent(t, result.Content())
	if got := strings.Count(markup, "data-margo-image-viewer"); got != 2 {
		t.Fatalf("viewers = %d, want 2: %s", got, markup)
	}
	for _, want := range []string{
		`aria-labelledby="margo-image-caption-0"`, `id="margo-image-caption-1"`,
		`alt="Dashboard details"`, `>Dashboard caption</figcaption>`,
		`<a href="full.png"><img src="thumb.png" alt="Thumbnail" title="Linked title"></a>`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("missing %q: %s", want, markup)
		}
	}
	viewerCount := 0
	for _, requirement := range result.projectedHTMLRequirements().List() {
		if requirement.ID == "margo.media-viewer" {
			viewerCount++
		}
		if strings.HasPrefix(requirement.ID, "margo.mermaid.") {
			t.Fatalf("image-only page requires Mermaid: %s", requirement.ID)
		}
	}
	if viewerCount != 1 {
		t.Fatalf("shared viewer requirements = %d, want 1", viewerCount)
	}
}

func TestImageViewerOptInAndStaticTargets(t *testing.T) {
	image := "![Details](dashboard.png \"Dashboard\")\n"
	for _, tc := range []struct {
		name, source string
		target       RenderTarget
	}{
		{"default", image, TargetHTML},
		{"disabled", "---\nmargo:\n  imageViewer: false\n---\n" + image, TargetSite},
		{"pdf", imageViewerFrontmatter + image, TargetPDF},
		{"deck", imageViewerFrontmatter + image, TargetDeck},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := mustRenderSource(t, tc.source, WithRenderTarget(tc.target))
			markup := renderComponent(t, result.Content())
			if strings.Contains(markup, "data-margo-image-viewer") || !strings.Contains(markup, `<img src="dashboard.png" alt="Details"`) {
				t.Fatalf("expected a static image: %s", markup)
			}
		})
	}
	// Enabling the preference alone must not load an unused runtime.
	result := mustRenderSource(t, imageViewerFrontmatter+"![Logo](logo.svg)\n")
	for _, requirement := range result.projectedHTMLRequirements().List() {
		if requirement.ID == "margo.media-viewer" {
			t.Fatal("uncaptioned image loads the viewer")
		}
	}
}

func TestImageViewerMetadataValidation(t *testing.T) {
	compiler := New()
	_, err := compiler.Compile(context.Background(), Source{Name: "bad.md", Content: []byte("---\nmargo:\n  imageViewer: all\n---\n")})
	if err == nil || !strings.Contains(err.Error(), "frontmatter.schema_invalid") {
		t.Fatalf("invalid imageViewer error = %v", err)
	}
	result := mustRenderSource(t, imageViewerFrontmatter+"# Images\n")
	if !result.Metadata().Margo.ImageViewer {
		t.Fatal("metadata lost the imageViewer preference")
	}
}

func TestImagesAndMermaidShareOneViewerRequirement(t *testing.T) {
	result := mustRenderSource(t, imageViewerFrontmatter+"![Details](dashboard.png \"Dashboard\")\n\n```mermaid\nflowchart LR\n A --> B\n```\n")
	count := 0
	for _, requirement := range result.projectedHTMLRequirements().List() {
		if requirement.ID == "margo.media-viewer" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("shared viewer requirement count = %d, want 1", count)
	}
}
