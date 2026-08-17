package ssg

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

type compositionTestFrame struct {
	name   string
	schema FrameSchema
}

func (frame compositionTestFrame) Schema(_ FrameContext) (FrameSchema, error) {
	return frame.schema, nil
}

func (frame compositionTestFrame) Render(input FrameInput) (FrameOutput, error) {
	var markup strings.Builder
	markup.WriteString(`<section data-frame="` + frame.name + `">`)
	for mountID, child := range input.ChildrenByMount {
		markup.WriteString(`<div data-mount="` + mountID + `" data-target="` + child.QualifiedTarget + `">`)
		if err := child.Fragment.Render(context.Background(), &markup); err != nil {
			return FrameOutput{}, err
		}
		markup.WriteString(`</div>`)
	}
	markup.WriteString(`</section>`)
	return FrameOutput{
		Fragment: templ.ComponentFunc(func(_ context.Context, writer io.Writer) error {
			_, err := io.WriteString(writer, markup.String())
			return err
		}),
		SchemaHash: input.SchemaHash,
	}, nil
}

func TestResolveCompositionHashesStructureAndRendersBottomUp(t *testing.T) {
	rootSchema := compositionTestSchema(true, []FrameMountDescriptor{{
		ID: "sidebar", HostArea: "top-nav", Target: "child-target", Required: true, Exclusive: true, Contract: FrameContract,
	}})
	childSchema := compositionTestSchema(false, nil)
	frames := map[string]Frame{
		"root":  compositionTestFrame{name: "root", schema: rootSchema},
		"child": compositionTestFrame{name: "child", schema: childSchema},
	}
	resolver := func(_ FrameContext, selector string, _ Values) (Frame, error) {
		return frames[selector], nil
	}
	composition, err := ResolveComposition(FrameComposition{Root: FrameNode{
		InstanceID: "root", Selector: "root", Children: []FrameNode{{
			InstanceID: "child", MountID: "sidebar", Selector: "child", Values: Values{"density": "compact"},
		}},
	}}, FrameContext{Locale: "en", Direction: "ltr"}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if composition.Hash == "" || len(composition.Root.Children) != 1 {
		t.Fatalf("composition = %#v", composition)
	}
	child := composition.Root.Children[0]
	if got, want := strings.Join(child.CompositionPath, "/"), "child"; got != want {
		t.Fatalf("child path = %q, want %q", got, want)
	}
	document, err := NewAreaBinding(composition.Root.SchemaHash, "index.html", BindingSpec{Kind: "document", Area: "main-content"}, 0, templ.Raw(`<article><h1>Home</h1></article>`))
	if err != nil {
		t.Fatal(err)
	}
	output, err := composition.Render(map[string]map[string][]AreaBinding{
		"root": {"main-content": {document}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var rendered strings.Builder
	if err := output.Fragment.Render(context.Background(), &rendered); err != nil {
		t.Fatal(err)
	}
	markup := rendered.String()
	if !strings.Contains(markup, `data-frame="root"`) || !strings.Contains(markup, `data-frame="child"`) || !strings.Contains(markup, `data-target="child--child-target"`) {
		t.Fatalf("rendered composition = %s", markup)
	}

	changed, err := ResolveComposition(FrameComposition{Root: FrameNode{
		InstanceID: "root", Selector: "root", Children: []FrameNode{{
			InstanceID: "child", MountID: "sidebar", Selector: "child", Values: Values{"density": "spacious"},
		}},
	}}, FrameContext{Locale: "en", Direction: "ltr"}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Hash == composition.Hash {
		t.Fatal("structural value change did not change composition hash")
	}
}

func TestResolveCompositionRejectsMountAndDocumentViolations(t *testing.T) {
	rootSchema := compositionTestSchema(true, []FrameMountDescriptor{{
		ID: "sidebar", HostArea: "top-nav", Target: "child-target", Required: true, Exclusive: true, Contract: FrameContract,
	}})
	childSchema := compositionTestSchema(false, nil)
	documentChildSchema := compositionTestSchema(true, nil)
	resolver := func(_ FrameContext, selector string, _ Values) (Frame, error) {
		switch selector {
		case "root":
			return compositionTestFrame{name: selector, schema: rootSchema}, nil
		case "child":
			return compositionTestFrame{name: selector, schema: childSchema}, nil
		case "document-child":
			return compositionTestFrame{name: selector, schema: documentChildSchema}, nil
		default:
			return nil, nil
		}
	}
	tests := []struct {
		name string
		root FrameNode
		want string
	}{
		{name: "required mount missing", root: FrameNode{InstanceID: "root", Selector: "root"}, want: "mount_required"},
		{name: "unresolved mount", root: FrameNode{InstanceID: "root", Selector: "root", Children: []FrameNode{{InstanceID: "child", MountID: "missing", Selector: "child"}}}, want: "mount_unresolved"},
		{name: "duplicate mount", root: FrameNode{InstanceID: "root", Selector: "root", Children: []FrameNode{{InstanceID: "one", MountID: "sidebar", Selector: "child"}, {InstanceID: "two", MountID: "sidebar", Selector: "child"}}}, want: "mount_duplicate"},
		{name: "document child", root: FrameNode{InstanceID: "root", Selector: "root", Children: []FrameNode{{InstanceID: "child", MountID: "sidebar", Selector: "document-child"}}}, want: "document_child"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ResolveComposition(FrameComposition{Root: test.root}, FrameContext{}, resolver); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestRenderCompositionRejectsExclusiveSemanticCollision(t *testing.T) {
	rootSchema := compositionTestSchema(true, []FrameMountDescriptor{{
		ID: "sidebar", HostArea: "top-nav", Target: "child-target", Required: false, Exclusive: true, Contract: FrameContract,
	}})
	childSchema := compositionTestSchema(false, nil)
	resolver := func(_ FrameContext, selector string, _ Values) (Frame, error) {
		if selector == "root" {
			return compositionTestFrame{name: "root", schema: rootSchema}, nil
		}
		return compositionTestFrame{name: "child", schema: childSchema}, nil
	}
	composition, err := ResolveComposition(FrameComposition{Root: FrameNode{
		InstanceID: "root", Selector: "root", Children: []FrameNode{{InstanceID: "child", MountID: "sidebar", Selector: "child"}},
	}}, FrameContext{}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	navigation, err := NewAreaBinding(composition.Root.SchemaHash, "index.html", BindingSpec{Kind: "navigation", Area: "top-nav"}, 0, templ.Raw(`<nav>home</nav>`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := composition.Render(map[string]map[string][]AreaBinding{
		"root": {"main-content": {mustDocumentBinding(t, composition.Root.SchemaHash)}, "top-nav": {navigation}},
	}); err == nil || !strings.Contains(err.Error(), "mount_collision") {
		t.Fatalf("collision error = %v", err)
	}
}

func mustDocumentBinding(t *testing.T, schemaHash string) AreaBinding {
	t.Helper()
	binding, err := NewAreaBinding(schemaHash, "index.html", BindingSpec{Kind: "document", Area: "main-content"}, 0, templ.Raw(`<article><h1>Home</h1></article>`))
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func compositionTestSchema(document bool, mounts []FrameMountDescriptor) FrameSchema {
	areas := []AreaDescriptor{}
	order := map[string][]string{}
	if document {
		areas = append(areas,
			AreaDescriptor{ID: "main-content", Role: "document", Required: true, Accepts: []string{"document"}, Target: "margo-document"},
			AreaDescriptor{ID: "top-nav", Accepts: []string{"navigation"}, Target: "top-nav"},
		)
		order["main-content"] = []string{"document"}
		order["top-nav"] = []string{"navigation"}
	} else {
		areas = append(areas, AreaDescriptor{ID: "child-content", Accepts: []string{"widget"}, Target: "child-content"})
		order["child-content"] = []string{"widget"}
	}
	for _, mount := range mounts {
		order[mount.HostArea] = append(order[mount.HostArea], "mount:"+mount.ID)
	}
	ids := make([]string, 0, len(areas))
	for _, area := range areas {
		ids = append(ids, area.ID)
	}
	placementFor := func() FramePlacement {
		regions := make([]FrameRegion, 0, len(ids))
		for index, id := range ids {
			regions = append(regions, FrameRegion{Area: id, RowStart: index + 1, RowEnd: index + 2, ColumnStart: 1, ColumnEnd: 2, Collapse: "none"})
		}
		rows := make([]string, len(ids))
		for index := range rows {
			rows[index] = "auto"
		}
		return FramePlacement{Rows: rows, Columns: []string{"fluid"}, Regions: regions, SourceOrder: append([]string(nil), ids...)}
	}
	return FrameSchema{
		Contract: FrameContract, Areas: areas, Mounts: mounts,
		Options: func() []FrameOptionDescriptor {
			if document {
				return nil
			}
			return []FrameOptionDescriptor{{Path: "density", Type: "enum", Default: "compact", Allowed: []string{"compact", "spacious"}, Description: "Test child density."}}
		}(),
		Layout: FrameLayoutDescriptor{
			Wide: placementFor(), Mid: placementFor(), Narrow: placementFor(),
			MainMeasure: "measure", GapToken: "gap", MinimumReadingBlock: "minimum",
			Breakpoints: []ContentBreakpoint{{Name: "narrow", MinCSSPx: 0, MaxCSSPx: intPtr(720)}, {Name: "mid", MinCSSPx: 720, MaxCSSPx: intPtr(1100)}, {Name: "wide", MinCSSPx: 1100}},
		},
		BindingOrder: order,
	}
}
