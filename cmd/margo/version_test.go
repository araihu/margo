package main

import (
	"bytes"
	"context"
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
		"engines chromium,native\n"
	if got := stdout.String(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
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
