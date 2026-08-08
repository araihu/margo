package margo

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicNoReplaceWriteFailureIsNotCommitted(t *testing.T) {
	target := filepath.Join(t.TempDir(), "artifact.html")
	digest := ArtifactDigestOf([]byte("expected"))
	result, err := (&AtomicFileSink{Target: target}).Commit(context.Background(), errorReader{}, digest)
	if err == nil {
		t.Fatal("Commit() error = nil")
	}
	if result.Outcome != CommitNotCommitted {
		t.Fatalf("outcome = %q, want %q", result.Outcome, CommitNotCommitted)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("target stat error = %v, want absent", statErr)
	}
}

func TestAtomicNoReplacePublishesExactDigest(t *testing.T) {
	target := filepath.Join(t.TempDir(), "artifact.html")
	want := []byte("complete artifact")
	digest := ArtifactDigestOf(want)
	result, err := (&AtomicFileSink{Target: target}).Commit(context.Background(), bytes.NewReader(want), digest)
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if result.Outcome != CommitCommitted || result.Digest != digest || result.Bytes != int64(len(want)) {
		t.Fatalf("result = %#v", result)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("target bytes = %q, want %q", got, want)
	}
}

func TestAtomicNoReplaceRefusesExistingDestination(t *testing.T) {
	target := filepath.Join(t.TempDir(), "artifact.html")
	prior := []byte("prior")
	if err := os.WriteFile(target, prior, 0o600); err != nil {
		t.Fatal(err)
	}
	next := []byte("next")
	result, err := (&AtomicFileSink{Target: target}).Commit(context.Background(), bytes.NewReader(next), ArtifactDigestOf(next))
	if err == nil {
		t.Fatal("Commit() error = nil")
	}
	if result.Outcome != CommitNotCommitted {
		t.Fatalf("outcome = %q, want %q", result.Outcome, CommitNotCommitted)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, prior) {
		t.Fatalf("existing target bytes = %q, want %q", got, prior)
	}
}

func TestAtomicNoReplaceCancellationBeforePublish(t *testing.T) {
	target := filepath.Join(t.TempDir(), "artifact.html")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := (&AtomicFileSink{Target: target}).Commit(ctx, bytes.NewReader([]byte("artifact")), ArtifactDigestOf([]byte("artifact")))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Commit() error = %v, want context.Canceled", err)
	}
	if result.Outcome != CommitNotCommitted {
		t.Fatalf("outcome = %q, want %q", result.Outcome, CommitNotCommitted)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("target stat error = %v, want absent", statErr)
	}
}

func TestAtomicNoReplaceRejectsDigestMismatch(t *testing.T) {
	target := filepath.Join(t.TempDir(), "artifact.html")
	result, err := (&AtomicFileSink{Target: target}).Commit(context.Background(), bytes.NewReader([]byte("actual")), ArtifactDigestOf([]byte("expected")))
	if err == nil {
		t.Fatal("Commit() error = nil")
	}
	if result.Outcome != CommitNotCommitted {
		t.Fatalf("outcome = %q, want %q", result.Outcome, CommitNotCommitted)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("target stat error = %v, want absent", statErr)
	}
}

func TestAtomicNoReplaceFailureCleansPrivateStage(t *testing.T) {
	targetDir := t.TempDir()
	target := filepath.Join(targetDir, "artifact.html")
	result, err := (&AtomicFileSink{Target: target}).Commit(context.Background(), errorReader{}, ArtifactDigestOf([]byte("expected")))
	if err == nil {
		t.Fatal("Commit() error = nil")
	}
	if result.Outcome != CommitNotCommitted {
		t.Fatalf("outcome = %q, want %q", result.Outcome, CommitNotCommitted)
	}
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if len(entry.Name()) >= len(".margo-artifact-") && entry.Name()[:len(".margo-artifact-")] == ".margo-artifact-" {
			t.Fatalf("private stage %q was not removed", entry.Name())
		}
	}
}

func TestAtomicForceReplacesExistingDestination(t *testing.T) {
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
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("target bytes = %q, want %q", got, want)
	}
}

func TestParentSyncFailureReportsVisibleNewBytes(t *testing.T) {
	target := filepath.Join(t.TempDir(), "artifact.html")
	want := []byte("visible")
	ops := &faultAtomicOps{inner: defaultAtomicOps(), syncErr: errors.New("sync failed")}
	result, err := (&AtomicFileSink{Target: target, ops: ops}).Commit(context.Background(), bytes.NewReader(want), ArtifactDigestOf(want))
	if err == nil {
		t.Fatal("Commit() error = nil")
	}
	if result.Outcome != CommitDurabilityUncertain {
		t.Fatalf("outcome = %q, want %q", result.Outcome, CommitDurabilityUncertain)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("target bytes = %q, want %q", got, want)
	}
}

func TestAtomicNoReplaceAmbiguousErrorUsesReadback(t *testing.T) {
	target := filepath.Join(t.TempDir(), "artifact.html")
	want := []byte("ambiguous")
	ops := &faultAtomicOps{inner: defaultAtomicOps(), ambiguous: true}
	result, err := (&AtomicFileSink{Target: target, ops: ops}).Commit(context.Background(), bytes.NewReader(want), ArtifactDigestOf(want))
	if err == nil {
		t.Fatal("Commit() error = nil")
	}
	if result.Outcome != CommitUnknown {
		t.Fatalf("outcome = %q, want %q", result.Outcome, CommitUnknown)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("target bytes = %q, want %q", got, want)
	}
}

func TestAtomicForceCancellationAfterVisibilityReportsCommitted(t *testing.T) {
	target := filepath.Join(t.TempDir(), "artifact.html")
	want := []byte("cancelled after visibility")
	ctx, cancel := context.WithCancel(context.Background())
	ops := &faultAtomicOps{inner: defaultAtomicOps(), afterPublish: cancel}
	result, err := (&AtomicFileSink{Target: target, Force: true, ops: ops}).Commit(ctx, bytes.NewReader(want), ArtifactDigestOf(want))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Commit() error = %v, want context.Canceled", err)
	}
	if result.Outcome != CommitCommitted {
		t.Fatalf("outcome = %q, want %q", result.Outcome, CommitCommitted)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("target bytes = %q, want %q", got, want)
	}
}

type faultAtomicOps struct {
	inner        atomicOps
	syncErr      error
	ambiguous    bool
	afterPublish func()
}

func (f *faultAtomicOps) createStage(dir string) (*os.File, error) {
	return f.inner.createStage(dir)
}

func (f *faultAtomicOps) publishNoReplace(stage, target string) (bool, error) {
	visible, err := f.inner.publishNoReplace(stage, target)
	if f.afterPublish != nil {
		f.afterPublish()
	}
	if f.ambiguous {
		return false, errors.New("ambiguous no-replace result")
	}
	return visible, err
}

func (f *faultAtomicOps) publishReplace(stage, target string) (bool, error) {
	visible, err := f.inner.publishReplace(stage, target)
	if f.afterPublish != nil {
		f.afterPublish()
	}
	return visible, err
}

func (f *faultAtomicOps) remove(path string) error { return f.inner.remove(path) }

func (f *faultAtomicOps) syncParent(dir string) error {
	if f.syncErr != nil {
		return f.syncErr
	}
	return f.inner.syncParent(dir)
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
