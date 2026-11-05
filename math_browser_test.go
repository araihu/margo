package margo

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/araihu/margo/internal/browserlaunch"
	"github.com/araihu/margo/internal/mathfonts"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

func TestMathBrowserOfflineRenderingAndFailures(t *testing.T) {
	path := installedPrintTestChromium()
	if path == "" {
		t.Skip("installed Chromium unavailable")
	}
	options := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	options = append(options, chromedp.ExecPath(path), chromedp.NoSandbox, chromedp.DisableGPU)
	a, cancel := browserlaunch.NewExecAllocator(context.Background(), options...)
	defer cancel()
	b, cancel := chromedp.NewContext(a)
	defer cancel()
	ctx, cancel := context.WithTimeout(b, 60*time.Second)
	defer cancel()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	directory, err := os.MkdirTemp(home, "margo-math-browser-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	cases := []struct {
		name, source string
		count        int
		failed       bool
	}{
		{"forms", "# Equations\n\nInline $E=mc^2$ and price $20 and $30.\n\n$$\\frac{1}{2}$$\n\n```math\n\\begin{aligned}a&=b+c\\\\d&=e+f\\end{aligned}\n```\n", 3, false},
		{"table", "# Table\n\n| $x$ |\n|---|\n| $2$ |\n| $1$ |\n", 3, false},
		{"wide", "# Wide\n\n$$" + strings.Repeat("x+", 200) + "x$$\n", 1, false},
		{"mixed", "# Mixed\n\n$x$\n\n```mermaid\nflowchart LR\n A --> B\n```\n", 1, false},
		{"invalid", "# Invalid\n\n$\\notARealCommand{x}$\n", 1, true},
		{"invalid-dark", "# Invalid\n\n$\\notARealCommand{x}$\n", 1, true},
		{"long-error", "# Invalid\n\n$\\" + strings.Repeat("unknown", 100) + "$\n", 1, true},
		{"trust", "# Trust\n\n$\\href{https://example.com}{x}$\n", 1, true},
		{"macros", "# Isolation\n\n$\\gdef\\custom{x}\\custom$ then $\\custom$\n", 2, true},
		{"loop", "# Expansion\n\n$\\def\\loop{\\loop}\\loop$\n", 1, true},
		{"budget", "# Budget\n\n$\\frac{1}{2}$\n", 1, true},
		{"broken-font", "# Fonts\n\n$x$\n", 1, true},
		{"missing-engine", "# Engine\n\n$x$\n", 1, true},
	}
	for _, theme := range []string{"modern", "goshtoso", "minimal"} {
		for _, mode := range []string{"light", "dark"} {
			cases = append(cases, struct {
				name, source string
				count        int
				failed       bool
			}{"theme-" + theme + "-" + mode, "# Theme\n\nInline $x_i^2$.\n\n$$\\frac{\\sqrt{\\pi}}{2}$$\n", 2, false})
		}
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			options := []Option{WithMath(true)}
			if tt.name == "budget" {
				options = append(options, WithHostPolicy(Policy{OutputBytes: 900}))
			}
			c := New(options...)
			doc, err := c.Compile(ctx, Source{Name: tt.name + ".md", Content: []byte(tt.source)})
			if err != nil {
				t.Fatal(err)
			}
			r, err := c.Render(ctx, doc)
			if err != nil {
				t.Fatal(err)
			}
			var standaloneOptions []any
			if tt.name == "invalid-dark" {
				standaloneOptions = append(standaloneOptions, WithStandaloneColorMode(ColorModeDark))
			}
			if strings.HasPrefix(tt.name, "theme-") {
				parts := strings.Split(tt.name, "-")
				standaloneOptions = append(standaloneOptions, WithStandaloneTheme(ThemeName(parts[1])), WithStandaloneColorMode(ColorMode(parts[2])))
			}
			page, err := RenderStandalone(r, standaloneOptions...)
			if err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(directory, tt.name+".html")
			markup := renderComponent(t, page)
			if tt.name == "missing-engine" {
				markup = strings.Replace(markup, "</head>", "<script>delete globalThis.katex;</script></head>", 1)
			}
			if tt.name == "broken-font" {
				start := strings.Index(markup, "src:url(data:font/woff2;base64,")
				end := start + strings.Index(markup[start:], ")")
				markup = markup[:start] + "src:url(data:font/woff2;base64,AAAA" + markup[end:]
			}
			if err := os.WriteFile(file, []byte(markup), 0600); err != nil {
				t.Fatal(err)
			}
			var state struct {
				Digest                                 string
				Status                                 string
				Math, Rendered, Errors, Fonts, Mermaid int
			}
			err = chromedp.Run(ctx, network.Enable(), network.SetBlockedURLs([]string{"http://*", "https://*"}), emulation.SetDeviceMetricsOverride(390, 844, 1, false), chromedp.Navigate((&url.URL{Scheme: "file", Path: filepath.ToSlash(file)}).String()), chromedp.Evaluate(`(async()=>{
		  await globalThis.margoRuntimeReady?.catch(()=>{});
		  const evidence = document.documentElement.dataset.margoRuntimeStatus === "ready" ? await globalThis.margoGetMathFontEvidence() : {fontBundleDigest:""};
		  return {digest:evidence.fontBundleDigest,status:document.documentElement.dataset.margoRuntimeStatus,math:document.querySelectorAll('.margo-math').length,rendered:document.querySelectorAll('.margo-math .katex-mathml math').length,errors:document.querySelectorAll('.margo-math__error:not([hidden])').length,fonts:(globalThis.margoMathFontChecks||[]).filter(x=>x.loaded).length,mermaid:document.querySelectorAll('.margo-mermaid__canvas svg').length};
		})()`, &state, func(p *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams { return p.WithAwaitPromise(true) }))
			if err != nil {
				t.Fatal(err)
			}
			if state.Math != tt.count {
				t.Fatalf("state: %+v", state)
			}
			if tt.failed {
				if state.Status != "failed" || state.Errors == 0 {
					t.Fatalf("failure state: %+v", state)
				}
			} else if state.Status != "ready" || state.Rendered != tt.count || state.Fonts != 20 {
				t.Fatalf("render state: %+v", state)
			}
			if tt.name == "invalid-dark" || tt.name == "long-error" {
				var readable bool
				if err := chromedp.Run(ctx, chromedp.Evaluate(`(()=>{
                  const error=document.querySelector('.margo-math__error:not([hidden])');
                  const rgb=value=>{const c=document.createElement("canvas");c.width=c.height=1;const x=c.getContext("2d");x.fillStyle=value;x.fillRect(0,0,1,1);return [...x.getImageData(0,0,1,1).data].slice(0,3)};
                  const luminance=color=>rgb(color).map(x=>{x/=255;return x<=.04045?x/12.92:((x+.055)/1.055)**2.4}).reduce((s,x,i)=>s+x*[.2126,.7152,.0722][i],0);
                  let parent=error,bg='rgb(255,255,255)';
                  while(parent){const color=getComputedStyle(parent).backgroundColor;if(color!=='rgba(0, 0, 0, 0)'&&color!=='transparent'){bg=color;break}parent=parent.parentElement}
                  const a=luminance(getComputedStyle(error).color),b=luminance(bg);
                  return (Math.max(a,b)+.05)/(Math.min(a,b)+.05)>=4.5 && document.documentElement.scrollWidth<=innerWidth;
                })()`, &readable)); err != nil {
					t.Fatal(err)
				}
				if !readable {
					t.Fatal("math failure text lacks contrast or overflows the mobile viewport")
				}
			}

			if tt.name == "mixed" && state.Mermaid != 1 {
				t.Fatalf("mixed: %+v", state)
			}
			if tt.name == "table" {
				var sorted bool
				if err := chromedp.Run(ctx, chromedp.Click("table thead th", chromedp.ByQuery), chromedp.Evaluate(`document.querySelectorAll('.margo-math math').length === 3 && document.querySelector('table tbody .margo-math__source').textContent === '1'`, &sorted)); err != nil {
					t.Fatal(err)
				}
				if !sorted {
					t.Fatal("sorting lost typeset math or used duplicate math text")
				}
			}
			if tt.name == "wide" {
				var accessible bool
				if err := chromedp.Run(ctx, chromedp.Evaluate(`(()=>{const n=document.querySelector('.margo-math--display');return n.tabIndex===0 && n.getAttribute('role')==='region' && n.querySelector('.katex-base').getBoundingClientRect().left >= n.getBoundingClientRect().left-1})()`, &accessible), chromedp.Focus(".margo-math--display", chromedp.ByQuery), chromedp.KeyEvent(kb.ArrowRight), chromedp.Poll(`document.querySelector('.margo-math--display').scrollLeft > 0`, nil)); err != nil {
					t.Fatal(err)
				}
				if !accessible {
					t.Fatal("wide math is cropped or lacks accessible scrolling")
				}
			}
			if !tt.failed {
				css, _ := EmbeddedAsset("math.css")
				expected, err := mathfonts.Digest(css.Content)
				if err != nil || state.Digest != expected {
					var metadata []string
					chromedp.Run(ctx, chromedp.Evaluate(`[...document.querySelector('[data-margo-requirement="margo.math.styles"]').sheet.cssRules].filter(r=>r.type===CSSRule.FONT_FACE_RULE).map(r=>[r.style.fontFamily,r.style.fontStyle,r.style.fontWeight].join("|"))`, &metadata))
					t.Fatalf("font digest browser=%s Go=%s error=%v metadata=%v", state.Digest, expected, err, metadata)
				}
			}
			if strings.HasPrefix(tt.name, "theme-") {
				var valid bool
				if err := chromedp.Run(ctx, emulation.SetDeviceMetricsOverride(1440, 900, 1, false), chromedp.Evaluate(`document.querySelectorAll('.margo-math math').length === 2 && [...document.querySelectorAll('.margo-math__output')].every(n=>n.getBoundingClientRect().width>0)`, &valid)); err != nil {
					t.Fatal(err)
				}
				if !valid {
					t.Fatal("wide theme projection lost math")
				}
			}
		})
	}
}
