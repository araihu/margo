package pdf

import (
	"math"
	"strings"
	"testing"
)

func TestPageConfigAcceptsSupportedMatrix(t *testing.T) {
	t.Parallel()

	for _, size := range []PageSize{PageA4, PageLetter} {
		for _, orientation := range []Orientation{Portrait, Landscape} {
			config := PageConfig{
				Size:        size,
				Orientation: orientation,
				Margins: Margins{
					Top: 12.5, Right: 10, Bottom: 14.25, Left: 10,
				},
			}
			if err := config.Validate(); err != nil {
				t.Fatalf("Validate(%q, %q) error = %v", size, orientation, err)
			}
		}
	}
}

func TestPageConfigAcceptsAndValidatesCustomSize(t *testing.T) {
	config := PageConfig{Custom: &CustomPageSize{WidthMM: 338.667, HeightMM: 190.5}}
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (PageConfig{Size: PageA4, Custom: &CustomPageSize{WidthMM: 10, HeightMM: 10}}).Validate(); err == nil {
		t.Fatal("named page with custom size unexpectedly accepted")
	}
}

func TestPageConfigRejectsUnsupportedSize(t *testing.T) {
	t.Parallel()

	err := (PageConfig{Size: PageSize("legal"), Orientation: Portrait}).Validate()
	requirePDFErrorCode(t, err, "pdf.page_size_unsupported")
}

func TestPageConfigRejectsUnsupportedOrientation(t *testing.T) {
	t.Parallel()

	err := (PageConfig{Size: PageA4, Orientation: Orientation("reverse-landscape")}).Validate()
	requirePDFErrorCode(t, err, "pdf.orientation_unsupported")
}

func TestPageConfigRejectsInvalidMargins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		margins Margins
	}{
		{name: "negative", margins: Margins{Top: -0.01}},
		{name: "not-a-number", margins: Margins{Right: Millimeters(math.NaN())}},
		{name: "positive-infinity", margins: Margins{Bottom: Millimeters(math.Inf(1))}},
		{name: "negative-infinity", margins: Margins{Left: Millimeters(math.Inf(-1))}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := (PageConfig{Size: PageLetter, Orientation: Landscape, Margins: test.margins}).Validate()
			requirePDFErrorCode(t, err, "pdf.margin_invalid")
		})
	}
}

func TestPageConfigAcceptsZeroMargins(t *testing.T) {
	t.Parallel()

	if err := (PageConfig{Size: PageA4, Orientation: Portrait}).Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestPageConfigAcceptsDefaultPortrait(t *testing.T) {
	t.Parallel()

	config := PageConfig{Size: PageA4}
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got := config.EffectiveImageOverflowPolicy(); got != ImageOverflowLimit {
		t.Fatalf("default image overflow policy = %q, want %q", got, ImageOverflowLimit)
	}
}

func TestPageConfigAcceptsExplicitImageOverflowPolicy(t *testing.T) {
	t.Parallel()

	for _, policy := range []ImageOverflowPolicy{ImageOverflowLimit, ImageOverflowAllow} {
		if err := (PageConfig{Size: PageA4, ImageOverflow: policy}).Validate(); err != nil {
			t.Fatalf("Validate(%q) error = %v", policy, err)
		}
	}
}

func TestPageConfigRejectsUnsupportedImageOverflowPolicy(t *testing.T) {
	t.Parallel()

	err := (PageConfig{Size: PageA4, ImageOverflow: ImageOverflowPolicy("stretch")}).Validate()
	requirePDFErrorCode(t, err, "pdf.image_overflow_unsupported")
}

func requirePDFErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), code) {
		t.Fatalf("error = %v, want code %q", err, code)
	}
}
