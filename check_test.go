package margo

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type checkMapReader map[string][]byte

func firstDiagnosticCode(err error) string {
	diagnostic := unwrapDiagnostic(err)
	if diagnostic != nil && len(diagnostic.Diagnostics) > 0 {
		return diagnostic.Diagnostics[0].Code
	}
	if err == nil {
		return ""
	}
	code, _, _ := strings.Cut(err.Error(), ":")
	return code
}

func (reader checkMapReader) ReadAsset(ctx context.Context, root, name string, limit int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, ok := reader[filepath.Clean(filepath.Join(root, filepath.FromSlash(name)))]
	if !ok {
		return nil, os.ErrNotExist
	}
	if int64(len(data)) > limit {
		return nil, ErrCheckAssetTooLarge
	}
	return append([]byte(nil), data...), nil
}

func TestCheckReportsActionableCompatibilityDiagnostics(t *testing.T) {
	root := filepath.Clean("/workspace/docs")
	source := Source{
		Name:    filepath.Join(root, "guide.md"),
		BaseURL: root,
		Content: []byte("---\nlanguage: en-US\n---\n\n<span>raw</span>\n\n![remote](https://cdn.example.com/image.png)\n![missing](missing.png)\n![unsafe](unsafe.svg)\n![](ok.png)\n[Guide](other.md)\n\n```mermaid\n%%{init: {\"theme\": \"dark\"}}%%\ngraph TD; A-->B\n```\n"),
	}
	reader := checkMapReader{
		filepath.Join(root, "unsafe.svg"): []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		filepath.Join(root, "ok.png"):     []byte("\x89PNG\r\n\x1a\n"),
	}

	diagnostics, err := Check(context.Background(), source, WithCheckAssetReader(reader))
	if err != nil {
		t.Fatal(err)
	}
	wantCodes := []string{
		"check.raw_html",
		"check.asset_remote",
		"check.asset_missing",
		"check.svg_incompatible",
		"check.image_alt_empty",
		"check.link_relative",
		"mermaid.configuration_forbidden",
	}
	gotCodes := make([]string, len(diagnostics))
	for index, diagnostic := range diagnostics {
		gotCodes[index] = diagnostic.Code
		if diagnostic.Source != source.Name || diagnostic.Line <= 0 || diagnostic.Column <= 0 {
			t.Errorf("diagnostic %q has incomplete source position: %+v", diagnostic.Code, diagnostic)
		}
		if diagnostic.Pointer == "" || diagnostic.Hint == "" {
			t.Errorf("diagnostic %q is not actionable: %+v", diagnostic.Code, diagnostic)
		}
	}
	if !reflect.DeepEqual(gotCodes, wantCodes) {
		t.Fatalf("codes = %#v, want %#v\ndiagnostics: %+v", gotCodes, wantCodes, diagnostics)
	}
	if diagnostics[6].Line != 14 || diagnostics[6].Pointer != "/mermaid/configuration" {
		t.Fatalf("Mermaid diagnostic = %+v", diagnostics[6])
	}
	if diagnostics[4].Severity != SeverityWarning || diagnostics[5].Severity != SeverityWarning {
		t.Fatalf("advisory severities = %q, %q", diagnostics[4].Severity, diagnostics[5].Severity)
	}
}

func TestCheckPolicyUsesHostOwnedRawHTMLAuthority(t *testing.T) {
	trusted := Source{Name: "trusted.md", Content: []byte("---\nlanguage: en\n---\n\n<span>trusted text</span>\n")}
	diagnostics, err := Check(context.Background(), trusted, WithCheckPolicy(Policy{RawHTML: RawHTMLSanitized, OutputBytes: MaxOutputBytes}))
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("host-authorized sanitized diagnostics = %+v", diagnostics)
	}

	diagnostics, err = Check(context.Background(), trusted, WithCheckPolicy(Policy{RawHTML: RawHTMLDeny, OutputBytes: MaxOutputBytes}))
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "policy.raw_html.denied" {
		t.Fatalf("host-denied diagnostics = %+v", diagnostics)
	}
}

func TestCheckPolicyMatchesCompileAndRenderRawHTMLFailures(t *testing.T) {
	tests := []struct {
		name   string
		policy Policy
		source Source
		code   string
	}{
		{
			name:   "host authorization cannot bypass HTML allowlist",
			policy: Policy{RawHTML: RawHTMLSanitized, OutputBytes: MaxOutputBytes},
			source: Source{Name: "unsafe.md", Content: []byte("---\nlanguage: en\n---\n\n<script>alert(1)</script>\n")},
			code:   "policy.html.invalid",
		},
		{
			name:   "host deny rejects raw HTML",
			policy: Policy{RawHTML: RawHTMLDeny, OutputBytes: MaxOutputBytes},
			source: Source{Name: "denied.md", Content: []byte("---\nlanguage: en\n---\n\n<span>raw</span>\n")},
			code:   "policy.raw_html.denied",
		},
		{
			name:   "unsafe HTML block is rejected",
			policy: Policy{RawHTML: RawHTMLSanitized, OutputBytes: MaxOutputBytes},
			source: Source{Name: "undeclared-block.md", Content: []byte("---\nlanguage: en\n---\n\n<script>alert(1)</script>\n")},
			code:   "policy.html.invalid",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diagnostics, err := Check(context.Background(), test.source, WithCheckPolicy(test.policy))
			if err != nil {
				t.Fatal(err)
			}
			if len(diagnostics) != 1 || diagnostics[0].Code != test.code {
				t.Fatalf("check diagnostics = %+v", diagnostics)
			}

			compiler := New(WithHostPolicy(test.policy))
			document, compileErr := compiler.Compile(context.Background(), test.source)
			var renderErr error
			if compileErr == nil {
				result, err := compiler.Render(context.Background(), document)
				if err == nil {
					var output strings.Builder
					renderErr = result.Content().Render(context.Background(), &output)
				} else {
					renderErr = err
				}
			}
			failure := compileErr
			if failure == nil {
				failure = renderErr
			}
			if got := firstDiagnosticCode(failure); got != test.code {
				t.Fatalf("compile/render diagnostic = %q, want %q, error = %v", got, test.code, failure)
			}
		})
	}
}

func TestCheckOptionsAreSafeForConcurrentReuse(t *testing.T) {
	policyOption := WithCheckPolicy(Policy{OutputBytes: MaxOutputBytes})
	extensionOption := WithCheckExtension(ExtensionRegistration{
		Identity: ExtensionIdentity{Name: "demo-check", Version: "v1"}, Fences: []string{"demo-check"},
		Check: func(context.Context, ExtensionNode) error { return nil },
	})
	source := Source{Name: "safe.md", Content: []byte("---\nlanguage: en\n---\n\n```demo-check\nsafe\n```\n")}
	const workers = 64
	failures := make(chan error, workers)
	var group sync.WaitGroup
	for index := 0; index < workers; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			diagnostics, err := Check(context.Background(), source, policyOption, extensionOption)
			if err != nil {
				failures <- err
				return
			}
			if len(diagnostics) != 0 {
				failures <- errors.New("concurrent check returned diagnostics")
			}
		}()
	}
	group.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
}

func TestWithCheckExtensionRejectsAmbiguousRegistrations(t *testing.T) {
	checker := func(context.Context, ExtensionNode) error { return nil }
	tests := []struct {
		name    string
		options []CheckOption
	}{
		{
			name: "empty fence",
			options: []CheckOption{WithCheckExtension(ExtensionRegistration{
				Identity: ExtensionIdentity{Name: "one", Version: "v1"}, Fences: []string{""}, Check: checker,
			})},
		},
		{
			name: "duplicate local fence",
			options: []CheckOption{WithCheckExtension(ExtensionRegistration{
				Identity: ExtensionIdentity{Name: "one", Version: "v1"}, Fences: []string{"demo", "demo"}, Check: checker,
			})},
		},
		{
			name: "duplicate identity",
			options: []CheckOption{
				WithCheckExtension(ExtensionRegistration{Identity: ExtensionIdentity{Name: "one", Version: "v1"}, Fences: []string{"first"}, Check: checker}),
				WithCheckExtension(ExtensionRegistration{Identity: ExtensionIdentity{Name: "one", Version: "v1"}, Fences: []string{"second"}, Check: checker}),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Check(context.Background(), Source{Name: "x.md", Content: []byte("# x\n")}, test.options...); err == nil || !strings.Contains(err.Error(), "check.extension_invalid") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestCheckExtensionPlainFailureUsesFenceSourcePosition(t *testing.T) {
	registration := ExtensionRegistration{
		Identity: ExtensionIdentity{Name: "demo", Version: "v1"}, Fences: []string{"demo"},
		Check: func(context.Context, ExtensionNode) error { return errors.New("invalid demo payload") },
	}
	source := Source{Name: "demo.md", Content: []byte("---\nlanguage: en\n---\n\n```demo\ninvalid\n```\n")}
	diagnostics, err := Check(context.Background(), source, WithCheckExtension(registration))
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "check.extension_invalid" || diagnostics[0].Line != 6 || diagnostics[0].Column != 1 {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestCheckExtensionReceivesRequestedTarget(t *testing.T) {
	var got RenderTarget
	registration := ExtensionRegistration{
		Identity: ExtensionIdentity{Name: "target-aware", Version: "v1"}, Fences: []string{"target-aware"},
		Check: func(_ context.Context, node ExtensionNode) error {
			got = node.Target
			return nil
		},
	}
	source := Source{Name: "deck.md", Content: []byte("---\nlanguage: en\n---\n\n```target-aware\ncontent\n```\n")}
	if _, err := Check(context.Background(), source, WithCheckTarget(TargetDeck), WithCheckExtension(registration)); err != nil {
		t.Fatal(err)
	}
	if got != TargetDeck {
		t.Fatalf("extension target = %q, want %q", got, TargetDeck)
	}
}

func TestCheckDoesNotDropSameLineFindings(t *testing.T) {
	source := Source{Name: "/workspace/guide.md", BaseURL: "/workspace", Content: []byte("![one](one.png) ![two](two.png) [x](x.md) [y](y.md)\n")}
	diagnostics, err := Check(context.Background(), source, WithCheckAssetReader(checkMapReader{}))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"check.language_missing", "check.asset_missing", "check.asset_missing", "check.link_relative", "check.link_relative"}
	got := make([]string, len(diagnostics))
	columns := make(map[int]struct{})
	for index, diagnostic := range diagnostics {
		got[index] = diagnostic.Code
		columns[diagnostic.Column] = struct{}{}
	}
	if !reflect.DeepEqual(got, want) || len(columns) != 5 {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestCheckLocatesLinkSyntaxInsteadOfEarlierDestinationText(t *testing.T) {
	source := Source{Name: "guide.md", Content: []byte("target appears here\n\n[link](target)\n")}
	diagnostics, err := Check(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 2 || diagnostics[1].Code != "check.link_relative" || diagnostics[1].Line != 3 {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestCheckSiteAllowsRelativeLinksForSiteBuildValidation(t *testing.T) {
	source := Source{Name: "index.md", Content: []byte("---\nlanguage: en\n---\n\n# Home\n\n[Post](post.md)\n")}
	diagnostics, err := Check(context.Background(), source, WithCheckTarget(TargetSite))
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "check.link_relative" {
			t.Fatalf("site check emitted standalone relative-link warning: %+v", diagnostic)
		}
	}
	if len(diagnostics) != 0 {
		t.Fatalf("site check diagnostics = %+v, want none", diagnostics)
	}
}

func TestCheckRetainsRelativeLinkWarningForStandaloneTargets(t *testing.T) {
	source := Source{Name: "guide.md", Content: []byte("---\nlanguage: en\n---\n\n[Guide](other.md)\n")}
	for _, target := range []RenderTarget{TargetHTML, TargetPDF, TargetDeck} {
		t.Run(string(target), func(t *testing.T) {
			diagnostics, err := Check(context.Background(), source, WithCheckTarget(target))
			if err != nil {
				t.Fatal(err)
			}
			if len(diagnostics) != 1 || diagnostics[0].Code != "check.link_relative" {
				t.Fatalf("%s diagnostics = %+v, want relative-link warning", target, diagnostics)
			}
		})
	}
}

func TestCheckSiteRetainsNonRelativeLinkDiagnostics(t *testing.T) {
	source := Source{Name: "index.md", Content: []byte("---\nlanguage: en\n---\n\n[Empty]()\n\n[Unsupported](javascript:alert(1))\n\n[Network path](//example.com/docs)\n")}
	diagnostics, err := Check(context.Background(), source, WithCheckTarget(TargetSite))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"check.link_destination_empty", "check.link_unsupported", "check.link_unsupported"}
	if len(diagnostics) != len(want) {
		t.Fatalf("site diagnostics = %+v, want codes %v", diagnostics, want)
	}
	for index, code := range want {
		if diagnostics[index].Code != code {
			t.Fatalf("site diagnostics[%d] = %+v, want code %q", index, diagnostics[index], code)
		}
	}
}

func TestCheckWarnsForMissingLanguageSkippedHeadingAndEmptyLink(t *testing.T) {
	source := Source{Name: "guide.md", Content: []byte("# Main\n\n### Skipped level\n\n[unfinished]()\n")}
	diagnostics, err := Check(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"check.language_missing", "check.heading_level_skipped", "check.link_destination_empty"}
	got := make([]string, len(diagnostics))
	for index := range diagnostics {
		got[index] = diagnostics[index].Code
		if diagnostics[index].Severity != SeverityWarning || diagnostics[index].Hint == "" {
			t.Fatalf("accessibility diagnostic is not actionable: %+v", diagnostics[index])
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestCheckEnforcesMetadataListLimitsAndSequencePositions(t *testing.T) {
	var many strings.Builder
	many.WriteString("---\nauthors:\n")
	for index := 0; index < 65; index++ {
		many.WriteString("  - author\n")
	}
	many.WriteString("---\n# x\n")
	diagnostics, err := Check(context.Background(), Source{Name: "many.md", Content: []byte(many.String())})
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "frontmatter.schema_invalid" || diagnostics[0].Pointer != "/authors" || diagnostics[0].Line != 3 {
		t.Fatalf("list limit diagnostics = %+v", diagnostics)
	}

	invalid := Source{Name: "invalid.md", Content: []byte("---\nauthors:\n  - valid\n  - 42\n---\n")}
	diagnostics, err = Check(context.Background(), invalid)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Pointer != "/authors/1" || diagnostics[0].Line != 4 {
		t.Fatalf("sequence diagnostics = %+v", diagnostics)
	}
}

func TestFilesystemCheckAssetReaderRejectsEscapeAndOversize(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "docs")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside.png")
	if err := os.WriteFile(outside, []byte("\x89PNG\r\n\x1a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape.png")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	huge := filepath.Join(root, "huge.png")
	file, err := os.Create(huge)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(MaxDocumentBytes + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	source := Source{Name: filepath.Join(root, "guide.md"), BaseURL: root, Content: []byte("![escape](escape.png)\n\n![huge](huge.png)\n")}
	diagnostics, err := Check(context.Background(), source, WithCheckAssetReader(FilesystemCheckAssetReader{}))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"check.language_missing", "check.asset_outside_root", "check.asset_too_large"}
	got := make([]string, len(diagnostics))
	for index := range diagnostics {
		got[index] = diagnostics[index].Code
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestCheckValidatesAssetContentAndDataSVG(t *testing.T) {
	root := filepath.Clean("/workspace")
	unsafeSVG := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	dataSVG := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(unsafeSVG)
	source := Source{Name: filepath.Join(root, "guide.md"), BaseURL: root, Content: []byte("![wrong name](unsafe.png)\n\n![bad bytes](broken.png)\n\n![data](" + dataSVG + ")\n")}
	reader := checkMapReader{
		filepath.Join(root, "unsafe.png"): unsafeSVG,
		filepath.Join(root, "broken.png"): []byte("not an image"),
	}
	diagnostics, err := Check(context.Background(), source, WithCheckAssetReader(reader))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"check.language_missing", "check.svg_incompatible", "check.asset_incompatible", "check.svg_incompatible"}
	got := make([]string, len(diagnostics))
	for index := range diagnostics {
		got[index] = diagnostics[index].Code
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestCheckRejectsMislabeledDataImage(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\n")
	mislabeled := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(png)
	diagnostics, err := Check(context.Background(), Source{Name: "guide.md", Content: []byte("![image](" + mislabeled + ")\n")})
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 2 || diagnostics[1].Code != "check.asset_incompatible" {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestCheckRejectsWindowsAbsoluteLink(t *testing.T) {
	diagnostics, err := Check(context.Background(), Source{Name: "guide.md", Content: []byte("[local](C:/docs/x.md)\n")})
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 2 || diagnostics[1].Code != "check.link_unsupported" || diagnostics[1].Severity != SeverityError {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestCheckReportsMalformedLinksInsideTableCells(t *testing.T) {
	source := Source{Name: "table.md", Content: []byte("---\nlanguage: en\n---\n\n# Table\n\n| Resource | Destination |\n| --- | --- |\n| Unsafe | [Open](javascript:alert(1)) |\n")}
	diagnostics, err := Check(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "check.link_unsupported" {
		t.Fatalf("diagnostics = %+v, want one unsupported-link diagnostic", diagnostics)
	}
}

type blockingCheckReader struct{ started chan struct{} }

func (reader blockingCheckReader) ReadAsset(ctx context.Context, _, _ string, _ int64) ([]byte, error) {
	close(reader.started)
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestCheckCancellationInterruptsAssetRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader := blockingCheckReader{started: make(chan struct{})}
	result := make(chan error, 1)
	go func() {
		_, err := Check(ctx, Source{Name: "/workspace/guide.md", BaseURL: "/workspace", Content: []byte("![x](x.png)\n")}, WithCheckAssetReader(reader))
		result <- err
	}()
	<-reader.started
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Check did not interrupt the asset read")
	}
}

func TestCheckIsDeterministicAndDoesNotMutateSource(t *testing.T) {
	source := Source{Name: "clean.md", BaseURL: "/workspace", Content: []byte("# Clean\n\n[external](https://example.com)\n")}
	original := append([]byte(nil), source.Content...)
	first, err := Check(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Check(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || first[0].Code != "check.language_missing" || !reflect.DeepEqual(first, second) || !reflect.DeepEqual(source.Content, original) {
		t.Fatalf("first=%+v second=%+v source=%q", first, second, source.Content)
	}
}

func TestCheckHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Check(ctx, Source{Name: "x.md", Content: []byte("# x")}); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}
