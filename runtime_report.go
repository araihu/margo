package margo

import (
	"sort"
	"strings"

	"github.com/araihu/margo/internal/canonicaljson"
)

type RuntimeStatus string

const (
	RuntimePending RuntimeStatus = "pending"
	RuntimeRunning RuntimeStatus = "running"
	RuntimeReady   RuntimeStatus = "ready"
	RuntimeFailed  RuntimeStatus = "failed"
)

type RuntimeTaskStatus string

const (
	RuntimeTaskPending   RuntimeTaskStatus = "pending"
	RuntimeTaskRunning   RuntimeTaskStatus = "running"
	RuntimeTaskSucceeded RuntimeTaskStatus = "succeeded"
	RuntimeTaskFailed    RuntimeTaskStatus = "failed"
)

type RuntimeTaskReport struct {
	ID           string            `json:"id"`
	Kind         string            `json:"kind"`
	InputSHA256  string            `json:"inputSHA256"`
	OutputSHA256 string            `json:"outputSHA256"`
	OutputBytes  int64             `json:"outputBytes"`
	Status       RuntimeTaskStatus `json:"status"`
	ErrorCode    string            `json:"errorCode"`
}

type FontCheck struct {
	Family string `json:"family"`
	Query  string `json:"query"`
	Loaded bool   `json:"loaded"`
}

type BlockedRequest struct {
	URL          string `json:"url"`
	ResourceType string `json:"resourceType"`
}

type RuntimeReport struct {
	Protocol            string              `json:"protocol"`
	DocumentFingerprint DocumentFingerprint `json:"documentFingerprint"`
	RenderInstanceID    RenderInstanceID    `json:"renderInstanceID"`
	ExecutionID         ExecutionID         `json:"executionID"`
	Status              RuntimeStatus       `json:"status"`
	Tasks               []RuntimeTaskReport `json:"tasks"`
	FontChecks          []FontCheck         `json:"fontChecks"`
	BlockedRequests     []BlockedRequest    `json:"blockedRequests"`
	Layout              LayoutMetrics       `json:"layout"`
	Diagnostic          *Diagnostic         `json:"diagnostic"`
}

func ParseRuntimeReport(data []byte) (RuntimeReport, error) {
	var report RuntimeReport
	if err := decodeRuntimeJSON(data, &report); err != nil {
		return RuntimeReport{}, runtimeDiagnostic("runtime.report_malformed", err.Error())
	}
	if err := validateRuntimeReportShape(report); err != nil {
		return RuntimeReport{}, err
	}
	return cloneRuntimeReportValue(report), nil
}

func ValidateRuntimeReport(descriptor RuntimeDescriptor, executionID ExecutionID, report RuntimeReport) error {
	if err := ValidateRuntimeDescriptor(descriptor); err != nil {
		return err
	}
	if executionID == "" || report.Protocol != descriptor.Protocol || report.DocumentFingerprint != descriptor.DocumentFingerprint || report.RenderInstanceID != descriptor.RenderInstanceID || report.ExecutionID != executionID {
		return runtimeDiagnostic("runtime.report_forged", "runtime report identity does not match the descriptor and execution")
	}
	if err := validateRuntimeReportShape(report); err != nil {
		return err
	}
	expected := make(map[string]RuntimeTask, len(descriptor.Tasks))
	for _, task := range descriptor.Tasks {
		expected[task.ID] = task
	}
	seen := make(map[string]struct{}, len(report.Tasks))
	for _, task := range report.Tasks {
		if _, duplicate := seen[task.ID]; duplicate {
			return runtimeDiagnostic("runtime.task_duplicate", "runtime report contains a duplicate task")
		}
		seen[task.ID] = struct{}{}
		expectedTask, exists := expected[task.ID]
		if !exists {
			return runtimeDiagnostic("runtime.task_unknown", "runtime report contains an unknown task")
		}
		if task.Kind != expectedTask.Kind || task.InputSHA256 != expectedTask.InputSHA256 {
			return runtimeDiagnostic("runtime.report_forged", "runtime task identity does not match the descriptor")
		}
	}
	if len(seen) != len(expected) {
		return runtimeDiagnostic("runtime.task_missing", "runtime report omitted an expected task")
	}
	return nil
}

func CanonicalRuntimeProjection(report RuntimeReport) ([]byte, error) {
	if err := validateRuntimeReportShape(report); err != nil {
		return nil, err
	}
	tasks := append([]RuntimeTaskReport(nil), report.Tasks...)
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Kind != tasks[j].Kind {
			return tasks[i].Kind < tasks[j].Kind
		}
		return tasks[i].ID < tasks[j].ID
	})
	fonts := append([]FontCheck(nil), report.FontChecks...)
	sort.Slice(fonts, func(i, j int) bool {
		if fonts[i].Family != fonts[j].Family {
			return fonts[i].Family < fonts[j].Family
		}
		return fonts[i].Query < fonts[j].Query
	})
	blocked := append([]BlockedRequest(nil), report.BlockedRequests...)
	sort.Slice(blocked, func(i, j int) bool {
		if blocked[i].URL != blocked[j].URL {
			return blocked[i].URL < blocked[j].URL
		}
		return blocked[i].ResourceType < blocked[j].ResourceType
	})
	taskProjection := make([]map[string]any, len(tasks))
	for index, task := range tasks {
		taskProjection[index] = map[string]any{
			"errorCode":    task.ErrorCode,
			"id":           task.ID,
			"inputSHA256":  task.InputSHA256,
			"kind":         task.Kind,
			"outputBytes":  task.OutputBytes,
			"outputSHA256": task.OutputSHA256,
			"status":       task.Status,
		}
	}
	fontProjection := make([]map[string]any, len(fonts))
	for index, font := range fonts {
		fontProjection[index] = map[string]any{"family": font.Family, "loaded": font.Loaded, "query": font.Query}
	}
	blockedProjection := make([]map[string]any, len(blocked))
	for index, request := range blocked {
		blockedProjection[index] = map[string]any{"resourceType": request.ResourceType, "url": request.URL}
	}
	diagnosticCode := ""
	if report.Diagnostic != nil {
		diagnosticCode = report.Diagnostic.Code
	}
	return canonicaljson.Marshal(map[string]any{
		"blockedRequests":     blockedProjection,
		"diagnosticCode":      diagnosticCode,
		"documentFingerprint": report.DocumentFingerprint.String(),
		"fontChecks":          fontProjection,
		"layout": map[string]any{
			"scrollHeight": report.Layout.ScrollHeight,
			"scrollWidth":  report.Layout.ScrollWidth,
		},
		"protocol":         report.Protocol,
		"renderInstanceID": report.RenderInstanceID,
		"status":           report.Status,
		"tasks":            taskProjection,
	})
}

func validateRuntimeReportShape(report RuntimeReport) error {
	if report.Protocol != RuntimeProtocolV1 || report.DocumentFingerprint == (DocumentFingerprint{}) || ValidateRenderInstanceID(report.RenderInstanceID) != nil || report.ExecutionID == "" {
		return runtimeDiagnostic("runtime.report_malformed", "runtime report identity is malformed")
	}
	if report.Status != RuntimeReady && report.Status != RuntimeFailed {
		return runtimeDiagnostic("runtime.report_malformed", "runtime report is not terminal")
	}
	if report.Tasks == nil || report.FontChecks == nil || report.BlockedRequests == nil || report.Layout.ScrollWidth < 0 || report.Layout.ScrollHeight < 0 {
		return runtimeDiagnostic("runtime.report_malformed", "runtime report contains an invalid collection or layout")
	}
	seenTasks := make(map[string]struct{}, len(report.Tasks))
	failedTask := false
	for _, task := range report.Tasks {
		if _, duplicate := seenTasks[task.ID]; duplicate {
			return runtimeDiagnostic("runtime.task_duplicate", "runtime report contains a duplicate task")
		}
		seenTasks[task.ID] = struct{}{}
		if !runtimeKindPattern.MatchString(task.Kind) || !runtimeDigestPattern.MatchString(task.InputSHA256) || task.OutputBytes < 0 {
			return runtimeDiagnostic("runtime.report_malformed", "runtime task report is malformed")
		}
		switch task.Status {
		case RuntimeTaskSucceeded:
			if !runtimeDigestPattern.MatchString(task.OutputSHA256) || task.ErrorCode != "" {
				return runtimeDiagnostic("runtime.report_malformed", "succeeded task has invalid output evidence")
			}
		case RuntimeTaskFailed:
			failedTask = true
			if task.OutputSHA256 != "" || task.OutputBytes != 0 || !validRuntimeDiagnosticCode(task.ErrorCode) {
				return runtimeDiagnostic("runtime.report_malformed", "failed task has invalid error evidence")
			}
		default:
			return runtimeDiagnostic("runtime.report_malformed", "runtime task report is not terminal")
		}
	}
	seenFonts := make(map[string]struct{}, len(report.FontChecks))
	for _, font := range report.FontChecks {
		key := font.Family + "\x00" + font.Query
		if font.Family == "" || font.Query == "" {
			return runtimeDiagnostic("runtime.report_malformed", "font check is malformed")
		}
		if _, duplicate := seenFonts[key]; duplicate {
			return runtimeDiagnostic("runtime.report_malformed", "font check is duplicated")
		}
		seenFonts[key] = struct{}{}
		if report.Status == RuntimeReady && !font.Loaded {
			return runtimeDiagnostic("runtime.report_malformed", "ready report contains a failed font check")
		}
	}
	if report.Status == RuntimeReady {
		if failedTask || report.Diagnostic != nil || len(report.BlockedRequests) != 0 {
			return runtimeDiagnostic("runtime.report_malformed", "ready report contains failure evidence")
		}
	} else if report.Diagnostic == nil || !validRuntimeDiagnosticCode(report.Diagnostic.Code) || report.Diagnostic.Severity != SeverityError {
		return runtimeDiagnostic("runtime.report_malformed", "failed report lacks a stable terminal diagnostic")
	}
	return nil
}

func validRuntimeDiagnosticCode(value string) bool {
	if value == "" || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '_' && character != '.' {
			return false
		}
	}
	return true
}

type runtimeTaskState struct {
	descriptor RuntimeTask
	status     RuntimeTaskStatus
	outputHash string
	outputSize int64
	errorCode  string
}

type runtimeState struct {
	descriptor RuntimeDescriptor
	status     RuntimeStatus
	tasks      map[string]*runtimeTaskState
	diagnostic *Diagnostic
}

func newRuntimeState(descriptor RuntimeDescriptor) (*runtimeState, error) {
	if err := ValidateRuntimeDescriptor(descriptor); err != nil {
		return nil, err
	}
	state := &runtimeState{descriptor: cloneRuntimeDescriptorValue(descriptor), status: RuntimePending, tasks: make(map[string]*runtimeTaskState, len(descriptor.Tasks))}
	for _, task := range descriptor.Tasks {
		state.tasks[task.ID] = &runtimeTaskState{descriptor: task, status: RuntimeTaskPending}
	}
	return state, nil
}

func (s *runtimeState) start() error {
	if s == nil || s.status != RuntimePending {
		return runtimeDiagnostic("runtime.transition_invalid", "runtime can start only from pending")
	}
	s.status = RuntimeRunning
	return nil
}

func (s *runtimeState) startTask(id string) error {
	if s == nil || s.status != RuntimeRunning {
		return runtimeDiagnostic("runtime.transition_invalid", "task can start only while runtime is running")
	}
	task, exists := s.tasks[id]
	if !exists {
		return runtimeDiagnostic("runtime.task_unknown", "runtime task is unknown")
	}
	if task.status != RuntimeTaskPending {
		return runtimeDiagnostic("runtime.transition_invalid", "task can start only from pending")
	}
	for _, dependency := range task.descriptor.DependsOn {
		if s.tasks[dependency].status != RuntimeTaskSucceeded {
			return runtimeDiagnostic("runtime.dependency_pending", "runtime dependency has not succeeded")
		}
	}
	task.status = RuntimeTaskRunning
	return nil
}

func (s *runtimeState) succeedTask(id, outputSHA256 string, outputBytes int64) error {
	if s == nil || s.status != RuntimeRunning {
		return runtimeDiagnostic("runtime.transition_invalid", "task can succeed only while runtime is running")
	}
	task, exists := s.tasks[id]
	if !exists {
		return runtimeDiagnostic("runtime.task_unknown", "runtime task is unknown")
	}
	if task.status != RuntimeTaskRunning || !runtimeDigestPattern.MatchString(outputSHA256) || outputBytes < 0 {
		return runtimeDiagnostic("runtime.transition_invalid", "task success transition or output evidence is invalid")
	}
	task.status = RuntimeTaskSucceeded
	task.outputHash = outputSHA256
	task.outputSize = outputBytes
	return nil
}

func (s *runtimeState) failTask(id, code string) error {
	if s == nil || s.status != RuntimeRunning {
		return runtimeDiagnostic("runtime.transition_invalid", "task can fail only while runtime is running")
	}
	task, exists := s.tasks[id]
	if !exists {
		return runtimeDiagnostic("runtime.task_unknown", "runtime task is unknown")
	}
	if task.status != RuntimeTaskRunning || !validRuntimeDiagnosticCode(code) {
		return runtimeDiagnostic("runtime.transition_invalid", "task failure transition or diagnostic is invalid")
	}
	task.status = RuntimeTaskFailed
	task.errorCode = code
	s.status = RuntimeFailed
	s.diagnostic = &Diagnostic{Code: code, Severity: SeverityError}
	return nil
}

func (s *runtimeState) ready() error {
	if s == nil || s.status != RuntimeRunning {
		return runtimeDiagnostic("runtime.transition_invalid", "runtime can become ready only from running")
	}
	for _, task := range s.tasks {
		if task.status != RuntimeTaskSucceeded {
			return runtimeDiagnostic("runtime.transition_invalid", "runtime cannot become ready before every task succeeds")
		}
	}
	s.status = RuntimeReady
	return nil
}

func (s *runtimeState) fail(code string) error {
	if s == nil || s.status != RuntimeRunning || !validRuntimeDiagnosticCode(code) {
		return runtimeDiagnostic("runtime.transition_invalid", "runtime can fail only once from running")
	}
	s.status = RuntimeFailed
	s.diagnostic = &Diagnostic{Code: code, Severity: SeverityError}
	return nil
}

func cloneRuntimeReportValue(value RuntimeReport) RuntimeReport {
	clone := value
	if value.Tasks != nil {
		clone.Tasks = append([]RuntimeTaskReport{}, value.Tasks...)
	}
	if value.FontChecks != nil {
		clone.FontChecks = append([]FontCheck{}, value.FontChecks...)
	}
	if value.BlockedRequests != nil {
		clone.BlockedRequests = append([]BlockedRequest{}, value.BlockedRequests...)
	}
	if value.Diagnostic != nil {
		diagnostic := *value.Diagnostic
		clone.Diagnostic = &diagnostic
	}
	return clone
}
