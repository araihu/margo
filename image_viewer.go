package margo

import (
	"fmt"
	"html"
	"io"
	"strconv"

	"github.com/yuin/goldmark/ast"
)

func mediaViewerRequirement() (HTMLRequirement, error) {
	asset, err := EmbeddedAsset("media-viewer.js")
	return HTMLRequirement{
		ID: "margo.media-viewer", Kind: HTMLScript,
		LocalURL:  "/margo-assets/media-viewer.js",
		LoadAfter: []string{"margo.document.styles"}, Inline: asset,
	}, err
}

func (r markdownRenderer) renderImageViewer(image *ast.Image) error {
	ordinal := r.runtimeTaskOrdinals["image"]
	r.runtimeTaskOrdinals["image"] = ordinal + 1
	captionID := fmt.Sprintf("margo-image-caption-%d", ordinal)
	if r.idAllocator != nil {
		captionID = r.idAllocator.Allocate("image-caption", strconv.FormatUint(uint64(ordinal), 10))
	}
	if _, err := fmt.Fprintf(r.out, `<figure class="margo-figure margo-image" data-margo-image-viewer><div class="margo-image__canvas" aria-labelledby="%s">`, html.EscapeString(captionID)); err != nil {
		return err
	}
	if err := r.renderImage(image); err != nil {
		return err
	}
	_, err := io.WriteString(r.out, `</div><figcaption id="`+html.EscapeString(captionID)+`" class="margo-figure-caption">`+html.EscapeString(string(image.Title))+`</figcaption></figure>`)
	return err
}
