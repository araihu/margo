package margo

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"github.com/araihu/margo/internal/canonicaljson"
)

// CompilerConfigFingerprint identifies the frozen compiler configuration.
type CompilerConfigFingerprint [32]byte

// DocumentFingerprint identifies the immutable compiled meaning.
type DocumentFingerprint [32]byte

// ArtifactFingerprint identifies the deterministic meaning of one emitted
// artifact. It intentionally excludes transport-only execution identity.
type ArtifactFingerprint [32]byte

// ArtifactDigest identifies the exact bytes emitted by an exporter.
type ArtifactDigest [32]byte

// LayoutMetrics is the quantized layout projection used by artifact identity.
// Runtime implementations may carry richer metrics in their own schema; C8
// only commits the stable dimensions needed by the core identity seam.
type LayoutMetrics struct {
	ScrollWidth  int64 `json:"scrollWidth"`
	ScrollHeight int64 `json:"scrollHeight"`
}

// TerminalReport is the immutable runtime projection consumed by artifact
// identity. ExecutionID routes a live execution and is deliberately excluded
// from the artifact preimage.
type TerminalReport struct {
	ProtocolVersion    string
	Document           DocumentFingerprint
	RenderInstanceID   string
	ExecutionID        string
	Kind               string
	Serializer         string
	Engine             string
	TerminalStatus     string
	TerminalDiagnostic string
	PageConfiguration  any
	TaskInputHashes    []string
	TaskOutputHashes   []string
	FontChecks         []string
	BlockedRequests    []string
	Layout             LayoutMetrics
}

// ArtifactDigestOf hashes the exact emitted bytes without a domain prefix.
func ArtifactDigestOf(data []byte) ArtifactDigest {
	return ArtifactDigest(sha256.Sum256(data))
}

func (f CompilerConfigFingerprint) String() string { return hex.EncodeToString(f[:]) }
func (f DocumentFingerprint) String() string       { return hex.EncodeToString(f[:]) }
func (f ArtifactFingerprint) String() string       { return hex.EncodeToString(f[:]) }
func (f ArtifactDigest) String() string            { return hex.EncodeToString(f[:]) }

func compilerConfigFingerprint(values map[string]any) CompilerConfigFingerprint {
	bytes, err := canonicaljson.Marshal(values)
	if err != nil {
		panic(err)
	}
	preimage := append([]byte("margo/compiler-config/v1\n"), bytes...)
	return CompilerConfigFingerprint(sha256.Sum256(preimage))
}

func documentFingerprint(source Source, compiler CompilerConfigFingerprint, values map[string]any) DocumentFingerprint {
	sourceHash := sha256.Sum256(source.Content)
	preimage := map[string]any{
		"baseURL":                   source.BaseURL,
		"compilerConfigFingerprint": compiler.String(),
		"sourceName":                source.Name,
		"sourceSHA256":              hex.EncodeToString(sourceHash[:]),
		"values":                    values,
	}
	bytes, err := canonicaljson.Marshal(preimage)
	if err != nil {
		panic(err)
	}
	data := append([]byte("margo/document/v1\n"), bytes...)
	return DocumentFingerprint(sha256.Sum256(data))
}

// artifactFingerprint is intentionally unexported until the runtime package
// owns the full terminal-report schema. The exported ArtifactFingerprint type
// remains usable by integration/reporting code.
func artifactFingerprint(report TerminalReport) ArtifactFingerprint {
	projection := map[string]any{
		"blockedRequests":    sortedStrings(report.BlockedRequests),
		"document":           report.Document.String(),
		"fontChecks":         sortedStrings(report.FontChecks),
		"layout":             report.Layout,
		"protocolVersion":    report.ProtocolVersion,
		"renderInstanceID":   report.RenderInstanceID,
		"taskInputHashes":    sortedStrings(report.TaskInputHashes),
		"taskOutputHashes":   sortedStrings(report.TaskOutputHashes),
		"terminalDiagnostic": report.TerminalDiagnostic,
		"terminalStatus":     report.TerminalStatus,
	}
	projectionBytes, err := canonicaljson.Marshal(projection)
	if err != nil {
		panic(err)
	}
	projectionHash := sha256.Sum256(projectionBytes)

	preimage := map[string]any{
		"documentFingerprint": report.Document.String(),
		"renderInstanceID":    report.RenderInstanceID,
		"kind":                report.Kind,
		"serializer":          report.Serializer,
		"pageConfiguration":   report.PageConfiguration,
		"engine":              report.Engine,
		"terminalProjection":  hex.EncodeToString(projectionHash[:]),
	}
	bytes, err := canonicaljson.Marshal(preimage)
	if err != nil {
		panic(err)
	}
	data := append([]byte("margo/artifact/v1\n"), bytes...)
	return ArtifactFingerprint(sha256.Sum256(data))
}

func sortedStrings(values []string) []string {
	if values == nil {
		return nil
	}
	copyValues := append([]string(nil), values...)
	sort.Strings(copyValues)
	return copyValues
}
