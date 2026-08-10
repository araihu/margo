package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReleaseShapeVerifierAcceptsBuiltMargoAndRejectsWrongName(t *testing.T) {
	directory := t.TempDir()
	name := "margo"
	if runtime.GOOS == "windows" {
		name = "margo.exe"
	}
	binary := filepath.Join(directory, name)
	build := exec.Command("go", "build", "-trimpath", "-o", binary, ".")
	build.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=readonly")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	verify := exec.Command("bash", "../../scripts/verify-release-shape.sh", directory)
	if output, err := verify.CombinedOutput(); err != nil {
		t.Fatalf("verify: %v\n%s", err, output)
	} else if !strings.Contains(string(output), "root version (devel)") {
		t.Fatalf("verify reported the wrong root version: %s", output)
	}
	wrong := filepath.Join(t.TempDir(), "not-margo")
	if err := os.WriteFile(wrong, []byte("not a binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	verify = exec.Command("bash", "../../scripts/verify-release-shape.sh", filepath.Dir(wrong))
	if err := verify.Run(); err == nil {
		t.Fatal("release verifier accepted a wrongly named artifact")
	}
}
