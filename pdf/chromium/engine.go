// Package chromium exports immutable Margo HTML through an explicitly selected
// installed Chromium-family executable. It never downloads a browser.
package chromium

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/assets"
	"github.com/araihu/margo/pdf"
)

const defaultTimeout = 30 * time.Second

// Config binds the engine to one already-installed browser executable.
type Config struct {
	ExecutablePath string
	Timeout        time.Duration
}

// Engine is an explicitly configured Chromium PDF renderer.
type Engine struct {
	executablePath string
	timeout        time.Duration
}

var _ pdf.Engine = (*Engine)(nil)

// New validates the executable identity without launching or downloading it.
func New(config Config) (*Engine, error) {
	path, err := filepath.Abs(strings.TrimSpace(config.ExecutablePath))
	if err != nil || strings.TrimSpace(config.ExecutablePath) == "" {
		return nil, chromiumError("pdf.chromium.path_invalid", "browser executable path is required")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return nil, chromiumError("pdf.chromium.path_invalid", "browser path is not an executable file")
	}
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Engine{executablePath: path, timeout: timeout}, nil
}

func (*Engine) Name() string { return "chromium" }

// Version reports the selected executable's own version string.
func (engine *Engine) Version(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, engine.timeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, engine.executablePath, "--version").CombinedOutput()
	if err != nil {
		return "", chromiumError("pdf.chromium.version_failed", strings.TrimSpace(string(output)))
	}
	version := strings.TrimSpace(string(output))
	if version == "" {
		return "", chromiumError("pdf.chromium.version_failed", "browser returned an empty version")
	}
	return version, nil
}

// Export launches only the selected executable, waits for document resources,
// and prints the fully loaded page. A render failure is terminal: callers must
// not silently fall back to another engine.
func (engine *Engine) Export(ctx context.Context, request pdf.Request) (pdf.Result, error) {
	request = request.Clone()
	if err := validateRequest(request); err != nil {
		return pdf.Result{}, err
	}
	for _, task := range request.Runtime.Tasks {
		if task.Kind != "mermaid" {
			return pdf.Result{}, chromiumError("pdf.runtime_unsupported", "runtime task kind "+task.Kind+" is not implemented")
		}
	}
	var err error
	request.HTML, err = rewriteDocumentLinks(request.HTML, request.RelativeLinks, request.BaseURL)
	if err != nil {
		return pdf.Result{}, err
	}
	request.HTML, err = injectPageGeometry(request.HTML, request.Page)
	if err != nil {
		return pdf.Result{}, err
	}

	exportCtx, cancelExport := context.WithTimeout(ctx, engine.timeout)
	defer cancelExport()
	server := documentServer(request.HTML)
	defer server.Close()

	profile, err := os.MkdirTemp("", "margo-chromium-profile-")
	if err != nil {
		return pdf.Result{}, chromiumError("pdf.chromium.profile_failed", err.Error())
	}
	defer os.RemoveAll(profile)

	options := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	options = append(options,
		chromedp.ExecPath(engine.executablePath),
		chromedp.UserDataDir(profile),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("host-resolver-rules", "MAP * 0.0.0.0, EXCLUDE 127.0.0.1, EXCLUDE localhost"),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
	)
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(exportCtx, options...)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx)
	defer cancelBrowser()

	var metrics margo.LayoutMetrics
	var runtimeOutput browserRuntimeOutput
	var pdfBytes []byte
	if err := chromedp.Run(browserCtx,
		chromedp.Navigate(server.URL),
		chromedp.Evaluate(runtimeExpression, &runtimeOutput, awaitPromise),
		chromedp.Evaluate(`(async () => {
			if (typeof globalThis.margoPreparePrint === "function") await globalThis.margoPreparePrint();
			if (typeof globalThis.margoPrepareDeckPrint === "function") await globalThis.margoPrepareDeckPrint();
			return true;
		})()`, nil, awaitPromise),
		chromedp.Evaluate(`(async () => {
			await document.fonts.ready;
			await Promise.all(Array.from(document.images).map((image) => image.complete ? true : new Promise((resolve, reject) => {
				image.addEventListener("load", resolve, {once: true});
				image.addEventListener("error", reject, {once: true});
			})));
			return {scrollWidth: document.documentElement.scrollWidth, scrollHeight: document.documentElement.scrollHeight};
		})()`, &metrics, awaitPromise),
		chromedp.ActionFunc(func(ctx context.Context) error {
			params := page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				WithGenerateTaggedPDF(true).
				WithGenerateDocumentOutline(true)
			var err error
			pdfBytes, _, err = params.Do(ctx)
			return err
		}),
	); err != nil {
		return pdf.Result{}, chromiumError("pdf.chromium.export_failed", err.Error())
	}
	if !strings.HasPrefix(string(pdfBytes), "%PDF-") {
		return pdf.Result{}, chromiumError("pdf.chromium.output_invalid", "browser returned invalid PDF bytes")
	}
	if len(runtimeOutput.SVG) != len(request.Runtime.Tasks) {
		return pdf.Result{}, chromiumError("pdf.runtime_task_mismatch", "document runtime markers do not match the descriptor")
	}
	version, err := engine.Version(exportCtx)
	if err != nil {
		return pdf.Result{}, err
	}
	report := margo.RuntimeReport{
		Protocol:            request.Runtime.Protocol,
		DocumentFingerprint: request.Runtime.DocumentFingerprint,
		RenderInstanceID:    request.Runtime.RenderInstanceID,
		ExecutionID:         request.ExecutionID,
		Status:              margo.RuntimeReady,
		Tasks:               runtimeTaskReports(request.Runtime.Tasks, runtimeOutput.SVG),
		FontChecks:          []margo.FontCheck{},
		BlockedRequests:     []margo.BlockedRequest{},
		Layout:              metrics,
	}
	if err := margo.ValidateRuntimeReport(request.Runtime, request.ExecutionID, report); err != nil {
		return pdf.Result{}, chromiumError("pdf.runtime_report_invalid", err.Error())
	}
	return pdf.Result{
		PDF:     pdfBytes,
		Runtime: report,
		Engine:  pdf.EngineInfo{Name: engine.Name(), Version: version},
	}, nil
}

func validateRequest(request pdf.Request) error {
	if len(request.HTML) == 0 || request.ExecutionID == "" {
		return chromiumError("pdf.request_invalid", "HTML and execution ID are required")
	}
	if err := request.Page.Validate(); err != nil {
		return chromiumError("pdf.request_invalid", err.Error())
	}
	if err := margo.ValidateRuntimeDescriptor(request.Runtime); err != nil {
		return chromiumError("pdf.request_invalid", err.Error())
	}
	if err := validateOfflineHTML(request.HTML); err != nil {
		return err
	}
	return nil
}

func documentServer(document []byte) *httptest.Server {
	mux := http.NewServeMux()
	mux.Handle("/margo-assets/", http.StripPrefix("/margo-assets", assets.MuambaHTTPHandler()))
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/" {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.Header().Set("Cache-Control", "no-store")
		_, _ = writer.Write(document)
	})
	return httptest.NewServer(mux)
}

type browserRuntimeOutput struct {
	SVG []string `json:"svg"`
}

const runtimeExpression = `(async () => {
	const nodes = Array.from(document.querySelectorAll('[data-margo-runtime-task="mermaid"]'));
	if (nodes.length === 0) return {svg: []};
	if (globalThis.margoRuntimeReady && typeof globalThis.margoRuntimeReady.then === 'function') {
		await globalThis.margoRuntimeReady;
		const embedded = nodes.map((node) => node.querySelector('.margo-mermaid__canvas svg')?.outerHTML ?? '');
		if (embedded.every((svg) => svg.length > 0)) return {svg: embedded};
	}
	const mermaid = (await import('/margo-assets/mermaid/11.16.1/mermaid.esm.min.mjs')).default;
	const outputs = [];
	for (let index = 0; index < nodes.length; index += 1) {
		const node = nodes[index];
		const sourceNode = node.querySelector('.margo-mermaid__source code');
		const target = node.querySelector('.margo-mermaid__canvas');
		if (!sourceNode || !target) throw new Error('malformed Mermaid runtime marker');
		mermaid.initialize({
			startOnLoad: false,
			securityLevel: 'strict',
			htmlLabels: false,
			flowchart: {htmlLabels: false},
			look: 'classic',
			layout: 'dagre',
			deterministicIds: true,
			deterministicIDSeed: 'margo-pdf-' + index
		});
		const rendered = await mermaid.render('margo-pdf-' + index, sourceNode.textContent);
		if (!rendered || typeof rendered.svg !== 'string' || rendered.svg.length === 0) throw new Error('Mermaid returned no SVG');
		target.innerHTML = rendered.svg;
		const source = node.querySelector('.margo-mermaid__source');
		if (source) source.hidden = true;
		outputs.push(rendered.svg);
	}
	return {svg: outputs};
})()`

func runtimeTaskReports(tasks []margo.RuntimeTask, outputs []string) []margo.RuntimeTaskReport {
	reports := make([]margo.RuntimeTaskReport, len(tasks))
	for index, task := range tasks {
		output := []byte(outputs[index])
		digest := sha256.Sum256(output)
		reports[index] = margo.RuntimeTaskReport{
			ID:           task.ID,
			Kind:         task.Kind,
			InputSHA256:  task.InputSHA256,
			OutputSHA256: hex.EncodeToString(digest[:]),
			OutputBytes:  int64(len(output)),
			Status:       margo.RuntimeTaskSucceeded,
		}
	}
	return reports
}

func injectPageGeometry(document []byte, config pdf.PageConfig) ([]byte, error) {
	orientation := config.Orientation
	if orientation == "" {
		orientation = pdf.Portrait
	}
	rule := fmt.Sprintf(`<style data-margo-page-geometry>@page { size: %s %s; margin: %smm %smm %smm %smm; }</style>`,
		config.Size, orientation,
		formatMillimeters(config.Margins.Top), formatMillimeters(config.Margins.Right),
		formatMillimeters(config.Margins.Bottom), formatMillimeters(config.Margins.Left),
	)
	lower := strings.ToLower(string(document))
	// Embedded runtimes may contain the literal text "</head>" inside script
	// source. The generated document's real head terminator is the final
	// occurrence; inserting at the first occurrence can leave @page inside JS.
	index := strings.LastIndex(lower, "</head>")
	if index < 0 {
		return nil, chromiumError("pdf.page_geometry_failed", "HTML document has no closing head element")
	}
	result := make([]byte, 0, len(document)+len(rule))
	result = append(result, document[:index]...)
	result = append(result, rule...)
	result = append(result, document[index:]...)
	return result, nil
}

func formatMillimeters(value pdf.Millimeters) string {
	return strconv.FormatFloat(float64(value), 'f', -1, 64)
}

func awaitPromise(parameters *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams {
	return parameters.WithAwaitPromise(true)
}

func chromiumError(code, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "Chromium operation failed"
	}
	return fmt.Errorf("%s: %s", code, message)
}
