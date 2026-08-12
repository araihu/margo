// Package main exposes Margo's portable build and validation pipelines.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"dagger.io/dagger"
	"dagger.io/dagger/dag"
)

const (
	goVersion       = "1.26.5"
	goImage         = "golang:1.26.5-bookworm@sha256:53eeac89074db483fdf0ab3be1df32bf6e47562263d2d0d6baa7f26acb4957dd"
	alpineGoImage   = "golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2"
	goreleaserImage = "goreleaser/goreleaser:v2.17.1@sha256:1098a0be4da1780f9616a85f4c5050447b53e3e74804d8017ec1e2bbb1fb697a"
	alpineImage     = "alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b"
)

var releaseTagPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$`)

var (
	cacheDomainPattern = regexp.MustCompile(`^(local|trusted-main|trusted-release|untrusted-pr-[0-9]+)$`)
	noncePattern       = regexp.MustCompile(`^[0-9]+-[0-9]+$`)
)

type executionContext struct {
	CacheDomain string `json:"cacheDomain"`
	Nonce       string `json:"nonce"`
	ReleaseTag  string `json:"releaseTag,omitempty"`
}

type Margo struct {
	Source *dagger.Directory
}

func New(
	// Repository source. Generated output, VCS metadata, and local caches are excluded.
	// +defaultPath="."
	// +ignore=[".git", ".dagger", ".dagger-ci-context.json", ".dagger-git.bundle", "dist", "test/browser/.cache", "go.work", "go.work.sum"]
	source *dagger.Directory,
) *Margo {
	return &Margo{Source: source}
}

// Required runs the complete required Go CI contract.
// +cache="never"
func (m *Margo) Required(
	ctx context.Context,
	// +optional
	ciContext *dagger.File,
) error {
	execution, err := readExecutionContext(ctx, ciContext)
	if err != nil {
		return err
	}
	_, err = m.goBase(execution).
		WithDirectory("/baseline", m.Source).
		WithExec([]string{"bash", "-euo", "pipefail", "-c", requiredScript}).
		Sync(ctx)
	return err
}

// Test runs all root-module tests without using a host Go installation.
// +cache="never"
func (m *Margo) Test(
	ctx context.Context,
	// +optional
	ciContext *dagger.File,
) error {
	execution, err := readExecutionContext(ctx, ciContext)
	if err != nil {
		return err
	}
	_, err = m.goBase(execution).WithExec([]string{"go", "test", "./...", "-count=1"}).Sync(ctx)
	return err
}

// Vet runs Go's static analyzer for every root-module package.
// +cache="never"
func (m *Margo) Vet(
	ctx context.Context,
	// +optional
	ciContext *dagger.File,
) error {
	execution, err := readExecutionContext(ctx, ciContext)
	if err != nil {
		return err
	}
	_, err = m.goBase(execution).WithExec([]string{"go", "vet", "./..."}).Sync(ctx)
	return err
}

// Build compiles every root-module package.
// +cache="never"
func (m *Margo) Build(
	ctx context.Context,
	// +optional
	ciContext *dagger.File,
) error {
	execution, err := readExecutionContext(ctx, ciContext)
	if err != nil {
		return err
	}
	_, err = m.goBase(execution).WithExec([]string{"go", "build", "./..."}).Sync(ctx)
	return err
}

// PortableReleaseShape builds and verifies the Linux release-shaped command.
// +cache="never"
func (m *Margo) PortableReleaseShape(
	ctx context.Context,
	// +optional
	ciContext *dagger.File,
) error {
	execution, err := readExecutionContext(ctx, ciContext)
	if err != nil {
		return err
	}
	_, err = m.goBase(execution).
		WithEnvVariable("CGO_ENABLED", "0").
		WithExec([]string{"bash", "-euo", "pipefail", "-c", `
mkdir -p dist
go build -trimpath -o dist/margo ./cmd/margo
go version -m dist/margo
scripts/verify-release-shape.sh dist
`}).
		Sync(ctx)
	return err
}

// Snapshot builds and verifies the six GoReleaser snapshot archives.
// GitBundle is a self-contained bundle created by scripts/prepare-dagger-git.sh.
// It works for both normal clones and linked worktrees without transferring a
// host-specific .git file or absolute gitdir pointer into the container.
// +cache="never"
func (m *Margo) Snapshot(
	ctx context.Context,
	// +defaultPath=".dagger-git.bundle"
	gitBundle *dagger.File,
	// +optional
	ciContext *dagger.File,
) (*dagger.Directory, error) {
	execution, err := readExecutionContext(ctx, ciContext)
	if err != nil {
		return nil, err
	}
	return m.goreleaserBase(gitBundle, execution).
		WithExec([]string{"sh", "-euc", snapshotScript}).
		Directory("/src/dist"), nil
}

// Musl verifies the portable Alpine build and no-browser failure contract.
// +cache="never"
func (m *Margo) Musl(
	ctx context.Context,
	// +optional
	ciContext *dagger.File,
) error {
	execution, err := readExecutionContext(ctx, ciContext)
	if err != nil {
		return err
	}
	_, err = m.goBaseWithImage(alpineGoImage, "alpine", execution).
		WithEnvVariable("CGO_ENABLED", "0").
		WithExec([]string{"sh", "-euc", muslScript}).
		Sync(ctx)
	return err
}

// PagesSite returns the exact schema-only site consumed by GitHub Pages.
func (m *Margo) PagesSite(ctx context.Context) (*dagger.Directory, error) {
	ctr, err := dag.Container().
		From(alpineImage).
		WithDirectory("/src", m.Source).
		WithWorkdir("/src").
		WithExec([]string{"sh", "-euc", pagesScript}).
		Sync(ctx)
	if err != nil {
		return nil, err
	}
	return ctr.Directory("/src/_site"), nil
}

// ReleaseVerify validates a release tag, complete Git checkout, and tagged source without publishing.
// +cache="never"
func (m *Margo) ReleaseVerify(
	ctx context.Context,
	// +defaultPath=".dagger-git.bundle"
	gitBundle *dagger.File,
	ciContext *dagger.File,
) error {
	execution, err := readExecutionContext(ctx, ciContext)
	if err != nil {
		return err
	}
	if err := validateReleaseTag(execution.ReleaseTag); err != nil {
		return err
	}
	_, err = m.goBase(execution).
		WithFile("/tmp/margo.bundle", gitBundle).
		WithExec([]string{"bash", "-euo", "pipefail", "-c", "rm -rf /src && git clone /tmp/margo.bundle /src && (git -C /src show-ref --verify --quiet refs/heads/main || git -C /src branch main refs/remotes/origin/main) && test \"$(git -C /src rev-parse refs/heads/main)\" = \"$(git -C /src rev-parse refs/remotes/origin/main)\""}).
		WithDirectory("/src", m.Source).
		WithWorkdir("/src").
		WithEnvVariable("MARGO_RELEASE_TAG", execution.ReleaseTag).
		WithDirectory("/baseline", m.Source).
		WithExec([]string{"bash", "-euo", "pipefail", "-c", releaseVerifyScript}).
		Sync(ctx)
	return err
}

func (m *Margo) goBase(execution executionContext) *dagger.Container {
	return m.goBaseWithImage(goImage, "bookworm", execution)
}

func (m *Margo) goBaseWithImage(image, cacheSuffix string, execution executionContext) *dagger.Container {
	return dag.Container().
		From(image).
		WithEnvVariable("GOWORK", "off").
		WithEnvVariable("GOFLAGS", "-mod=readonly").
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithEnvVariable("GOCACHE", "/root/.cache/go-build").
		WithEnvVariable("MARGO_RUN_NONCE", execution.Nonce).
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("margo-go-mod-"+goVersion+"-"+execution.CacheDomain)).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("margo-go-build-"+goVersion+"-"+cacheSuffix+"-"+execution.CacheDomain)).
		WithDirectory("/src", m.Source).
		WithWorkdir("/src")
}

func (m *Margo) goreleaserBase(gitBundle *dagger.File, execution executionContext) *dagger.Container {
	return dag.Container().
		From(goreleaserImage).
		WithoutEntrypoint().
		WithoutDefaultArgs().
		WithEnvVariable("GOWORK", "off").
		WithEnvVariable("GOFLAGS", "-mod=readonly").
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithEnvVariable("GOCACHE", "/root/.cache/go-build").
		WithEnvVariable("MARGO_RUN_NONCE", execution.Nonce).
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("margo-go-mod-"+goVersion+"-"+execution.CacheDomain)).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("margo-goreleaser-build-"+goVersion+"-"+execution.CacheDomain)).
		WithFile("/tmp/margo.bundle", gitBundle).
		WithExec([]string{"sh", "-euc", "git clone /tmp/margo.bundle /src && (git -C /src show-ref --verify --quiet refs/heads/main || git -C /src branch main refs/remotes/origin/main) && test \"$(git -C /src rev-parse refs/heads/main)\" = \"$(git -C /src rev-parse refs/remotes/origin/main)\""}).
		WithDirectory("/src", m.Source).
		WithWorkdir("/src")
}

func readExecutionContext(ctx context.Context, file *dagger.File) (executionContext, error) {
	if file == nil {
		return executionContext{
			CacheDomain: "local",
			Nonce:       fmt.Sprintf("%d-1", time.Now().UTC().UnixNano()),
		}, nil
	}
	contents, err := file.Contents(ctx)
	if err != nil {
		return executionContext{}, fmt.Errorf("read execution context: %w", err)
	}
	var execution executionContext
	if err := json.Unmarshal([]byte(contents), &execution); err != nil {
		return executionContext{}, fmt.Errorf("decode execution context: %w", err)
	}
	execution.ReleaseTag = strings.TrimSpace(execution.ReleaseTag)
	if err := validateExecutionContext(execution); err != nil {
		return executionContext{}, err
	}
	return execution, nil
}

func validateExecutionContext(execution executionContext) error {
	if !cacheDomainPattern.MatchString(execution.CacheDomain) {
		return fmt.Errorf("invalid cache trust domain %q", execution.CacheDomain)
	}
	if !noncePattern.MatchString(execution.Nonce) {
		return fmt.Errorf("invalid execution nonce %q", execution.Nonce)
	}
	return nil
}

func validateReleaseTag(tag string) error {
	if !releaseTagPattern.MatchString(tag) {
		return &invalidReleaseTagError{tag: tag}
	}
	return nil
}

type invalidReleaseTagError struct{ tag string }

func (e *invalidReleaseTagError) Error() string {
	return "invalid root Margo release tag: " + e.tag
}

const requiredScript = `
module_files="$(find . -name go.mod -not -path './.git/*' -not -path '*/vendor/*' -not -path './dagger/*' -print | sort)"
test "$module_files" = './go.mod'
go mod verify
go vet ./...
go test ./... -count=1
go build ./...
go generate ./...
unformatted="$(find . -name '*.go' -not -path './.git/*' -not -path './dagger/*' -exec gofmt -l {} +)"
test -z "$unformatted" || { printf 'gofmt drift in:\n%s\n' "$unformatted" >&2; exit 1; }
diff -ruN --exclude=.git --exclude=dagger --exclude=dist --exclude=test/browser/.cache /baseline /src
(cd dagger && go mod verify && test -z "$(gofmt -l .)" && go vet ./... && go test ./... -count=1)
`

const releaseVerifyScript = `
test "$(git rev-list -n 1 "$MARGO_RELEASE_TAG")" = "$(git rev-parse HEAD)"
git merge-base --is-ancestor HEAD refs/heads/main
go mod verify
go vet ./...
go test ./... -count=1
unformatted="$(find . -name '*.go' -not -path './.git/*' -not -path './dagger/*' -exec gofmt -l {} +)"
test -z "$unformatted" || { printf 'gofmt drift in:\n%s\n' "$unformatted" >&2; exit 1; }
diff -ruN --exclude=.git --exclude=dagger --exclude=dist --exclude=test/browser/.cache /baseline /src
`

const snapshotScript = `
goreleaser release --snapshot --clean --skip=publish
cd dist
sha256sum -c checksums.txt
archive_count="$(find . -maxdepth 1 -type f \( -name 'margo_*.tar.gz' -o -name 'margo_*.zip' \) | wc -l | tr -d ' ')"
test "$archive_count" -eq 6
for archive in margo_*.tar.gz; do
  tar -tzf "$archive" | grep -qx 'margo'
  tar -tzf "$archive" | grep -qx 'LICENSE'
  tar -tzf "$archive" | grep -qx 'README.md'
done
for archive in margo_*.zip; do
  unzip -Z1 "$archive" | grep -qx 'margo.exe'
  unzip -Z1 "$archive" | grep -qx 'LICENSE'
  unzip -Z1 "$archive" | grep -qx 'README.md'
done
`

const muslScript = `
go test ./pdf/native -count=1
go build -trimpath -o /tmp/margo ./cmd/margo
/tmp/margo doctor --diagnostics json | grep -q pdf.native.compiled_out
if printf '# no browser\n' | /tmp/margo pdf - --output /tmp/should-not-exist.pdf; then
  echo 'PDF unexpectedly succeeded without an engine' >&2
  exit 1
fi
test ! -e /tmp/should-not-exist.pdf
`

const pagesScript = `
mkdir -p _site/schema
cp -R schema/v1 _site/schema/v1
test "$(find _site -type f | LC_ALL=C sort)" = "$(printf '_site/schema/v1/document.json\n_site/schema/v1/policy.json')"
cmp schema/v1/document.json _site/schema/v1/document.json
cmp schema/v1/policy.json _site/schema/v1/policy.json
`
