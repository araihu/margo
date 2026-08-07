package margo

// Metadata is the immutable normalized metadata projection exposed by a
// RenderResult. Additional frontmatter fields are added by the parser task.
type Metadata struct {
	Name        string
	BaseURL     string
	Title       string
	Description string
}

func (m Metadata) clone() Metadata { return m }

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
	SeverityInfo  Severity = "info"
	SeverityError Severity = "error"
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
