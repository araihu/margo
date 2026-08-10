package deck

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/araihu/margo"
	"gopkg.in/yaml.v3"
)

type fenceState struct {
	marker byte
	length int
	line   int
}

func Parse(name string, source []byte) (*Document, error) {
	snapshot := append([]byte(nil), source...)
	lines := bytes.SplitAfter(snapshot, []byte("\n"))
	metadata, bodyStart, err := parseMetadata(name, lines)
	if err != nil {
		return nil, err
	}
	slides, err := parseSlides(name, lines[bodyStart:], bodyStart+1)
	if err != nil {
		return nil, err
	}
	return &Document{name: name, metadata: metadata, slides: slides}, nil
}

func parseMetadata(name string, lines [][]byte) (Metadata, int, error) {
	if len(lines) == 0 || lineContent(lines[0]) != "---" {
		return Metadata{}, 0, nil
	}
	closeIndex := -1
	for index := 1; index < len(lines); index++ {
		if lineContent(lines[index]) == "---" {
			closeIndex = index
			break
		}
	}
	if closeIndex < 0 {
		return Metadata{}, 0, deckError("deck.frontmatter_invalid", name, 1, "opening frontmatter is not closed")
	}
	encoded := bytes.Join(lines[1:closeIndex], nil)
	var node yaml.Node
	if err := yaml.Unmarshal(encoded, &node); err != nil {
		return Metadata{}, 0, deckError("deck.frontmatter_invalid", name, 1, err.Error())
	}
	if len(node.Content) > 0 && node.Content[0].Kind != yaml.MappingNode {
		return Metadata{}, 0, deckError("deck.frontmatter_invalid", name, 1, "frontmatter must be a mapping")
	}
	var metadata Metadata
	if err := yaml.Unmarshal(encoded, &metadata); err != nil {
		return Metadata{}, 0, deckError("deck.frontmatter_invalid", name, 1, err.Error())
	}
	return metadata, closeIndex + 1, nil
}

func parseSlides(name string, lines [][]byte, firstLine int) ([]Slide, error) {
	var slides []Slide
	var current bytes.Buffer
	var fence *fenceState
	for index, line := range lines {
		lineNumber := firstLine + index
		if fence != nil {
			if closesFence(line, *fence) {
				fence = nil
			}
			_, _ = current.Write(line)
			continue
		}
		if marker, length, ok := opensFence(line); ok {
			fence = &fenceState{marker: marker, length: length, line: lineNumber}
			_, _ = current.Write(line)
			continue
		}
		if lineContent(line) == "---" {
			var err error
			slides, err = appendSlide(slides, current.Bytes(), name, lineNumber)
			if err != nil {
				return nil, err
			}
			current.Reset()
			continue
		}
		_, _ = current.Write(line)
	}
	if fence != nil {
		return nil, deckError("deck.fence_unclosed", name, fence.line, "fenced code block is not closed")
	}
	return appendSlide(slides, current.Bytes(), name, firstLine+len(lines))
}

func appendSlide(slides []Slide, markdown []byte, name string, line int) ([]Slide, error) {
	if len(bytes.TrimSpace(markdown)) == 0 {
		return nil, deckError("deck.slide_empty", name, line, "slides must contain Markdown content")
	}
	ordinal := len(slides) + 1
	return append(slides, Slide{
		ordinal:  ordinal,
		id:       fmt.Sprintf("slide-%04d", ordinal),
		markdown: append([]byte(nil), markdown...),
	}), nil
}

func lineContent(line []byte) string {
	return strings.TrimSuffix(strings.TrimSuffix(string(line), "\n"), "\r")
}

func opensFence(line []byte) (byte, int, bool) {
	content := lineContent(line)
	indent := 0
	for indent < len(content) && indent < 4 && content[indent] == ' ' {
		indent++
	}
	if indent > 3 || indent == len(content) {
		return 0, 0, false
	}
	marker := content[indent]
	if marker != '`' && marker != '~' {
		return 0, 0, false
	}
	length := markerRun(content[indent:], marker)
	if length < 3 {
		return 0, 0, false
	}
	if marker == '`' && strings.Contains(content[indent+length:], "`") {
		return 0, 0, false
	}
	return marker, length, true
}

func closesFence(line []byte, fence fenceState) bool {
	content := lineContent(line)
	indent := 0
	for indent < len(content) && indent < 4 && content[indent] == ' ' {
		indent++
	}
	if indent > 3 || indent == len(content) || content[indent] != fence.marker {
		return false
	}
	length := markerRun(content[indent:], fence.marker)
	return length >= fence.length && strings.TrimSpace(content[indent+length:]) == ""
}

func markerRun(value string, marker byte) int {
	length := 0
	for length < len(value) && value[length] == marker {
		length++
	}
	return length
}

func deckError(code, source string, line int, message string) error {
	return &margo.DiagnosticError{Diagnostics: []margo.Diagnostic{{
		Code:     code,
		Severity: margo.SeverityError,
		Source:   source,
		Line:     line,
		Column:   1,
		Message:  message,
	}}}
}
