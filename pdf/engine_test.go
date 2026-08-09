package pdf

import (
	"context"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
)

type contractEngine struct{}

var _ Engine = contractEngine{}

func (contractEngine) Name() string { return "contract" }

func (contractEngine) Version(context.Context) (string, error) { return "1.0.0", nil }

func (contractEngine) Export(_ context.Context, request Request) (Result, error) {
	return Result{
		PDF:     append([]byte(nil), request.HTML...),
		Runtime: validContractRuntimeReport(request.Runtime, request.ExecutionID),
		Engine:  EngineInfo{Name: "contract", Version: "1.0.0"},
	}, nil
}

func TestEngineContractUsesRootRuntimeValues(t *testing.T) {
	t.Parallel()

	request := validContractRequest()
	var engine Engine = contractEngine{}
	result, err := engine.Export(context.Background(), request)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if result.Runtime.ExecutionID != request.ExecutionID || result.Runtime.DocumentFingerprint != request.Runtime.DocumentFingerprint {
		t.Fatalf("result runtime identity = %+v, request = %+v", result.Runtime, request.Runtime)
	}
	if result.Engine != (EngineInfo{Name: "contract", Version: "1.0.0"}) {
		t.Fatalf("Engine = %+v", result.Engine)
	}
}

func TestEngineContractClonesMutablePayloads(t *testing.T) {
	t.Parallel()

	request := validContractRequest()
	clonedRequest := request.Clone()
	request.HTML[0] = 'X'
	request.Runtime.Tasks[0].DependsOn[0] = "changed"
	if string(clonedRequest.HTML) != "<html>contract</html>" {
		t.Fatalf("cloned HTML = %q", clonedRequest.HTML)
	}
	if got := clonedRequest.Runtime.Tasks[0].DependsOn[0]; got != "ri-00000000:mermaid:00000001:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("cloned dependency = %q", got)
	}

	result := Result{
		PDF:     []byte("%PDF-contract"),
		Runtime: validContractRuntimeReport(clonedRequest.Runtime, clonedRequest.ExecutionID),
		Engine:  EngineInfo{Name: "contract", Version: "1.0.0"},
	}
	clonedResult := result.Clone()
	result.PDF[0] = 'X'
	result.Runtime.Tasks[0].OutputSHA256 = "changed"
	result.Runtime.FontChecks[0].Family = "changed"
	result.Runtime.BlockedRequests = append(result.Runtime.BlockedRequests, margo.BlockedRequest{URL: "https://example.invalid"})
	if string(clonedResult.PDF) != "%PDF-contract" {
		t.Fatalf("cloned PDF = %q", clonedResult.PDF)
	}
	if got := clonedResult.Runtime.Tasks[0].OutputSHA256; got != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("cloned output digest = %q", got)
	}
	if got := clonedResult.Runtime.FontChecks[0].Family; got != "Inter" {
		t.Fatalf("cloned font family = %q", got)
	}
	if len(clonedResult.Runtime.BlockedRequests) != 0 {
		t.Fatalf("cloned blocked requests = %+v", clonedResult.Runtime.BlockedRequests)
	}
}

func TestEngineInfoRejectsIncompleteIdentity(t *testing.T) {
	t.Parallel()

	for _, info := range []EngineInfo{
		{},
		{Name: "contract"},
		{Version: "1.0.0"},
	} {
		if err := info.Validate(); err == nil || !strings.HasPrefix(err.Error(), "pdf.engine_identity_invalid") {
			t.Fatalf("Validate(%+v) error = %v", info, err)
		}
	}
	if err := (EngineInfo{Name: "contract", Version: "1.0.0"}).Validate(); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}
}

func validContractRequest() Request {
	descriptor := margo.RuntimeDescriptor{
		Protocol:            margo.RuntimeProtocolV1,
		DocumentFingerprint: margo.DocumentFingerprint{1},
		RenderInstanceID:    "ri-00000000",
		Tasks: []margo.RuntimeTask{
			{
				ID:          "ri-00000000:mermaid:00000001:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Kind:        "mermaid",
				InputSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				DependsOn:   []string{"ri-00000000:mermaid:00000001:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			},
			{
				ID:          "ri-00000000:mermaid:00000001:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Kind:        "mermaid",
				InputSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				DependsOn:   []string{},
			},
		},
	}
	return Request{
		HTML:        []byte("<html>contract</html>"),
		Runtime:     descriptor,
		ExecutionID: "execution-contract",
		Page:        PageConfig{Size: PageA4, Orientation: Portrait},
	}
}

func validContractRuntimeReport(descriptor margo.RuntimeDescriptor, executionID margo.ExecutionID) margo.RuntimeReport {
	return margo.RuntimeReport{
		Protocol:            descriptor.Protocol,
		DocumentFingerprint: descriptor.DocumentFingerprint,
		RenderInstanceID:    descriptor.RenderInstanceID,
		ExecutionID:         executionID,
		Status:              margo.RuntimeReady,
		Tasks: []margo.RuntimeTaskReport{
			{
				ID:           descriptor.Tasks[0].ID,
				Kind:         descriptor.Tasks[0].Kind,
				InputSHA256:  descriptor.Tasks[0].InputSHA256,
				OutputSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				OutputBytes:  1,
				Status:       margo.RuntimeTaskSucceeded,
			},
			{
				ID:           descriptor.Tasks[1].ID,
				Kind:         descriptor.Tasks[1].Kind,
				InputSHA256:  descriptor.Tasks[1].InputSHA256,
				OutputSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				OutputBytes:  1,
				Status:       margo.RuntimeTaskSucceeded,
			},
		},
		FontChecks:      []margo.FontCheck{{Family: "Inter", Query: "12px Inter", Loaded: true}},
		BlockedRequests: []margo.BlockedRequest{},
		Layout:          margo.LayoutMetrics{ScrollWidth: 1280, ScrollHeight: 720},
	}
}
