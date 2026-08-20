package deck

import "testing"

func TestParseCompositionImplicitLayout(t *testing.T) {
	source := []byte("<!-- composition: media-split -->\n<!-- slot: media -->\nMedia\n<!-- slot: content -->\nContent\n")
	doc, err := Parse("composition-implicit.md", source)
	if err != nil {
		t.Fatal(err)
	}
	slide := doc.Slides()[0]
	if got := slide.Layout(); got == nil || got.Class != "columns" {
		t.Fatalf("layout = %#v want columns", got)
	}
	if got := slide.Composition(); got.Name != "media-split" || got.LayoutClass != "columns" {
		t.Fatalf("composition = %#v", got)
	}
	if slots := slide.Layout().Slots; len(slots) != 2 || slots[0].Name != "media" || slots[1].Name != "content" {
		t.Fatalf("slots = %#v", slots)
	}
}

func TestParseCompositionAcceptsMatchingLayoutMarker(t *testing.T) {
	source := []byte("<!-- composition: compare-grid -->\n<!-- layout: grid -->\n<!-- slot: item-1 -->\nOne\n<!-- slot: item-2 -->\nTwo\n<!-- /layout -->\n")
	doc, err := Parse("composition-explicit-layout.md", source)
	if err != nil {
		t.Fatal(err)
	}
	if got := doc.Slides()[0].Layout().Class; got != "grid" {
		t.Fatalf("layout class = %q want grid", got)
	}
}

func TestParseCompositionRejectsClassLayoutConflict(t *testing.T) {
	source := []byte("<!-- composition: hero -->\n<!-- class: quote -->\n# Opening\n")
	_, err := Parse("composition-class-conflict.md", source)
	if got, want := deckDiagnosticCode(err), "deck.composition_conflict"; got != want {
		t.Fatalf("diagnostic = %q want %q (err=%v)", got, want, err)
	}
}

func TestParseCompositionRejectsCompositionSlotErrors(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name:   "missing media split slot",
			source: "<!-- composition: media-split -->\n<!-- slot: media -->\nMedia\n",
		},
		{
			name:   "unknown image grid slot",
			source: "<!-- composition: image-grid -->\n<!-- slot: image-1 -->\nOne\n<!-- slot: wrong -->\nTwo\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.name+".md", []byte(tc.source))
			code := deckDiagnosticCode(err)
			if code != "deck.composition_slots_required" && code != "deck.composition_slot_invalid" {
				t.Fatalf("diagnostic = %q (err=%v)", code, err)
			}
		})
	}
}

func TestParseCompositionBodyCompositionsDoNotRequireSlots(t *testing.T) {
	source := []byte("<!-- composition: hero -->\n# Opening\n")
	doc, err := Parse("composition-body.md", source)
	if err != nil {
		t.Fatal(err)
	}
	slide := doc.Slides()[0]
	if got := slide.Composition(); got.Name != "hero" || got.LayoutClass != "lead" {
		t.Fatalf("composition = %#v", got)
	}
	if slide.Layout() != nil {
		t.Fatalf("body composition unexpectedly has structural layout: %#v", slide.Layout())
	}
}

func TestParseCompositionCatalogLayouts(t *testing.T) {
	cases := []struct {
		name        string
		composition string
		slots       []string
		class       string
		body        bool
	}{
		{name: "agenda", composition: "agenda", slots: []string{"item-1", "item-2", "item-3"}, class: "timeline"},
		{name: "media stage", composition: "media-stage", slots: []string{"media", "content"}, class: "columns"},
		{name: "steps", composition: "steps", slots: []string{"step-1", "step-2", "step-3"}, class: "timeline"},
		{name: "compare grid", composition: "compare-grid", slots: []string{"item-1", "item-2"}, class: "grid"},
		{name: "image grid", composition: "image-grid", slots: []string{"image-1", "image-2"}, class: "grid"},
		{name: "content", composition: "content", body: true},
		{name: "highlight", composition: "highlight", class: "section", body: true},
		{name: "hero", composition: "hero", class: "lead", body: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "<!-- composition: " + tc.composition + " -->\n"
			if tc.body {
				source += "# Body\n"
			} else {
				for _, slot := range tc.slots {
					source += "<!-- slot: " + slot + " -->\n" + slot + " content\n"
				}
			}
			doc, err := Parse(tc.name+".md", []byte(source))
			if err != nil {
				t.Fatal(err)
			}
			slide := doc.Slides()[0]
			if got := slide.Composition(); got.Name != CompositionName(tc.composition) {
				t.Fatalf("composition = %#v", got)
			}
			if tc.body {
				if slide.Layout() != nil {
					t.Fatalf("body composition unexpectedly has layout: %#v", slide.Layout())
				}
				if classes := slide.Directives().Classes; len(classes) != boolToInt(tc.class != "") {
					t.Fatalf("body classes = %#v", classes)
				}
				if tc.class != "" && slide.Directives().Classes[0] != tc.class {
					t.Fatalf("body class = %#v want %q", slide.Directives().Classes, tc.class)
				}
				return
			}
			layout := slide.Layout()
			if layout == nil || layout.Class != tc.class || len(layout.Slots) != len(tc.slots) {
				t.Fatalf("layout = %#v want class %q with %d slots", layout, tc.class, len(tc.slots))
			}
		})
	}
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
