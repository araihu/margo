package htmlpolicy

import "testing"

func TestSanitizerBypassCorpus(t *testing.T) {
	for _, input := range []string{`<svg><script>x</script></svg>`, `<iframe src="x"></iframe>`, `<img src="data:text/html,x">`} {
		if err := Validate([]byte(input)); err == nil {
			t.Fatalf("bypass input accepted: %s", input)
		}
	}
}
