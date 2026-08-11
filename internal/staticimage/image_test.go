package staticimage

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDetectContextHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := DetectContext(ctx, []byte(`<svg xmlns="http://www.w3.org/2000/svg"><path/></svg>`)); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}

func TestDetectRejectsSVGExternalCSSReferences(t *testing.T) {
	fixtures := []string{
		`<svg xmlns="http://www.w3.org/2000/svg"><style>@import url(https://example.com/a.css);</style></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><style>.x { fill: url( //example.com/a.svg#x ) }</style></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><path style="fill:url('https://example.com/a.svg#x')"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><style>.x { fill: u\72l(https://example.com/a.svg) }</style></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><style>.x { fill: u&#114;l(https://example.com/a.svg) }</style></svg>`,
	}
	for _, fixture := range fixtures {
		_, err := Detect([]byte(fixture))
		var imageErr *Error
		if !errors.As(err, &imageErr) || imageErr.Kind != SVGActive || !strings.Contains(imageErr.Message, "style") {
			t.Fatalf("fixture %q error = %v", fixture, err)
		}
	}
}

func TestDetectAllowsInternalSVGStyleReference(t *testing.T) {
	fixture := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><defs><linearGradient id="g"/></defs><style>.x { fill: url(#g) }</style><path class="x"/></svg>`)
	if mediaType, err := Detect(fixture); err != nil || mediaType != "image/svg+xml" {
		t.Fatalf("media type = %q error = %v", mediaType, err)
	}
}
