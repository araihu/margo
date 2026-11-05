package chromium

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/araihu/margo"
	"github.com/araihu/margo/pdf"
)

type browserMathTask struct {
	Source      margo.SourcePosition `json:"source"`
	ID          string               `json:"id"`
	InputSHA256 string               `json:"inputSHA256"`
	Output      string               `json:"output"`
	Status      string               `json:"status"`
	ErrorCode   string               `json:"errorCode"`
	Message     string               `json:"message"`
}
type browserMathOutput struct {
	Tasks      []browserMathTask `json:"tasks"`
	FontChecks []margo.FontCheck `json:"fontChecks"`
}

func mathRuntimeExpression(descriptor margo.RuntimeDescriptor) string {
	encoded, _ := json.Marshal(descriptor)
	return `(async()=>{
      const descriptor = ` + string(encoded) + `;
      const expected = descriptor.tasks.filter(task=>task.kind === "math");
      const allNodes = [...document.querySelectorAll('[data-margo-runtime-task="math"]')];
      const nodes = [...document.querySelectorAll('.margo-document')].flatMap(article =>
        [...article.querySelectorAll('[data-margo-runtime-task="math"]')]
          .filter(node => node.closest('.margo-document') === article)
          .sort((a,b) => Number(a.dataset.margoMathOrdinal)-Number(b.dataset.margoMathOrdinal)));
      if (new Set(allNodes.map(node=>node.id)).size !== allNodes.length || nodes.length !== allNodes.length) throw new Error("runtime.task_duplicate: math markers have invalid occurrence identities");
      if (nodes.length !== expected.length) throw new Error("runtime.task_missing: math markers do not match descriptor");
      if (globalThis.margoRuntimeReady) await globalThis.margoRuntimeReady.catch(()=>{});
      const results = typeof globalThis.margoMathResults === "function" ? globalThis.margoMathResults() : [];
      const byElement = new Map(results.filter(Boolean).map(result=>[result.elementID,result]));
      if (byElement.size !== results.filter(Boolean).length) throw new Error("runtime.task_duplicate: duplicate math results");
      const tasks = expected.map((task,index)=>{
        const node=nodes[index], result=byElement.get(node.id);
        if (node.dataset.margoMathInput !== task.inputSHA256) throw new Error("runtime.report_forged: math input differs from descriptor");
        if (!result) return {...task,status:"failed",errorCode:"math.runtime_unavailable",output:"",message:"Math executor returned no result."};
        if (result.inputSHA256 !== task.inputSHA256 || result.elementID !== node.id) throw new Error("runtime.report_forged: math result identity differs from marker");
        return {...result,id:task.id};
      });
      if (results.length !== expected.length && results.length !== 0) throw new Error("runtime.task_duplicate: unexpected math results");
      return {tasks,fontChecks:globalThis.margoMathFontChecks||[]};
    })()`
}

func validateMathOutput(descriptor margo.RuntimeDescriptor, output browserMathOutput) error {
	expected := map[string]margo.RuntimeTask{}
	for _, task := range descriptor.Tasks {
		if task.Kind == "math" {
			expected[task.ID] = task
		}
	}
	seen := map[string]bool{}
	totalOutput := 0
	if len(expected) > 1024 {
		return chromiumError("math.document_limit_exceeded", "too many math tasks")
	}
	for _, task := range output.Tasks {
		match, ok := expected[task.ID]
		if !ok || seen[task.ID] || match.InputSHA256 != task.InputSHA256 {
			return chromiumError("pdf.runtime_task_mismatch", "math task evidence does not match descriptor")
		}
		seen[task.ID] = true
		if task.Status != "succeeded" {
			code := task.ErrorCode
			if !strings.HasPrefix(code, "math.") {
				code = "math.expression_invalid"
			}
			return &margo.DiagnosticError{Diagnostics: []margo.Diagnostic{{Code: code, Severity: margo.SeverityError, Source: task.Source.Source, Line: task.Source.Line, Column: task.Source.Column, Pointer: "/math", Message: task.Message, Hint: "Correct the equation and retry the export."}}}
		}
		if task.Output == "" {
			return chromiumError("pdf.runtime_task_mismatch", "math task returned no HTML/MathML")
		}
		totalOutput += len(task.Output)
		if len(task.Output) > 256<<10 || totalOutput > 8<<20 {
			return chromiumError("math.output_limit_exceeded", "math output evidence exceeds the publication limit")
		}
	}
	if len(seen) != len(expected) {
		return chromiumError("pdf.runtime_task_mismatch", "missing math task evidence")
	}
	if len(expected) > 0 && len(output.FontChecks) != 20 {
		return chromiumError("math.font_unavailable", "math font evidence is incomplete")
	}
	seenFonts := map[string]bool{}
	for _, font := range output.FontChecks {
		key := font.Family + "\x00" + font.Query
		if !font.Loaded || !strings.HasPrefix(font.Family, "KaTeX_") || font.Query == "" || seenFonts[key] {
			return chromiumError("math.font_unavailable", "a required math font did not load")
		}
		seenFonts[key] = true
	}
	return nil
}

func mathTaskReports(tasks []margo.RuntimeTask, output browserMathOutput) []margo.RuntimeTaskReport {
	byID := map[string]browserMathTask{}
	for _, task := range output.Tasks {
		byID[task.ID] = task
	}
	reports := make([]margo.RuntimeTaskReport, 0, len(tasks))
	for _, task := range tasks {
		if task.Kind != "math" {
			continue
		}
		actual, ok := byID[task.ID]
		report := margo.RuntimeTaskReport{ID: task.ID, Kind: "math", InputSHA256: task.InputSHA256, Status: margo.RuntimeTaskFailed, ErrorCode: "math.runtime_unavailable"}
		if ok && actual.Status == "succeeded" && actual.InputSHA256 == task.InputSHA256 && actual.Output != "" {
			digest := sha256.Sum256([]byte(actual.Output))
			report.Status = margo.RuntimeTaskSucceeded
			report.ErrorCode = ""
			report.OutputSHA256 = hex.EncodeToString(digest[:])
			report.OutputBytes = int64(len(actual.Output))
		} else if actual.ErrorCode != "" {
			report.ErrorCode = actual.ErrorCode
		}
		reports = append(reports, report)
	}
	return reports
}

func failedMathReport(request pdf.Request, output browserMathOutput, err error) margo.RuntimeReport {
	reports := mathTaskReports(request.Runtime.Tasks, output)
	for _, task := range request.Runtime.Tasks {
		if task.Kind != "math" {
			reports = append(reports, margo.RuntimeTaskReport{ID: task.ID, Kind: task.Kind, InputSHA256: task.InputSHA256, Status: margo.RuntimeTaskFailed, ErrorCode: "runtime.dependency_failed"})
		}
	}
	fonts := append([]margo.FontCheck{}, output.FontChecks...)
	diagnostic := margo.Diagnostic{Code: "math.expression_invalid", Severity: margo.SeverityError, Message: err.Error()}
	var structured *margo.DiagnosticError
	if errors.As(err, &structured) && len(structured.Diagnostics) > 0 {
		diagnostic = structured.Diagnostics[0]
	}
	return margo.RuntimeReport{Protocol: request.Runtime.Protocol, DocumentFingerprint: request.Runtime.DocumentFingerprint, RenderInstanceID: request.Runtime.RenderInstanceID, ExecutionID: request.ExecutionID, Status: margo.RuntimeFailed, Tasks: reports, FontChecks: fonts, BlockedRequests: []margo.BlockedRequest{}, Diagnostic: &diagnostic}
}

func mathPrintExpression(config pdf.PageConfig) string {
	width, height := 210.0, 297.0
	if config.Size == pdf.PageLetter {
		width, height = 215.9, 279.4
	}
	if config.Custom != nil {
		width, height = float64(config.Custom.WidthMM), float64(config.Custom.HeightMM)
	} else if config.Orientation == pdf.Landscape {
		width, height = height, width
	}
	width -= float64(config.Margins.Left + config.Margins.Right)
	height -= float64(config.Margins.Top + config.Margins.Bottom)
	return fmt.Sprintf(`typeof globalThis.margoValidateMathPrint === "function" ? globalThis.margoValidateMathPrint(%g,%g) : ""`, width*96/25.4, height*96/25.4)
}

func mathPrintFailure(detail string) error {
	var evidence struct {
		Source                                         string `json:"source"`
		Line                                           int    `json:"line"`
		Column                                         int    `json:"column"`
		Width, Height, AvailableWidth, AvailableHeight float64
	}
	if err := json.Unmarshal([]byte(detail), &evidence); err != nil {
		return chromiumError("math.print_overflow", "equation exceeds the printable content box")
	}
	return &margo.DiagnosticError{Diagnostics: []margo.Diagnostic{{Code: "math.print_overflow", Severity: margo.SeverityError,
		Source: evidence.Source, Line: evidence.Line, Column: evidence.Column, Pointer: "/math",
		Message: fmt.Sprintf("equation (%.1f × %.1f px) exceeds or is clipped by the printable content box (%.1f × %.1f px)", evidence.Width, evidence.Height, evidence.AvailableWidth, evidence.AvailableHeight),
		Hint:    "Shorten or split the equation, or choose a larger page or slide size."}}}
}
