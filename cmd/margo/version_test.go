package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
)

func TestVersionWritesCompleteBuildIdentityToStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
		Stderr: &stderr,
		Build:  testBuildInfo(),
	})
	cmd.SetArgs([]string{"version"})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := "margo v0.1.0\n" +
		"module github.com/araihu/margo\n" +
		"commit abc123\n" +
		"go go1.26.5\n" +
		"platform darwin/arm64\n" +
		"compiled engines chromium,native\n" +
		"external engine discovery run \"margo doctor\"\n"
	if got := stdout.String(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRootVersionFlagAliasesVersionCommand(t *testing.T) {
	var commandOutput, flagOutput bytes.Buffer
	for _, test := range []struct {
		args   []string
		output *bytes.Buffer
	}{
		{args: []string{"version"}, output: &commandOutput},
		{args: []string{"--version"}, output: &flagOutput},
	} {
		command := NewRootCommand(Dependencies{Stdout: test.output, Build: testBuildInfo()})
		command.SetArgs(test.args)
		if err := command.ExecuteContext(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if commandOutput.String() != flagOutput.String() {
		t.Fatalf("--version output = %q, version output = %q", flagOutput.String(), commandOutput.String())
	}
}

func testBuildInfo() BuildInfo {
	return BuildInfo{
		Module:    "github.com/araihu/margo",
		Version:   "v0.1.0",
		Commit:    "abc123",
		GoVersion: "go1.26.5",
		GOOS:      "darwin",
		GOARCH:    "arm64",
		Engines:   []string{"native", "chromium"},
	}
}

func TestBuildInfoFromGoMetadataNormalizesDevelopmentIdentity(t *testing.T) {
	info := &debug.BuildInfo{
		GoVersion: "go1.26.5",
		Main: debug.Module{
			Path:    "github.com/araihu/margo/cmd/margo",
			Version: "(devel)",
		},
		Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "deadbeef"}},
	}
	got := buildInfoFromGoMetadata(info, "darwin", "arm64", []string{"chromium"})
	want := BuildInfo{
		Module:    "github.com/araihu/margo",
		Version:   "dev",
		Commit:    "deadbeef",
		GoVersion: "go1.26.5",
		GOOS:      "darwin",
		GOARCH:    "arm64",
		Engines:   []string{"chromium"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestReadBuildInfoReportsRuntimeAndCapabilities(t *testing.T) {
	got := ReadBuildInfo([]string{"chromium"})
	if got.Module != "github.com/araihu/margo" {
		t.Fatalf("module = %q", got.Module)
	}
	if got.Version == "" || got.Commit == "" || got.GoVersion == "" {
		t.Fatalf("incomplete build info: %#v", got)
	}
	if got.GOOS != runtime.GOOS || got.GOARCH != runtime.GOARCH {
		t.Fatalf("platform = %s/%s", got.GOOS, got.GOARCH)
	}
	if !reflect.DeepEqual(got.Engines, []string{"chromium"}) {
		t.Fatalf("engines = %v", got.Engines)
	}
}

func TestReleaseLinkerIdentityOverridesDevelopmentMetadata(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "margo")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-trimpath", "-ldflags", "-X main.releaseVersion=v0.0.3 -X main.releaseCommit=0123456789abcdef", "-o", binary, ".")
	build.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=readonly", "CGO_ENABLED=0")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build release binary: %v\n%s", err, output)
	}
	command := exec.Command(binary, "version")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run release binary: %v\n%s", err, output)
	}
	identity := string(output)
	if !strings.Contains(identity, "margo v0.0.3\n") || !strings.Contains(identity, "commit 0123456789abcdef\n") {
		t.Fatalf("release identity missing linker values:\n%s", identity)
	}
}
