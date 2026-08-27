package staticimage

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"strings"
	"testing"
)

func avifFixture(major string, compatible ...string) []byte {
	data := make([]byte, 16+4*len(compatible))
	binary.BigEndian.PutUint32(data[:4], uint32(len(data)))
	copy(data[4:8], "ftyp")
	copy(data[8:12], major)
	for index, brand := range compatible {
		copy(data[16+index*4:], brand)
	}
	return data
}

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

func TestDetectRecognizesAVIFMajorAndCompatibleBrands(t *testing.T) {
	for name, fixture := range map[string][]byte{
		"major brand":         avifFixture("avif"),
		"sequence brand":      avifFixture("avis"),
		"compatible brand":    avifFixture("mif1", "avif"),
		"compatible sequence": avifFixture("mif1", "avis"),
	} {
		t.Run(name, func(t *testing.T) {
			mediaType, err := Detect(fixture)
			if err != nil || mediaType != "image/avif" {
				t.Fatalf("media type = %q error = %v", mediaType, err)
			}
		})
	}
}

func TestDetectRejectsMalformedOrUnsupportedISOContainers(t *testing.T) {
	fixtures := map[string][]byte{
		"unsupported brand": avifFixture("isom", "mp41"),
		"truncated ftyp":    avifFixture("mif1", "avif")[:15],
		"declared size too large": func() []byte {
			fixture := avifFixture("avif")
			binary.BigEndian.PutUint32(fixture[:4], uint32(len(fixture)+1))
			return fixture
		}(),
	}
	for name, fixture := range fixtures {
		t.Run(name, func(t *testing.T) {
			mediaType, err := Detect(fixture)
			var imageErr *Error
			if mediaType != "" || !errors.As(err, &imageErr) || imageErr.Kind != FormatUnsupported {
				t.Fatalf("media type = %q error = %v", mediaType, err)
			}
		})
	}
}

func TestValidateDataURLAcceptsAVIFAndChecksDeclaration(t *testing.T) {
	data := avifFixture("mif1", "avif")
	value := "data:image/avif;base64," + base64.StdEncoding.EncodeToString(data)
	decoded, mediaType, err := ValidateDataURL(context.Background(), value, int64(len(data)))
	if err != nil || mediaType != "image/avif" || string(decoded) != string(data) {
		t.Fatalf("decoded media type = %q error = %v", mediaType, err)
	}

	wrong := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
	if _, _, err := ValidateDataURL(context.Background(), wrong, int64(len(data))); err == nil || !strings.Contains(err.Error(), "declared media type") {
		t.Fatalf("mislabeled AVIF error = %v", err)
	}
}
