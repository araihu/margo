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
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/assets"
	"github.com/araihu/margo/internal/browserlaunch"
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
	mermaidTasks := 0
	for _, task := range request.Runtime.Tasks {
		if task.Kind == "mermaid" {
			mermaidTasks++
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

	options := chromiumAllocatorOptions(engine.executablePath, profile)
	allocatorCtx, cancelAllocator := browserlaunch.NewExecAllocator(exportCtx, options...)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx)
	defer cancelBrowser()

	var metrics margo.LayoutMetrics
	var runtimeOutput browserRuntimeOutput
	var printOverflow string
	var pdfBytes []byte
	stage := "initializing"
	stageAction := func(name string, action chromedp.Action) chromedp.Action {
		return chromedp.ActionFunc(func(ctx context.Context) error {
			stage = name
			return action.Do(ctx)
		})
	}
	actions := []chromedp.Action{}
	if request.Runtime.Protocol == margo.RuntimeProtocolV2 {
		validationRequest := request.Runtime.ValidationRequest
		actions = append(actions, emulation.SetDeviceMetricsOverride(
			int64(validationRequest.ViewportWidth), int64(validationRequest.ViewportHeight), validationRequest.DeviceScaleFactor, false,
		))
	}
	actions = append(actions,
		stageAction("navigate", chromedp.Navigate(server.URL)),
		stageAction("runtime tasks", chromedp.Evaluate(runtimeExpression, &runtimeOutput, awaitPromise)),
		stageAction("print preparation", chromedp.Evaluate(`(async () => {
			if (typeof globalThis.margoPreparePrint === "function") await globalThis.margoPreparePrint();
			if (typeof globalThis.margoPrepareDeckPrint === "function") await globalThis.margoPrepareDeckPrint();
			return true;
		})()`, nil, awaitPromise)),
		stageAction("print media", emulation.SetEmulatedMedia().WithMedia("print")),
		stageAction("wait for fonts and images", chromedp.Evaluate(`(async () => {
			await document.fonts.ready;
			await Promise.all(Array.from(document.images).map((image) => image.complete ? true : new Promise((resolve, reject) => {
				image.addEventListener("load", resolve, {once: true});
				image.addEventListener("error", reject, {once: true});
			})))
			return {scrollWidth: document.documentElement.scrollWidth, scrollHeight: document.documentElement.scrollHeight};
		})()`, &metrics, awaitPromise)),
		stageAction("validate print overflow", chromedp.Evaluate(`(() => {
			if (typeof globalThis.margoValidateDeckPrint !== "function") return "";
			return globalThis.margoValidateDeckPrint();
		})()`, &printOverflow)),
		chromedp.ActionFunc(func(context.Context) error {
			stage = "check print overflow"
			if printOverflow != "" {
				return fmt.Errorf("deck print overflow")
			}
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			stage = "print PDF"
			params := page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				WithGenerateTaggedPDF(true).
				WithGenerateDocumentOutline(true)
			var err error
			pdfBytes, _, err = params.Do(ctx)
			return err
		}),
	)
	if err := chromedp.Run(browserCtx, actions...); err != nil {
		if printOverflow != "" {
			return pdf.Result{}, chromiumError("pdf.deck_print_overflow", printOverflow)
		}
		return pdf.Result{}, chromiumError("pdf.chromium.export_failed", fmt.Sprintf("stage=%s executable=%s timeout=%s html_bytes=%d mermaid_tasks=%d: %v", stage, engine.executablePath, engine.timeout, len(request.HTML), mermaidTasks, err))
	}
	if printOverflow != "" {
		return pdf.Result{}, chromiumError("pdf.deck_print_overflow", printOverflow)
	}
	if !strings.HasPrefix(string(pdfBytes), "%PDF-") {
		return pdf.Result{}, chromiumError("pdf.chromium.output_invalid", "browser returned invalid PDF bytes")
	}
	if len(runtimeOutput.SVG) != mermaidTasks {
		return pdf.Result{}, chromiumError("pdf.runtime_task_mismatch", "document runtime markers do not match the descriptor")
	}
	version, err := engine.Version(exportCtx)
	if err != nil {
		return pdf.Result{}, err
	}
	status := margo.RuntimeReady
	var diagnostic *margo.Diagnostic
	fontDigest := runtimeOutput.FontBundleDigest
	for _, check := range runtimeOutput.FontChecks {
		if !check.Loaded {
			status = margo.RuntimeFailed
			diagnostic = &margo.Diagnostic{Code: "deck.fonts_unavailable", Severity: margo.SeverityError, Message: "a required deck font face did not load"}
			break
		}
	}
	if request.Runtime.Protocol == margo.RuntimeProtocolV2 && fontDigest != request.Runtime.ValidationRequest.ExpectedFontBundleDigest {
		status = margo.RuntimeFailed
		diagnostic = &margo.Diagnostic{Code: "deck.font_bundle_mismatch", Severity: margo.SeverityError, Message: "observed deck font bundle differs from the descriptor lock"}
	}
	report := margo.RuntimeReport{
		Protocol:            request.Runtime.Protocol,
		DocumentFingerprint: request.Runtime.DocumentFingerprint,
		RenderInstanceID:    request.Runtime.RenderInstanceID,
		ExecutionID:         request.ExecutionID,
		Status:              status,
		Tasks:               runtimeTaskReports(request.Runtime.Tasks, runtimeOutput.SVG, metrics),
		FontChecks:          runtimeOutput.FontChecks,
		BlockedRequests:     []margo.BlockedRequest{},
		Layout:              metrics,
		Diagnostic:          diagnostic,
	}
	if request.Runtime.Protocol == margo.RuntimeProtocolV2 {
		platformProfile := runtime.GOOS + "-" + runtime.GOARCH
		if validRuntimeDigest(fontDigest) {
			report.ValidationIdentity = &margo.RuntimeValidationIdentity{
				BrowserProfile:   request.Runtime.ValidationRequest.BrowserProfile,
				EngineName:       engine.Name(),
				EngineVersion:    version,
				PlatformProfile:  platformProfile,
				FontBundleDigest: fontDigest,
			}
		}
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

func chromiumAllocatorOptions(executablePath, profile string) []chromedp.ExecAllocatorOption {
	options := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	options = append(options,
		chromedp.ExecPath(executablePath),
		chromedp.UserDataDir(profile),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("host-resolver-rules", "MAP * 0.0.0.0, EXCLUDE 127.0.0.1, EXCLUDE localhost"),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
	)
	if runtime.GOOS == "linux" {
		options = append(options, chromedp.NoSandbox, chromedp.DisableGPU)
	}
	return options
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
	SVG              []string          `json:"svg"`
	FontChecks       []margo.FontCheck `json:"fontChecks"`
	FontBundleDigest string            `json:"fontBundleDigest"`
}

const runtimeExpression = `(async () => {
	function materializeTreeViewIcons(svg) {
		// Mermaid strict mode strips <use>; only restore fixed built-in paths.
		const tree = svg.querySelector('.tree-view');
		if (!tree) return;
		const paths = Object.freeze({
			folder: 'M10.59 4.59A2 2 0 0 0 9.17 4H4a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.17z',
			file: 'M6 2a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8.83a2 2 0 0 0-.59-1.42l-4.82-4.82A2 2 0 0 0 13.17 2H6Zm7.5 1.9l4.6 4.6h-3.6a1 1 0 0 1-1-1V3.9Z'
		});
		for (const group of [...tree.children]) {
			if (group.localName !== 'g') continue;
			const label = [...group.children].find((child) => child.localName === 'text' && child.classList.contains('treeView-node-label'));
			if (!label || [...group.children].some((child) => child.classList.contains('treeView-node-icon'))) continue;
			const x = Number(label.getAttribute('x'));
			const y = Number(label.getAttribute('y'));
			if (!Number.isFinite(x) || !Number.isFinite(y)) continue;
			const remainder = ((x - 5) % 15 + 15) % 15;
			if (Math.abs(remainder - 3) > 0.5) continue;
			const type = label.classList.contains('treeView-node-dir') ? 'folder' : 'file';
			const icon = svg.ownerDocument.createElementNS('http://www.w3.org/2000/svg', 'path');
			icon.setAttribute('class', 'treeView-node-icon');
			icon.setAttribute('d', paths[type]);
			icon.setAttribute('fill', 'currentColor');
			if (type === 'file') {
				icon.setAttribute('fill-rule', 'evenodd');
				icon.setAttribute('clip-rule', 'evenodd');
			}
			icon.setAttribute('transform', 'translate(' + (x - 18) + ' ' + (y - 7) + ') scale(' + (14 / 24) + ')');
			group.insertBefore(icon, label);
		}
	}

	function margoMermaidConfiguration() {
		const styles = getComputedStyle(document.documentElement);
		const read = (name) => styles.getPropertyValue(name).trim();
		const canvas = read('--margo-mermaid-canvas');
		const node = read('--margo-mermaid-node');
		const nodeBorder = read('--margo-mermaid-node-border');
		const text = read('--margo-mermaid-text');
		const edge = read('--margo-mermaid-edge');
		const edgeLabel = read('--margo-mermaid-edge-label');
		const edgeLabelBackground = read('--margo-mermaid-edge-label-background');
		if (!canvas || !node || !nodeBorder || !text || !edge || !edgeLabel || !edgeLabelBackground) return {};
		const themeVariables = {
			background: canvas,
			darkMode: document.documentElement.classList.contains('dark'),
			fontFamily: read('--font-body'),
			primaryColor: node,
			primaryTextColor: text,
			primaryBorderColor: nodeBorder,
			secondaryColor: canvas,
			secondaryTextColor: text,
			secondaryBorderColor: nodeBorder,
			tertiaryColor: canvas,
			tertiaryTextColor: text,
			tertiaryBorderColor: nodeBorder,
			textColor: edgeLabel,
			titleColor: text,
			lineColor: edge,
			defaultLinkColor: edge,
			arrowheadColor: edge,
			nodeBkg: node,
			nodeBorder,
			nodeTextColor: text,
			clusterBkg: canvas,
			clusterBorder: nodeBorder,
			edgeLabelBackground,
			labelBackground: edgeLabelBackground
		};
		return {theme: 'base', themeVariables};
	}

	const fontEvidence = typeof globalThis.margoGetDeckFontEvidence === 'function'
		? await globalThis.margoGetDeckFontEvidence()
		: {fontChecks: [], fontBundleDigest: ''};
	const nodes = Array.from(document.querySelectorAll('[data-margo-runtime-task="mermaid"]'));
	if (nodes.length === 0) return {svg: [], ...fontEvidence};
	if (globalThis.margoRuntimeReady && typeof globalThis.margoRuntimeReady.then === 'function') {
		await globalThis.margoRuntimeReady;
		const embedded = nodes.map((node) => node.querySelector('.margo-mermaid__canvas svg')?.outerHTML ?? '');
		if (embedded.every((svg) => svg.length > 0)) return {svg: embedded, ...fontEvidence};
	}
	const mermaid = (await import('/margo-assets/mermaid/11.16.1/mermaid.esm.min.mjs')).default;
	const mermaidConfiguration = margoMermaidConfiguration();
	const outputs = [];
	for (let index = 0; index < nodes.length; index += 1) {
		const node = nodes[index];
		const sourceNode = node.querySelector('.margo-mermaid__source code');
		const target = node.querySelector('.margo-mermaid__canvas');
		if (!sourceNode || !target) throw new Error('malformed Mermaid runtime marker');
		mermaid.initialize({
			...mermaidConfiguration,
			startOnLoad: false,
			securityLevel: 'strict',
			htmlLabels: false,
			flowchart: {htmlLabels: false},
			look: 'classic',
			layout: 'dagre',
			treeView: {showIcons: true},
			deterministicIds: true,
			deterministicIDSeed: 'margo-pdf-' + index
		});
		const rendered = await mermaid.render('margo-pdf-' + index, sourceNode.textContent);
		if (!rendered || typeof rendered.svg !== 'string' || rendered.svg.length === 0) throw new Error('Mermaid returned no SVG');
		target.innerHTML = rendered.svg;
		const svg = target.querySelector('svg');
		if (svg) materializeTreeViewIcons(svg);
		const source = node.querySelector('.margo-mermaid__source');
		if (source) source.hidden = true;
		outputs.push(svg ? svg.outerHTML : rendered.svg);
	}
	return {svg: outputs, ...fontEvidence};
})()`

func runtimeTaskReports(tasks []margo.RuntimeTask, outputs []string, metrics margo.LayoutMetrics) []margo.RuntimeTaskReport {
	reports := make([]margo.RuntimeTaskReport, len(tasks))
	mermaidIndex := 0
	for index, task := range tasks {
		output := []byte(fmt.Sprintf("margo-runtime-task/%s/%d/%d", task.Kind, metrics.ScrollWidth, metrics.ScrollHeight))
		if task.Kind == "mermaid" {
			if mermaidIndex >= len(outputs) {
				output = nil
			} else {
				output = []byte(outputs[mermaidIndex])
			}
			mermaidIndex++
		}
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

func validRuntimeDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func injectPageGeometry(document []byte, config pdf.PageConfig) ([]byte, error) {
	orientation := config.Orientation
	if orientation == "" {
		orientation = pdf.Portrait
	}
	pageSize := string(config.Size)
	if config.Custom != nil {
		pageSize = fmt.Sprintf("%smm %smm", formatMillimeters(config.Custom.WidthMM), formatMillimeters(config.Custom.HeightMM))
		orientation = ""
	}
	margins := fmt.Sprintf("%smm %smm %smm %smm",
		formatMillimeters(config.Margins.Top), formatMillimeters(config.Margins.Right),
		formatMillimeters(config.Margins.Bottom), formatMillimeters(config.Margins.Left),
	)
	maxImageHeight := fmt.Sprintf("%dvh", pdf.DefaultImageMaxHeightPercent)
	if config.EffectiveImageOverflowPolicy() == pdf.ImageOverflowAllow {
		maxImageHeight = "none"
	}
	imageRule := fmt.Sprintf(`@media print { .margo-document img { max-block-size: %s !important; max-height: %s !important; max-inline-size: 100%% !important; max-width: 100%% !important; inline-size: auto !important; width: auto !important; block-size: auto !important; height: auto !important; aspect-ratio: auto !important; object-fit: contain !important; page-break-inside: avoid; break-inside: avoid-page; } }`, maxImageHeight, maxImageHeight)
	orientationSuffix := ""
	if orientation != "" {
		orientationSuffix = " " + string(orientation)
	}
	rule := fmt.Sprintf(`<style data-margo-page-geometry>@page { size: %s%s; margin: %s; } @page margo-diagram-landscape { size: %s landscape; margin: %s; } %s</style>`,
		pageSize, orientationSuffix, margins, pageSize, margins, imageRule,
	)
	index := htmlHeadEnd(document)
	if index < 0 {
		return nil, chromiumError("pdf.page_geometry_failed", "HTML document has no closing head element")
	}
	result := make([]byte, 0, len(document)+len(rule))
	result = append(result, document[:index]...)
	result = append(result, rule...)
	result = append(result, document[index:]...)
	return result, nil
}

// htmlHeadEnd returns the byte offset of the real closing head tag. A simple
// strings.Index is insufficient because generated runtime bundles may contain
// the literal text "</head>" inside script or style source. The scan is
// intentionally lexical: raw-text elements are skipped, while the first
// closing head tag outside them is the document boundary where @page CSS must
// be inserted.
func htmlHeadEnd(document []byte) int {
	lower := strings.ToLower(string(document))
	headStart := strings.Index(lower, "<head")
	if headStart < 0 {
		return -1
	}
	headOpenEnd := strings.IndexByte(lower[headStart:], '>')
	if headOpenEnd < 0 {
		return -1
	}
	position := headStart + headOpenEnd + 1
	for position < len(lower) {
		closingHead := strings.Index(lower[position:], "</head>")
		if closingHead < 0 {
			return -1
		}
		closingHead += position
		for _, rawName := range []string{"script", "style"} {
			rawStart := findHTMLTag(lower, position, "<"+rawName)
			if rawStart < 0 || rawStart >= closingHead {
				continue
			}
			rawOpenEnd := strings.IndexByte(lower[rawStart:], '>')
			if rawOpenEnd < 0 {
				return -1
			}
			rawClose := strings.Index(lower[rawStart+rawOpenEnd+1:], "</"+rawName+">")
			if rawClose < 0 {
				return -1
			}
			position = rawStart + rawOpenEnd + 1 + rawClose + len(rawName) + 3
			goto nextCandidate
		}
		return closingHead
	nextCandidate:
	}
	return -1
}

func findHTMLTag(lower string, start int, prefix string) int {
	for position := start; position < len(lower); {
		found := strings.Index(lower[position:], prefix)
		if found < 0 {
			return -1
		}
		found += position
		after := found + len(prefix)
		if after == len(lower) || lower[after] == ' ' || lower[after] == '\t' || lower[after] == '\n' || lower[after] == '\r' || lower[after] == '>' || lower[after] == '/' {
			return found
		}
		position = after
	}
	return -1
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
