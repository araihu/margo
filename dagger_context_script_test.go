package margo

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWriteDaggerContextUsesValidatedEnvironmentChannel(t *testing.T) {
	script, err := filepath.Abs("scripts/write-dagger-context.sh")
	if err != nil {
		t.Fatal(err)
	}
	workdir := t.TempDir()
	cmd := exec.Command(script)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(),
		"MARGO_CACHE_DOMAIN=untrusted-pr-42",
		"MARGO_RUN_ID=12345",
		"MARGO_RUN_ATTEMPT=2",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("write context: %v\n%s", err, output)
	}
	data, err := os.ReadFile(filepath.Join(workdir, ".dagger-ci-context.json"))
	if err != nil {
		t.Fatal(err)
	}
	var context struct {
		CacheDomain string `json:"cacheDomain"`
		Nonce       string `json:"nonce"`
	}
	if err := json.Unmarshal(data, &context); err != nil {
		t.Fatal(err)
	}
	if context.CacheDomain != "untrusted-pr-42" || context.Nonce != "12345-2" {
		t.Fatalf("context = %#v", context)
	}
}

func TestWriteDaggerContextRejectsUntrustedShellText(t *testing.T) {
	script, err := filepath.Abs("scripts/write-dagger-context.sh")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(script)
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(),
		"MARGO_CACHE_DOMAIN=trusted-main; touch injected",
		"MARGO_RUN_ID=12345",
		"MARGO_RUN_ATTEMPT=2",
	)
	if err := cmd.Run(); err == nil {
		t.Fatal("context writer accepted shell text")
	}
}

func TestWriteDaggerContextRejectsUntrustedReleaseTag(t *testing.T) {
	script, err := filepath.Abs("scripts/write-dagger-context.sh")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(script)
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(),
		"MARGO_CACHE_DOMAIN=trusted-release",
		"MARGO_RUN_ID=12345",
		"MARGO_RUN_ATTEMPT=2",
		"MARGO_RELEASE_TAG=v1.2.3; touch injected",
	)
	if err := cmd.Run(); err == nil {
		t.Fatal("context writer accepted release-tag shell text")
	}
}
