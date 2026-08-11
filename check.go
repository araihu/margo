package margo

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	internalmermaid "github.com/araihu/margo/internal/mermaid"
	goldast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

// CheckAssetReader supplies local assets to Check without coupling library
// users to the host filesystem.
type CheckAssetReader interface {
	ReadFile(string) ([]byte, error)
}

type checkConfig struct {
	assetReader CheckAssetReader
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

// Check performs read-only compatibility analysis without rendering. Findings
// are deterministic and carry stable source positions and remediation hints.
func Check(ctx context.Context, source Source, options ...CheckOption) ([]Diagnostic, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	config := checkConfig{}
	for index, option := range options {
		if option == nil {
			return nil, fmt.Errorf("check.option_invalid: nil option at index %d", index)
		}
		if err := option(&config); err != nil {
			return nil, err
		}
	}
	snapshot := source.clone()
	frontmatter, err := parseFrontmatter(snapshot)
	if err != nil {
		return actionableCheckFailure(snapshot, err), nil
	}
	root := newMarkdownParser().Parse(text.NewReader(frontmatter.body))
	diagnostics := checkMetadata(snapshot, frontmatter)
	locator := checkLocator{source: frontmatter.body}
	walkErr := goldast.Walk(root, func(node goldast.Node, entering bool) (goldast.WalkStatus, error) {
		if !entering {
			return goldast.WalkContinue, nil
		}
		if err := contextError(ctx); err != nil {
			return goldast.WalkStop, err
		}
		switch value := node.(type) {
		case *goldast.HTMLBlock:
			offset := frontmatter.bodyOffset + segmentAtStart(value.Lines())
			diagnostics = append(diagnostics, checkDiagnostic(snapshot, "check.raw_html", SeverityError, "/rawHTML", "raw HTML is not accepted by the default CLI policy", "Replace the fragment with Markdown before rendering.", offset))
		case *goldast.RawHTML:
			offset := frontmatter.bodyOffset + segmentAtStart(value.Segments)
			diagnostics = append(diagnostics, checkDiagnostic(snapshot, "check.raw_html", SeverityError, "/rawHTML", "raw HTML is not accepted by the default CLI policy", "Replace the fragment with Markdown before rendering.", offset))
		case *goldast.Image:
			offset := frontmatter.bodyOffset + locator.find(value.Destination)
			diagnostics = append(diagnostics, checkImage(snapshot, config, frontmatter.body, value, offset)...)
		case *goldast.Link:
			offset := frontmatter.bodyOffset + locator.find(value.Destination)
			if checkRelativeReference(string(value.Destination)) {
				diagnostics = append(diagnostics, checkDiagnostic(snapshot, "check.link_relative", SeverityWarning, "/link/destination", fmt.Sprintf("relative link %q requires an output policy", value.Destination), "Choose a base URL or an explicit strip, error, or keep policy for the target format.", offset))
			}
		case *goldast.FencedCodeBlock:
			if string(value.Language(frontmatter.body)) != "mermaid" {
				break
			}
			payloadStart := segmentAtStart(value.Lines())
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
	return dedupeCheckDiagnostics(diagnostics), nil
}

func dedupeCheckDiagnostics(diagnostics []Diagnostic) []Diagnostic {
	result := make([]Diagnostic, 0, len(diagnostics))
	seen := make(map[string]struct{}, len(diagnostics))
	for _, diagnostic := range diagnostics {
		key := fmt.Sprintf("%s\x00%s\x00%d\x00%s", diagnostic.Source, diagnostic.Code, diagnostic.Line, diagnostic.Pointer)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, diagnostic)
	}
	return result
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
	for _, field := range fields {
		if len([]byte(normalizeHTMLText(field.value))) <= field.limit {
			continue
		}
		line, column := frontmatterPointerPosition(source.Content, field.pointer)
		diagnostics := Diagnostic{Code: "source.metadata_invalid", Severity: SeverityError, Source: source.Name, Line: line, Column: column, Pointer: field.pointer, Message: fmt.Sprintf("%s exceeds its %d-byte HTML limit", strings.TrimPrefix(field.pointer, "/"), field.limit), Hint: fmt.Sprintf("Shorten %s to at most %d UTF-8 bytes.", field.pointer, field.limit)}
		return []Diagnostic{diagnostics}
	}
	return nil
}

func checkImage(source Source, config checkConfig, body []byte, image *goldast.Image, offset int) []Diagnostic {
	destination := strings.TrimSpace(string(image.Destination))
	result := make([]Diagnostic, 0, 2)
	if strings.TrimSpace(plainInlineText(image, body)) == "" {
		result = append(result, checkDiagnostic(source, "check.image_alt_empty", SeverityWarning, "/image/alt", "image alternative text is empty", "Add concise alternative text between the image brackets.", offset))
	}
	parsed, err := url.Parse(destination)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || strings.HasPrefix(destination, "//") || filepath.IsAbs(parsed.Path) {
		if err == nil && strings.EqualFold(parsed.Scheme, "data") {
			return result
		}
		return append(result, checkDiagnostic(source, "check.asset_remote", SeverityError, "/image/destination", fmt.Sprintf("image source %q is not a local relative asset", destination), "Download the image and reference a local path relative to this Markdown file.", offset))
	}
	if parsed.Path == "" || config.assetReader == nil {
		return result
	}
	target, safe := checkAssetPath(source.BaseURL, parsed.Path)
	if !safe {
		return append(result, checkDiagnostic(source, "check.asset_missing", SeverityError, "/image/destination", fmt.Sprintf("image source %q escapes or lacks an input root", destination), "Move the asset under the Markdown file directory and use a contained relative path.", offset))
	}
	data, readErr := config.assetReader.ReadFile(target)
	if readErr != nil {
		return append(result, checkDiagnostic(source, "check.asset_missing", SeverityError, "/image/destination", fmt.Sprintf("local image %q cannot be read", destination), "Add the asset at the referenced path or correct the image destination.", offset))
	}
	if strings.EqualFold(filepath.Ext(parsed.Path), ".svg") {
		if svgErr := validateCheckStaticSVG(data); svgErr != nil {
			return append(result, checkDiagnostic(source, "check.svg_incompatible", SeverityError, "/image/destination", fmt.Sprintf("SVG %q is incompatible: %v", destination, svgErr), "Use a well-formed static SVG without scripts, active elements, event handlers, or external references.", offset))
		}
	}
	return result
}

func checkRelativeReference(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return false
	}
	parsed, err := url.Parse(trimmed)
	return err == nil && parsed.Scheme == "" && parsed.Host == "" && !strings.HasPrefix(trimmed, "//")
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
		return locator.next
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
	for _, segment := range segments {
		if node.Kind == yaml.MappingNode {
			found := false
			for index := 0; index+1 < len(node.Content); index += 2 {
				if node.Content[index].Value == segment {
					node = node.Content[index]
					found = true
					break
				}
			}
			if !found {
				return 1, 1
			}
			continue
		}
		return node.Line + 1, node.Column
	}
	return node.Line + 1, node.Column
}

func validateCheckStaticSVG(data []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	rootSeen := false
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF && rootSeen {
				return nil
			}
			if err == io.EOF {
				return errors.New("SVG root element is missing")
			}
			return err
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		name := strings.ToLower(start.Name.Local)
		if !rootSeen {
			rootSeen = true
			if name != "svg" {
				return fmt.Errorf("XML root element %s is not svg", name)
			}
		}
		switch name {
		case "script", "foreignobject", "iframe", "object", "embed":
			return fmt.Errorf("element %s is forbidden", name)
		}
		for _, attribute := range start.Attr {
			attributeName := strings.ToLower(attribute.Name.Local)
			value := strings.TrimSpace(attribute.Value)
			lowerValue := strings.ToLower(value)
			if strings.HasPrefix(attributeName, "on") || ((attributeName == "href" || attributeName == "src") && value != "" && !strings.HasPrefix(value, "#") && !strings.HasPrefix(lowerValue, "data:")) {
				return fmt.Errorf("active attribute %s is forbidden", attributeName)
			}
			if attributeName == "style" && (strings.Contains(lowerValue, "url(http:") || strings.Contains(lowerValue, "url(https:")) {
				return errors.New("external style URL is forbidden")
			}
		}
	}
}
