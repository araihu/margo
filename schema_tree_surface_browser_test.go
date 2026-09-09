package margo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/araihu/margo/internal/browserlaunch"
	"github.com/chromedp/chromedp"
)

func TestSchemaTreeBranchesKeepNeutralSurfaces(t *testing.T) {
	browserPath := installedPrintTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	result := mustRenderSource(t, "```jsonschema\n"+`{"type":"object","properties":{"pages":{"type":"array","items":{"type":"object","properties":{"actions":{"type":"object","properties":{"markdown":{"type":"boolean","enum":[true,false],"description":"Use the `+"`markdown`"+` option."}}}}}}}}`+"\n```\n")
	component, err := RenderStandalone(result)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, component)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(markup))
	}))
	defer server.Close()
	options := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	options = append(options, chromedp.ExecPath(browserPath))
	allocator, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), options...)
	defer cancelAllocator()
	browser, cancelBrowser := chromedp.NewContext(allocator)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browser, 30*time.Second)
	defer cancel()
	if err := chromedp.Run(ctx, chromedp.Navigate(server.URL), chromedp.WaitVisible(".gs-schema-tree", chromedp.ByQuery)); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"light", "dark"} {
		t.Run(mode, func(t *testing.T) {
			var state struct {
				Branches    int  `json:"branches"`
				Neutral     bool `json:"neutral"`
				ProseStyled bool `json:"proseStyled"`
				Guides      bool `json:"guides"`
				CodeOwned   bool `json:"codeOwned"`
				ProseCode   bool `json:"proseCode"`
			}
			script := `(() => {
				document.documentElement.classList.toggle('dark', '` + mode + `' === 'dark');
				const prose = document.querySelector('.margo-document');
				let disclosure = document.getElementById('ordinary-disclosure');
				if (!disclosure) {
					disclosure = document.createElement('details');
					disclosure.id = 'ordinary-disclosure';
					disclosure.innerHTML = '<summary>Ordinary disclosure</summary><code>Body</code>';
					prose.append(disclosure);
				}
				const branches = [...document.querySelectorAll('details.gs-schema-tree-node')];
				const constraints = [...document.querySelectorAll('.gs-schema-tree-constraints code')];
				const names = [...document.querySelectorAll('.gs-schema-tree-name')];
				const descriptionCode = document.querySelector('.gs-schema-tree-description code');
				return {
					branches: branches.length,
					neutral: branches.every(node => getComputedStyle(node).backgroundColor === 'rgba(0, 0, 0, 0)'),
					proseStyled: getComputedStyle(disclosure).backgroundColor !== 'rgba(0, 0, 0, 0)',
					guides: branches.every(node =>
						parseFloat(getComputedStyle(node, '::before').borderInlineStartWidth) > 0 ||
						parseFloat(getComputedStyle(node.querySelector(':scope > .gs-schema-tree-children')).borderInlineStartWidth) > 0),
					codeOwned: constraints.length > 0 && names.length > 0 && [...constraints, ...names].every(node => {
						const style = getComputedStyle(node);
						return style.borderTopWidth === '0px' && style.paddingLeft === '0px' && style.marginLeft === '0px';
					}),
					proseCode: !!descriptionCode && [descriptionCode, disclosure.querySelector('code')].every(node => getComputedStyle(node).borderTopWidth === '1px')
				};
			})()`
			if err := chromedp.Run(ctx, chromedp.Evaluate(script, &state)); err != nil {
				t.Fatal(err)
			}
			if state.Branches < 2 || !state.Neutral || !state.ProseStyled || !state.Guides || !state.CodeOwned || !state.ProseCode {
				t.Fatalf("schema tree surface state = %+v", state)
			}
		})
	}
}
