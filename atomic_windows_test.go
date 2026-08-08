//go:build windows

package margo

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWindowsForceUsesWriteThroughReplacement(t *testing.T) {
	target := filepath.Join(t.TempDir(), "artifact.html")
	if err := os.WriteFile(target, []byte("prior"), 0o600); err != nil {
		t.Fatal(err)
	}
	want := []byte("replacement")
	result, err := (&AtomicFileSink{Target: target, Force: true}).Commit(context.Background(), bytes.NewReader(want), ArtifactDigestOf(want))
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if result.Outcome != CommitCommitted {
		t.Fatalf("outcome = %q, want %q", result.Outcome, CommitCommitted)
	}
}
