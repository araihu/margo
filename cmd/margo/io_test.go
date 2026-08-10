package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
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
