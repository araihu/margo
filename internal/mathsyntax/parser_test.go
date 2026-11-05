package mathsyntax

import (
	"strings"
	"testing"

	"github.com/yuin/goldmark/ast"
)

func TestGrammar(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		want         []string
		issue        string
	}{
		{"inline", `The $E = mc^2$ and $2 + 2 = 4$.`, []string{"E = mc^2", "2 + 2 = 4"}, ""},
		{"currency", `Price: $20 and $30.`, nil, ""},
		{"invalid", `$ x$ $x $ $x$2 $x$$y$ $$$x$$$`, nil, ""},
		{"escape", `\$literal $x\$y$`, []string{`x\$y`}, ""},
		{"opaque", `$x_1 * y &amp; <script> \frac{a}{b}$`, []string{`x_1 * y &amp; <script> \frac{a}{b}`}, ""},
		{"code", "`$x$`\n\n```tex\n$x$\n```\n", nil, ""},
		{"compact", "$$ E = mc^2 $$\n", []string{"E = mc^2"}, ""},
		{"prose", `Text $$x$$ text`, nil, ""},
		{"multiline", "$$\r\nx+y\r\n\r\nz\r\n$$\r\n", []string{"x+y\n\nz\n"}, ""},
		{"interrupt", "Paragraph\n$$\nx\n$$\n", []string{"x\n"}, ""},
		{"quote", "> $$\n> x+y\n> $$\n", []string{"x+y\n"}, ""},
		{"list", "- $$\n  x+y\n  $$\n", []string{"x+y\n"}, ""},
		{"indented", "    $$x$$\n", nil, ""},
		{"fence", "```math\nx+y\n```\n", []string{"x+y\n"}, ""},
		{"tilde", "~~~math\nx+y\n~~~\n", []string{"x+y\n"}, ""},
		{"image", `![$x$](image.png) and [$y$](url)`, []string{"y"}, ""},
		{"image-html", `![$x <span title="a]b">y</span>$](image.png)`, nil, ""},
		{"empty", "$$\n\n$$\n", []string{"\n"}, "math.empty_expression"},
		{"unclosed", "$$\nx\n", []string{"x\n"}, "math.delimiter_unclosed"},
		{"unclosed-fence", "```math\nx\n", []string{"x\n"}, "math.delimiter_unclosed"},
		{"large", "$" + strings.Repeat("x", MaxExpressionBytes+1) + "$", nil, "math.expression_too_large"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := Parse([]byte(tt.source))
			var got []string
			_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
				if entering {
					if d := Of(n); d != nil {
						got = append(got, string(d.Payload))
						if d.Offset < 0 || d.End > d.Offset+len(tt.source) {
							t.Fatal("invalid span")
						}
					}
				}
				return ast.WalkContinue, nil
			})
			if tt.name != "large" && strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Fatalf("payloads: %#v, want %#v", got, tt.want)
			}
			issues := Validate(root)
			if tt.issue == "" && len(issues) > 0 {
				t.Fatalf("unexpected: %+v", issues)
			}
			if tt.issue != "" && (len(issues) == 0 || issues[0].Code != tt.issue) {
				t.Fatalf("issues: %+v, want %s", issues, tt.issue)
			}
		})
	}
}

func TestMaskProtectsDeckSyntax(t *testing.T) {
	source := []byte("# Heading\n$$\n---\n<!-- theme: minimal -->\n$$\n---\n```math\n# Hidden\n```\n")
	masked, issues := Mask(source)
	if len(issues) > 0 || len(masked) != len(source) {
		t.Fatalf("mask: %s %+v", masked, issues)
	}
	if strings.Count(string(masked), "---") != 1 || strings.Contains(string(masked), "theme:") || strings.Contains(string(masked), "Hidden") {
		t.Fatalf("unprotected: %s", masked)
	}
}
