package deck

import "testing"

func TestCompositionCatalogR1ContainsEveryApprovedName(t *testing.T) {
	if got, want := CompositionCatalogVersion, "r1"; got != want {
		t.Fatalf("catalog version = %q want %q", got, want)
	}
	names := []CompositionName{
		"content", "agenda", "media-split", "media-stage", "steps",
		"highlight", "compare-grid", "hero", "image-grid",
	}
	for _, name := range names {
		spec, err := ResolveComposition(name)
		if err != nil {
			t.Fatalf("ResolveComposition(%q): %v", name, err)
		}
		if spec.Name != name {
			t.Fatalf("spec name = %q want %q", spec.Name, name)
		}
		if spec.CatalogVersion != CompositionCatalogVersion {
			t.Fatalf("%q catalog version = %q want %q", name, spec.CatalogVersion, CompositionCatalogVersion)
		}
	}
}

func TestResolveCompositionRejectsUnknownName(t *testing.T) {
	_, err := ResolveComposition(CompositionName("unknown"))
	if got, want := deckDiagnosticCode(err), "deck.composition_invalid"; got != want {
		t.Fatalf("diagnostic = %q want %q (err=%v)", got, want, err)
	}
}

func TestResolveCompositionMapsGridVariants(t *testing.T) {
	tests := []struct {
		name       CompositionName
		variant    string
		min, max   int
		slotPrefix string
	}{
		{name: "compare-grid", variant: "compare", min: 2, max: 4, slotPrefix: "item-"},
		{name: "image-grid", variant: "image", min: 2, max: 4, slotPrefix: "image-"},
	}
	for _, tc := range tests {
		t.Run(string(tc.name), func(t *testing.T) {
			spec, err := ResolveComposition(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if spec.LayoutClass != "grid" || spec.Variant != tc.variant {
				t.Fatalf("spec = %#v", spec)
			}
			if spec.MinSlots != tc.min || spec.MaxSlots != tc.max {
				t.Fatalf("cardinality = %d-%d want %d-%d", spec.MinSlots, spec.MaxSlots, tc.min, tc.max)
			}
			if len(spec.Slots) != tc.max {
				t.Fatalf("slot definitions = %d want %d", len(spec.Slots), tc.max)
			}
			for index, slot := range spec.Slots {
				want := tc.slotPrefix + string(rune('1'+index))
				if slot.Name != want {
					t.Fatalf("slot %d = %q want %q", index, slot.Name, want)
				}
			}
		})
	}
}

func TestCompositionSpecCloneDoesNotShareSlots(t *testing.T) {
	spec, err := ResolveComposition("media-split")
	if err != nil {
		t.Fatal(err)
	}
	clone := cloneCompositionSpec(spec)
	if len(clone.Slots) != len(spec.Slots) {
		t.Fatalf("clone slots = %d want %d", len(clone.Slots), len(spec.Slots))
	}
	clone.Slots[0].Name = "changed"
	if spec.Slots[0].Name == "changed" {
		t.Fatal("composition slot clone shares backing storage")
	}
}
