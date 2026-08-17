package ssg

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"sort"
	"strings"

	"github.com/a-h/templ"
)

var builtinFrameNames = []string{
	"main",
	"top-main",
	"top-main-footer",
	"top-left-main-footer",
	"top-left-main-right-footer",
	"main-footer",
}

// BuiltinFrame returns one of the six structural v1 frames.
func BuiltinFrame(name string) (Frame, error) {
	for _, candidate := range builtinFrameNames {
		if name == candidate {
			return builtinFrame{name: name}, nil
		}
	}
	return nil, fmt.Errorf("ssg.layout_unknown: unknown builtin frame %q", name)
}

// BuiltinFrameNames returns the stable catalog order.
func BuiltinFrameNames() []string { return append([]string(nil), builtinFrameNames...) }

type builtinFrame struct{ name string }

func (f builtinFrame) Schema(_ FrameContext) (FrameSchema, error) {
	return builtinSchema(f.name)
}

func (f builtinFrame) Render(input FrameInput) (FrameOutput, error) {
	schema, err := builtinSchema(f.name)
	if err != nil {
		return FrameOutput{}, err
	}
	expectedHash, err := SchemaHashForValues(schema, input.Values)
	if err != nil {
		return FrameOutput{}, err
	}
	if input.SchemaHash == "" {
		input.SchemaHash = expectedHash
	} else if input.SchemaHash != expectedHash {
		return FrameOutput{}, fmt.Errorf("ssg.schema_hash: frame input hash %q does not match %q", input.SchemaHash, expectedHash)
	}
	if input.Bindings == nil {
		input.Bindings = map[string][]AreaBinding{}
	}
	if err := ValidateBindings(schema, input.Bindings); err != nil {
		return FrameOutput{}, err
	}
	data, err := renderBuiltinFragment(schema, input)
	if err != nil {
		return FrameOutput{}, err
	}
	return FrameOutput{
		Fragment: templ.ComponentFunc(func(_ context.Context, writer io.Writer) error {
			_, writeErr := writer.Write(data)
			return writeErr
		}),
		SchemaHash: input.SchemaHash,
	}, nil
}

func builtinSchema(name string) (FrameSchema, error) {
	document := AreaDescriptor{
		ID: "main-content", Role: "document", Required: true, MaxBindings: 1,
		Accepts: []string{"document"}, Target: "margo-document",
		AllowedSwaps: stableSwaps(), Live: "off", Focus: "area", Swap: SwapInnerHTML,
		Slots: []SlotDescriptor{{ID: "after-article", Accepts: []string{"pagination"}, Order: 1}},
	}
	top := AreaDescriptor{
		ID: "top-nav", Multiple: true, MaxBindings: 4,
		MaxBindingsByKind: map[string]int{"navigation": 1, "breadcrumbs": 1, "theme_controls": 1, "locale_controls": 1},
		Accepts:           []string{"navigation", "breadcrumbs", "theme_controls", "locale_controls"},
		Target:            "top-nav", AllowedSwaps: stableSwaps(), Live: "off", Focus: "area", Swap: SwapInnerHTML,
	}
	left := AreaDescriptor{
		ID: "left-nav", MaxBindings: 1, Accepts: []string{"navigation"}, Target: "left-nav",
		AllowedSwaps: stableSwaps(), Live: "off", Focus: "area", Swap: SwapInnerHTML,
	}
	right := AreaDescriptor{
		ID: "right-nav", MaxBindings: 1, Accepts: []string{"toc"}, Target: "right-nav",
		AllowedSwaps: stableSwaps(), Live: "off", Focus: "area", Swap: SwapInnerHTML,
	}
	footer := AreaDescriptor{
		ID: "footer", MaxBindings: 1, Accepts: []string{"footer"}, Target: "footer",
		AllowedSwaps: stableSwaps(), Live: "off", Focus: "area", Swap: SwapInnerHTML,
	}
	areas := map[string]AreaDescriptor{
		"main-content": document, "top-nav": top, "left-nav": left, "right-nav": right, "footer": footer,
	}
	order := map[string][]string{
		"main-content": {"document", "pagination"},
		"top-nav":      {"navigation", "breadcrumbs", "theme_controls", "locale_controls"},
		"left-nav":     {"navigation"}, "right-nav": {"toc"}, "footer": {"footer"},
	}
	var selected []AreaDescriptor
	var source []string
	var defaults map[string]string
	switch name {
	case "main":
		selected, source = []AreaDescriptor{areas["main-content"]}, []string{"main-content"}
		defaults = map[string]string{"document": "main-content", "pagination": "main-content"}
	case "top-main":
		selected, source = []AreaDescriptor{areas["top-nav"], areas["main-content"]}, []string{"top-nav", "main-content"}
		defaults = map[string]string{"document": "main-content", "pagination": "main-content", "navigation": "top-nav", "breadcrumbs": "top-nav", "theme_controls": "top-nav", "locale_controls": "top-nav"}
	case "top-main-footer":
		selected, source = []AreaDescriptor{areas["top-nav"], areas["main-content"], areas["footer"]}, []string{"top-nav", "main-content", "footer"}
		defaults = map[string]string{"document": "main-content", "pagination": "main-content", "navigation": "top-nav", "breadcrumbs": "top-nav", "theme_controls": "top-nav", "locale_controls": "top-nav", "footer": "footer"}
	case "top-left-main-footer":
		selected, source = []AreaDescriptor{areas["top-nav"], areas["left-nav"], areas["main-content"], areas["footer"]}, []string{"top-nav", "left-nav", "main-content", "footer"}
		defaults = map[string]string{"document": "main-content", "pagination": "main-content", "navigation": "left-nav", "breadcrumbs": "top-nav", "theme_controls": "top-nav", "locale_controls": "top-nav", "footer": "footer"}
	case "top-left-main-right-footer":
		selected, source = []AreaDescriptor{areas["top-nav"], areas["left-nav"], areas["main-content"], areas["right-nav"], areas["footer"]}, []string{"top-nav", "left-nav", "main-content", "right-nav", "footer"}
		defaults = map[string]string{"document": "main-content", "pagination": "main-content", "navigation": "left-nav", "breadcrumbs": "top-nav", "theme_controls": "top-nav", "locale_controls": "top-nav", "toc": "right-nav", "footer": "footer"}
	case "main-footer":
		selected, source = []AreaDescriptor{areas["main-content"], areas["footer"]}, []string{"main-content", "footer"}
		defaults = map[string]string{"document": "main-content", "pagination": "main-content", "footer": "footer"}
	default:
		return FrameSchema{}, fmt.Errorf("ssg.layout_unknown: unknown builtin frame %q", name)
	}
	for index := range selected {
		selected[index].Slots = append([]SlotDescriptor(nil), selected[index].Slots...)
		selected[index].Accepts = append([]string(nil), selected[index].Accepts...)
	}
	schema := FrameSchema{
		Contract: FrameContract, Areas: selected, Layout: builtinLayout(name),
		Options:         builtinOptions(name),
		BindingDefaults: defaults, BindingOrder: make(map[string][]string, len(source)),
	}
	for _, area := range source {
		schema.BindingOrder[area] = append([]string(nil), order[area]...)
	}
	if err := ValidateFrameSchema(schema, ""); err != nil {
		return FrameSchema{}, err
	}
	return schema, nil
}

func builtinOptions(name string) []FrameOptionDescriptor {
	areas := map[string]struct{}{}
	switch name {
	case "top-main", "top-main-footer":
		areas["top-nav"] = struct{}{}
	case "top-left-main-footer":
		areas["top-nav"], areas["left-nav"] = struct{}{}, struct{}{}
	case "top-left-main-right-footer":
		areas["top-nav"], areas["left-nav"], areas["right-nav"] = struct{}{}, struct{}{}, struct{}{}
	case "main-footer":
		areas["footer"] = struct{}{}
	}
	if name == "main" {
		return nil
	}
	options := make([]FrameOptionDescriptor, 0, len(areas)*3)
	appendSticky := func(area, edge string) {
		if _, exists := areas[area]; !exists {
			return
		}
		options = append(options,
			FrameOptionDescriptor{Path: "areas." + area + ".sticky.enabled", Type: "boolean", Default: false, Description: "Enable structural sticky positioning for this area."},
			FrameOptionDescriptor{Path: "areas." + area + ".sticky.edge", Type: "enum", Default: edge, Allowed: []string{"block-start", "block-end"}, Description: "Logical edge used by sticky positioning."},
			FrameOptionDescriptor{Path: "areas." + area + ".sticky.offset", Type: "length", Default: "0", Description: "Bounded logical offset for sticky positioning."},
		)
	}
	appendSticky("top-nav", "block-start")
	appendSticky("footer", "block-end")
	for _, area := range []string{"left-nav", "right-nav"} {
		if _, exists := areas[area]; exists {
			options = append(options, FrameOptionDescriptor{Path: "areas." + area + ".collapse_at", Type: "breakpoint", Default: "narrow", Allowed: []string{"narrow", "mid", "wide"}, Description: "Content breakpoint at which this area changes collapse behavior."})
		}
	}
	return options
}

func stableSwaps() []SwapMode {
	return []SwapMode{SwapInnerHTML, SwapOuterHTML, SwapAfterBegin, SwapBeforeEnd}
}

func builtinLayout(name string) FrameLayoutDescriptor {
	breakpoints := []ContentBreakpoint{{Name: "narrow", MinCSSPx: 0, MaxCSSPx: intPtr(720)}, {Name: "mid", MinCSSPx: 720, MaxCSSPx: intPtr(1100)}, {Name: "wide", MinCSSPx: 1100}}
	fluid := func(rows ...string) FramePlacement {
		return FramePlacement{Rows: rows, Columns: []string{"fluid"}, Regions: []FrameRegion{{Area: "main-content", RowStart: 1, RowEnd: len(rows) + 1, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}}, SourceOrder: []string{"main-content"}}
	}
	wide := FramePlacement{}
	mid := FramePlacement{}
	narrow := FramePlacement{}
	switch name {
	case "main":
		wide, mid, narrow = FramePlacement{Rows: []string{"1fr"}, Columns: []string{"main-wide"}, Regions: []FrameRegion{{Area: "main-content", RowStart: 1, RowEnd: 2, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}}, SourceOrder: []string{"main-content"}}, FramePlacement{Rows: []string{"1fr"}, Columns: []string{"main-wide"}, Regions: []FrameRegion{{Area: "main-content", RowStart: 1, RowEnd: 2, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}}, SourceOrder: []string{"main-content"}}, FramePlacement{Rows: []string{"1fr"}, Columns: []string{"narrow-content"}, Regions: []FrameRegion{{Area: "main-content", RowStart: 1, RowEnd: 2, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}}, SourceOrder: []string{"main-content"}}
	case "top-main", "top-main-footer", "main-footer":
		areas := map[string]string{"top-main": "top-nav", "top-main-footer": "top-nav", "main-footer": "main-content"}
		_ = areas
		if name == "top-main" {
			wide = placement([]string{"auto", "1fr"}, []string{"fluid"}, []FrameRegion{{Area: "top-nav", RowStart: 1, RowEnd: 2, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}, {Area: "main-content", RowStart: 2, RowEnd: 3, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}}, []string{"top-nav", "main-content"})
			mid = wide
			narrow = placement([]string{"auto", "1fr"}, []string{"fluid"}, []FrameRegion{{Area: "top-nav", RowStart: 1, RowEnd: 2, ColumnStart: 1, ColumnEnd: 2, Collapse: "stack-before"}, {Area: "main-content", RowStart: 2, RowEnd: 3, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}}, []string{"top-nav", "main-content"})
		} else if name == "top-main-footer" {
			wide = placement([]string{"auto", "1fr", "auto"}, []string{"fluid"}, []FrameRegion{{Area: "top-nav", RowStart: 1, RowEnd: 2, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}, {Area: "main-content", RowStart: 2, RowEnd: 3, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}, {Area: "footer", RowStart: 3, RowEnd: 4, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}}, []string{"top-nav", "main-content", "footer"})
			mid = wide
			narrow = placement([]string{"auto", "1fr", "auto"}, []string{"fluid"}, []FrameRegion{{Area: "top-nav", RowStart: 1, RowEnd: 2, ColumnStart: 1, ColumnEnd: 2, Collapse: "stack-before"}, {Area: "main-content", RowStart: 2, RowEnd: 3, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}, {Area: "footer", RowStart: 3, RowEnd: 4, ColumnStart: 1, ColumnEnd: 2, Collapse: "stack-after"}}, []string{"top-nav", "main-content", "footer"})
		} else {
			wide = placement([]string{"1fr", "auto"}, []string{"fluid"}, []FrameRegion{{Area: "main-content", RowStart: 1, RowEnd: 2, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}, {Area: "footer", RowStart: 2, RowEnd: 3, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}}, []string{"main-content", "footer"})
			mid = wide
			narrow = placement([]string{"1fr", "auto"}, []string{"fluid"}, []FrameRegion{{Area: "main-content", RowStart: 1, RowEnd: 2, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}, {Area: "footer", RowStart: 2, RowEnd: 3, ColumnStart: 1, ColumnEnd: 2, Collapse: "stack-after"}}, []string{"main-content", "footer"})
		}
	case "top-left-main-footer":
		wide = placement([]string{"auto", "1fr", "auto"}, []string{"sidebar-wide", "main-wide"}, []FrameRegion{{Area: "top-nav", RowStart: 1, RowEnd: 2, ColumnStart: 1, ColumnEnd: 3, Collapse: "none"}, {Area: "left-nav", RowStart: 2, RowEnd: 3, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}, {Area: "main-content", RowStart: 2, RowEnd: 3, ColumnStart: 2, ColumnEnd: 3, Collapse: "none"}, {Area: "footer", RowStart: 3, RowEnd: 4, ColumnStart: 1, ColumnEnd: 3, Collapse: "none"}}, []string{"top-nav", "left-nav", "main-content", "footer"})
		mid = placement([]string{"auto", "1fr", "auto"}, []string{"sidebar-mid", "main-mid"}, []FrameRegion{{Area: "top-nav", RowStart: 1, RowEnd: 2, ColumnStart: 1, ColumnEnd: 3, Collapse: "none"}, {Area: "left-nav", RowStart: 2, RowEnd: 3, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}, {Area: "main-content", RowStart: 2, RowEnd: 3, ColumnStart: 2, ColumnEnd: 3, Collapse: "none"}, {Area: "footer", RowStart: 3, RowEnd: 4, ColumnStart: 1, ColumnEnd: 3, Collapse: "none"}}, []string{"top-nav", "left-nav", "main-content", "footer"})
		narrow = placement([]string{"auto", "auto", "1fr", "auto"}, []string{"fluid"}, []FrameRegion{{Area: "top-nav", RowStart: 1, RowEnd: 2, ColumnStart: 1, ColumnEnd: 2, Collapse: "stack-before"}, {Area: "left-nav", RowStart: 2, RowEnd: 3, ColumnStart: 1, ColumnEnd: 2, Collapse: "drawer-inline-start"}, {Area: "main-content", RowStart: 3, RowEnd: 4, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}, {Area: "footer", RowStart: 4, RowEnd: 5, ColumnStart: 1, ColumnEnd: 2, Collapse: "stack-after"}}, []string{"top-nav", "left-nav", "main-content", "footer"})
	case "top-left-main-right-footer":
		wide = placement([]string{"auto", "1fr", "auto"}, []string{"sidebar-wide", "main-wide", "sidebar-wide"}, []FrameRegion{{Area: "top-nav", RowStart: 1, RowEnd: 2, ColumnStart: 1, ColumnEnd: 4, Collapse: "none"}, {Area: "left-nav", RowStart: 2, RowEnd: 3, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}, {Area: "main-content", RowStart: 2, RowEnd: 3, ColumnStart: 2, ColumnEnd: 3, Collapse: "none"}, {Area: "right-nav", RowStart: 2, RowEnd: 3, ColumnStart: 3, ColumnEnd: 4, Collapse: "none"}, {Area: "footer", RowStart: 3, RowEnd: 4, ColumnStart: 1, ColumnEnd: 4, Collapse: "none"}}, []string{"top-nav", "left-nav", "main-content", "right-nav", "footer"})
		mid = placement([]string{"auto", "1fr", "auto"}, []string{"sidebar-mid", "main-three-column-mid", "sidebar-mid"}, wide.Regions, wide.SourceOrder)
		narrow = placement([]string{"auto", "auto", "1fr", "auto", "auto"}, []string{"fluid"}, []FrameRegion{{Area: "top-nav", RowStart: 1, RowEnd: 2, ColumnStart: 1, ColumnEnd: 2, Collapse: "stack-before"}, {Area: "left-nav", RowStart: 2, RowEnd: 3, ColumnStart: 1, ColumnEnd: 2, Collapse: "drawer-inline-start"}, {Area: "main-content", RowStart: 3, RowEnd: 4, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"}, {Area: "right-nav", RowStart: 4, RowEnd: 5, ColumnStart: 1, ColumnEnd: 2, Collapse: "stack-after"}, {Area: "footer", RowStart: 5, RowEnd: 6, ColumnStart: 1, ColumnEnd: 2, Collapse: "stack-after"}}, []string{"top-nav", "left-nav", "main-content", "right-nav", "footer"})
	}
	if wide.Rows == nil {
		wide, mid, narrow = fluid("1fr"), fluid("1fr"), fluid("1fr")
	}
	return FrameLayoutDescriptor{
		Wide: wide, Mid: mid, Narrow: narrow, MainMeasure: "token://layout.tokens.main-measure",
		GapToken: "token://layout.tokens.gap", MinimumReadingBlock: "token://layout.tokens.minimum-reading-block",
		StickyOrder: []string{"top-nav", "footer"}, Breakpoints: breakpoints,
		DrawerInlineSize: "token://layout.tokens.drawer-inline-size", DrawerMaxInlineSize: "min(22rem, 85vi)",
	}
}

func placement(rows, columns []string, regions []FrameRegion, order []string) FramePlacement {
	return FramePlacement{Rows: rows, Columns: columns, Regions: regions, SourceOrder: order}
}

func intPtr(value int) *int { return &value }

func renderBuiltinFragment(schema FrameSchema, input FrameInput) ([]byte, error) {
	var buffer bytes.Buffer
	write := func(value string) { _, _ = buffer.WriteString(value) }
	name := frameName(schema)
	write(`<div class="margo-frame margo-frame--` + html.EscapeString(name) + `" data-margo-frame="` + html.EscapeString(name) + `">`)
	for _, area := range schema.Layout.Wide.SourceOrder {
		descriptor := findArea(schema, area)
		if descriptor.ID == "" {
			continue
		}
		if descriptor.Role == "document" {
			write(`<main id="margo-document" class="margo-area margo-area--` + html.EscapeString(descriptor.ID) + `" tabindex="-1"` + areaAttributes(descriptor, input.Values) + `>`)
		} else {
			write(`<div id="` + html.EscapeString(descriptor.Target) + `" class="margo-area margo-area--` + html.EscapeString(descriptor.ID) + `"` + areaAttributes(descriptor, input.Values) + `>`)
		}
		bindings := append([]AreaBinding(nil), input.Bindings[descriptor.ID]...)
		sort.SliceStable(bindings, func(i, j int) bool {
			left := bindingRank(schema.BindingOrder[descriptor.ID], bindings[i])
			right := bindingRank(schema.BindingOrder[descriptor.ID], bindings[j])
			if left != right {
				return left < right
			}
			return bindings[i].Token < bindings[j].Token
		})
		for _, binding := range bindings {
			if binding.Component == nil {
				return nil, fmt.Errorf("ssg.binding_invalid: %s binding has no fragment", binding.Kind)
			}
			write(`<!--margo.ssg.area-payload:start ` + html.EscapeString(binding.Token) + `-->`)
			if err := binding.Component.Render(context.Background(), &buffer); err != nil {
				return nil, err
			}
			write(`<!--margo.ssg.area-payload:end ` + html.EscapeString(binding.Token) + `-->`)
		}
		if descriptor.Role == "document" {
			write(`</main>`)
		} else {
			write(`</div>`)
		}
	}
	write(`</div>`)
	return buffer.Bytes(), nil
}

func areaAttributes(area AreaDescriptor, values Values) string {
	allowed := make([]string, len(area.AllowedSwaps))
	for index, swap := range area.AllowedSwaps {
		allowed[index] = string(swap)
	}
	attributes := ` data-margo-area="` + html.EscapeString(area.ID) + `" data-margo-target="#` + html.EscapeString(area.Target) + `" data-margo-swap="` + html.EscapeString(string(area.Swap)) + `" data-margo-allowed-swaps="` + html.EscapeString(strings.Join(allowed, ",")) + `" data-margo-live="` + html.EscapeString(area.Live) + `" data-margo-focus="` + html.EscapeString(area.Focus) + `" hx-target="#` + html.EscapeString(area.Target) + `" hx-swap="` + html.EscapeString(string(area.Swap)) + `"`
	if enabled, ok := valueAt(values, "areas", area.ID, "sticky", "enabled"); ok {
		if enabledValue, isBool := enabled.(bool); isBool && enabledValue {
			edge, _ := valueAt(values, "areas", area.ID, "sticky", "edge")
			offset, _ := valueAt(values, "areas", area.ID, "sticky", "offset")
			attributes += ` data-margo-sticky="true" data-margo-sticky-edge="` + html.EscapeString(fmt.Sprint(edge)) + `" data-margo-sticky-offset="` + html.EscapeString(fmt.Sprint(offset)) + `"`
			if offsetString, isString := offset.(string); isString && cssLengthPattern.MatchString(offsetString) {
				attributes += ` style="--margo-sticky-offset:` + html.EscapeString(offsetString) + `"`
			}
		}
	}
	if collapse, ok := valueAt(values, "areas", area.ID, "collapse_at"); ok {
		attributes += ` data-margo-collapse-at="` + html.EscapeString(fmt.Sprint(collapse)) + `"`
	}
	return attributes
}

func frameName(schema FrameSchema) string {
	if len(schema.Layout.Wide.SourceOrder) == 1 {
		return "main"
	}
	for _, name := range builtinFrameNames {
		candidate, _ := builtinSchema(name)
		if strings.Join(candidate.Layout.Wide.SourceOrder, "\x00") == strings.Join(schema.Layout.Wide.SourceOrder, "\x00") {
			return name
		}
	}
	return "custom"
}

func findArea(schema FrameSchema, id string) AreaDescriptor {
	for _, area := range schema.Areas {
		if area.ID == id {
			return area
		}
	}
	return AreaDescriptor{}
}

func bindingRank(order []string, binding AreaBinding) int {
	for index, kind := range order {
		if kind == binding.Kind {
			return index
		}
	}
	return len(order) + 1
}
