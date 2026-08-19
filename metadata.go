package margo

// Metadata is the immutable normalized metadata projection exposed by a
// RenderResult. Additional frontmatter fields are added by the parser task.
type Metadata struct {
	Name        string
	BaseURL     string
	Title       string
	Description string
	Language    string
	Slug        string
	Authors     []string
	PublishedAt string
	ModifiedAt  string
	Tags        []string
	Margo       DocumentPreferences
	Additional  map[string]any
}

type DocumentPreferences struct {
	Page    *PagePreference
	Actions *PageActions
	Site    *SitePreference
}

type SitePreference struct {
	Layout string
}

// PDFMode selects how a site's PDF action is fulfilled.
type PDFMode string

const (
	PDFModePreRendered PDFMode = "pre-rendered"
	PDFModeClient      PDFMode = "client"
)

// PageActions selects optional artifacts and controls emitted by a site
// generator. PDF publication also retains the Markdown source for the page.
type PageActions struct {
	Markdown bool    `json:"markdown,omitempty"`
	PDF      bool    `json:"pdf,omitempty"`
	PDFMode  PDFMode `json:"pdfMode,omitempty"`
}

func (actions PageActions) EffectivePDFMode() PDFMode {
	if actions.PDFMode == "" {
		return PDFModePreRendered
	}
	return actions.PDFMode
}

func (actions PageActions) UsesClientPDF() bool {
	return actions.PDF && actions.EffectivePDFMode() == PDFModeClient
}

type PagePreference struct {
	Size          string
	Orientation   string
	ImageOverflow string
	Margins       *PageMarginPreference
}

// PageMarginPreference keeps every side optional so an author can override
// one side without discarding the built-in values for the others. Pointers
// distinguish an omitted side from an explicit zero used for full bleed.
type PageMarginPreference struct {
	Top    *float64
	Right  *float64
	Bottom *float64
	Left   *float64
}

func (m Metadata) clone() Metadata {
	m.Authors = append([]string(nil), m.Authors...)
	m.Tags = append([]string(nil), m.Tags...)
	if m.Margo.Page != nil {
		page := *m.Margo.Page
		if page.Margins != nil {
			margins := *page.Margins
			margins.Top = cloneFloat64Pointer(margins.Top)
			margins.Right = cloneFloat64Pointer(margins.Right)
			margins.Bottom = cloneFloat64Pointer(margins.Bottom)
			margins.Left = cloneFloat64Pointer(margins.Left)
			page.Margins = &margins
		}
		m.Margo.Page = &page
	}
	if m.Margo.Actions != nil {
		actions := *m.Margo.Actions
		m.Margo.Actions = &actions
	}
	if m.Margo.Site != nil {
		site := *m.Margo.Site
		m.Margo.Site = &site
	}
	if len(m.Additional) > 0 {
		m.Additional = cloneStringAnyMap(m.Additional)
	}
	return m
}

func cloneFloat64Pointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

// AssetSet is the defensive asset identity projection for a result.
type AssetSet struct {
	IDs []string
}

func (a AssetSet) clone() AssetSet {
	return AssetSet{IDs: append([]string(nil), a.IDs...)}
}

// Severity is the stable diagnostic severity vocabulary.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Diagnostic is a stable, serializable problem projection.
type Diagnostic struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Source   string   `json:"source"`
	Line     int      `json:"line"`
	Column   int      `json:"column"`
	Pointer  string   `json:"pointer"`
	Message  string   `json:"message"`
	Hint     string   `json:"hint"`
}

func cloneDiagnostics(in []Diagnostic) []Diagnostic {
	if len(in) == 0 {
		return nil
	}
	out := make([]Diagnostic, len(in))
	copy(out, in)
	return out
}
