package deck

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/araihu/margo"
)

func TestParseSplitsOnlyExactSeparatorsOutsideFences(t *testing.T) {
	source := []byte("---\ntitle: Demo\ndescription: Two slides\n---\n# One\n\n```text\n---\n```\n---\n# Two\n")
	doc, err := Parse("demo.md", source)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := doc.Metadata().Title, "Demo"; got != want {
		t.Fatalf("title = %q want %q", got, want)
	}
	if got, want := doc.Metadata().Description, "Two slides"; got != want {
		t.Fatalf("description = %q want %q", got, want)
	}
	slides := doc.Slides()
	if len(slides) != 2 {
		t.Fatalf("slides = %d", len(slides))
	}
	if !bytes.Contains(slides[0].Markdown(), []byte("```text\n---\n```")) {
		t.Fatal("fenced separator was removed")
	}
	if slides[0].Ordinal() != 1 || slides[1].Ordinal() != 2 {
		t.Fatal("unstable ordinals")
	}
	if slides[0].ID() != "slide-0001" || slides[1].ID() != "slide-0002" {
		t.Fatalf("slide IDs = %q, %q", slides[0].ID(), slides[1].ID())
	}
}

func TestParseSupportsTildeFencesAndCRLF(t *testing.T) {
	doc, err := Parse("windows.md", []byte("# One\r\n~~~~yaml\r\n---\r\n~~~~\r\n---\r\n# Two\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(doc.Slides()); got != 2 {
		t.Fatalf("slides = %d", got)
	}
}

func TestParseRejectsEmptySlides(t *testing.T) {
	for _, source := range []string{"\n---\n# Two\n", "# One\n---\n  \n", "# One\n---\n---\n# Three\n"} {
		_, err := Parse("empty.md", []byte(source))
		if got := deckDiagnosticCode(err); got != "deck.slide_empty" {
			t.Fatalf("source %q code = %q", source, got)
		}
	}
}

func TestParseRejectsInvalidFrontmatterAndUnclosedFence(t *testing.T) {
	_, err := Parse("metadata.md", []byte("---\ntitle: [\n---\n# One\n"))
	if got := deckDiagnosticCode(err); got != "deck.frontmatter_invalid" {
		t.Fatalf("frontmatter code = %q", got)
	}
	_, err = Parse("fence.md", []byte("# One\n```text\ncontent\n"))
	if got := deckDiagnosticCode(err); got != "deck.fence_unclosed" {
		t.Fatalf("fence code = %q", got)
	}
}

func TestDeckActivationDetectionAndConflict(t *testing.T) {
	active, err := Detect("active.md", []byte("---\nmarp: true\n---\n# One\n"))
	if err != nil || !active {
		t.Fatalf("active = %v err = %v", active, err)
	}
	inactive, err := Detect("inactive.md", []byte("---\nmarp: false\n---\n# One\n"))
	if err != nil || inactive {
		t.Fatalf("inactive = %v err = %v", inactive, err)
	}
	if _, err := Parse("contradictory.md", []byte("---\nmarp: false\n---\n# One\n")); deckDiagnosticCode(err) != "deck.activation_conflict" {
		t.Fatalf("conflict error = %v", err)
	}
}

func TestParseFrontmatterLocalDirectivesMatchDirectiveComments(t *testing.T) {
	frontmatter := []byte(`---
paginate: true
header: Atlas review
footer: Internal review
class: lead
color: ink
backgroundColor: surface
backgroundImage: gradient-blue
backgroundPosition: bottom-right
backgroundRepeat: no-repeat
backgroundSize: cover
backgroundDecorative: true
---
# One
---
# Two
`)
	comments := []byte(`<!-- paginate: true -->
<!-- header: Atlas review -->
<!-- footer: Internal review -->
<!-- class: lead -->
<!-- color: ink -->
<!-- backgroundColor: surface -->
<!-- backgroundImage: gradient-blue -->
<!-- backgroundPosition: bottom-right -->
<!-- backgroundRepeat: no-repeat -->
<!-- backgroundSize: cover -->
<!-- backgroundDecorative: true -->
# One
---
# Two
`)
	frontmatterDocument, err := Parse("frontmatter.md", frontmatter)
	if err != nil {
		t.Fatal(err)
	}
	commentDocument, err := Parse("comments.md", comments)
	if err != nil {
		t.Fatal(err)
	}
	frontmatterSlides := frontmatterDocument.Slides()
	commentSlides := commentDocument.Slides()
	if len(frontmatterSlides) != 2 || len(commentSlides) != 2 {
		t.Fatalf("slide counts = %d and %d, want 2", len(frontmatterSlides), len(commentSlides))
	}
	for index := range frontmatterSlides {
		if got, want := frontmatterSlides[index].Directives(), commentSlides[index].Directives(); !reflect.DeepEqual(got, want) {
			t.Fatalf("slide %d frontmatter directives = %#v, comment directives = %#v", index+1, got, want)
		}
	}
}

func TestParseBoundsHeaderAndFooterDirectiveBytes(t *testing.T) {
	for _, key := range []string{"header", "footer"} {
		t.Run(key, func(t *testing.T) {
			accepted := strings.Repeat("x", 240)
			if _, err := Parse(key+"-accepted.md", []byte("<!-- "+key+": "+accepted+" -->\n# One\n")); err != nil {
				t.Fatalf("240-byte value rejected: %v", err)
			}
			overLimit := strings.Repeat("x", 241)
			if _, err := Parse(key+"-over-limit.md", []byte("<!-- "+key+": "+overLimit+" -->\n# One\n")); deckDiagnosticCode(err) != "deck.directive_invalid" {
				t.Fatalf("241-byte value code = %q; err = %v", deckDiagnosticCode(err), err)
			}
		})
	}
}

func TestParseRejectsRecognizedLegacyDirectiveComments(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "paginate", value: "true"},
		{name: "theme", value: "modern"},
		{name: "lang", value: "pt-BR"},
		{name: "colorMode", value: "dark"},
		{name: "headingDivider", value: "2"},
		{name: "size", value: "16:9"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := []byte(fmt.Sprintf("<!-- $%s: %s -->\n# One\n", test.name, test.value))
			_, err := Parse(test.name+".md", source)
			if err == nil {
				t.Fatal("legacy directive comment was accepted")
			}
			var diagnosticError *margo.DiagnosticError
			if !errors.As(err, &diagnosticError) || len(diagnosticError.Diagnostics) != 1 {
				t.Fatalf("error = %v; want one diagnostic", err)
			}
			diagnostic := diagnosticError.Diagnostics[0]
			if diagnostic.Code != "deck.directive_invalid" {
				t.Fatalf("code = %q", diagnostic.Code)
			}
			wantHint := fmt.Sprintf("Use the unprefixed `%s` directive.", test.name)
			if diagnostic.Hint != wantHint {
				t.Fatalf("hint = %q want %q", diagnostic.Hint, wantHint)
			}
			if diagnostic.Line != 1 || diagnostic.Source != test.name+".md" {
				t.Fatalf("location = %s:%d", diagnostic.Source, diagnostic.Line)
			}
		})
	}
}

func TestParseRejectsWhitespaceSeparatedLegacyDirectiveComments(t *testing.T) {
	tests := []struct {
		name    string
		comment string
		hint    string
	}{
		{name: "space", comment: "$ paginate: true", hint: "paginate"},
		{name: "multiple-spaces", comment: "$  paginate: true", hint: "paginate"},
		{name: "tab", comment: "$	paginate: true", hint: "paginate"},
		{name: "spot-space", comment: "$ _paginate: true", hint: "_paginate"},
		{name: "flow-map", comment: "{ $ paginate: true }", hint: "paginate"},
		{name: "quoted-flow-map-key", comment: `{ "$ paginate": true }`, hint: "paginate"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse(test.name+".md", []byte("<!-- "+test.comment+" -->\n# One\n"))
			if err == nil {
				t.Fatal("legacy directive comment was accepted")
			}
			var diagnosticError *margo.DiagnosticError
			if !errors.As(err, &diagnosticError) || len(diagnosticError.Diagnostics) != 1 {
				t.Fatalf("error = %v; want one diagnostic", err)
			}
			diagnostic := diagnosticError.Diagnostics[0]
			if diagnostic.Code != "deck.directive_invalid" {
				t.Fatalf("code = %q", diagnostic.Code)
			}
			wantHint := fmt.Sprintf("Use the unprefixed `%s` directive.", test.hint)
			if diagnostic.Hint != wantHint {
				t.Fatalf("hint = %q want %q", diagnostic.Hint, wantHint)
			}
		})
	}
}

func TestParseKeepsUnknownDollarCommentsAsNotes(t *testing.T) {
	source := []byte("<!-- $custom: true -->\n<!-- $ unknown: true -->\n<!-- $ paginate is useful -->\n<!-- A note mentioning $paginate: true -->\n<!-- paginate: true -->\n# One\n")
	doc, err := Parse("notes.md", source)
	if err != nil {
		t.Fatal(err)
	}
	slides := doc.Slides()
	if len(slides) != 1 {
		t.Fatalf("slides = %d", len(slides))
	}
	if got, want := slides[0].Notes(), []string{"$custom: true", "$ unknown: true", "$ paginate is useful", "A note mentioning $paginate: true"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("notes = %#v want %#v", got, want)
	}
	if got := slides[0].Directives().Paginate; got != "true" {
		t.Fatalf("paginate = %q", got)
	}
}

func TestParseKeepsUnknownDollarFrontmatter(t *testing.T) {
	source := []byte("---\n$custom: true\ntitle: One\npaginate: true\n---\n# One\n")
	doc, err := Parse("frontmatter.md", source)
	if err != nil {
		t.Fatal(err)
	}
	if got := doc.Metadata().Title; got != "One" {
		t.Fatalf("title = %q", got)
	}
	if got := doc.Directives().Paginate; got != "true" {
		t.Fatalf("paginate = %q", got)
	}
}

func TestParseRejectsRecognizedLegacyDirectiveFrontmatter(t *testing.T) {
	_, err := Parse("frontmatter.md", []byte("---\n$theme: modern\n---\n# One\n"))
	if err == nil {
		t.Fatal("legacy frontmatter directive was accepted")
	}
	var diagnosticError *margo.DiagnosticError
	if !errors.As(err, &diagnosticError) || len(diagnosticError.Diagnostics) != 1 {
		t.Fatalf("error = %v; want one diagnostic", err)
	}
	diagnostic := diagnosticError.Diagnostics[0]
	if diagnostic.Code != "deck.directive_invalid" || diagnostic.Hint != "Use the unprefixed `theme` directive." {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}
}

func TestDocumentAndSlidesReturnDefensiveCopies(t *testing.T) {
	doc, err := Parse("copy.md", []byte("# One\n---\n# Two\n"))
	if err != nil {
		t.Fatal(err)
	}
	first := doc.Slides()
	first[0].markdown[0] = 'X'
	first = append(first, Slide{})
	second := doc.Slides()
	if got := string(second[0].Markdown()); got != "# One\n" {
		t.Fatalf("markdown = %q", got)
	}
	if len(second) != 2 {
		t.Fatalf("slides = %d", len(second))
	}
}

func deckDiagnosticCode(err error) string {
	if err == nil {
		return ""
	}
	var diagnostic *margo.DiagnosticError
	if errors.As(err, &diagnostic) && len(diagnostic.Diagnostics) > 0 {
		return diagnostic.Diagnostics[0].Code
	}
	return err.Error()
}
