package margo

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestStdoutReceivesZeroBytesBeforeValidatedCommit(t *testing.T) {
	out := new(bytes.Buffer)
	result, err := (StdoutSink{Writer: out}).Commit(context.Background(), errorReader{}, ArtifactDigestOf([]byte("expected")))
	if err == nil {
		t.Fatal("Commit() error = nil")
	}
	if result.Outcome != CommitNotCommitted {
		t.Fatalf("outcome = %q, want %q", result.Outcome, CommitNotCommitted)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout bytes = %d, want 0", out.Len())
	}
}

func TestStdoutPublishesExactValidatedSpool(t *testing.T) {
	want := []byte("complete stdout artifact")
	out := new(bytes.Buffer)
	result, err := (StdoutSink{Writer: out}).Commit(context.Background(), bytes.NewReader(want), ArtifactDigestOf(want))
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if result.Outcome != CommitCommitted || result.Bytes != int64(len(want)) {
		t.Fatalf("result = %#v", result)
	}
	if !bytes.Equal(out.Bytes(), want) {
		t.Fatalf("stdout bytes = %q, want %q", out.Bytes(), want)
	}
}

func TestStdoutDigestMismatchPublishesZeroBytes(t *testing.T) {
	out := new(bytes.Buffer)
	result, err := (StdoutSink{Writer: out}).Commit(context.Background(), strings.NewReader("actual"), ArtifactDigestOf([]byte("expected")))
	if err == nil {
		t.Fatal("Commit() error = nil")
	}
	if result.Outcome != CommitNotCommitted {
		t.Fatalf("outcome = %q, want %q", result.Outcome, CommitNotCommitted)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout bytes = %d, want 0", out.Len())
	}
}

func TestStdoutPartialWriteReportsObservedPrefix(t *testing.T) {
	want := []byte("partial stdout artifact")
	out := &partialWriter{limit: 5, err: errors.New("pipe closed")}
	result, err := (StdoutSink{Writer: out}).Commit(context.Background(), bytes.NewReader(want), ArtifactDigestOf(want))
	if err == nil || !strings.Contains(err.Error(), "margo.stdout.partial_write") {
		t.Fatalf("Commit() error = %v, want partial-write diagnostic", err)
	}
	if result.Outcome != CommitUnknown {
		t.Fatalf("outcome = %q, want %q", result.Outcome, CommitUnknown)
	}
	if result.Bytes != int64(out.limit) || out.buf.Len() != out.limit {
		t.Fatalf("result/output = %#v/%d, want observed prefix %d", result, out.buf.Len(), out.limit)
	}
}

func TestStdoutCancellationBeforeCommitPublishesZeroBytes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out := new(bytes.Buffer)
	want := []byte("cancelled")
	result, err := (StdoutSink{Writer: out}).Commit(ctx, bytes.NewReader(want), ArtifactDigestOf(want))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Commit() error = %v, want context.Canceled", err)
	}
	if result.Outcome != CommitNotCommitted || out.Len() != 0 {
		t.Fatalf("result/output = %#v/%d, want not committed and zero bytes", result, out.Len())
	}
}

type partialWriter struct {
	buf   bytes.Buffer
	limit int
	err   error
}

func (w *partialWriter) Write(data []byte) (int, error) {
	if len(data) > w.limit-w.buf.Len() {
		data = data[:w.limit-w.buf.Len()]
	}
	if len(data) > 0 {
		_, _ = w.buf.Write(data)
	}
	if w.buf.Len() == w.limit {
		return len(data), w.err
	}
	return len(data), nil
}

var _ io.Writer = (*partialWriter)(nil)
