package htmlpolicy

import (
	"strings"
	"testing"
)

func TestSanitizerBypassCorpus(t *testing.T) {
	for _, input := range []string{`<svg><script>x</script></svg>`, `<iframe src="x"></iframe>`, `<img src="data:text/html,x">`} {
		if err := Validate([]byte(input)); err == nil {
			t.Fatalf("bypass input accepted: %s", input)
		}
	}
}

func TestNormalizeSerializesFreshCanonicalFragment(t *testing.T) {
	input := []byte(`<SPAN TITLE='A &amp; B'>ok<BR></SPAN>`)
	output, err := Normalize(input)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(output), `<span title="A &amp; B">ok<br/></span>`; got != want {
		t.Fatalf("normalized fragment = %q, want %q", got, want)
	}
	if strings.Contains(string(output), "SPAN") || strings.Contains(string(output), "'") {
		t.Fatalf("original spelling leaked: %s", output)
	}
}

func TestNormalizeTokensPreservesSplitInlineTagSemantics(t *testing.T) {
	opening, err := NormalizeTokens([]byte(`<SPAN TITLE='A &amp; B'>`))
	if err != nil {
		t.Fatal(err)
	}
	closing, err := NormalizeTokens([]byte(`</SPAN>`))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(opening) + "text" + string(closing); got != `<span title="A &amp; B">text</span>` {
		t.Fatalf("split normalized fragment = %q", got)
	}
}
