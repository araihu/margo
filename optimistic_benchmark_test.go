package margo

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	goldast "github.com/yuin/goldmark/ast"
	extensionast "github.com/yuin/goldmark/extension/ast"
	"golang.org/x/net/html"
)

type optimisticMermaidNegativeVector struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

func TestOptimisticBenchmarkPresentsColorModeProjectionBeforeFeatureTour(t *testing.T) {
	source, err := os.ReadFile("testdata/markdown/margo-full-feature-set.md")
	if err != nil {
		t.Fatal(err)
	}
	projection := bytes.Index(source, []byte("Color mode projection edge cases"))
	featureTour := bytes.Index(source, []byte("## 1. Document anatomy and navigation"))
	if projection < 0 || featureTour < 0 || projection >= featureTour {
		t.Fatal("color-mode projection contract must appear before the feature tour")
	}
	for _, required := range []string{
		"same immutable document",
		"light PDF",
		"dark PDF",
		"unsupported color mode",
		"before output",
	} {
		if !bytes.Contains(source, []byte(required)) {
			t.Errorf("optimistic benchmark missing color-mode contract %q", required)
		}
	}
}

func TestOptimisticBenchmarkPresentsMermaidEdgeCasesBeforeHappyPath(t *testing.T) {
	manifestBytes, err := os.ReadFile("testdata/mermaid/negative/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []optimisticMermaidNegativeVector
	if err := json.Unmarshal(manifestBytes, &vectors); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"testdata/markdown/slices/05-mermaid-profile.md",
		"testdata/markdown/margo-full-feature-set.md",
	} {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		edgeCases := bytes.Index(source, []byte("Edge cases first"))
		happyPath := bytes.Index(source, []byte("```mermaid"))
		if edgeCases < 0 || happyPath < 0 || edgeCases >= happyPath {
			t.Fatalf("%s must present Mermaid edge cases before its first executable Mermaid fence", path)
		}
		for _, vector := range vectors {
			for _, required := range []string{"`" + vector.Name + "`", "`" + vector.Code + "`"} {
				if !bytes.Contains(source, []byte(required)) {
					t.Errorf("%s missing negative vector contract %s", path, required)
				}
			}
		}
		for _, required := range []string{
			`stroke-width="1pt"`,
			`stroke-width="1cm"`,
			"canonical base64 ASCII JSON",
			"profile fingerprint mismatch",
			"unsupported diagram family",
			"byte limit",
			"element limit",
			"attribute limit",
			"CSS rule limit",
			"selector-byte limit",
			"`mermaid.svg_resource_limit`",
		} {
			if !bytes.Contains(source, []byte(required)) {
				t.Errorf("%s missing Mermaid boundary %q", path, required)
			}
		}
	}
}

func TestOptimisticBenchmarkExercisesFullMarkdownProfile(t *testing.T) {
	source, err := os.ReadFile("testdata/markdown/margo-full-feature-set.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(source) < 12_000 {
		t.Fatalf("benchmark source is only %d bytes; want long-form integration corpus", len(source))
	}

	compiler := New(WithHostPolicy(Policy{RawHTML: RawHTMLSanitized, OutputBytes: MaxOutputBytes}))
	document, err := compiler.Compile(context.Background(), Source{Name: "margo-full-feature-set.md", Content: source})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	parsed := document.parsed.(normalizedMarkdown)

	headingLevels := map[int]bool{}
	languages := map[string]bool{}
	var images, tables, ordered, unordered, tasks, footnoteLinks, footnoteLists, rawHTML, thematicBreaks int
	var maxListDepth int
	err = goldast.Walk(parsed.root, func(node goldast.Node, entering bool) (goldast.WalkStatus, error) {
		if !entering {
			return goldast.WalkContinue, nil
		}
		switch value := node.(type) {
		case *goldast.Heading:
			headingLevels[value.Level] = true
		case *goldast.FencedCodeBlock:
			languages[string(value.Language(parsed.frontmatter.body))] = true
		case *goldast.Image:
			images++
		case *goldast.List:
			if value.IsOrdered() {
				ordered++
			} else {
				unordered++
			}
			depth := 1
			for parent := value.Parent(); parent != nil; parent = parent.Parent() {
				if _, ok := parent.(*goldast.List); ok {
					depth++
				}
			}
			if depth > maxListDepth {
				maxListDepth = depth
			}
		case *extensionast.TaskCheckBox:
			tasks++
		case *extensionast.Table:
			tables++
		case *extensionast.FootnoteLink:
			footnoteLinks++
		case *extensionast.FootnoteList:
			footnoteLists++
		case *goldast.HTMLBlock, *goldast.RawHTML:
			rawHTML++
		case *goldast.ThematicBreak:
			thematicBreaks++
		}
		return goldast.WalkContinue, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for level := 1; level <= 6; level++ {
		if !headingLevels[level] {
			t.Errorf("heading level %d missing", level)
		}
	}
	for _, language := range []string{"go", "bash", "json", "yaml", "html", "text", "mermaid"} {
		if !languages[language] {
			t.Errorf("fenced language %q missing", language)
		}
	}
	if images < 2 || tables < 2 || ordered < 3 || unordered < 5 || maxListDepth < 2 || tasks < 6 || footnoteLinks < 3 || footnoteLists != 1 || rawHTML < 2 || thematicBreaks < 2 {
		t.Fatalf("incomplete AST coverage images=%d tables=%d ordered=%d unordered=%d listDepth=%d tasks=%d footnoteLinks=%d footnoteLists=%d rawHTML=%d breaks=%d", images, tables, ordered, unordered, maxListDepth, tasks, footnoteLinks, footnoteLists, rawHTML, thematicBreaks)
	}
	if len(document.plan.nodes) != 3 {
		t.Fatalf("Mermaid runtime tasks = %d, want 3", len(document.plan.nodes))
	}
	for _, node := range document.plan.nodes {
		if node.Fence != "mermaid" || node.compiled == nil {
			t.Fatalf("uncompiled Mermaid node: %#v", node)
		}
	}

	result, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	markup := renderComponent(t, result.Content())
	root, err := html.Parse(strings.NewReader(markup))
	if err != nil {
		t.Fatal(err)
	}

	for _, tag := range []string{"h1", "h2", "h3", "h4", "h5", "h6", "strong", "em", "del", "blockquote", "ol", "ul", "table", "img", "details", "mark", "kbd", "sup", "sub"} {
		if countElements(root, tag) == 0 {
			t.Errorf("rendered benchmark missing <%s>", tag)
		}
	}
	if got := countElements(root, "input"); got < 6 {
		t.Errorf("rendered task checkboxes = %d, want at least 6", got)
	}
	if got := countElementsWithClass(root, "section", "footnotes"); got != 1 {
		t.Errorf("rendered footnote sections = %d, want 1", got)
	}
	if !bytes.Contains([]byte(markup), []byte(`src="../../assets/logo.svg" alt="Margo mark used as a compact vector figure" title="Margo vector mark"`)) {
		t.Errorf("vector image semantics missing")
	}
	if !bytes.Contains([]byte(markup), []byte(`data-margo-runtime-task="mermaid"`)) {
		t.Errorf("Mermaid runtime placement missing")
	}
}

func countElements(root *html.Node, tag string) int {
	count := 0
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == tag {
			count++
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return count
}

func countElementsWithClass(root *html.Node, tag, class string) int {
	count := 0
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == tag {
			for _, attribute := range node.Attr {
				if attribute.Key == "class" && strings.Contains(" "+attribute.Val+" ", " "+class+" ") {
					count++
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return count
}
