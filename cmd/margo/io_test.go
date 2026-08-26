package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPublishRefusesExistingDestinationWithoutForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "page.html")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := publish(context.Background(), []byte("new"), outputOptions{Path: path}, io.Discard)
	if got := cliDiagnosticCode(err); got != "margo.atomic.destination_exists" {
		t.Fatalf("code = %q error = %v", got, err)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "old" {
		t.Fatalf("destination = %q error = %v", got, err)
	}
}

func TestPublishExistingDestinationDiagnosticIsActionableInTextAndJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "page.html")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, failure := publish(context.Background(), []byte("new"), outputOptions{Path: path}, io.Discard)
	if failure == nil {
		t.Fatal("publish() error = nil")
	}

	var text bytes.Buffer
	if err := writeDiagnostic(&text, diagnosticText, failure); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"margo.atomic.destination_exists", path, "--force", "new destination"} {
		if !strings.Contains(text.String(), want) {
			t.Fatalf("text diagnostic %q missing %q", text.String(), want)
		}
	}

	var encoded bytes.Buffer
	if err := writeDiagnostic(&encoded, diagnosticJSON, failure); err != nil {
		t.Fatal(err)
	}
	var projection diagnosticProjection
	if err := json.Unmarshal(encoded.Bytes(), &projection); err != nil {
		t.Fatalf("JSON diagnostic = %q: %v", encoded.String(), err)
	}
	if projection.Code != "margo.atomic.destination_exists" || !strings.Contains(projection.Message, path) || !strings.Contains(projection.Message, "--force") || !strings.Contains(projection.Message, "new destination") {
		t.Fatalf("JSON projection = %+v", projection)
	}
}

func TestPublishWritesCompleteArtifactToStdout(t *testing.T) {
	var output bytes.Buffer
	result, err := publish(context.Background(), []byte("complete"), outputOptions{Path: "-"}, &output)
	if err != nil {
		t.Fatal(err)
	}
	if output.String() != "complete" || result.Bytes != 8 {
		t.Fatalf("output = %q result = %+v", output.String(), result)
	}
}

func TestReadInputUsesInjectedStdin(t *testing.T) {
	source, err := readInput(context.Background(), nil, bytes.NewBufferString("# stdin"), "-")
	if err != nil {
		t.Fatal(err)
	}
	if source.Name != "<stdin>" || string(source.Content) != "# stdin" {
		t.Fatalf("source = %+v", source)
	}
}

func TestOSSourceReaderStopsAfterCallerLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large-policy.json")
	content := bytes.Repeat([]byte("x"), maxPolicyBytes*2)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := (osSourceReader{}).ReadFile(path, maxPolicyBytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != maxPolicyBytes+1 {
		t.Fatalf("bounded read returned %d bytes", len(data))
	}
}

func TestOSSourceReaderRejectsSpecialFilesBeforeReading(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable null device path differs on Windows")
	}
	_, err := (osSourceReader{}).ReadFile("/dev/zero", maxPolicyBytes)
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("special-file error = %v", err)
	}
}
