package margo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInstallDaggerReusesExactEmbeddedCLIWithoutDownload(t *testing.T) {
	script, err := filepath.Abs("scripts/install-dagger.sh")
	if err != nil {
		t.Fatal(err)
	}
	tools := t.TempDir()
	embedded := filepath.Join(tools, "dagger")
	if err := os.WriteFile(embedded, []byte("#!/bin/sh\necho 'dagger v0.21.8 embedded'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tools, "curl"), []byte("#!/bin/sh\necho download-called >&2\nexit 99\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	installDirectory := t.TempDir()
	cmd := exec.Command(script, installDirectory)
	cmd.Env = append(os.Environ(), "PATH="+tools+":"+os.Getenv("PATH"))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("reuse embedded Dagger: %v\n%s", err, output)
	}
	output, err := exec.Command(filepath.Join(installDirectory, "dagger"), "version").CombinedOutput()
	if err != nil || string(output) != "dagger v0.21.8 embedded\n" {
		t.Fatalf("installed Dagger: %v %q", err, output)
	}
}
