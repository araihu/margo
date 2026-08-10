package main

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func TestCLIArtifactsAndGeneratedHTMLBrowser(t *testing.T) {
	workspace := t.TempDir()
	binary := filepath.Join(workspace, "margo")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-trimpath", "-o", binary, ".")
	build.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=readonly")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build margo: %v\n%s", err, output)
	}

	article, deckSource := writeCLIFixture(t, workspace)
	articleHTML := filepath.Join(workspace, "article.html")
	deckHTML := filepath.Join(workspace, "deck.html")
	runCLI(t, binary, []string{"html", article, "--output", articleHTML})
	runCLI(t, binary, []string{"deck", deckSource, "--output", deckHTML})
	doctor := runCLI(t, binary, []string{"doctor", "--diagnostics", "json"})
	if !bytes.Contains(doctor.stdout.Bytes(), []byte(`"candidates"`)) || doctor.stderr.Len() != 0 {
		t.Fatalf("doctor stdout = %s stderr = %s", doctor.stdout.Bytes(), doctor.stderr.Bytes())
	}
	version := runCLI(t, binary, []string{"version"})
	if !bytes.Contains(version.stdout.Bytes(), []byte("module github.com/araihu/margo")) || version.stderr.Len() != 0 {
		t.Fatalf("version stdout = %s stderr = %s", version.stdout.Bytes(), version.stderr.Bytes())
	}

	before, err := os.ReadFile(articleHTML)
	if err != nil {
		t.Fatal(err)
	}
	refused := runCLIWantFailure(t, binary, []string{"html", article, "--output", articleHTML})
	if !bytes.Contains(refused.stderr.Bytes(), []byte("margo.atomic.destination_exists")) || refused.stdout.Len() != 0 {
		t.Fatalf("refusal stdout = %q stderr = %q", refused.stdout.String(), refused.stderr.String())
	}
	after, err := os.ReadFile(articleHTML)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("refused overwrite changed the destination")
	}

	invalidOutput := filepath.Join(workspace, "invalid.html")
	invalid := filepath.Join(workspace, "invalid.md")
	if err := os.WriteFile(invalid, []byte("```mermaid\n%%{init: {}}%%\ngraph TD; A-->B\n```\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runCLIWantFailure(t, binary, []string{"html", invalid, "--output", invalidOutput, "--diagnostics", "json"})
	if _, err := os.Stat(invalidOutput); !os.IsNotExist(err) {
		t.Fatalf("invalid output stat = %v", err)
	}

	browser := installedBrowserPath()
	if browser == "" {
		t.Skip("installed Chromium unavailable; process-level HTML assertions passed")
	}
	assertGeneratedArticleDOM(t, browser, articleHTML)
	assertGeneratedDeckDOM(t, browser, deckHTML)

	articlePDF := filepath.Join(workspace, "article.pdf")
	deckPDF := filepath.Join(workspace, "deck.pdf")
	runCLI(t, binary, []string{"pdf", article, "--output", articlePDF, "--engine", "chromium", "--engine-path", browser})
	runCLI(t, binary, []string{"deck", deckSource, "--format", "pdf", "--output", deckPDF, "--engine", "chromium", "--engine-path", browser})
	for _, artifact := range []string{articlePDF, deckPDF} {
		data, err := os.ReadFile(artifact)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(data, []byte("%PDF-")) || len(data) < 1000 {
			t.Fatalf("%s has %d invalid bytes", artifact, len(data))
		}
	}
}

type cliRun struct {
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func runCLI(t *testing.T, binary string, args []string) cliRun {
	t.Helper()
	result, err := executeCLI(binary, args)
	if err != nil {
		t.Fatalf("margo %s: %v\nstdout: %s\nstderr: %s", strings.Join(args, " "), err, result.stdout.String(), result.stderr.String())
	}
	return result
}

func runCLIWantFailure(t *testing.T, binary string, args []string) cliRun {
	t.Helper()
	result, err := executeCLI(binary, args)
	if err == nil {
		t.Fatalf("margo %s unexpectedly succeeded", strings.Join(args, " "))
	}
	return result
}

func executeCLI(binary string, args []string) (cliRun, error) {
	var result cliRun
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, args...)
	command.Stdout = &result.stdout
	command.Stderr = &result.stderr
	err := command.Run()
	return result, err
}

func writeCLIFixture(t *testing.T, root string) (string, string) {
	t.Helper()
	images := filepath.Join(root, "images")
	if err := os.Mkdir(images, 0o700); err != nil {
		t.Fatal(err)
	}
	fixtures := map[string]string{
		"sample.png":  "../../examples/blog/site/assets/format-study.png",
		"sample.jpg":  "../../examples/blog/site/assets/atelier-hero.jpg",
		"sample.webp": "../../examples/blog/site/assets/atelier-hero.webp",
		"sample.gif":  "../../examples/blog/site/assets/format-study.gif",
	}
	for name, source := range fixtures {
		data, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(images, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(images, "sample.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="32" height="20"><rect width="32" height="20" fill="#d84f8b"/></svg>`), 0o600); err != nil {
		t.Fatal(err)
	}
	chart := "```goshtosochart\nschemaVersion: 1\ntype: bar\ntitle: Revenue\ncategories: [Q1]\nseries:\n  - name: Actual\n    values: [12]\n```\n"
	articleText := "# Generated article\n\n[External link](https://example.com/post)\n\n" +
		"![PNG](images/sample.png)\n![JPEG](images/sample.jpg)\n![WebP](images/sample.webp)\n![GIF](images/sample.gif)\n![SVG](images/sample.svg)\n\n" +
		"| Format | Ready |\n|---|---|\n| HTML | yes |\n\n" + chart + "\n```mermaid\ngraph TD; A-->B\n```\n"
	deckText := "# First slide\n\n```mermaid\ngraph TD; A-->B\n```\n---\n# Second slide\n\n![PNG](images/sample.png)\n\n" + chart
	article := filepath.Join(root, "article.md")
	deckPath := filepath.Join(root, "deck.md")
	if err := os.WriteFile(article, []byte(articleText), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(deckPath, []byte(deckText), 0o600); err != nil {
		t.Fatal(err)
	}
	return article, deckPath
}

func installedBrowserPath() string {
	for _, path := range []string{"/opt/homebrew/bin/chromium", "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"} {
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}

func browserContext(t *testing.T, executable string) context.Context {
	t.Helper()
	ctx, cancelTimeout := context.WithTimeout(context.Background(), 25*time.Second)
	t.Cleanup(cancelTimeout)
	options := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	options = append(options, chromedp.ExecPath(executable), chromedp.Flag("allow-file-access-from-files", true))
	allocator, cancelAllocator := chromedp.NewExecAllocator(ctx, options...)
	t.Cleanup(cancelAllocator)
	browser, cancelBrowser := chromedp.NewContext(allocator)
	t.Cleanup(cancelBrowser)
	return browser
}

func assertGeneratedArticleDOM(t *testing.T, browser, path string) {
	t.Helper()
	ctx := browserContext(t, browser)
	location := (&url.URL{Scheme: "file", Path: path}).String()
	var state struct {
		Runtime string `json:"runtime"`
		Images  int    `json:"images"`
		Loaded  int    `json:"loaded"`
		Mermaid int    `json:"mermaid"`
		Charts  int    `json:"charts"`
		Tables  int    `json:"tables"`
		Heading string `json:"heading"`
	}
	err := chromedp.Run(ctx,
		chromedp.Navigate(location),
		chromedp.Evaluate(`(async () => {
			if (globalThis.margoRuntimeReady) await globalThis.margoRuntimeReady;
			const images = [...document.images];
			await Promise.all(images.map((image) => image.decode ? image.decode().catch(() => {}) : Promise.resolve()));
			return {
				runtime: document.documentElement.dataset.margoRuntimeStatus || "",
				images: images.length,
				loaded: images.filter((image) => image.naturalWidth > 0 && image.naturalHeight > 0).length,
				mermaid: document.querySelectorAll('.margo-mermaid__canvas svg').length,
				charts: document.querySelectorAll('[data-goshtoso-chart-content] svg').length,
				tables: document.querySelectorAll('table').length,
				heading: document.querySelector('h1')?.textContent.trim() || ""
			};
		})()`, &state, awaitBrowserPromise),
	)
	if err != nil {
		t.Fatal(err)
	}
	if state.Runtime != "ready" || state.Images != 5 || state.Loaded != 5 || state.Mermaid != 1 || state.Charts < 1 || state.Tables < 2 || state.Heading != "Generated article" {
		t.Fatalf("article DOM = %+v", state)
	}
}

func assertGeneratedDeckDOM(t *testing.T, browser, path string) {
	t.Helper()
	ctx := browserContext(t, browser)
	location := (&url.URL{Scheme: "file", Path: path}).String()
	var state struct {
		Runtime  string `json:"runtime"`
		Mermaid  int    `json:"mermaid"`
		Previous int    `json:"previous"`
		Next     int    `json:"next"`
		Print    int    `json:"print"`
		PrintCSS bool   `json:"printCSS"`
		Keys     []int  `json:"keys"`
	}
	err := chromedp.Run(ctx,
		chromedp.Navigate(location),
		chromedp.Evaluate(`(async () => {
			if (globalThis.margoRuntimeReady) await globalThis.margoRuntimeReady;
			const current = () => Number(document.querySelector('[data-margo-slide][aria-current="page"]')?.dataset.margoSlide ?? -1);
			const keys = [];
			for (const key of ['ArrowRight', 'End', 'Home', 'ArrowLeft']) {
				document.dispatchEvent(new KeyboardEvent('keydown', {key, bubbles: true}));
				keys.push(current());
			}
			return {
				runtime: document.documentElement.dataset.margoRuntimeStatus || "",
				mermaid: document.querySelectorAll('.margo-mermaid__canvas svg').length,
				previous: document.querySelectorAll('[data-margo-deck-previous]').length,
				next: document.querySelectorAll('[data-margo-deck-next]').length,
				print: document.querySelectorAll('[data-margo-deck-print]').length,
				printCSS: [...document.styleSheets].some((sheet) => { try { return [...sheet.cssRules].some((rule) => rule.media?.mediaText.includes('print')); } catch { return false; } }),
				keys
			};
		})()`, &state, awaitBrowserPromise),
	)
	if err != nil {
		t.Fatal(err)
	}
	if state.Runtime != "ready" || state.Mermaid != 1 || state.Previous != 1 || state.Next != 1 || state.Print != 1 || !state.PrintCSS || !reflect.DeepEqual(state.Keys, []int{1, 1, 0, 0}) {
		t.Fatalf("deck DOM = %+v", state)
	}
}

func awaitBrowserPromise(parameters *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams {
	return parameters.WithAwaitPromise(true)
}

func (run cliRun) String() string {
	return fmt.Sprintf("stdout=%q stderr=%q", run.stdout.String(), run.stderr.String())
}
