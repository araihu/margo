package margo

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
)

func TestSpoolCrossesMemoryLimitWithoutPublishing(t *testing.T) {
	tempDir := t.TempDir()
	spool := NewSpool(SpoolOptions{MemoryLimit: 64, MaximumBytes: 1024, TempDir: tempDir})
	t.Cleanup(func() { _ = spool.Close() })

	if err := spool.WriteAll(context.Background(), bytes.Repeat([]byte("x"), 65)); err != nil {
		t.Fatalf("WriteAll() error = %v", err)
	}
	if !spool.UsesPrivateFile() {
		t.Fatal("spool did not cross the private-file threshold")
	}
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("private staging entries = %d, want 1", len(entries))
	}
	if entries[0].Type().Perm() != 0 {
		// Type().Perm() is zero for regular files; the mode assertion below
		// verifies the actual private permission contract.
		t.Fatalf("unexpected staging entry type: %s", entries[0].Type())
	}
	info, err := entries[0].Info()
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("private staging mode = %o, want 600", got)
	}
}

func TestSpoolRefusesMaximumOverflowWithoutAppending(t *testing.T) {
	spool := NewSpool(SpoolOptions{MemoryLimit: 64, MaximumBytes: 4, TempDir: t.TempDir()})
	t.Cleanup(func() { _ = spool.Close() })
	if err := spool.WriteAll(context.Background(), []byte("four")); err != nil {
		t.Fatalf("initial WriteAll() error = %v", err)
	}
	if err := spool.WriteAll(context.Background(), []byte("!")); err == nil {
		t.Fatal("overflow WriteAll() error = nil")
	}
	got, err := readSpool(spool)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "four" {
		t.Fatalf("spool bytes after rejected overflow = %q, want four", got)
	}
}

func TestSpoolCancellationCleansPrivateFile(t *testing.T) {
	tempDir := t.TempDir()
	spool := NewSpool(SpoolOptions{MemoryLimit: 1, MaximumBytes: 1024, TempDir: tempDir})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := spool.WriteAll(ctx, []byte("cancelled")); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled WriteAll() error = %v, want context.Canceled", err)
	}
	if err := spool.Close(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("staging entries after cancellation = %d, want 0", len(entries))
	}
}

func TestSpoolReplayAndDigest(t *testing.T) {
	spool := NewSpool(SpoolOptions{MemoryLimit: 4, MaximumBytes: 1024, TempDir: t.TempDir()})
	t.Cleanup(func() { _ = spool.Close() })
	want := []byte("bounded artifact bytes")
	if err := spool.WriteAll(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got, err := readSpool(spool)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("replay bytes = %q, want %q", got, want)
	}
	if got, wantDigest := spool.Digest(), ArtifactDigestOf(want); got != wantDigest {
		t.Fatalf("digest = %s, want %s", got, wantDigest)
	}
	if got := spool.Size(); got != int64(len(want)) {
		t.Fatalf("size = %d, want %d", got, len(want))
	}
}

func readSpool(spool *Spool) ([]byte, error) {
	reader, err := spool.Reader()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
