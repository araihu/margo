package margo

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestDaggerModuleAndAdaptersStayPinned(t *testing.T) {
	configData, err := os.ReadFile("dagger.json")
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Name          string `json:"name"`
		EngineVersion string `json:"engineVersion"`
		Source        string `json:"source"`
		SDK           struct {
			Source string `json:"source"`
		} `json:"sdk"`
	}
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatal(err)
	}
	if config.Name != "margo" || config.EngineVersion != "v0.21.8" || config.Source != "dagger" || config.SDK.Source != "go" {
		t.Fatalf("unexpected Dagger configuration: %#v", config)
	}

	for _, path := range []string{
		".github/workflows/ci.yml",
		".github/workflows/pages.yml",
		".github/workflows/release.yml",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		workflow := string(data)
		if !strings.Contains(workflow, "scripts/install-dagger.sh") ||
			!strings.Contains(workflow, `test "$(dagger version | awk 'NR == 1 {print $2}')" = 'v0.21.8'`) {
			t.Fatalf("%s does not install and gate exact Dagger CLI", path)
		}
		if strings.Contains(workflow, "dagger/dagger-for-github") {
			t.Fatalf("%s retains permissive dagger-for-github adapter", path)
		}
		if strings.Contains(strings.ToLower(workflow), "coderabbit") {
			t.Fatalf("%s retains CodeRabbit configuration", path)
		}
	}
}

func TestDaggerModuleKeepsPublishEffectsAtProviderBoundary(t *testing.T) {
	data, err := os.ReadFile("dagger/main.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, forbidden := range []string{"GITHUB_TOKEN", "Publish(ctx", "deploy-pages", "github release"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("Dagger module contains provider-side effect %q", forbidden)
		}
	}
	for _, required := range []string{
		"WithMountedCache", "margo-go-mod-", "margo-go-build-", "margo-goreleaser-build-",
		"execution.CacheDomain", "execution.Nonce", `+cache="never"`,
		`gitBundle *dagger.File`, `git clone /tmp/margo.bundle /src`,
		`git merge-base --is-ancestor HEAD refs/heads/main`,
		`cd dagger && go mod verify`, `gofmt -l .`,
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("Dagger module missing cache contract %q", required)
		}
	}
}

func TestSnapshotUsesBusyBoxCompatibleZipListing(t *testing.T) {
	data, err := os.ReadFile("dagger/main.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	if strings.Contains(source, "unzip -Z") {
		t.Fatal("snapshot archive verification uses unsupported BusyBox unzip -Z")
	}
	for _, required := range []string{`unzip -l "$archive"`, `awk '$1 ~ /^[0-9]+$/ { print $4 }'`} {
		if !strings.Contains(source, required) {
			t.Fatalf("snapshot archive verification misses %q", required)
		}
	}
}

func TestDaggerModuleUsesGeneratedSDKBindingsConsistently(t *testing.T) {
	data, err := os.ReadFile("dagger/main.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	if !strings.Contains(source, `"dagger/margo/internal/dagger"`) {
		t.Fatal("Dagger module does not use generated internal SDK bindings")
	}
	for _, forbidden := range []string{
		`"dagger.io/dagger"`,
		`"dagger.io/dagger/dag"`,
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("Dagger module imports incompatible external SDK binding %s", forbidden)
		}
	}
	for _, signature := range []string{
		`Source *dagger.Directory`,
		`source *dagger.Directory`,
		`ciContext *dagger.File`,
		`gitBundle *dagger.File`,
	} {
		if !strings.Contains(source, signature) {
			t.Fatalf("Dagger module misses generated SDK type contract %q", signature)
		}
	}
	if !strings.Contains(source, "dag.Container()") || !strings.Contains(source, "dag.CacheVolume(") {
		t.Fatal("Dagger module does not use generated global client")
	}
}

func TestDaggerInstallerPinsOfficialLinuxArchive(t *testing.T) {
	data, err := os.ReadFile("scripts/install-dagger.sh")
	if err != nil {
		t.Fatal(err)
	}
	script := string(data)
	for _, required := range []string{
		`dagger_version="0.21.8"`,
		`dagger_v${dagger_version}_linux_amd64.tar.gz`,
		`53e226c7da8fb75171e58c35759d736d961ce8b3a12db0baa7b7107954fccc5a`,
		`curl --proto '=https' --tlsv1.2`,
		`actual_sha256`, `actual_version`, `v${dagger_version}`,
		`command -v dagger`, `embedded_version`, `ln -sf`,
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("installer missing %q", required)
		}
	}
}

func TestDaggerSourceExcludesExplicitGitBundle(t *testing.T) {
	data, err := os.ReadFile("dagger/main.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `+ignore=[".git", ".dagger", ".dagger-ci-context.json", ".dagger-git.bundle"`) {
		t.Fatal("Dagger source does not exclude explicit Git bundle argument")
	}
}

func TestDaggerAdaptersUseTrustedContextChannel(t *testing.T) {
	for _, path := range []string{".github/workflows/ci.yml", ".github/workflows/release.yml"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		workflow := string(data)
		for _, line := range strings.Split(workflow, "\n") {
			if strings.Contains(line, "args:") && strings.Contains(line, "${{") {
				t.Fatalf("%s interpolates GitHub context into action args: %s", path, line)
			}
		}
		for _, required := range []string{
			"MARGO_CACHE_DOMAIN:", "MARGO_RUN_ID: ${{ github.run_id }}",
			"MARGO_RUN_ATTEMPT: ${{ github.run_attempt }}", "scripts/write-dagger-context.sh",
			"--ci-context=.dagger-ci-context.json",
		} {
			if !strings.Contains(workflow, required) {
				t.Fatalf("%s missing trusted context contract %q", path, required)
			}
		}
	}
}

func TestRequiredCheckDisplayNamesRemainStable(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, name := range []string{
		"name: Single-module CI", "name: Release shape (portable-linux)",
		"name: Release shape (${{ matrix.label }})", "name: GoReleaser snapshot",
		"name: Portable musl and no-browser behavior",
	} {
		if !strings.Contains(workflow, name) {
			t.Fatalf("required CI missing stable check %q", name)
		}
	}
}
