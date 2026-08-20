package deck

import "testing"

func TestParseHeadingDividerRulersAndSetextPrecedence(t *testing.T) {
	source := []byte("---\nheadingDivider: 2\n---\n# One\n## One detail\n### Still one\n   ***\n# Two\n    ---\n# Three\n")
	doc, err := Parse("split.md", source)
	if err != nil {
		t.Fatal(err)
	}
	slides := doc.Slides()
	if got, want := len(slides), 4; got != want {
		t.Fatalf("slides = %d want %d", got, want)
	}
	if got := string(slides[0].Markdown()); got != "# One\n" {
		t.Fatalf("slide 1 markdown = %q", got)
	}
	if got := string(slides[1].Markdown()); got != "## One detail\n### Still one\n" {
		t.Fatalf("slide 2 markdown = %q", got)
	}
	if got := string(slides[2].Markdown()); got != "# Two\n    ---\n" {
		t.Fatalf("slide 3 markdown = %q", got)
	}
	if got := string(slides[3].Markdown()); got != "# Three\n" {
		t.Fatalf("slide 4 markdown = %q", got)
	}
}

func TestParseDirectiveStateNotesAndBackgroundReset(t *testing.T) {
	source := []byte("<!-- theme: goshtoso -->\n<!-- paginate: true -->\n<!-- class: lead -->\n<!-- backgroundImage: images/one.png -->\n<!-- backgroundDecorative: false -->\n<!-- backgroundAlt: First image -->\n<!-- speaker note one -->\n# One\n---\n<!-- paginate: none -->\n<!-- class: none -->\n<!-- backgroundImage: images/two.png -->\n<!-- backgroundDecorative: false -->\n<!-- backgroundAlt: Second image -->\n<!-- speaker note two -->\n# Two\n")
	doc, err := Parse("directives.md", source)
	if err != nil {
		t.Fatal(err)
	}
	if got := doc.Directives().Theme; got != "goshtoso" {
		t.Fatalf("document theme = %q", got)
	}
	slides := doc.Slides()
	if len(slides) != 2 {
		t.Fatalf("slides = %d", len(slides))
	}
	first := slides[0].Directives()
	if got := first.Paginate; got != "true" {
		t.Fatalf("first paginate = %q", got)
	}
	if got := first.Classes; len(got) != 1 || got[0] != "lead" {
		t.Fatalf("first classes = %#v", got)
	}
	if got := first.Background.Source; got != "images/one.png" {
		t.Fatalf("first background = %q", got)
	}
	if got := first.Background.Alt; got != "First image" {
		t.Fatalf("first alt = %q", got)
	}
	if got := slides[0].Notes(); len(got) != 1 || got[0] != "speaker note one" {
		t.Fatalf("first notes = %#v", got)
	}
	second := slides[1].Directives()
	if second.Paginate != "" || len(second.Classes) != 0 {
		t.Fatalf("second reset state = %#v", second)
	}
	if second.Background.Source != "images/two.png" || second.Background.Alt != "Second image" {
		t.Fatalf("second background = %#v", second.Background)
	}
	if got := slides[1].Notes(); len(got) != 1 || got[0] != "speaker note two" {
		t.Fatalf("second notes = %#v", got)
	}
}

func TestParseStructuralLayoutSlots(t *testing.T) {
	source := []byte("<!-- _class: columns -->\n<!-- layout: columns -->\n<!-- slot: left -->\n## Left\nLeft content\n<!-- slot: right -->\n## Right\nRight content\n<!-- /layout -->\n")
	doc, err := Parse("columns.md", source)
	if err != nil {
		t.Fatal(err)
	}
	slide := doc.Slides()[0]
	layout := slide.Layout()
	if layout == nil || layout.Class != "columns" {
		t.Fatalf("layout = %#v", layout)
	}
	if len(layout.Slots) != 2 || layout.Slots[0].Name != "left" || layout.Slots[1].Name != "right" {
		t.Fatalf("slots = %#v", layout.Slots)
	}
	if string(layout.Slots[0].Markdown) != "## Left\nLeft content\n" {
		t.Fatalf("left markdown = %q", layout.Slots[0].Markdown)
	}
	if string(layout.Slots[1].Markdown) != "## Right\nRight content\n" {
		t.Fatalf("right markdown = %q", layout.Slots[1].Markdown)
	}
	if got := string(slide.Markdown()); got != "" {
		t.Fatalf("structural slide body = %q", got)
	}
}

func TestParseRejectsMalformedDirectivesAndLayoutCardinality(t *testing.T) {
	cases := []struct {
		name   string
		source string
		code   string
	}{
		{name: "malformed", source: "<!-- headingDivider: [2, -->\n# One\n", code: "deck.directive_comment_invalid"},
		{name: "unsupported", source: "<!-- style: custom -->\n# One\n", code: "deck.directive_unsupported"},
		{name: "missing slot", source: "<!-- class: columns -->\n<!-- layout: columns -->\n<!-- slot: left -->\n# Left\n<!-- /layout -->\n", code: "deck.layout_slots_required"},
		{name: "duplicate slot", source: "<!-- class: columns -->\n<!-- layout: columns -->\n<!-- slot: left -->\n# Left\n<!-- slot: left -->\n# Again\n<!-- slot: right -->\n# Right\n<!-- /layout -->\n", code: "deck.slot_invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.name+".md", []byte(tc.source))
			if got := deckDiagnosticCode(err); got != tc.code {
				t.Fatalf("code = %q want %q (err=%v)", got, tc.code, err)
			}
		})
	}
}
