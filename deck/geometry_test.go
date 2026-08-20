package deck

import "testing"

func TestDeckGeometryPresetsAndCustomUnits(t *testing.T) {
	cases := []struct {
		value         string
		preset        string
		width, height float64
	}{
		{value: "16:9", preset: "16:9", width: 1280, height: 720},
		{value: "4:3", preset: "4:3", width: 960, height: 720},
		{value: "1280x800px", preset: "custom", width: 1280, height: 800},
		{value: "10x5in", preset: "custom", width: 960, height: 480},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			geometry, err := ParseDeckGeometry(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			if geometry.Preset != tc.preset || geometry.Width != tc.width || geometry.Height != tc.height {
				t.Fatalf("geometry = %#v", geometry)
			}
			if err := geometry.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDeckGeometryRejectsUnsafeDimensions(t *testing.T) {
	for _, value := range []string{"319x720px", "1280x800%", "10000x800px", "10x0in", "1x100in", "1280x800vw", "1280x800px; color:red"} {
		if _, err := ParseDeckGeometry(value); err == nil {
			t.Fatalf("ParseDeckGeometry(%q) unexpectedly succeeded", value)
		}
	}
}

func TestFrontmatterCustomGeometryUsesFiniteAbsoluteUnits(t *testing.T) {
	document, err := Parse("custom.md", []byte("---\nsize:\n  width: 12.5\n  height: 8\n  unit: in\n---\n# One\n"))
	if err != nil {
		t.Fatal(err)
	}
	geometry, err := ParseDeckGeometry(document.Directives().Size)
	if err != nil {
		t.Fatal(err)
	}
	if geometry.Preset != "custom" || geometry.Width != 1200 || geometry.Height != 768 {
		t.Fatalf("geometry = %#v", geometry)
	}
}
