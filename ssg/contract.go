// Package ssg contains the layout-neutral contract used by Margo static sites.
//
// The package deliberately knows nothing about Goshtoso components, Markdown
// routing, or document metadata. A site binds semantic fragments to a
// discovered frame or shell schema; the selected layout only places them.
package ssg

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

const (
	FrameContract = "margo.ssg.frame/v1"
	ShellContract = "margo.ssg.shell/v1"
	DocsProfile   = "margo-docs"
)

// Values are structural options supplied to a frame or shell.
type Values map[string]any

type SwapMode string

const (
	SwapInnerHTML   SwapMode = "innerHTML"
	SwapOuterHTML   SwapMode = "outerHTML"
	SwapBeforeBegin SwapMode = "beforebegin"
	SwapAfterBegin  SwapMode = "afterbegin"
	SwapBeforeEnd   SwapMode = "beforeend"
	SwapAfterEnd    SwapMode = "afterend"
)

type SlotDescriptor struct {
	ID      string   `json:"id"`
	Accepts []string `json:"accepts"`
	Order   int      `json:"order"`
}

type AreaDescriptor struct {
	ID                string           `json:"id"`
	Role              string           `json:"role,omitempty"`
	Required          bool             `json:"required"`
	Multiple          bool             `json:"multiple"`
	MaxBindings       int              `json:"maxBindings,omitempty"`
	MaxBindingsByKind map[string]int   `json:"maxBindingsByKind,omitempty"`
	Accepts           []string         `json:"accepts"`
	Slots             []SlotDescriptor `json:"slots,omitempty"`
	Target            string           `json:"target,omitempty"`
	Triggers          []string         `json:"triggers,omitempty"`
	AllowedSwaps      []SwapMode       `json:"allowedSwaps,omitempty"`
	Live              string           `json:"live,omitempty"`
	Focus             string           `json:"focus,omitempty"`
	Swap              SwapMode         `json:"swap,omitempty"`
}

type FrameOptionDescriptor struct {
	Path        string   `json:"path"`
	Type        string   `json:"type"`
	Default     any      `json:"default,omitempty"`
	Allowed     []string `json:"allowed,omitempty"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
	Description string   `json:"description"`
}

type FrameMountDescriptor struct {
	ID        string `json:"id"`
	HostArea  string `json:"hostArea"`
	Target    string `json:"target"`
	Required  bool   `json:"required"`
	Exclusive bool   `json:"exclusive"`
	Contract  string `json:"contract"`
}

type FrameRegion struct {
	Area        string `json:"area"`
	RowStart    int    `json:"rowStart"`
	RowEnd      int    `json:"rowEnd"`
	ColumnStart int    `json:"columnStart"`
	ColumnEnd   int    `json:"columnEnd"`
	Collapse    string `json:"collapse"`
}

type FramePlacement struct {
	Rows        []string      `json:"rows"`
	Columns     []string      `json:"columns"`
	Regions     []FrameRegion `json:"regions"`
	SourceOrder []string      `json:"sourceOrder"`
}

type ContentBreakpoint struct {
	Name     string `json:"name"`
	MinCSSPx int    `json:"minCSSPx"`
	MaxCSSPx *int   `json:"maxCSSPx,omitempty"`
}

type FrameLayoutDescriptor struct {
	Wide                FramePlacement      `json:"wide"`
	Mid                 FramePlacement      `json:"mid"`
	Narrow              FramePlacement      `json:"narrow"`
	MainMeasure         string              `json:"mainMeasure"`
	GapToken            string              `json:"gapToken"`
	MinimumReadingBlock string              `json:"minimumReadingBlock"`
	StickyOrder         []string            `json:"stickyOrder"`
	Breakpoints         []ContentBreakpoint `json:"breakpoints"`
	DrawerInlineSize    string              `json:"drawerInlineSize,omitempty"`
	DrawerMaxInlineSize string              `json:"drawerMaxInlineSize,omitempty"`
}

type ResourceRequirement struct {
	Placement  string            `json:"placement"`
	Kind       string            `json:"kind"`
	URL        string            `json:"url,omitempty"`
	Inline     string            `json:"inline,omitempty"`
	Integrity  string            `json:"integrity,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type FrameSchema struct {
	Contract        string                  `json:"contract"`
	Areas           []AreaDescriptor        `json:"areas"`
	Options         []FrameOptionDescriptor `json:"options,omitempty"`
	Mounts          []FrameMountDescriptor  `json:"mounts,omitempty"`
	Layout          FrameLayoutDescriptor   `json:"layout"`
	BindingDefaults map[string]string       `json:"bindingDefaults,omitempty"`
	BindingOrder    map[string][]string     `json:"bindingOrder,omitempty"`
	Resources       []ResourceRequirement   `json:"resources,omitempty"`
}

type ThemeContext struct {
	Name             string   `json:"name"`
	Available        []string `json:"available"`
	AllowSwitchTheme bool     `json:"allowSwitchTheme"`
	ColorMode        string   `json:"colorMode"`
}

type FrameContext struct {
	Locale          string       `json:"locale"`
	Direction       string       `json:"direction"`
	Theme           ThemeContext `json:"theme"`
	Profile         string       `json:"profile"`
	Root            bool         `json:"root"`
	InstanceID      string       `json:"instanceID"`
	CompositionPath []string     `json:"compositionPath,omitempty"`
}

type FrameInput struct {
	SchemaHash          string                       `json:"schemaHash"`
	RootCompositionHash string                       `json:"rootCompositionHash"`
	InstanceID          string                       `json:"instanceID"`
	CompositionPath     []string                     `json:"compositionPath,omitempty"`
	Values              Values                       `json:"values,omitempty"`
	Bindings            map[string][]AreaBinding     `json:"bindings"`
	ChildrenByMount     map[string]FrameChildBinding `json:"childrenByMount,omitempty"`
}

type FrameChildBinding struct {
	MountID         string
	HostArea        string
	Target          string
	QualifiedTarget string
	InstanceID      string
	CompositionPath []string
	SchemaHash      string
	Digest          string
	Fragment        templ.Component
}

type FrameOutput struct {
	Fragment   templ.Component
	Assets     AssetSet
	SchemaHash string
}

type Frame interface {
	Schema(FrameContext) (FrameSchema, error)
	Render(FrameInput) (FrameOutput, error)
}

type ShellSchema struct {
	Contract  string                       `json:"contract"`
	Areas     []AreaDescriptor             `json:"areas"`
	Locales   []string                     `json:"locales"`
	Labels    map[string]map[string]string `json:"labels"`
	Resources []ResourceRequirement        `json:"resources,omitempty"`
}

type Route struct {
	Path   string `json:"path"`
	Locale string `json:"locale"`
}

type PageMetadata struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Canonical   string `json:"canonical"`
}

type Brand struct {
	Name      string `json:"name"`
	LogoURL   string `json:"logoURL"`
	IconURL   string `json:"iconURL"`
	SocialURL string `json:"socialURL"`
}

type LocaleContext struct {
	Current    string            `json:"current"`
	Supported  []string          `json:"supported"`
	Alternates map[string]string `json:"alternates,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

type AssetSet struct {
	Paths []string `json:"paths,omitempty"`
}

type AreaBinding struct {
	Kind            string          `json:"kind"`
	CompositionPath []string        `json:"compositionPath,omitempty"`
	Slot            string          `json:"slot,omitempty"`
	Token           string          `json:"token"`
	Digest          string          `json:"digest"`
	Component       templ.Component `json:"-"`
}

type BindingSpec struct {
	Kind            string         `json:"kind"`
	CompositionPath []string       `json:"compositionPath,omitempty"`
	Area            string         `json:"area"`
	Slot            string         `json:"slot,omitempty"`
	Props           map[string]any `json:"props,omitempty"`
}

type FrameNode struct {
	InstanceID string      `json:"instanceID"`
	MountID    string      `json:"mountID,omitempty"`
	Selector   string      `json:"selector"`
	Values     Values      `json:"values,omitempty"`
	Children   []FrameNode `json:"children,omitempty"`
}

type FrameComposition struct {
	Root FrameNode `json:"root"`
}

type ShellContext struct {
	Locale    string       `json:"locale"`
	Direction string       `json:"direction"`
	Theme     ThemeContext `json:"theme"`
}

type ShellInput struct {
	Route      Route                    `json:"route"`
	Metadata   PageMetadata             `json:"metadata"`
	Brand      Brand                    `json:"brand"`
	Navigation any                      `json:"navigation"`
	Locale     LocaleContext            `json:"locale"`
	Theme      ThemeContext             `json:"theme"`
	Assets     AssetSet                 `json:"assets"`
	SchemaHash string                   `json:"schemaHash"`
	Bindings   map[string][]AreaBinding `json:"bindings"`
}

type ShellOutput struct {
	Fragment   templ.Component
	Assets     AssetSet
	SchemaHash string
}

type Shell interface {
	Schema(ShellContext) (ShellSchema, error)
	Render(ShellInput) (ShellOutput, error)
}

// Component is a small helper for layout adapters that need a context-aware
// fragment without exposing templ implementation details in their contract.
func Component(render func(context.Context, io.Writer) error) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		return render(ctx, writer)
	})
}
