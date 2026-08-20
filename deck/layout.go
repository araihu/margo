package deck

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

var structuralLayoutClasses = map[string]struct{}{
	"columns": {}, "sidebar": {}, "compare": {}, "metrics": {}, "timeline": {}, "demo": {}, "grid": {},
}

var styleLayoutClasses = map[string]struct{}{
	"lead": {}, "section": {}, "chapter": {}, "quote": {}, "invert": {},
}

var structuralCommentPattern = regexp.MustCompile(`^<!--\s*(layout|slot|/layout)\s*:\s*([^>]*)-->$`)
var closeLayoutCommentPattern = regexp.MustCompile(`^<!--\s*/layout\s*-->$`)

type layoutMarker struct {
	kind  string
	value string
	line  int
}

func validLayoutClass(value string) bool {
	_, structural := structuralLayoutClasses[value]
	_, style := styleLayoutClasses[value]
	return structural || style
}

func isStructuralClass(value string) bool {
	_, ok := structuralLayoutClasses[value]
	return ok
}

func parseStructuralComment(line []byte) (layoutMarker, bool, error) {
	trimmed := strings.TrimSpace(lineContent(line))
	if closeLayoutCommentPattern.MatchString(trimmed) {
		return layoutMarker{kind: "end"}, true, nil
	}
	matches := structuralCommentPattern.FindStringSubmatch(trimmed)
	if len(matches) == 0 {
		if strings.HasPrefix(trimmed, "<!--") && strings.HasSuffix(trimmed, "-->") {
			body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "<!--"), "-->"))
			if strings.HasPrefix(body, "layout") || strings.HasPrefix(body, "slot") || strings.HasPrefix(body, "/layout") {
				return layoutMarker{}, true, fmt.Errorf("malformed structural layout marker")
			}
		}
		return layoutMarker{}, false, nil
	}
	value := strings.TrimSpace(matches[2])
	if value == "" {
		return layoutMarker{}, true, fmt.Errorf("structural marker value is empty")
	}
	return layoutMarker{kind: matches[1], value: value}, true, nil
}

type layoutBuilder struct {
	class         string
	markerClass   string
	composition   CompositionName
	started       bool
	closed        bool
	activeSlot    string
	activeLine    int
	current       bytes.Buffer
	content       bytes.Buffer
	slots         []LayoutSlot
	seen          map[string]struct{}
	firstLine     int
	markerPresent bool
}

func newLayoutBuilder() *layoutBuilder {
	return &layoutBuilder{seen: make(map[string]struct{})}
}

func (builder *layoutBuilder) write(line []byte) {
	if builder.started && builder.activeSlot != "" {
		_, _ = builder.current.Write(line)
		return
	}
	_, _ = builder.content.Write(line)
}

func (builder *layoutBuilder) start(class string, line int) error {
	if builder.started {
		if builder.composition != "" && !builder.markerPresent && builder.markerClass == class {
			builder.markerPresent = true
			return nil
		}
		return fmt.Errorf("nested layout marker")
	}
	builder.started = true
	builder.markerPresent = true
	builder.markerClass = class
	builder.firstLine = line
	return nil
}

// setComposition binds the current slide to a catalog composition. Structural
// compositions create an implicit layout; body compositions keep the normal
// markdown path. The binding may be replaced only before any content or an
// explicit layout marker has been authored.
func (builder *layoutBuilder) setComposition(name CompositionName, line int) error {
	if name == builder.composition {
		return nil
	}
	spec := CompositionSpec{}
	if name != "" {
		resolved, err := ResolveComposition(name)
		if err != nil {
			return err
		}
		spec = resolved
	}
	if builder.composition != "" {
		if builder.markerPresent || len(bytes.TrimSpace(builder.content.Bytes())) > 0 || len(bytes.TrimSpace(builder.current.Bytes())) > 0 || len(builder.slots) > 0 {
			return fmt.Errorf("composition cannot change after layout content begins")
		}
		if builder.started && builder.markerPresent && spec.LayoutClass != builder.markerClass {
			return fmt.Errorf("composition does not agree with layout marker")
		}
	}
	if builder.started && builder.markerPresent {
		if !isStructuralClass(spec.LayoutClass) || spec.LayoutClass != builder.markerClass {
			return fmt.Errorf("composition does not agree with layout marker")
		}
	}
	structural := isStructuralClass(spec.LayoutClass)
	if builder.started && !builder.markerPresent && structural && spec.LayoutClass != builder.markerClass {
		builder.started = false
		builder.closed = false
		builder.markerClass = ""
		builder.firstLine = 0
		builder.activeSlot = ""
		builder.activeLine = 0
		builder.current.Reset()
		builder.content.Reset()
		builder.slots = nil
		builder.seen = make(map[string]struct{})
	}
	if builder.started && !structural {
		builder.started = false
		builder.closed = false
		builder.markerClass = ""
		builder.firstLine = 0
		builder.activeSlot = ""
		builder.activeLine = 0
		builder.current.Reset()
		builder.content.Reset()
		builder.slots = nil
		builder.seen = make(map[string]struct{})
	}
	builder.composition = name
	if !structural {
		return nil
	}
	if !builder.started {
		builder.started = true
		builder.markerPresent = false
		builder.markerClass = spec.LayoutClass
		builder.firstLine = line
	}
	return nil
}

func (builder *layoutBuilder) slot(name string, line int) error {
	if !builder.started || builder.closed {
		return fmt.Errorf("slot marker is outside an active layout")
	}
	if builder.activeSlot != "" {
		if err := builder.finishSlot(); err != nil {
			return err
		}
	}
	if _, exists := builder.seen[name]; exists {
		return fmt.Errorf("duplicate slot %q", name)
	}
	builder.seen[name] = struct{}{}
	builder.activeSlot = name
	builder.activeLine = line
	builder.current.Reset()
	return nil
}

func (builder *layoutBuilder) end() error {
	if !builder.started || builder.closed {
		return fmt.Errorf("layout end marker is outside an active layout")
	}
	if err := builder.finishSlot(); err != nil {
		return err
	}
	builder.closed = true
	return nil
}

func (builder *layoutBuilder) finishSlot() error {
	if builder.activeSlot == "" {
		return nil
	}
	markdown := bytes.TrimSpace(builder.current.Bytes())
	if len(markdown) == 0 {
		return fmt.Errorf("slot %q is empty", builder.activeSlot)
	}
	builder.slots = append(builder.slots, LayoutSlot{
		Name:       builder.activeSlot,
		Markdown:   append(append([]byte(nil), markdown...), '\n'),
		SourceLine: builder.activeLine,
	})
	builder.activeSlot = ""
	builder.current.Reset()
	return nil
}

func (builder *layoutBuilder) finish(name string, line int, classes []string, spec CompositionSpec) (*Layout, []byte, error) {
	if err := validateClassCombination(name, line, classes); err != nil {
		return nil, nil, err
	}
	structural := ""
	for _, class := range classes {
		if isStructuralClass(class) {
			structural = class
		}
	}
	if spec.Name != "" {
		if spec.BodyRole != "" {
			if builder.started || len(builder.slots) > 0 || builder.activeSlot != "" {
				return nil, nil, deckError("deck.composition_slot_invalid", name, builder.firstLine, "body composition cannot contain structural slots")
			}
			return nil, append([]byte(nil), builder.content.Bytes()...), nil
		}
		if structural != spec.LayoutClass || builder.composition != spec.Name {
			return nil, nil, deckError("deck.composition_conflict", name, line, "composition and layout class do not agree")
		}
		if !builder.started {
			return nil, nil, deckError("deck.composition_slots_required", name, line, "structural composition requires slots")
		}
		if builder.markerPresent && !builder.closed {
			return nil, nil, deckError("deck.layout_invalid", name, line, "layout marker is not closed")
		}
		if !builder.closed {
			if err := builder.finishSlot(); err != nil {
				return nil, nil, deckError("deck.composition_slot_invalid", name, builder.activeLine, err.Error())
			}
		}
		if builder.content.Len() > 0 && len(bytes.TrimSpace(builder.content.Bytes())) > 0 {
			return nil, nil, deckError("deck.composition_slot_invalid", name, builder.firstLine, "unmarked content is not allowed inside a composition layout")
		}
		if err := validateCompositionSlots(name, spec, builder.slots, builder.firstLine); err != nil {
			return nil, nil, err
		}
		return &Layout{Class: structural, Slots: append([]LayoutSlot(nil), builder.slots...)}, nil, nil
	}
	if !builder.started {
		if structural != "" {
			return nil, nil, deckError("deck.layout_slots_required", name, line, "structural class requires a layout marker")
		}
		return nil, append([]byte(nil), builder.content.Bytes()...), nil
	}
	if structural == "" || builder.markerClass != structural {
		return nil, nil, deckError("deck.layout_invalid", name, builder.firstLine, "layout marker and class do not agree")
	}
	if !builder.closed {
		return nil, nil, deckError("deck.layout_invalid", name, line, "layout marker is not closed")
	}
	if builder.content.Len() > 0 && len(bytes.TrimSpace(builder.content.Bytes())) > 0 {
		return nil, nil, deckError("deck.layout_invalid", name, builder.firstLine, "unmarked content is not allowed inside a structural layout")
	}
	if err := validateSlots(name, structural, builder.slots, builder.firstLine); err != nil {
		return nil, nil, err
	}
	return &Layout{Class: structural, Slots: append([]LayoutSlot(nil), builder.slots...)}, nil, nil
}

func validateCompositionSlots(name string, spec CompositionSpec, slots []LayoutSlot, line int) error {
	if len(slots) < spec.MinSlots || len(slots) > spec.MaxSlots {
		return deckError("deck.composition_slots_required", name, line, fmt.Sprintf("%s requires %d-%d slots", spec.Name, spec.MinSlots, spec.MaxSlots))
	}
	allowed := make(map[string]CompositionSlot, len(spec.Slots))
	for _, slot := range spec.Slots {
		allowed[slot.Name] = slot
	}
	for index, slot := range slots {
		expected, ok := allowed[slot.Name]
		if !ok || index >= len(spec.Slots) || spec.Slots[index].Name != slot.Name {
			return deckError("deck.composition_slot_invalid", name, slot.SourceLine, "slot is not valid for composition "+string(spec.Name))
		}
		if !expected.Required && index < spec.MinSlots {
			return deckError("deck.composition_slot_invalid", name, slot.SourceLine, "required composition slots are out of order")
		}
	}
	return nil
}

func validateSlots(name, class string, slots []LayoutSlot, line int) error {
	if len(slots) == 0 {
		return deckError("deck.layout_slots_required", name, line, "structural layout requires slots")
	}
	expected := map[string]struct{}{}
	min, max := 0, 0
	switch class {
	case "columns", "sidebar", "compare", "demo":
		min, max = 2, 2
		if class == "sidebar" {
			expected["main"], expected["rail"] = struct{}{}, struct{}{}
		} else if class == "demo" {
			expected["code"], expected["result"] = struct{}{}, struct{}{}
		} else {
			expected["left"], expected["right"] = struct{}{}, struct{}{}
		}
	case "metrics":
		min, max = 3, 4
	case "timeline":
		min, max = 3, 6
	case "grid":
		min, max = 2, 4
	}
	if len(slots) < min || len(slots) > max {
		return deckError("deck.layout_slots_required", name, line, fmt.Sprintf("%s requires %d-%d slots", class, min, max))
	}
	seen := make(map[string]struct{}, len(slots))
	for index, slot := range slots {
		if _, duplicate := seen[slot.Name]; duplicate {
			return deckError("deck.slot_invalid", name, slot.SourceLine, "slot is duplicated")
		}
		seen[slot.Name] = struct{}{}
		if len(expected) > 0 {
			if _, ok := expected[slot.Name]; !ok {
				return deckError("deck.slot_invalid", name, slot.SourceLine, "slot is not valid for layout "+class)
			}
		} else if class == "metrics" {
			if slot.Name != fmt.Sprintf("metric-%d", index+1) {
				return deckError("deck.slot_invalid", name, slot.SourceLine, "metrics slots must be metric-1 through metric-4 in order")
			}
		} else if class == "timeline" {
			if slot.Name != fmt.Sprintf("step-%d", index+1) {
				return deckError("deck.slot_invalid", name, slot.SourceLine, "timeline slots must be step-1 through step-6 in order")
			}
		}
	}
	if len(expected) > 0 && len(seen) != len(expected) {
		return deckError("deck.layout_slots_required", name, line, "required layout slots are missing")
	}
	return nil
}
