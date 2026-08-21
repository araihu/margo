package deck

import "github.com/araihu/margo"

type Metadata struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Marp        *bool  `yaml:"marp"`
}

// HeadingDivider describes the accepted Margo Marpit-compatible heading
// divider. Scalar values use inclusive H1-through-HN semantics; Levels uses
// exact heading levels when Scalar is zero.
type HeadingDivider struct {
	Scalar int
	Levels []int
}

// BackgroundState is the typed background projection for one slide. Source,
// accessibility metadata, and sizing fields are reset atomically when a new
// background image is assigned.
type BackgroundState struct {
	Source     string
	Alt        string
	Decorative bool
	Position   string
	Repeat     string
	Size       string
}

// DirectiveState is the immutable, normalized state effective for a slide.
// Empty local fields mean that the theme default is active.
type DirectiveState struct {
	Theme           margo.ThemeName
	Lang            string
	ColorMode       margo.ColorMode
	HeadingDivider  HeadingDivider
	Size            string
	Composition     CompositionName
	Paginate        string
	Header          string
	Footer          string
	Classes         []string
	Color           string
	BackgroundColor string
	Background      BackgroundState
}

// LayoutSlot is one source-ordered structural layout slot.
type LayoutSlot struct {
	Name       string
	Markdown   []byte
	SourceLine int
}

// Layout is the normalized structural layout model. A nil layout denotes the
// ordinary unstructured slide path.
type Layout struct {
	Class string
	Slots []LayoutSlot
}

type Slide struct {
	ordinal     int
	id          string
	markdown    []byte
	directives  DirectiveState
	composition CompositionSpec
	notes       []string
	layout      *Layout
}

func (s Slide) Ordinal() int {
	return s.ordinal
}

func (s Slide) ID() string {
	return s.id
}

func (s Slide) Markdown() []byte {
	return append([]byte(nil), s.markdown...)
}

// Directives returns a defensive copy of the effective slide state.
func (s Slide) Directives() DirectiveState {
	return cloneDirectiveState(s.directives)
}

// Notes returns presenter notes in source order.
func (s Slide) Notes() []string {
	return append([]string(nil), s.notes...)
}

// Layout returns a defensive copy of the normalized structural layout.
func (s Slide) Layout() *Layout {
	return cloneLayout(s.layout)
}

// Composition returns a defensive copy of the resolved R1 composition.
func (s Slide) Composition() CompositionSpec {
	return cloneCompositionSpec(s.composition)
}

type Document struct {
	name       string
	metadata   Metadata
	directives DirectiveState
	slides     []Slide
}

func (d *Document) Metadata() Metadata {
	if d == nil {
		return Metadata{}
	}
	return cloneMetadata(d.metadata)
}

// Directives returns the final deck-wide global directive state.
func (d *Document) Directives() DirectiveState {
	if d == nil {
		return DirectiveState{}
	}
	return cloneDirectiveState(d.directives)
}

func (d *Document) Slides() []Slide {
	if d == nil {
		return nil
	}
	result := make([]Slide, len(d.slides))
	for index, slide := range d.slides {
		result[index] = Slide{
			ordinal:     slide.ordinal,
			id:          slide.id,
			markdown:    slide.Markdown(),
			directives:  cloneDirectiveState(slide.directives),
			composition: cloneCompositionSpec(slide.composition),
			notes:       slide.Notes(),
			layout:      slide.Layout(),
		}
	}
	return result
}

func defaultDirectiveState() DirectiveState {
	return DirectiveState{
		Theme:     margo.ThemeModern,
		Lang:      "en",
		ColorMode: margo.ColorModeLight,
		Size:      "16:9",
	}
}

func cloneDirectiveState(state DirectiveState) DirectiveState {
	state.Classes = append([]string(nil), state.Classes...)
	state.HeadingDivider.Levels = append([]int(nil), state.HeadingDivider.Levels...)
	return state
}

func cloneMetadata(metadata Metadata) Metadata {
	if metadata.Marp != nil {
		active := *metadata.Marp
		metadata.Marp = &active
	}
	return metadata
}

func cloneLayout(layout *Layout) *Layout {
	if layout == nil {
		return nil
	}
	clone := &Layout{Class: layout.Class, Slots: make([]LayoutSlot, len(layout.Slots))}
	for index, slot := range layout.Slots {
		clone.Slots[index] = LayoutSlot{
			Name:       slot.Name,
			Markdown:   append([]byte(nil), slot.Markdown...),
			SourceLine: slot.SourceLine,
		}
	}
	return clone
}
