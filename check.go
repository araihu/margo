package margo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	internalmermaid "github.com/araihu/margo/internal/mermaid"
	"github.com/araihu/margo/internal/staticimage"
	goldast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

// CheckAssetReader supplies local assets to Check without coupling library
// users to the host filesystem.
type CheckAssetReader interface {
	ReadAsset(context.Context, string, string, int64) ([]byte, error)
}

var (
	ErrCheckAssetOutsideRoot = errors.New("check asset is outside its source root")
	ErrCheckAssetTooLarge    = errors.New("check asset exceeds its byte limit")
	ErrCheckAssetNotRegular  = errors.New("check asset is not a regular file")
)

// FilesystemCheckAssetReader reads bounded regular files after resolving
// symlinks and proving that the real target remains below the real root.
type FilesystemCheckAssetReader struct{}

func (FilesystemCheckAssetReader) ReadAsset(ctx context.Context, root, name string, limit int64) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	target, safe := checkAssetPath(root, name)
	if !safe {
		return nil, ErrCheckAssetOutsideRoot
	}
	realRoot, err := filepath.EvalSymlinks(filepath.Clean(root))
	if err != nil {
		return nil, err
	}
	realTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		return nil, err
	}
	relative, err := filepath.Rel(realRoot, realTarget)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, ErrCheckAssetOutsideRoot
	}
	file, err := os.Open(realTarget)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, ErrCheckAssetNotRegular
	}
	if limit < 0 || info.Size() > limit {
		return nil, ErrCheckAssetTooLarge
	}
	data := make([]byte, 0, info.Size())
	buffer := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		read, readErr := file.Read(buffer)
		if read > 0 {
			if int64(len(data)+read) > limit {
				return nil, ErrCheckAssetTooLarge
			}
			data = append(data, buffer[:read]...)
		}
		if readErr != nil {
			if errors.Is(readErr, os.ErrClosed) {
				return nil, readErr
			}
			if errors.Is(readErr, io.EOF) {
				return data, nil
			}
			return nil, readErr
		}
	}
}

type checkConfig struct {
	assetReader CheckAssetReader
	assetBytes  int64
	policy      Policy
	policySet   bool
	extensions  []ExtensionRegistration
	target      RenderTarget
}

// WithCheckTarget selects the output projection analyzed by Check.
func WithCheckTarget(target RenderTarget) CheckOption {
	return func(config *checkConfig) error {
		switch target {
		case TargetHTML, TargetSite, TargetPDF, TargetDeck:
			config.target = target
			return nil
		default:
			return fmt.Errorf("check.target_invalid: unsupported target %q", target)
		}
	}
}

// CheckOption configures compatibility analysis.
type CheckOption func(*checkConfig) error

// WithCheckAssetReader enables missing-asset and SVG compatibility checks.
func WithCheckAssetReader(reader CheckAssetReader) CheckOption {
	return func(config *checkConfig) error {
		if reader == nil {
			return errors.New("check.asset_reader_invalid: asset reader is nil")
		}
		config.assetReader = reader
		return nil
	}
}

// WithCheckPolicy evaluates compatibility against the same host capability
// ceiling used for compilation. Document metadata has no capability authority.
func WithCheckPolicy(policy Policy) CheckOption {
	if policy.RawHTML == "" {
		policy.RawHTML = RawHTMLDeny
	}
	frozen := clonePolicy(policy)
	return func(config *checkConfig) error {
		if err := validateRawHTMLMode(frozen.RawHTML); err != nil {
			return err
		}
		if frozen.OutputBytes < MinOutputBytes || frozen.OutputBytes > MaxOutputBytes {
			return policyDiagnostic("policy.output_bytes_invalid", "output byte limit must be between 1 and 67108864")
		}
		config.policy = clonePolicy(frozen)
		config.policySet = true
		return nil
	}
}

// WithCheckExtension enables an extension's read-only fence validation during
// compatibility analysis.
func WithCheckExtension(registration ExtensionRegistration) CheckOption {
	frozen := cloneRegistration(registration)
	return func(config *checkConfig) error {
		candidate := cloneRegistration(frozen)
		if candidate.Check == nil {
			return fmt.Errorf("check.extension_invalid: extension %q has no checker", candidate.Identity.Name)
		}
		if candidate.Identity.Name == "" || candidate.Identity.Version == "" || len(candidate.Fences) == 0 {
			return fmt.Errorf("check.extension_invalid: extension identity and fences are required")
		}
		localFences := make(map[string]struct{}, len(candidate.Fences))
		for _, fence := range candidate.Fences {
			if fence == "" {
				return fmt.Errorf("check.extension_invalid: extension fence is empty")
			}
			if _, exists := localFences[fence]; exists {
				return fmt.Errorf("check.extension_invalid: duplicate fence %q", fence)
			}
			localFences[fence] = struct{}{}
		}
		for _, existing := range config.extensions {
			if existing.Identity.Name == candidate.Identity.Name {
				return fmt.Errorf("check.extension_invalid: duplicate extension name %q", candidate.Identity.Name)
			}
			for _, left := range existing.Fences {
				for _, right := range candidate.Fences {
					if left == right {
						return fmt.Errorf("check.extension_invalid: duplicate fence %q", left)
					}
				}
			}
		}
		config.extensions = append(config.extensions, candidate)
		return nil
	}
}

// Check performs read-only compatibility analysis without rendering. Findings
// are deterministic and carry stable source positions and remediation hints.
func Check(ctx context.Context, source Source, options ...CheckOption) ([]Diagnostic, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	config := checkConfig{}
	config.target = TargetHTML
	for index, option := range options {
		if option == nil {
			return nil, fmt.Errorf("check.option_invalid: nil option at index %d", index)
		}
		if err := option(&config); err != nil {
			return nil, err
		}
	}
	snapshot := source.clone()
	if ctx == nil {
		ctx = context.Background()
	}
	frontmatter, err := parseFrontmatter(snapshot)
	if err != nil {
		return actionableCheckFailure(snapshot, err), nil
	}
	root := newMarkdownParser().Parse(text.NewReader(frontmatter.body))
	effectivePolicy, policyDiagnostics := checkPolicyCompatibility(config, snapshot, frontmatter, root)
	allowRawHTML := config.policySet && len(policyDiagnostics) == 0 && effectivePolicy.RawHTML == RawHTMLSanitized
	extensionChecks := make(map[string]ExtensionCheck)
	for _, registration := range config.extensions {
		for _, fence := range registration.Fences {
			extensionChecks[fence] = registration.Check
		}
	}
	diagnostics := append(checkMetadata(snapshot, frontmatter), policyDiagnostics...)
	locator := checkLocator{source: frontmatter.body}
	lastHeadingLevel := 0
	rawContainers := make(map[goldast.Node]struct{})
	walkErr := goldast.Walk(root, func(node goldast.Node, entering bool) (goldast.WalkStatus, error) {
		if !entering {
			return goldast.WalkContinue, nil
		}
		if err := contextError(ctx); err != nil {
			return goldast.WalkStop, err
		}
		switch value := node.(type) {
		case *goldast.Text:
			if index := bytes.Index(value.Value(frontmatter.body), []byte("<!--")); index >= 0 && !insideCodeSpan(value) {
				offset := frontmatter.bodyOffset + value.Segment.Start + index
				diagnostics = append(diagnostics, checkDiagnostic(snapshot, "source.html_comment_malformed", SeverityError, "/rawHTML", "HTML comment must end with -->", "Close the HTML comment with -->.", offset))
			}
		case *goldast.Heading:
			offset := frontmatter.bodyOffset + segmentAtStart(value.Lines())
			if value.Level > lastHeadingLevel+1 {
				diagnostics = append(diagnostics, checkDiagnostic(snapshot, "check.heading_level_skipped", SeverityWarning, "/heading/level", fmt.Sprintf("heading level jumps from %d to %d", lastHeadingLevel, value.Level), "Use consecutive heading levels without skipping an intermediate level.", offset))
			}
			lastHeadingLevel = value.Level
		case *goldast.HTMLBlock:
			fragment := value.Text(frontmatter.body)
			remaining, commentErr := stripHTMLComments(fragment)
			if commentErr != nil {
				offset := frontmatter.bodyOffset + segmentAtStart(value.Lines())
				diagnostics = append(diagnostics, checkDiagnostic(snapshot, "source.html_comment_malformed", SeverityError, "/rawHTML", commentErr.Error(), "Close the HTML comment with -->.", offset))
				break
			}
			if strings.TrimSpace(string(remaining)) == "" {
				break
			}
			if embed, recognized, embedErr := parseIframeFragment(remaining); recognized {
				offset := frontmatter.bodyOffset + segmentAtStart(value.Lines())
				diagnostics = append(diagnostics, checkIframeDiagnostics(snapshot, config.target, effectivePolicy.Iframe, embed, embedErr, offset)...)
				break
			}
			if allowRawHTML {
				if validationErr := ValidateHTML(string(remaining)); validationErr != nil {
					offset := frontmatter.bodyOffset + segmentAtStart(value.Lines())
					diagnostics = append(diagnostics, checkDiagnostic(snapshot, "policy.html.invalid", SeverityError, "/rawHTML", validationErr.Error(), "Use only elements and attributes accepted by the margo-html-v1 allowlist.", offset))
				}
				break
			}
			if config.policySet {
				break
			}
			offset := frontmatter.bodyOffset + segmentAtStart(value.Lines())
			diagnostics = append(diagnostics, checkDiagnostic(snapshot, "check.raw_html", SeverityError, "/rawHTML", "raw HTML is not accepted by the default CLI policy", "Replace the fragment with Markdown before rendering.", offset))
		case *goldast.RawHTML:
			fragment := value.Segments.Value(frontmatter.body)
			remaining, commentErr := stripHTMLComments(fragment)
			if commentErr != nil {
				offset := frontmatter.bodyOffset + segmentAtStart(value.Segments)
				diagnostics = append(diagnostics, checkDiagnostic(snapshot, "source.html_comment_malformed", SeverityError, "/rawHTML", commentErr.Error(), "Close the HTML comment with -->.", offset))
				break
			}
			if strings.TrimSpace(string(remaining)) == "" {
				break
			}
			if embed, recognized, embedErr := parseIframeFragment(remaining); recognized {
				offset := frontmatter.bodyOffset + segmentAtStart(value.Segments)
				diagnostics = append(diagnostics, checkIframeDiagnostics(snapshot, config.target, effectivePolicy.Iframe, embed, embedErr, offset)...)
				break
			}
			if allowRawHTML {
				if validationErr := ValidateHTML(string(remaining)); validationErr != nil {
					offset := frontmatter.bodyOffset + segmentAtStart(value.Segments)
					diagnostics = append(diagnostics, checkDiagnostic(snapshot, "policy.html.invalid", SeverityError, "/rawHTML", validationErr.Error(), "Use only elements and attributes accepted by the margo-html-v1 allowlist.", offset))
				}
				break
			}
			if config.policySet {
				break
			}
			container := value.Parent()
			if _, exists := rawContainers[container]; exists {
				break
			}
			rawContainers[container] = struct{}{}
			offset := frontmatter.bodyOffset + segmentAtStart(value.Segments)
			diagnostics = append(diagnostics, checkDiagnostic(snapshot, "check.raw_html", SeverityError, "/rawHTML", "raw HTML is not accepted by the default CLI policy", "Replace the fragment with Markdown before rendering.", offset))
		case *goldast.Image:
			offset := frontmatter.bodyOffset + locator.find(value.Destination)
			imageDiagnostics, imageErr := checkImage(ctx, snapshot, &config, frontmatter.body, value, offset)
			if imageErr != nil {
				return goldast.WalkStop, imageErr
			}
			diagnostics = append(diagnostics, imageDiagnostics...)
		case *goldast.Link:
			offset := frontmatter.bodyOffset + locator.find(value.Destination)
			if code, severity, message, hint := checkLinkReference(config.target, string(value.Destination)); code != "" {
				diagnostics = append(diagnostics, checkDiagnostic(snapshot, code, severity, "/link/destination", message, hint, offset))
			}
		case *goldast.FencedCodeBlock:
			language := string(value.Language(frontmatter.body))
			payloadStart := segmentAtStart(value.Lines())
			if checker := extensionChecks[language]; checker != nil {
				node := ExtensionNode{
					Fence: language, Payload: append([]byte(nil), value.Lines().Value(frontmatter.body)...),
					Source: SourcePosition{Source: snapshot.Name, Line: lineAtOffset(snapshot.Content, frontmatter.bodyOffset+payloadStart), Column: 1},
					Target: config.target,
				}
				failure := checker(ctx, node)
				if errors.Is(failure, context.Canceled) || errors.Is(failure, context.DeadlineExceeded) {
					return goldast.WalkStop, failure
				}
				diagnostics = append(diagnostics, extensionCheckDiagnostics(snapshot, node, failure)...)
			}
			if language != "mermaid" {
				break
			}
			payload := value.Lines().Value(frontmatter.body)
			if preflightErr := internalmermaid.Preflight(payload); preflightErr != nil {
				var mermaidDiagnostic *internalmermaid.DiagnosticError
				if errors.As(preflightErr, &mermaidDiagnostic) {
					offset := frontmatter.bodyOffset + payloadStart + mermaidDiagnostic.Offset
					diagnostics = append(diagnostics, checkDiagnostic(snapshot, mermaidDiagnostic.Code, SeverityError, "/mermaid/configuration", mermaidDiagnostic.Message, "Remove the legacy init directive; Margo applies a fixed safe Mermaid configuration.", offset))
				}
			}
		}
		return goldast.WalkContinue, nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	sort.SliceStable(diagnostics, func(left, right int) bool {
		if diagnostics[left].Line != diagnostics[right].Line {
			return diagnostics[left].Line < diagnostics[right].Line
		}
		if diagnostics[left].Column != diagnostics[right].Column {
			return diagnostics[left].Column < diagnostics[right].Column
		}
		return diagnostics[left].Code < diagnostics[right].Code
	})
	return diagnostics, nil
}

func checkIframeDiagnostics(source Source, target RenderTarget, policy *IframePolicy, embed iframeEmbed, failure error, offset int) []Diagnostic {
	if failure == nil {
		failure = authorizeIframe(policy, embed)
	}
	if failure == nil && iframeProjection(policy, target) == ProjectionDeny {
		failure = fmt.Errorf("iframe projection is deny for target %s", target)
	}
	if failure != nil {
		return []Diagnostic{checkDiagnostic(source, "policy.iframe_denied", SeverityError, "/policy/iframe", failure.Error(), "Authorize the exact HTTPS origin and target projection in the trusted host policy, or remove the iframe.", offset)}
	}
	if embed.Title == "" {
		return []Diagnostic{checkDiagnostic(source, "check.iframe_title_missing", SeverityWarning, "/iframe/title", "iframe has no accessible title", "Add a concise title attribute.", offset)}
	}
	return nil
}

func extensionCheckDiagnostics(source Source, node ExtensionNode, failure error) []Diagnostic {
	if failure == nil {
		return nil
	}
	var diagnosticFailure *DiagnosticError
	if errors.As(failure, &diagnosticFailure) && len(diagnosticFailure.Diagnostics) > 0 {
		diagnostics := cloneDiagnostics(diagnosticFailure.Diagnostics)
		for index := range diagnostics {
			if diagnostics[index].Source == "" {
				diagnostics[index].Source = source.Name
			}
			if diagnostics[index].Line <= 0 {
				diagnostics[index].Line = node.Source.Line
			}
			if diagnostics[index].Column <= 0 {
				diagnostics[index].Column = node.Source.Column
			}
		}
		return diagnostics
	}
	return []Diagnostic{{
		Code: "check.extension_invalid", Severity: SeverityError, Source: source.Name,
		Line: node.Source.Line, Column: node.Source.Column, Pointer: "/extension",
		Message: failure.Error(), Hint: "Correct the extension payload and run margo check again.",
	}}
}

func checkPolicyCompatibility(config checkConfig, source Source, frontmatter frontmatterResult, root goldast.Node) (EffectivePolicy, []Diagnostic) {
	if !config.policySet {
		return EffectivePolicy{}, nil
	}
	compilerConfig := newCompilerConfig()
	compilerConfig.values["hostPolicy"] = config.policy
	normalized := sourceNormalization{
		parsed: normalizedMarkdown{frontmatter: frontmatter, root: root}, sourceBytes: int64(len(source.Content)), skipRemoteImages: true,
	}
	effective, err := defaultEvaluatePolicy(compilerConfig, normalized)
	if err == nil {
		return effective, nil
	}
	var diagnosticFailure *DiagnosticError
	if !errors.As(err, &diagnosticFailure) || len(diagnosticFailure.Diagnostics) == 0 {
		return EffectivePolicy{}, []Diagnostic{checkDiagnostic(source, "check.policy_invalid", SeverityError, "/policy/rawHTML", err.Error(), "Update the trusted host policy.", 0)}
	}
	diagnostics := cloneDiagnostics(diagnosticFailure.Diagnostics)
	for index := range diagnostics {
		diagnostics[index].Source = source.Name
		diagnostics[index].Line = 1
		diagnostics[index].Column = 1
		if diagnostics[index].Pointer == "" {
			diagnostics[index].Pointer = "/policy/rawHTML"
		}
		diagnostics[index].Hint = "Update the trusted host policy or replace raw HTML with Markdown."
	}
	return EffectivePolicy{}, diagnostics
}

func actionableCheckFailure(source Source, failure error) []Diagnostic {
	diagnosticFailure := unwrapDiagnostic(failure)
	if diagnosticFailure == nil || len(diagnosticFailure.Diagnostics) == 0 {
		return []Diagnostic{checkDiagnostic(source, "check.parse_failed", SeverityError, "/document", failure.Error(), "Correct the Markdown syntax and run margo check again.", 0)}
	}
	diagnostics := cloneDiagnostics(diagnosticFailure.Diagnostics)
	for index := range diagnostics {
		diagnostics[index].Source = source.Name
		if diagnostics[index].Line <= 0 {
			diagnostics[index].Line = 1
		}
		if diagnostics[index].Column <= 0 {
			diagnostics[index].Column = 1
		}
		if diagnostics[index].Pointer == "" {
			diagnostics[index].Pointer = "/frontmatter"
		}
		diagnostics[index].Hint = "Correct the frontmatter field and run margo check again."
	}
	return diagnostics
}

func checkMetadata(source Source, frontmatter frontmatterResult) []Diagnostic {
	metadata, err := normalizeSourceMetadata(source, frontmatter.values)
	if err != nil {
		diagnostics := actionableCheckFailure(source, err)
		for index := range diagnostics {
			diagnostics[index].Line, diagnostics[index].Column = frontmatterPointerPosition(source.Content, diagnostics[index].Pointer)
			diagnostics[index].Hint = "Correct this frontmatter field to the documented type and format."
		}
		return diagnostics
	}
	fields := []struct {
		pointer string
		value   string
		limit   int
	}{
		{pointer: "/title", value: metadata.Title, limit: 256},
		{pointer: "/description", value: metadata.Description, limit: 512},
		{pointer: "/language", value: metadata.Language, limit: 64},
		{pointer: "/slug", value: metadata.Slug, limit: 128},
	}
	diagnostics := make([]Diagnostic, 0)
	if metadata.Language == "" {
		diagnostics = append(diagnostics, Diagnostic{Code: "check.language_missing", Severity: SeverityWarning, Source: source.Name, Line: 1, Column: 1, Pointer: "/language", Message: "document language is not declared", Hint: "Add a BCP 47 language tag such as language: en or language: pt-BR to frontmatter, or pass --lang when rendering."})
	}
	for _, field := range fields {
		if len([]byte(normalizeHTMLText(field.value))) <= field.limit {
			continue
		}
		line, column := frontmatterPointerPosition(source.Content, field.pointer)
		diagnostics = append(diagnostics, Diagnostic{Code: "source.metadata_invalid", Severity: SeverityError, Source: source.Name, Line: line, Column: column, Pointer: field.pointer, Message: fmt.Sprintf("%s exceeds its %d-byte HTML limit", strings.TrimPrefix(field.pointer, "/"), field.limit), Hint: fmt.Sprintf("Shorten %s to at most %d UTF-8 bytes.", field.pointer, field.limit)})
	}
	lists := []struct {
		pointer string
		values  []string
	}{
		{pointer: "/authors", values: metadata.Authors},
		{pointer: "/tags", values: metadata.Tags},
	}
	for _, list := range lists {
		if len(list.values) > 64 {
			line, column := frontmatterPointerPosition(source.Content, list.pointer)
			diagnostics = append(diagnostics, Diagnostic{Code: "source.metadata_invalid", Severity: SeverityError, Source: source.Name, Line: line, Column: column, Pointer: list.pointer, Message: fmt.Sprintf("%s exceeds its 64-item HTML limit", strings.TrimPrefix(list.pointer, "/")), Hint: "Reduce this metadata list to at most 64 entries."})
			continue
		}
		for index, value := range list.values {
			normalized := normalizeHTMLText(value)
			if normalized != "" && len([]byte(normalized)) <= 128 {
				continue
			}
			pointer := fmt.Sprintf("%s/%d", list.pointer, index)
			line, column := frontmatterPointerPosition(source.Content, pointer)
			diagnostics = append(diagnostics, Diagnostic{Code: "source.metadata_invalid", Severity: SeverityError, Source: source.Name, Line: line, Column: column, Pointer: pointer, Message: fmt.Sprintf("%s entry is empty or exceeds its 128-byte HTML limit", strings.TrimPrefix(list.pointer, "/")), Hint: "Use a non-empty value of at most 128 UTF-8 bytes."})
		}
	}
	return diagnostics
}

func checkImage(ctx context.Context, source Source, config *checkConfig, body []byte, image *goldast.Image, offset int) ([]Diagnostic, error) {
	destination := strings.TrimSpace(string(image.Destination))
	result := make([]Diagnostic, 0, 2)
	if strings.TrimSpace(plainInlineText(image, body)) == "" {
		result = append(result, checkDiagnostic(source, "check.image_alt_empty", SeverityWarning, "/image/alt", "image alternative text is empty", "Add concise alternative text between the image brackets.", offset))
	}
	parsed, err := url.Parse(destination)
	if err == nil && strings.EqualFold(parsed.Scheme, "data") {
		data, _, decodeErr := staticimage.ValidateDataURL(ctx, destination, MaxDocumentBytes-config.assetBytes)
		if decodeErr != nil {
			if errors.Is(decodeErr, staticimage.ErrDataTooLarge) {
				return append(result, checkDiagnostic(source, "check.asset_too_large", SeverityError, "/image/destination", "data image exceeds the remaining document asset limit", "Reduce or remove the embedded image.", offset)), nil
			}
			return append(result, checkImageValidationDiagnostic(source, "data image", decodeErr, offset)), nil
		}
		config.assetBytes += int64(len(data))
		contentDiagnostics, contentErr := checkImageContent(ctx, source, destination, data, offset)
		return append(result, contentDiagnostics...), contentErr
	}
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || strings.HasPrefix(destination, "//") || filepath.IsAbs(parsed.Path) {
		return append(result, checkDiagnostic(source, "check.asset_remote", SeverityError, "/image/destination", fmt.Sprintf("image source %q is not a local relative asset", destination), "Download the image and reference a local path relative to this Markdown file.", offset)), nil
	}
	if parsed.Path == "" || config.assetReader == nil {
		return result, nil
	}
	data, readErr := config.assetReader.ReadAsset(ctx, source.BaseURL, parsed.Path, MaxDocumentBytes-config.assetBytes)
	if readErr != nil {
		if errors.Is(readErr, context.Canceled) || errors.Is(readErr, context.DeadlineExceeded) {
			return nil, readErr
		}
		code, message, hint := "check.asset_missing", fmt.Sprintf("local image %q cannot be read", destination), "Add the asset at the referenced path or correct the image destination."
		if errors.Is(readErr, ErrCheckAssetOutsideRoot) {
			code, message, hint = "check.asset_outside_root", fmt.Sprintf("local image %q resolves outside its source root", destination), "Move the asset under the Markdown file directory and use a contained relative path."
		} else if errors.Is(readErr, ErrCheckAssetTooLarge) {
			code, message, hint = "check.asset_too_large", fmt.Sprintf("local image %q exceeds the remaining document asset limit", destination), "Reduce the image size or remove other embedded images."
		}
		return append(result, checkDiagnostic(source, code, SeverityError, "/image/destination", message, hint, offset)), nil
	}
	config.assetBytes += int64(len(data))
	contentDiagnostics, contentErr := checkImageContent(ctx, source, destination, data, offset)
	return append(result, contentDiagnostics...), contentErr
}

func checkImageContent(ctx context.Context, source Source, destination string, data []byte, offset int) ([]Diagnostic, error) {
	_, err := staticimage.DetectContext(ctx, data)
	if err == nil {
		return nil, nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	}
	var imageErr *staticimage.Error
	if errors.As(err, &imageErr) && (imageErr.Kind == staticimage.SVGInvalid || imageErr.Kind == staticimage.SVGActive) {
		return []Diagnostic{checkImageValidationDiagnostic(source, fmt.Sprintf("SVG %q", destination), err, offset)}, nil
	}
	return []Diagnostic{checkDiagnostic(source, "check.asset_incompatible", SeverityError, "/image/destination", fmt.Sprintf("image %q is incompatible: %v", destination, err), "Use a supported PNG, JPEG, GIF, WebP, or static SVG image.", offset)}, nil
}

func checkImageValidationDiagnostic(source Source, subject string, failure error, offset int) Diagnostic {
	var imageErr *staticimage.Error
	if errors.As(failure, &imageErr) && (imageErr.Kind == staticimage.SVGInvalid || imageErr.Kind == staticimage.SVGActive) {
		return checkDiagnostic(source, "check.svg_incompatible", SeverityError, "/image/destination", fmt.Sprintf("%s is incompatible: %v", subject, failure), "Use a well-formed static SVG without scripts, active elements, event handlers, or external references.", offset)
	}
	return checkDiagnostic(source, "check.asset_incompatible", SeverityError, "/image/destination", fmt.Sprintf("%s is incompatible: %v", subject, failure), "Use a supported PNG, JPEG, GIF, WebP, or static SVG image.", offset)
}

func checkLinkReference(target RenderTarget, value string) (string, Severity, string, string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "check.link_destination_empty", SeverityWarning, "link destination is empty", "Add a destination URL or remove the link markup."
	}
	if strings.HasPrefix(trimmed, "#") {
		return "", "", "", ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || strings.HasPrefix(trimmed, "//") || parsed.Host != "" && parsed.Scheme == "" {
		return "check.link_unsupported", SeverityError, fmt.Sprintf("link %q is not a supported URL", value), "Use a relative path, fragment, or an http, https, mailto, or tel URL."
	}
	if parsed.Scheme == "" {
		if target == TargetSite {
			// Relative Markdown links are the input contract for multi-page
			// sites. The site builder resolves and validates them after it has
			// indexed every source, so a single-document check cannot add useful
			// information here. In particular, do not suggest PDF-only link
			// policies for a target that has no such flags.
			return "", "", "", ""
		}
		return "check.link_relative", SeverityWarning, fmt.Sprintf("relative link %q requires an output policy", value), "Choose a base URL or an explicit strip, error, or keep policy for the target format."
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "mailto", "tel":
		return "", "", "", ""
	default:
		return "check.link_unsupported", SeverityError, fmt.Sprintf("link %q uses unsupported scheme %q", value, parsed.Scheme), "Use a relative path, fragment, or an http, https, mailto, or tel URL."
	}
}

func checkAssetPath(root, name string) (string, bool) {
	if root == "" {
		return "", false
	}
	cleanRoot := filepath.Clean(root)
	target := filepath.Clean(filepath.Join(cleanRoot, filepath.FromSlash(name)))
	relative, err := filepath.Rel(cleanRoot, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}
	return target, true
}

func checkDiagnostic(source Source, code string, severity Severity, pointer, message, hint string, offset int) Diagnostic {
	line, column := sourcePositionAtOffset(source.Content, offset)
	return Diagnostic{Code: code, Severity: severity, Source: source.Name, Line: line, Column: column, Pointer: pointer, Message: message, Hint: hint}
}

func sourcePositionAtOffset(source []byte, offset int) (int, int) {
	if offset < 0 || offset > len(source) {
		offset = 0
	}
	line := lineAtOffset(source, offset)
	lineStart := bytes.LastIndexByte(source[:offset], '\n') + 1
	return line, offset - lineStart + 1
}

type checkLocator struct {
	source []byte
	next   int
}

func (locator *checkLocator) find(value []byte) int {
	if len(value) == 0 {
		if index := bytes.Index(locator.source[locator.next:], []byte("](")); index >= 0 {
			offset := locator.next + index + 2
			locator.next = offset
			return offset
		}
		return locator.next
	}
	for _, prefix := range [][]byte{append([]byte("]("), value...), append([]byte("](<"), value...)} {
		if index := bytes.Index(locator.source[locator.next:], prefix); index >= 0 {
			offset := locator.next + index + len(prefix) - len(value)
			locator.next = offset + len(value)
			return offset
		}
	}
	if index := bytes.Index(locator.source[locator.next:], value); index >= 0 {
		offset := locator.next + index
		locator.next = offset + len(value)
		return offset
	}
	if index := bytes.Index(locator.source, value); index >= 0 {
		return index
	}
	return locator.next
}

func frontmatterPointerPosition(source []byte, pointer string) (int, int) {
	lines := bytes.SplitAfter(source, []byte("\n"))
	if len(lines) < 2 || strings.TrimSpace(string(lines[0])) != "---" {
		return 1, 1
	}
	closeIndex := -1
	for index := 1; index < len(lines); index++ {
		marker := strings.TrimSpace(string(lines[index]))
		if marker == "---" || marker == "..." {
			closeIndex = index
			break
		}
	}
	if closeIndex < 0 {
		return 1, 1
	}
	var document yaml.Node
	if err := yaml.Unmarshal(bytes.Join(lines[1:closeIndex], nil), &document); err != nil || len(document.Content) == 0 {
		return 1, 1
	}
	node := document.Content[0]
	segments := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	for segmentIndex, segment := range segments {
		if node.Kind == yaml.MappingNode {
			found := false
			for index := 0; index+1 < len(node.Content); index += 2 {
				if node.Content[index].Value == segment {
					if segmentIndex == len(segments)-1 {
						return node.Content[index].Line + 1, node.Content[index].Column
					}
					node = node.Content[index+1]
					found = true
					break
				}
			}
			if !found {
				return 1, 1
			}
			continue
		}
		if node.Kind == yaml.SequenceNode {
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(node.Content) {
				return 1, 1
			}
			node = node.Content[index]
			if segmentIndex == len(segments)-1 {
				return node.Line + 1, node.Column
			}
			continue
		}
		return node.Line + 1, node.Column
	}
	return node.Line + 1, node.Column
}
