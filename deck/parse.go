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
	metadata, directives, bodyStart, err := parseMetadata(name, lines)
	if err != nil {
		return nil, err
	}
	if metadata.Marp != nil && !*metadata.Marp {
		return nil, deckError("deck.activation_conflict", name, 1, "marp: false contradicts explicit deck parsing")
	}
	if err := scanGlobalDirectives(name, lines[bodyStart:], bodyStart+1, &directives); err != nil {
		return nil, err
	}
	slides, err := parseSlides(name, lines[bodyStart:], bodyStart+1, directives)
	if err != nil {
		return nil, err
	}
	if err := validateDeckReferences(name, slides); err != nil {
		return nil, err
	}
	return &Document{name: name, metadata: metadata, directives: directives, slides: slides}, nil
}

// Detect reports whether opening frontmatter explicitly opts into deck
// routing with `marp: true`. It does not parse or render the document body.
func Detect(name string, source []byte) (bool, error) {
	snapshot := append([]byte(nil), source...)
	lines := bytes.SplitAfter(snapshot, []byte("\n"))
	metadata, _, _, err := parseMetadata(name, lines)
	if err != nil {
		return false, err
	}
	return metadata.Marp != nil && *metadata.Marp, nil
}

func parseMetadata(name string, lines [][]byte) (Metadata, DirectiveState, int, error) {
	directives := defaultDirectiveState()
	if len(lines) == 0 || lineContent(lines[0]) != "---" {
		return Metadata{}, directives, 0, nil
	}
	closeIndex := -1
	for index := 1; index < len(lines); index++ {
		if lineContent(lines[index]) == "---" {
			closeIndex = index
			break
		}
	}
	if closeIndex < 0 {
		return Metadata{}, directives, 0, deckError("deck.frontmatter_invalid", name, 1, "opening frontmatter is not closed")
	}
	encoded := bytes.Join(lines[1:closeIndex], nil)
	var node yaml.Node
	if err := yaml.Unmarshal(encoded, &node); err != nil {
		return Metadata{}, directives, 0, deckError("deck.frontmatter_invalid", name, 1, err.Error())
	}
	if len(node.Content) > 0 && node.Content[0].Kind != yaml.MappingNode {
		return Metadata{}, directives, 0, deckError("deck.frontmatter_invalid", name, 1, "frontmatter must be a mapping")
	}
	var metadata Metadata
	if err := yaml.Unmarshal(encoded, &metadata); err != nil {
		return Metadata{}, directives, 0, deckError("deck.frontmatter_invalid", name, 1, err.Error())
	}
	if len(node.Content) > 0 {
		mapping := node.Content[0]
		for index := 0; index+1 < len(mapping.Content); index += 2 {
			key := mapping.Content[index].Value
			if key == "title" || key == "description" {
				continue
			}
			if key == "marp" {
				value := mapping.Content[index+1]
				if value.Kind != yaml.ScalarNode || value.Tag != "!!bool" {
					return Metadata{}, directives, 0, deckError("deck.frontmatter_invalid", name, 1, "marp must be boolean")
				}
				active := value.Value == "true"
				metadata.Marp = &active
				continue
			}
			if strings.HasPrefix(key, "$") {
				return Metadata{}, directives, 0, deckError("deck.directive_invalid", name, 1, "legacy $ directives are not supported; use the unprefixed form")
			}
			if !isDeckDirective(key) {
				// Existing Margo frontmatter namespaces (for example margo.page)
				// remain host metadata. Deck-owned keys are validated below while
				// unrelated frontmatter is preserved for the CLI projection.
				continue
			}
			event := directiveEvent{name: key, node: mapping.Content[index+1], line: 1}
			if err := validateDirectiveNode(name, 1, key, event.node); err != nil {
				return Metadata{}, directives, 0, err
			}
			if err := applyDirectiveEvent(&directives, event); err != nil {
				return Metadata{}, directives, 0, err
			}
		}
	}
	return metadata, directives, closeIndex + 1, nil
}

func scanGlobalDirectives(name string, lines [][]byte, firstLine int, directives *DirectiveState) error {
	var fence *fenceState
	for index, line := range lines {
		lineNumber := firstLine + index
		if fence != nil {
			if closesFence(line, *fence) {
				fence = nil
			}
			continue
		}
		if marker, length, ok := opensFence(line); ok {
			fence = &fenceState{marker: marker, length: length, line: lineNumber}
			continue
		}
		comment, ok := htmlComment(line)
		if !ok {
			continue
		}
		kind, events, _, err := parseDirectiveComment(name, lineNumber, comment)
		if err != nil {
			return err
		}
		if kind != directiveCommentMapping {
			continue
		}
		for _, event := range events {
			if _, global := globalDirectiveNames[event.name]; !global {
				continue
			}
			if event.spot {
				return deckError("deck.directive_invalid", name, event.line, "global directives cannot use the spot prefix")
			}
			if err := applyDirectiveEvent(directives, event); err != nil {
				return err
			}
		}
	}
	if fence != nil {
		return deckError("deck.fence_unclosed", name, fence.line, "fenced code block is not closed")
	}
	return nil
}

func parseSlides(name string, lines [][]byte, firstLine int, global DirectiveState) ([]Slide, error) {
	var slides []Slide
	builder := newLayoutBuilder()
	inherited := cloneDirectiveState(global)
	if err := builder.setComposition(inherited.Composition, firstLine); err != nil {
		return nil, deckError("deck.composition_conflict", name, firstLine, err.Error())
	}
	var spot []directiveEvent
	var notes []string
	var fence *fenceState
	previousLine := ""
	for index, line := range lines {
		lineNumber := firstLine + index
		if fence != nil {
			builder.write(line)
			if closesFence(line, *fence) {
				fence = nil
			}
			previousLine = lineContent(line)
			continue
		}
		if marker, length, ok := opensFence(line); ok {
			fence = &fenceState{marker: marker, length: length, line: lineNumber}
			builder.write(line)
			previousLine = lineContent(line)
			continue
		}
		if marker, handled, markerErr := parseStructuralComment(line); handled {
			if markerErr != nil {
				return nil, deckError("deck.layout_invalid", name, lineNumber, markerErr.Error())
			}
			var err error
			switch marker.kind {
			case "layout":
				if !validLayoutClass(marker.value) || !isStructuralClass(marker.value) {
					return nil, deckError("deck.layout_invalid", name, lineNumber, "layout marker names an unknown structural class")
				}
				if builder.composition != "" {
					spec, resolveErr := ResolveComposition(builder.composition)
					if resolveErr != nil || !isStructuralClass(spec.LayoutClass) || spec.LayoutClass != marker.value {
						return nil, deckError("deck.composition_conflict", name, lineNumber, "layout marker does not match composition")
					}
				}
				err = builder.start(marker.value, lineNumber)
			case "slot":
				if builder.composition != "" {
					spec, resolveErr := ResolveComposition(builder.composition)
					if resolveErr != nil || !isStructuralClass(spec.LayoutClass) {
						return nil, deckError("deck.composition_slot_invalid", name, lineNumber, "body composition cannot contain structural slots")
					}
				}
				err = builder.slot(marker.value, lineNumber)
			case "end":
				err = builder.end()
			}
			if err != nil {
				code := "deck.layout_invalid"
				if builder.composition != "" {
					if marker.kind == "slot" || strings.Contains(err.Error(), "empty") {
						code = "deck.composition_slot_invalid"
					}
				} else if marker.kind == "slot" && (strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "empty")) {
					code = "deck.slot_invalid"
				}
				return nil, deckError(code, name, lineNumber, err.Error())
			}
			previousLine = lineContent(line)
			continue
		}
		if comment, ok := htmlComment(line); ok {
			kind, events, note, err := parseDirectiveComment(name, lineNumber, comment)
			if err != nil {
				return nil, err
			}
			switch kind {
			case directiveCommentNote:
				if note != "" {
					notes = append(notes, note)
				}
			case directiveCommentMapping:
				for _, event := range events {
					if _, global := globalDirectiveNames[event.name]; global {
						continue
					}
					if event.spot {
						spot = append(spot, event)
						if event.name == "composition" {
							if err := builder.setComposition(compositionName(event), event.line); err != nil {
								return nil, deckError("deck.composition_conflict", name, event.line, err.Error())
							}
						}
						continue
					}
					if err := applyDirectiveEvent(&inherited, event); err != nil {
						return nil, err
					}
					if event.name == "composition" {
						if err := builder.setComposition(inherited.Composition, event.line); err != nil {
							return nil, deckError("deck.composition_conflict", name, event.line, err.Error())
						}
					}
				}
			}
			previousLine = lineContent(line)
			continue
		}
		if shouldSplitHeading(line, global.HeadingDivider) && hasLayoutContent(builder) {
			var err error
			slides, err = appendScannedSlide(slides, builder, name, lineNumber, global, inherited, spot, notes)
			if err != nil {
				return nil, err
			}
			builder = newLayoutBuilder()
			if err := builder.setComposition(inherited.Composition, lineNumber); err != nil {
				return nil, deckError("deck.composition_conflict", name, lineNumber, err.Error())
			}
			spot = nil
			notes = nil
			previousLine = ""
		}
		if isThematicBreak(line) && !isSetextUnderline(previousLine, line) {
			var err error
			slides, err = appendScannedSlide(slides, builder, name, lineNumber, global, inherited, spot, notes)
			if err != nil {
				return nil, err
			}
			builder = newLayoutBuilder()
			if err := builder.setComposition(inherited.Composition, lineNumber); err != nil {
				return nil, deckError("deck.composition_conflict", name, lineNumber, err.Error())
			}
			spot = nil
			notes = nil
			previousLine = ""
			continue
		}
		builder.write(line)
		previousLine = lineContent(line)
	}
	if fence != nil {
		return nil, deckError("deck.fence_unclosed", name, fence.line, "fenced code block is not closed")
	}
	return appendScannedSlide(slides, builder, name, firstLine+len(lines), global, inherited, spot, notes)
}

func appendScannedSlide(slides []Slide, builder *layoutBuilder, name string, line int, global, inherited DirectiveState, spot []directiveEvent, notes []string) ([]Slide, error) {
	state, err := effectiveState(global, inherited, spot)
	if err != nil {
		return nil, err
	}
	spec := CompositionSpec{}
	if state.Composition != "" {
		spec, err = ResolveComposition(state.Composition)
		if err != nil {
			return nil, deckError("deck.composition_invalid", name, line, err.Error())
		}
	}
	classes, err := normalizeCompositionClasses(name, line, state.Classes, spec)
	if err != nil {
		return nil, err
	}
	state.Classes = classes
	if spec.Name == "" {
		for _, class := range classes {
			if class == "grid" {
				return nil, deckError("deck.class_unsupported", name, line, "unsupported deck class grid without a composition")
			}
		}
	}
	layout, markdown, err := builder.finish(name, line, classes, spec)
	if err != nil {
		return nil, err
	}
	if state.Background.Source != "" && !state.Background.Decorative && strings.TrimSpace(state.Background.Alt) == "" {
		return nil, deckError("deck.background_alt_required", name, line, "non-decorative local backgrounds require backgroundAlt")
	}
	foreground, background := state.Color, state.BackgroundColor
	if foreground == "" {
		foreground = "ink"
	}
	if background == "" || background == "transparent" {
		background = "surface"
	}
	if err := ValidateThemeColorPair(state.Theme, state.ColorMode, foreground, background); err != nil {
		return nil, deckError("deck.contrast_invalid", name, line, err.Error())
	}
	if len(bytes.TrimSpace(markdown)) == 0 && layout == nil {
		return nil, deckError("deck.slide_empty", name, line, "slides must contain Markdown content")
	}
	ordinal := len(slides) + 1
	return append(slides, Slide{
		ordinal:     ordinal,
		id:          fmt.Sprintf("slide-%04d", ordinal),
		markdown:    append([]byte(nil), markdown...),
		directives:  state,
		composition: cloneCompositionSpec(spec),
		notes:       append([]string(nil), notes...),
		layout:      cloneLayout(layout),
	}), nil
}

func compositionName(event directiveEvent) CompositionName {
	value, _ := directiveScalar(event.node, true)
	if value == "none" {
		return ""
	}
	return CompositionName(value)
}

func normalizeCompositionClasses(name string, line int, classes []string, spec CompositionSpec) ([]string, error) {
	result := append([]string(nil), classes...)
	if spec.Name == "" {
		return result, nil
	}
	for _, class := range result {
		if class == "invert" {
			continue
		}
		if spec.LayoutClass == "" || class != spec.LayoutClass {
			return nil, deckError("deck.composition_conflict", name, line, "class does not match composition")
		}
	}
	if spec.LayoutClass != "" {
		found := false
		for _, class := range result {
			if class == spec.LayoutClass {
				found = true
				break
			}
		}
		if !found {
			result = append(result, spec.LayoutClass)
		}
	}
	if err := validateClassCombination(name, line, result); err != nil {
		return nil, deckError("deck.composition_conflict", name, line, err.Error())
	}
	return result, nil
}

func htmlComment(line []byte) (string, bool) {
	trimmed := strings.TrimSpace(lineContent(line))
	if !strings.HasPrefix(trimmed, "<!--") || !strings.HasSuffix(trimmed, "-->") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "<!--"), "-->")), true
}

func hasLayoutContent(builder *layoutBuilder) bool {
	if builder == nil {
		return false
	}
	if len(bytes.TrimSpace(builder.content.Bytes())) > 0 || len(bytes.TrimSpace(builder.current.Bytes())) > 0 {
		return true
	}
	return len(builder.slots) > 0 || builder.started
}

func shouldSplitHeading(line []byte, divider HeadingDivider) bool {
	level, ok := headingLevel(line)
	if !ok {
		return false
	}
	if divider.Scalar > 0 {
		return level <= divider.Scalar
	}
	for _, candidate := range divider.Levels {
		if level == candidate {
			return true
		}
	}
	return false
}

func headingLevel(line []byte) (int, bool) {
	content := lineContent(line)
	indent := 0
	for indent < len(content) && indent < 4 && content[indent] == ' ' {
		indent++
	}
	if indent > 3 || indent == len(content) {
		return 0, false
	}
	index := indent
	for index < len(content) && content[index] == '#' {
		index++
	}
	level := index - indent
	if level < 1 || level > 6 {
		return 0, false
	}
	if index < len(content) && content[index] != ' ' && content[index] != '\t' {
		return 0, false
	}
	return level, true
}

func isThematicBreak(line []byte) bool {
	content := lineContent(line)
	indent := 0
	for indent < len(content) && content[indent] == ' ' {
		indent++
	}
	if indent > 3 || indent == len(content) {
		return false
	}
	if content[indent] == '>' || content[indent] == '+' || content[indent] >= '0' && content[indent] <= '9' {
		return false
	}
	marker := content[indent]
	if marker != '-' && marker != '_' && marker != '*' {
		return false
	}
	count := 0
	for _, character := range content[indent:] {
		switch character {
		case rune(marker):
			count++
		case ' ', '\t':
		default:
			return false
		}
	}
	return count >= 3
}

func isSetextUnderline(previous string, line []byte) bool {
	previous = strings.TrimSpace(previous)
	if previous == "" || strings.HasPrefix(previous, "#") || strings.HasPrefix(previous, ">") {
		return false
	}
	if strings.HasPrefix(previous, "```") || strings.HasPrefix(previous, "~~~") {
		return false
	}
	if strings.HasPrefix(previous, "-") || strings.HasPrefix(previous, "*") || strings.HasPrefix(previous, "+") || strings.HasPrefix(previous, "[") {
		return false
	}
	content := strings.TrimSpace(lineContent(line))
	if content == "" {
		return false
	}
	for _, character := range content {
		if character != '-' && character != ' ' && character != '\t' {
			return false
		}
	}
	return strings.Count(content, "-") >= 3
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
