package deck

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/araihu/margo"
	"gopkg.in/yaml.v3"
)

var bcp47Pattern = regexp.MustCompile(`^[A-Za-z]{2,8}(?:-[A-Za-z0-9]{1,8})*$`)

var (
	globalDirectiveNames = map[string]struct{}{
		"theme": {}, "lang": {}, "colorMode": {}, "headingDivider": {}, "size": {}, "style": {},
	}
	localDirectiveNames = map[string]struct{}{
		"paginate": {}, "header": {}, "footer": {}, "class": {}, "color": {}, "backgroundColor": {},
		"backgroundImage": {}, "backgroundPosition": {}, "backgroundRepeat": {}, "backgroundSize": {},
		"backgroundDecorative": {}, "backgroundAlt": {},
	}
	colorTokens = map[string]struct{}{
		"surface": {}, "surface-alt": {}, "ink": {}, "ink-muted": {}, "accent": {}, "accent-strong": {},
		"positive": {}, "warning": {}, "negative": {}, "info": {}, "transparent": {},
	}
	gradientTokens = map[string]struct{}{
		"gradient-blue": {}, "gradient-violet": {}, "gradient-sunset": {}, "gradient-forest": {},
	}
)

type directiveEvent struct {
	name string
	spot bool
	node *yaml.Node
	line int
}

type directiveCommentKind uint8

const (
	directiveCommentNote directiveCommentKind = iota
	directiveCommentMapping
)

// parseDirectiveComment parses one complete HTML comment. A comment without a
// recognized key is a presenter note; a malformed comment that starts with a
// recognized key is a positioned directive error.
func parseDirectiveComment(name string, line int, comment string) (directiveCommentKind, []directiveEvent, string, error) {
	body := strings.TrimSpace(comment)
	if body == "" {
		return directiveCommentNote, nil, "", nil
	}
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(body), &document); err != nil {
		if directiveLooksRecognized(body) {
			return 0, nil, "", deckError("deck.directive_comment_invalid", name, line, err.Error())
		}
		return directiveCommentNote, nil, body, nil
	}
	if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		if directiveLooksRecognized(body) {
			return 0, nil, "", deckError("deck.directive_comment_invalid", name, line, "directive comment must be one YAML mapping")
		}
		return directiveCommentNote, nil, body, nil
	}
	mapping := document.Content[0]
	var events []directiveEvent
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		keyNode := mapping.Content[index]
		valueNode := mapping.Content[index+1]
		if keyNode.Kind != yaml.ScalarNode || keyNode.Tag != "!!str" {
			return 0, nil, "", deckError("deck.directive_comment_invalid", name, line, "directive keys must be strings")
		}
		key := keyNode.Value
		spot := strings.HasPrefix(key, "_")
		if spot {
			key = strings.TrimPrefix(key, "_")
			if key == "" || strings.HasPrefix(key, "_") {
				return 0, nil, "", deckError("deck.directive_invalid", name, line, "directive spot prefix is invalid")
			}
		}
		if _, ok := globalDirectiveNames[key]; !ok {
			if _, ok := localDirectiveNames[key]; !ok {
				continue
			}
		}
		if err := validateDirectiveNode(name, line, key, valueNode); err != nil {
			return 0, nil, "", err
		}
		events = append(events, directiveEvent{name: key, spot: spot, node: valueNode, line: line})
	}
	if len(events) == 0 {
		return directiveCommentNote, nil, body, nil
	}
	return directiveCommentMapping, events, "", nil
}

func directiveLooksRecognized(body string) bool {
	trimmed := strings.TrimSpace(body)
	if strings.HasPrefix(trimmed, "$") {
		trimmed = strings.TrimPrefix(trimmed, "$")
	}
	if strings.HasPrefix(trimmed, "_") {
		trimmed = strings.TrimPrefix(trimmed, "_")
	}
	for key := range globalDirectiveNames {
		if strings.HasPrefix(trimmed, key+":") {
			return true
		}
	}
	for key := range localDirectiveNames {
		if strings.HasPrefix(trimmed, key+":") {
			return true
		}
	}
	return false
}

func validateDirectiveNode(name string, line int, key string, node *yaml.Node) error {
	if node == nil {
		return deckError("deck.directive_invalid", name, line, "directive value is missing")
	}
	if node.Anchor != "" || node.Kind == yaml.AliasNode || node.Tag == "!!map" || node.Tag == "!!seq" && key != "headingDivider" && key != "class" {
		if key != "size" {
			return deckError("deck.directive_invalid", name, line, "directive value uses an unsupported YAML feature")
		}
	}
	switch key {
	case "theme":
		value, err := directiveScalar(node, true)
		if err != nil || (value != string(margo.ThemeModern) && value != string(margo.ThemeGoshtoso) && value != string(margo.ThemeMinimal)) {
			return deckError("deck.directive_invalid", name, line, "theme must be modern, goshtoso, or minimal")
		}
	case "lang":
		value, err := directiveScalar(node, true)
		if err != nil || !bcp47Pattern.MatchString(value) {
			return deckError("deck.directive_invalid", name, line, "lang must be a BCP 47-style language tag")
		}
	case "colorMode":
		value, err := directiveScalar(node, true)
		if err != nil || (value != string(margo.ColorModeLight) && value != string(margo.ColorModeDark)) {
			return deckError("deck.directive_invalid", name, line, "colorMode must be light or dark")
		}
	case "headingDivider":
		if _, err := parseHeadingDivider(node); err != nil {
			return deckError("deck.directive_invalid", name, line, err.Error())
		}
	case "size":
		if _, err := parseSize(node); err != nil {
			return deckError("deck.directive_invalid", name, line, err.Error())
		}
	case "style":
		return deckError("deck.directive_unsupported", name, line, "document-authored style is not supported")
	case "paginate":
		if node.Tag != "!!null" {
			if node.Tag == "!!bool" {
				break
			}
			value, err := directiveScalar(node, true)
			if err != nil || (value != "hold" && value != "skip" && value != "none") {
				return deckError("deck.directive_invalid", name, line, "paginate must be true, false, hold, skip, or none")
			}
		}
	case "header", "footer":
		if node.Tag != "!!null" {
			value, err := directiveScalar(node, true)
			if err != nil || len(value) > 240 {
				return deckError("deck.directive_invalid", name, line, key+" must be bounded inline text")
			}
		}
	case "class":
		values, err := directiveStringList(node)
		if err != nil {
			return deckError("deck.directive_invalid", name, line, "class must be a string or string list")
		}
		for _, value := range values {
			if value == "none" {
				continue
			}
			if !validLayoutClass(value) {
				return deckError("deck.class_unsupported", name, line, "unsupported deck class "+value)
			}
		}
	case "color", "backgroundColor":
		value, err := directiveScalarOrNone(node)
		if err != nil {
			return deckError("deck.color_invalid", name, line, "color must use a finite token")
		}
		if value != "" {
			if _, ok := colorTokens[value]; !ok {
				return deckError("deck.color_invalid", name, line, "color must use a finite token")
			}
		}
	case "backgroundImage":
		value, err := directiveScalarOrNone(node)
		if err != nil || value != "" && !validBackgroundSource(value) {
			return deckError("deck.background_invalid", name, line, "backgroundImage must be a local asset, gradient token, or none")
		}
	case "backgroundPosition":
		value, err := directiveScalarOrNone(node)
		if err != nil || !oneOf(value, "", "center", "top", "bottom", "left", "right", "top-left", "top-right", "bottom-left", "bottom-right") {
			return deckError("deck.background_invalid", name, line, "backgroundPosition is not supported")
		}
	case "backgroundRepeat":
		value, err := directiveScalarOrNone(node)
		if err != nil || !oneOf(value, "", "no-repeat", "repeat", "repeat-x", "repeat-y") {
			return deckError("deck.background_invalid", name, line, "backgroundRepeat is not supported")
		}
	case "backgroundSize":
		value, err := directiveScalarOrNone(node)
		if err != nil || !oneOf(value, "", "cover", "contain", "auto") {
			return deckError("deck.background_invalid", name, line, "backgroundSize is not supported")
		}
	case "backgroundDecorative":
		if node.Tag != "!!null" && node.Tag != "!!bool" {
			if value, err := directiveScalar(node, true); err != nil || value != "none" {
				return deckError("deck.background_invalid", name, line, "backgroundDecorative must be boolean or none")
			}
		}
	case "backgroundAlt":
		if node.Tag != "!!null" {
			value, err := directiveScalar(node, true)
			if err != nil || len(value) > 240 {
				return deckError("deck.background_invalid", name, line, "backgroundAlt must be bounded text")
			}
		}
	}
	return nil
}

func applyDirectiveEvent(state *DirectiveState, event directiveEvent) error {
	if state == nil {
		return fmt.Errorf("nil directive state")
	}
	switch event.name {
	case "theme":
		value, _ := directiveScalar(event.node, true)
		state.Theme = margo.ThemeName(value)
	case "lang":
		value, _ := directiveScalar(event.node, true)
		state.Lang = value
	case "colorMode":
		value, _ := directiveScalar(event.node, true)
		state.ColorMode = margo.ColorMode(value)
	case "headingDivider":
		value, _ := parseHeadingDivider(event.node)
		state.HeadingDivider = value
	case "size":
		value, _ := parseSize(event.node)
		state.Size = value
	case "paginate":
		if event.node.Tag == "!!null" {
			state.Paginate = ""
		} else if event.node.Tag == "!!bool" {
			state.Paginate = event.node.Value
		} else {
			value, _ := directiveScalar(event.node, true)
			if value == "none" {
				state.Paginate = ""
			} else {
				state.Paginate = value
			}
		}
	case "header", "footer":
		value, _ := directiveScalarOrNone(event.node)
		if event.name == "header" {
			state.Header = value
		} else {
			state.Footer = value
		}
	case "class":
		values, _ := directiveStringList(event.node)
		if len(values) == 0 || (len(values) == 1 && values[0] == "none") {
			state.Classes = nil
		} else {
			state.Classes = append([]string(nil), values...)
		}
	case "color", "backgroundColor":
		value, _ := directiveScalarOrNone(event.node)
		if event.name == "color" {
			state.Color = value
		} else {
			state.BackgroundColor = value
		}
	case "backgroundImage":
		value, _ := directiveScalarOrNone(event.node)
		if value == "" {
			state.Background = BackgroundState{}
		} else {
			_, gradient := gradientTokens[value]
			state.Background = BackgroundState{Source: value, Decorative: gradient}
		}
	case "backgroundPosition", "backgroundRepeat", "backgroundSize":
		value, _ := directiveScalarOrNone(event.node)
		switch event.name {
		case "backgroundPosition":
			state.Background.Position = value
		case "backgroundRepeat":
			state.Background.Repeat = value
		case "backgroundSize":
			state.Background.Size = value
		}
	case "backgroundDecorative":
		if event.node.Tag == "!!null" {
			_, gradient := gradientTokens[state.Background.Source]
			state.Background.Decorative = gradient
		} else if event.node.Tag == "!!bool" {
			state.Background.Decorative = event.node.Value == "true"
		} else {
			_, gradient := gradientTokens[state.Background.Source]
			state.Background.Decorative = gradient
		}
	case "backgroundAlt":
		value, _ := directiveScalarOrNone(event.node)
		state.Background.Alt = value
	}
	return nil
}

func directiveScalar(node *yaml.Node, rejectNull bool) (string, error) {
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag == "!!map" || node.Tag == "!!seq" || rejectNull && node.Tag == "!!null" {
		return "", fmt.Errorf("directive value must be a scalar")
	}
	return node.Value, nil
}

func directiveScalarOrNone(node *yaml.Node) (string, error) {
	if node == nil || node.Tag == "!!null" {
		return "", nil
	}
	value, err := directiveScalar(node, true)
	if err != nil {
		return "", err
	}
	if strings.EqualFold(value, "none") {
		return "", nil
	}
	return value, nil
}

func directiveStringList(node *yaml.Node) ([]string, error) {
	if node == nil || node.Tag == "!!null" {
		return nil, nil
	}
	if node.Kind == yaml.ScalarNode {
		value := strings.TrimSpace(node.Value)
		if value == "" {
			return nil, nil
		}
		if value == "none" {
			return []string{"none"}, nil
		}
		parts := strings.Fields(value)
		return parts, nil
	}
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("directive value must be a string list")
	}
	values := make([]string, 0, len(node.Content))
	for _, item := range node.Content {
		value, err := directiveScalar(item, true)
		if err != nil || strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("directive list contains a non-string value")
		}
		values = append(values, strings.TrimSpace(value))
	}
	return values, nil
}

func parseHeadingDivider(node *yaml.Node) (HeadingDivider, error) {
	if node == nil || node.Tag == "!!null" {
		return HeadingDivider{}, nil
	}
	if node.Kind == yaml.ScalarNode {
		value, err := strconv.Atoi(node.Value)
		if err != nil || value < 1 || value > 6 {
			return HeadingDivider{}, fmt.Errorf("headingDivider scalar must be an integer from 1 through 6")
		}
		return HeadingDivider{Scalar: value}, nil
	}
	if node.Kind != yaml.SequenceNode || len(node.Content) == 0 {
		return HeadingDivider{}, fmt.Errorf("headingDivider must be an integer or non-empty integer list")
	}
	levels := make([]int, 0, len(node.Content))
	seen := make(map[int]struct{}, len(node.Content))
	for _, item := range node.Content {
		value, err := strconv.Atoi(item.Value)
		if item.Kind != yaml.ScalarNode || err != nil || value < 1 || value > 6 {
			return HeadingDivider{}, fmt.Errorf("headingDivider levels must be integers from 1 through 6")
		}
		if _, ok := seen[value]; ok {
			return HeadingDivider{}, fmt.Errorf("headingDivider levels must be unique")
		}
		seen[value] = struct{}{}
		levels = append(levels, value)
	}
	return HeadingDivider{Levels: levels}, nil
}

func parseSize(node *yaml.Node) (string, error) {
	if node == nil || node.Tag == "!!null" {
		return "16:9", nil
	}
	if node.Kind == yaml.ScalarNode {
		value := strings.TrimSpace(node.Value)
		if value == "16:9" || value == "4:3" {
			return value, nil
		}
		return "", fmt.Errorf("size must be 16:9, 4:3, or bounded custom dimensions")
	}
	if node.Kind != yaml.MappingNode {
		return "", fmt.Errorf("size must be 16:9, 4:3, or bounded custom dimensions")
	}
	var width, height float64
	hasWidth, hasHeight, hasUnit := false, false, false
	unit := ""
	for index := 0; index+1 < len(node.Content); index += 2 {
		key := node.Content[index].Value
		valueNode := node.Content[index+1]
		switch key {
		case "width":
			if hasWidth || valueNode.Kind != yaml.ScalarNode || valueNode.Tag != "!!int" && valueNode.Tag != "!!float" {
				return "", fmt.Errorf("custom size width must be a finite number")
			}
			value, err := strconv.ParseFloat(valueNode.Value, 64)
			if err != nil || value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
				return "", fmt.Errorf("custom size width must be a finite positive number")
			}
			width, hasWidth = value, true
		case "height":
			if hasHeight || valueNode.Kind != yaml.ScalarNode || valueNode.Tag != "!!int" && valueNode.Tag != "!!float" {
				return "", fmt.Errorf("custom size height must be a finite number")
			}
			value, err := strconv.ParseFloat(valueNode.Value, 64)
			if err != nil || value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
				return "", fmt.Errorf("custom size height must be a finite positive number")
			}
			height, hasHeight = value, true
		case "unit":
			if valueNode.Kind != yaml.ScalarNode || valueNode.Tag != "!!str" {
				return "", fmt.Errorf("custom size unit must be an absolute CSS unit")
			}
			if hasUnit {
				return "", fmt.Errorf("custom size unit is duplicated")
			}
			unit = valueNode.Value
			hasUnit = true
		default:
			return "", fmt.Errorf("custom size only accepts width, height, and unit")
		}
	}
	if !hasWidth || !hasHeight || !hasUnit {
		return "", fmt.Errorf("custom size requires width, height, and unit")
	}
	if _, ok := deckUnitFactor(DeckUnit(unit)); !ok {
		return "", fmt.Errorf("custom size unit is not supported")
	}
	normalized := strconv.FormatFloat(width, 'f', -1, 64) + "x" + strconv.FormatFloat(height, 'f', -1, 64) + unit
	if _, err := ParseDeckGeometry(normalized); err != nil {
		return "", err
	}
	return normalized, nil
}

func validBackgroundSource(value string) bool {
	if _, ok := gradientTokens[value]; ok {
		return true
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.ContainsAny(trimmed, "\r\n{};<>") || strings.Contains(trimmed, "\\") {
		return false
	}
	for _, prefix := range []string{"http:", "https:", "//", "data:", "javascript:"} {
		if strings.HasPrefix(strings.ToLower(trimmed), prefix) {
			return false
		}
	}
	return true
}

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func effectiveState(global, inherited DirectiveState, spot []directiveEvent) (DirectiveState, error) {
	state := cloneDirectiveState(global)
	state.Paginate = inherited.Paginate
	state.Header = inherited.Header
	state.Footer = inherited.Footer
	state.Classes = append([]string(nil), inherited.Classes...)
	state.Color = inherited.Color
	state.BackgroundColor = inherited.BackgroundColor
	state.Background = inherited.Background
	for _, event := range spot {
		if err := applyDirectiveEvent(&state, event); err != nil {
			return DirectiveState{}, err
		}
	}
	return state, nil
}

func validateClassCombination(name string, line int, classes []string) error {
	structural := 0
	style := 0
	for _, class := range classes {
		if class == "invert" {
			continue
		}
		if isStructuralClass(class) {
			structural++
		} else {
			style++
		}
	}
	if structural > 1 || (structural > 0 && style > 0) || style > 1 {
		return deckError("deck.class_combination_invalid", name, line, "deck classes contain incompatible structural/style combinations")
	}
	return nil
}
