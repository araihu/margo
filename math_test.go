package margo

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestMathCompileAndCheck(t *testing.T) {
	ctx := context.Background()
	source := Source{Name: "math.md", Content: []byte("---\ntitle: Math\n---\n\n# $x$\n\nInline $E=mc^2$.\n\n$$x^2$$\n\n```math\n\\frac{1}{2}\n```\n")}
	c := New(WithMath(true))
	doc, err := c.Compile(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	result, err := c.Render(ctx, doc)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, result.content)
	if strings.Count(markup, `data-margo-runtime-task="math"`) != 4 {
		t.Fatalf("markers: %s", markup)
	}
	descriptor, err := result.RuntimeDescriptor("ri-00000001")
	if err != nil {
		t.Fatal(err)
	}
	if len(descriptor.Tasks) != 4 || descriptor.Tasks[0].Kind != "math" {
		t.Fatalf("tasks: %+v", descriptor.Tasks)
	}
	for _, id := range []string{"margo.math.engine", "margo.math.styles", "margo.math.execute", "margo.runtime.coordinator"} {
		found := false
		for _, r := range result.projectedHTMLRequirements().List() {
			found = found || r.ID == id
		}
		if !found {
			t.Fatal("missing requirement", id)
		}
	}
	findings, err := Check(ctx, source, WithCheckMath(true))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range findings {
		if d.Code == "check.math_runtime_validation_required" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing runtime-validation warning")
	}
}

func TestMathDisabledAndUnused(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		c := New(WithMath(enabled))
		source := "# Plain\n\nCost $20 and $30.\n"
		if !enabled {
			source += "\n$x$\n\n```math\nx\n```\n"
		}
		doc, err := c.Compile(context.Background(), Source{Content: []byte(source)})
		if err != nil {
			t.Fatal(err)
		}
		result, err := c.Render(context.Background(), doc)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range result.projectedHTMLRequirements().List() {
			if strings.HasPrefix(r.ID, "margo.math.") {
				t.Fatal("unused math requirement", r.ID)
			}
		}
		if len(result.runtimeTasks) != 0 {
			t.Fatal("unused math task")
		}
	}
	if New().fingerprint != New(WithMath(false)).fingerprint {
		t.Fatal("disabled setting changes fingerprint")
	}
}

func TestMathDiagnosticsUseSourcePositions(t *testing.T) {
	source := Source{Name: "bad.md", Content: []byte("---\ntitle: Bad\n---\n\n$$\nx\n")}
	_, err := New(WithMath(true)).Compile(context.Background(), source)
	diagnostic := unwrapDiagnostic(err)
	if diagnostic == nil || diagnostic.Diagnostics[0].Code != "math.delimiter_unclosed" || diagnostic.Diagnostics[0].Line != 5 || diagnostic.Diagnostics[0].Column != 1 {
		t.Fatalf("diagnostics: %+v", diagnostic)
	}
	findings, err := Check(context.Background(), source, WithCheckMath(true))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range findings {
		found = found || d.Code == "math.delimiter_unclosed" && d.Line == 5
	}
	if !found {
		t.Fatalf("check: %+v", findings)
	}
}

func TestMathPublicationSourceLimits(t *testing.T) {
	for _, source := range []string{strings.Repeat("$x$ ", 1025), strings.Repeat("$"+strings.Repeat("x", 16<<10)+"$\n", 65)} {
		_, err := New(WithMath(true)).Compile(context.Background(), Source{Name: "limits.md", Content: []byte(source)})
		diagnostic := unwrapDiagnostic(err)
		if diagnostic == nil || diagnostic.Diagnostics[0].Code != "math.document_limit_exceeded" {
			t.Fatalf("limits: %v", err)
		}
	}
}

func TestMathImageAltRetainsOrdinaryMarkdown(t *testing.T) {
	for _, source := range []string{
		`![$*em*$ and $` + "`code`" + `$](image.png)`,
		`![$a [label](target) b$](image.png)`,
		`![$x](first.png) y$](second.png)`,
		`![$x$](first.png) and ![$**y**$](second.png)`,
		`![$*x*$][image]` + "\n\n[image]: image.png\n",
	} {
		var output [2]string
		for i, enabled := range []bool{false, true} {
			c := New(WithMath(enabled))
			doc, err := c.Compile(context.Background(), Source{Name: "alt.md", Content: []byte(source)})
			if err != nil {
				t.Fatal(err)
			}
			r, err := c.Render(context.Background(), doc)
			if err != nil {
				t.Fatal(err)
			}
			output[i] = renderComponent(t, r.Content())
			if len(r.runtimeTasks) != 0 {
				t.Fatalf("alt text became math: %q", source)
			}
		}
		if output[0] != output[1] {
			t.Fatalf("math changed image semantics for %q:\n%s\n%s", source, output[0], output[1])
		}
	}
}

func TestMathPlainTextIncludesAllFormsOnce(t *testing.T) {
	c := New(WithMath(true))
	doc, err := c.Compile(context.Background(), Source{Name: "text.md", Content: []byte("# Formula $x^2$\n\nInline $y_i$.\n\n$$a+b$$\n\n```math\nc+d\n```\n")})
	if err != nil {
		t.Fatal(err)
	}
	r, err := c.Render(context.Background(), doc)
	if err != nil {
		t.Fatal(err)
	}
	editorial, err := RenderHTML(r)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := editorial.PlainText(), "Formula x^2 Inline y_i. a+b c+d"; got != want {
		t.Fatalf("plain text = %q, want %q", got, want)
	}
	if editorial.Metadata().Title != "Formula x^2" {
		t.Fatalf("title = %q", editorial.Metadata().Title)
	}
}

func TestMathOrdinalsFollowSourceAfterFootnoteRelocation(t *testing.T) {
	c := New(WithMath(true))
	doc, err := c.Compile(context.Background(), Source{Name: "order.md", Content: []byte("See[^f].\n\n[^f]: $x$\n\n$y$\n")})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.plan.mathTasks) != 2 || doc.plan.mathTasks[0].Position.Line != 3 || doc.plan.mathTasks[1].Position.Line != 5 {
		t.Fatalf("math tasks lost source order: %+v", doc.plan.mathTasks)
	}
}

func TestMathConcurrentCompilerAndRender(t *testing.T) {
	c := New(WithMath(true))
	source := Source{Name: "concurrent.md", Content: []byte("# Shared\n\n$x$\n\n$$y$$\n")}
	ctx := context.Background()
	doc, err := c.Compile(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := c.Render(ctx, doc)
	if err != nil {
		t.Fatal(err)
	}
	var expected bytes.Buffer
	if err := baseline.Content().Render(ctx, &expected); err != nil {
		t.Fatal(err)
	}
	failures := make(chan error, 16)
	var workers sync.WaitGroup
	for i := 0; i < 16; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			compiled, err := c.Compile(ctx, source)
			if err != nil {
				failures <- err
				return
			}
			for _, d := range []*Document{compiled, doc} {
				result, err := c.Render(ctx, d)
				if err != nil {
					failures <- err
					return
				}
				var got bytes.Buffer
				if err := result.Content().Render(ctx, &got); err != nil {
					failures <- err
					return
				}
				if !bytes.Equal(expected.Bytes(), got.Bytes()) {
					failures <- fmt.Errorf("concurrent math output changed")
					return
				}
			}
		}()
	}
	workers.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
}
