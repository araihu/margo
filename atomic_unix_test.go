//go:build darwin || linux

package margo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicUnixDefaultOperationsAreAvailable(t *testing.T) {
	targetDir := t.TempDir()
	stage := filepath.Join(targetDir, ".stage")
	if err := os.WriteFile(stage, []byte("stage"), 0o600); err != nil {
		t.Fatal(err)
	}
	visible, err := defaultAtomicOps().publishNoReplace(stage, filepath.Join(targetDir, "target"))
	if err != nil {
		t.Fatalf("publishNoReplace() error = %v", err)
	}
	if !visible {
		t.Fatal("publishNoReplace() visible = false, want true")
	}
	if _, err := os.Stat(stage); !os.IsNotExist(err) {
		t.Fatalf("stage stat error = %v, want absent", err)
	}
}

func TestAtomicUnixStageUsesPrivateMode(t *testing.T) {
	stage, err := defaultAtomicOps().createStage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	name := stage.Name()
	defer func() {
		_ = stage.Close()
		_ = os.Remove(name)
	}()
	info, err := stage.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o600); got != want {
		t.Fatalf("stage mode = %o, want %o", got, want)
	}
}
