package margo

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type goreleaserContract struct {
	Version     int    `yaml:"version"`
	ProjectName string `yaml:"project_name"`
	Git         struct {
		IgnoreTags []string `yaml:"ignore_tags"`
	} `yaml:"git"`
	Builds []struct {
		ID      string   `yaml:"id"`
		Main    string   `yaml:"main"`
		Binary  string   `yaml:"binary"`
		Env     []string `yaml:"env"`
		GOOS    []string `yaml:"goos"`
		GOARCH  []string `yaml:"goarch"`
		Flags   []string `yaml:"flags"`
		LDFlags []string `yaml:"ldflags"`
	} `yaml:"builds"`
	Archives []struct {
		Formats         []string `yaml:"formats"`
		FormatOverrides []struct {
			GOOS    string   `yaml:"goos"`
			Formats []string `yaml:"formats"`
		} `yaml:"format_overrides"`
	} `yaml:"archives"`
	Checksum struct {
		NameTemplate string `yaml:"name_template"`
	} `yaml:"checksum"`
}

func TestGoReleaserBuildsOnePortableMargoMatrix(t *testing.T) {
	data, err := os.ReadFile(".goreleaser.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var config goreleaserContract
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if config.Version != 2 || config.ProjectName != "margo" || len(config.Builds) != 1 {
		t.Fatalf("unexpected GoReleaser root contract: %#v", config)
	}
	if !reflect.DeepEqual(config.Git.IgnoreTags, []string{"*/v*"}) {
		t.Fatalf("ignored historical submodule tags = %v", config.Git.IgnoreTags)
	}
	build := config.Builds[0]
	if build.ID != "margo" || build.Main != "./cmd/margo" || build.Binary != "margo" {
		t.Fatalf("unexpected build target: %#v", build)
	}
	if !reflect.DeepEqual(build.Env, []string{"CGO_ENABLED=0", "GOWORK=off", "GOFLAGS=-mod=readonly"}) {
		t.Fatalf("build environment = %v", build.Env)
	}
	if !reflect.DeepEqual(build.GOOS, []string{"linux", "darwin", "windows"}) || !reflect.DeepEqual(build.GOARCH, []string{"amd64", "arm64"}) {
		t.Fatalf("build matrix = %v/%v", build.GOOS, build.GOARCH)
	}
	for _, required := range []string{"-trimpath", "main.releaseVersion=v{{ .Version }}", "main.releaseCommit={{ .FullCommit }}"} {
		if !strings.Contains(strings.Join(append(build.Flags, build.LDFlags...), " "), required) {
			t.Fatalf("build flags missing %q: flags=%v ldflags=%v", required, build.Flags, build.LDFlags)
		}
	}
	if len(config.Archives) != 1 || !reflect.DeepEqual(config.Archives[0].Formats, []string{"tar.gz"}) {
		t.Fatalf("default archive format = %#v", config.Archives)
	}
	if len(config.Archives[0].FormatOverrides) != 1 || config.Archives[0].FormatOverrides[0].GOOS != "windows" || !reflect.DeepEqual(config.Archives[0].FormatOverrides[0].Formats, []string{"zip"}) {
		t.Fatalf("Windows archive override = %#v", config.Archives[0].FormatOverrides)
	}
	if config.Checksum.NameTemplate != "checksums.txt" {
		t.Fatalf("checksum name = %q", config.Checksum.NameTemplate)
	}
}

func TestRequiredCIValidatesGoReleaserSnapshot(t *testing.T) {
	workflowData, err := os.ReadFile(".github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	daggerData, err := os.ReadFile("dagger/main.go")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(workflowData)
	for _, required := range []string{
		"GoReleaser snapshot", "fetch-depth: 0",
		"scripts/install-dagger.sh", "scripts/prepare-dagger-git.sh",
		"dagger call snapshot --git-bundle=.dagger-git.bundle --ci-context=.dagger-ci-context.json sync",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("required CI missing %q", required)
		}
	}
	module := string(daggerData)
	for _, required := range []string{
		"goreleaser/goreleaser:v2.18.0@sha256:",
		"goreleaser release --snapshot --clean --skip=publish",
		"sha256sum -c checksums.txt", "archive_count", "margo_*.tar.gz", "margo_*.zip",
	} {
		if !strings.Contains(module, required) {
			t.Fatalf("Dagger snapshot function missing %q", required)
		}
	}
}

func TestReleaseWorkflowUsesPinnedGoReleaserForRootSemverTags(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, required := range []string{
		"tags:", "v[0-9]*.[0-9]*.[0-9]*", "contents: write", "fetch-depth: 0",
		"actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e",
		"dagger call release-verify --git-bundle=.dagger-git.bundle --ci-context=.dagger-ci-context.json",
		"goreleaser/goreleaser-action@f06c13b6b1a9625abc9e6e439d9c05a8f2190e94",
		"version: v2.18.0", "args: release --clean",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("release workflow missing %q", required)
		}
	}
	for _, forbidden := range []string{"pdf/v", "charts/v", "cmd/margo/v"} {
		if strings.Contains(workflow, forbidden) {
			t.Fatalf("release workflow retains submodule tag %q", forbidden)
		}
	}
}
