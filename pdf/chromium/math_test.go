package chromium

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/araihu/margo"
	"github.com/araihu/margo/deck"
	"github.com/araihu/margo/internal/browserlaunch"
	"github.com/araihu/margo/pdf"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

func mathRequest(t *testing.T, source string) pdf.Request {
	t.Helper()
	ctx := context.Background()
	c := margo.New(margo.WithMath(true))
	doc, err := c.Compile(ctx, margo.Source{Name: "math.md", Content: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	r, err := c.Render(ctx, doc, margo.WithRenderTarget(margo.TargetPDF))
	if err != nil {
		t.Fatal(err)
	}
	component, err := margo.RenderStandalone(r)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := component.Render(ctx, &output); err != nil {
		t.Fatal(err)
	}
	descriptor, err := r.RuntimeDescriptor("ri-math0001")
	if err != nil {
		t.Fatal(err)
	}
	return pdf.Request{HTML: output.Bytes(), Runtime: descriptor, ExecutionID: "exec-math", Page: pdf.PageConfig{Size: pdf.PageA4, Margins: pdf.Margins{Top: 10, Bottom: 10, Left: 10, Right: 10}}}
}

func TestMathDeckScreenIgnoresAccessibleDuplicate(t *testing.T) {
	path := installedChromium()
	if path == "" {
		t.Skip("installed Chromium unavailable")
	}
	source := "# Energy\n\n$E=mc^2$\n\n---\n\n# Integral\n\n$$\\int_0^\\infty e^{-x^2}\\,dx=\\frac{\\sqrt{\\pi}}2$$\n"
	result, err := deck.Render(context.Background(), margo.New(margo.WithMath(true)), deck.RenderInput{Name: "screen.md", Markdown: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	server := documentServer(result.HTML())
	defer server.Close()
	options := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	options = append(options, chromedp.ExecPath(path), chromedp.NoSandbox, chromedp.DisableGPU)
	a, cancel := browserlaunch.NewExecAllocator(context.Background(), options...)
	defer cancel()
	b, cancel := chromedp.NewContext(a)
	defer cancel()
	ctx, cancel := context.WithTimeout(b, 30*time.Second)
	defer cancel()
	for _, width := range []int64{1440, 390} {
		var valid bool
		if err := chromedp.Run(ctx, emulation.SetDeviceMetricsOverride(width, 900, 1, false), chromedp.Navigate(server.URL),
			chromedp.Evaluate(`(async()=>{await margoRuntimeReady;await document.fonts.ready;return true})()`, nil, awaitPromise),
			chromedp.Poll(`document.querySelector('.margo-deck').dataset.margoScreenRuntime === 'ready'`, nil),
			chromedp.Evaluate(`!document.querySelector('[data-margo-screen-diagnostic]') && margoValidateDeckScreen()`, &valid),
			chromedp.Click("[data-margo-deck-next]", chromedp.ByQuery),
			chromedp.Evaluate(`!document.querySelector('[data-margo-screen-diagnostic]') && margoValidateDeckScreen()`, &valid),
		); err != nil {
			t.Fatal(err)
		}
		if !valid {
			t.Fatalf("valid equation marked as slide overflow at %dpx", width)
		}
		// Excluding the MathML duplicate must not hide real visual overflow.
		if err := chromedp.Run(ctx, chromedp.Evaluate(`(()=>{const n=document.querySelector('.margo-deck__slide:not([hidden]) .katex-html');n.style.display='block';n.style.width='5000px';return margoValidateDeckScreen()})()`, &valid)); err != nil {
			t.Fatal(err)
		}
		if valid {
			t.Fatal("visible math overflow was accepted")
		}
	}
}

func TestMathPDFEvidenceAndFailures(t *testing.T) {
	path := installedChromium()
	if path == "" {
		t.Skip("installed Chromium unavailable")
	}
	engine, err := New(Config{ExecutablePath: path, Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct{ name, source, wantError string }{
		{"valid", "# Math\n\n$x$\n\n$$\\frac{1}{2}$$\n\n```math\n\\sqrt{x}\n```\n", ""},
		{"footnote-order", "See[^f].\n\n[^f]: $x$\n\n$y$\n", ""},
		{"mixed", "# Mixed\n\n$x$\n\n```mermaid\nflowchart LR\n A --> B\n```\n", ""},
		{"invalid", "# Invalid\n\n$\\invalidMath{x}$\n", "math.expression_invalid"},
		{"forbidden", "# Forbidden\n\n$\\href{https://example.com}{x}$\n", "math.command_forbidden"},
		{"overflow", "# Wide\n\n$$" + strings.Repeat("x+", 200) + "x$$\n", "math.print_overflow"},
		{"inline-height", "# Tall\n\n$\\rule{1em}{20em}$\n", "math.print_overflow"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := mathRequest(t, tt.source)
			if tt.name == "inline-height" {
				request.Page.Size = ""
				request.Page.Custom = &pdf.CustomPageSize{WidthMM: 100, HeightMM: 60}
			}
			result, err := engine.Export(context.Background(), request)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) || len(result.PDF) > 0 {
					t.Fatalf("error=%v PDF=%d runtime=%+v", err, len(result.PDF), result.Runtime)
				}
				if tt.wantError == "math.print_overflow" {
					if result.Runtime.Diagnostic == nil || result.Runtime.Diagnostic.Source != "math.md" || result.Runtime.Diagnostic.Line != 3 {
						t.Fatalf("missing source-positioned print diagnostic: %+v", result.Runtime)
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasPrefix(result.PDF, []byte("%PDF-")) || result.Runtime.Status != margo.RuntimeReady {
				t.Fatal("missing ready PDF")
			}
			if len(result.Runtime.FontChecks) != 20 {
				t.Fatalf("font evidence: %+v", result.Runtime.FontChecks)
			}
			for _, task := range result.Runtime.Tasks {
				if task.Kind == "math" && (task.OutputBytes < 100 || task.OutputSHA256 == "" || task.Status != margo.RuntimeTaskSucceeded) {
					t.Fatalf("math evidence: %+v", task)
				}
			}
		})
	}
}

func TestMathDeckPDFIncludesFontLock(t *testing.T) {
	path := installedChromium()
	if path == "" {
		t.Skip("installed Chromium unavailable")
	}
	c := margo.New(margo.WithMath(true))
	result, err := deck.Render(context.Background(), c, deck.RenderInput{Name: "slides.md", Markdown: []byte("# First\n\n$x$\n\n---\n\n# Second\n\n$$\\frac{1}{2}$$\n")})
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := result.RuntimeDescriptor("ri-math0001")
	if err != nil {
		t.Fatal(err)
	}
	engine, err := New(Config{ExecutablePath: path, Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	request := pdf.Request{HTML: result.HTML(), Runtime: descriptor, ExecutionID: "exec-math", Page: pdf.PageConfig{Custom: &pdf.CustomPageSize{WidthMM: 338.6666667, HeightMM: 190.5}}}
	exported, err := engine.Export(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if exported.Runtime.ValidationIdentity == nil || exported.Runtime.ValidationIdentity.FontBundleDigest != descriptor.ValidationRequest.ExpectedFontBundleDigest {
		t.Fatalf("font lock: %+v", exported.Runtime)
	}
	if len(exported.Runtime.FontChecks) <= 20 {
		t.Fatal("missing deck or math font evidence")
	}
}

func TestMathDeckPrintRejectsClippedInlineEquation(t *testing.T) {
	path := installedChromium()
	if path == "" {
		t.Skip("installed Chromium unavailable")
	}
	r, err := deck.Render(context.Background(), margo.New(margo.WithMath(true)), deck.RenderInput{Name: "slides.md", Markdown: []byte("# Tall\n\n" + strings.Repeat("A paragraph.\n\n", 20) + "$\\rule{1em}{16em}$\n")})
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := r.RuntimeDescriptor("ri-math0001")
	if err != nil {
		t.Fatal(err)
	}
	engine, err := New(Config{ExecutablePath: path, Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.Export(context.Background(), pdf.Request{HTML: r.HTML(), Runtime: descriptor, ExecutionID: "exec-math", Page: pdf.PageConfig{Custom: &pdf.CustomPageSize{WidthMM: 338.6666667, HeightMM: 190.5}}})
	if err == nil || !strings.Contains(err.Error(), "math.print_overflow") || len(result.PDF) > 0 {
		t.Fatalf("clipped equation exported: error=%v PDF=%d", err, len(result.PDF))
	}
	if result.Runtime.Diagnostic == nil || result.Runtime.Diagnostic.Source != "slides.md" || result.Runtime.Diagnostic.Line != 43 {
		t.Fatalf("missing original position: %+v", result.Runtime)
	}
}

func TestMathRejectsMissingDuplicateAndForgedEvidence(t *testing.T) {
	request := mathRequest(t, "# Evidence\n\n$x$\n")
	task := request.Runtime.Tasks[0]
	valid := browserMathTask{ID: task.ID, InputSHA256: task.InputSHA256, Output: "typeset math", Status: "succeeded"}
	fonts := make([]margo.FontCheck, 20)
	for i := range fonts {
		fonts[i] = margo.FontCheck{Family: fmt.Sprintf("KaTeX_Face%d", i), Query: fmt.Sprintf("400 16px Face%d", i), Loaded: true}
	}
	for _, test := range []struct {
		name   string
		tasks  []browserMathTask
		checks []margo.FontCheck
	}{
		{"missing", nil, fonts},
		{"duplicate", []browserMathTask{valid, valid}, fonts},
		{"wrong-input", []browserMathTask{{ID: task.ID, InputSHA256: strings.Repeat("a", 64), Output: "typeset math", Status: "succeeded"}}, fonts},
		{"empty-output", []browserMathTask{{ID: task.ID, InputSHA256: task.InputSHA256, Status: "succeeded"}}, fonts},
		{"missing-fonts", []browserMathTask{valid}, nil},
		{"duplicate-fonts", []browserMathTask{valid}, append(append([]margo.FontCheck{}, fonts[:19]...), fonts[0])},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateMathOutput(request.Runtime, browserMathOutput{Tasks: test.tasks, FontChecks: test.checks}); err == nil {
				t.Fatal("invalid evidence accepted")
			}
		})
	}
	request.Runtime.Tasks[0].Kind = "unknown-kind"
	if err := validateRequest(request); err == nil || !strings.Contains(err.Error(), "runtime.task_unknown") {
		t.Fatalf("unknown executor accepted: %v", err)
	}
}
