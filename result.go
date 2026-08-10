package margo

import "github.com/a-h/templ"

// RenderResult is an immutable render projection safe for concurrent access.
type RenderResult struct {
	content             templ.Component
	metadata            Metadata
	assets              AssetSet
	diagnostics         []Diagnostic
	htmlRequirements    HTMLRequirements
	documentFingerprint DocumentFingerprint
	runtimeTasks        []runtimeTaskTemplate
}

// Content returns the immutable templ component.
func (r *RenderResult) Content() templ.Component {
	if r == nil {
		return nil
	}
	return r.content
}

// Metadata returns a defensive metadata copy.
func (r *RenderResult) Metadata() Metadata {
	if r == nil {
		return Metadata{}
	}
	return r.metadata.clone()
}

// Assets returns a defensive asset copy.
func (r *RenderResult) Assets() AssetSet {
	if r == nil {
		return AssetSet{}
	}
	return r.assets.clone()
}

// Diagnostics returns a defensive diagnostic slice.
func (r *RenderResult) Diagnostics() []Diagnostic {
	if r == nil {
		return nil
	}
	return cloneDiagnostics(r.diagnostics)
}

func (r *RenderResult) projectedHTMLRequirements() HTMLRequirements {
	if r == nil {
		return HTMLRequirements{}
	}
	return HTMLRequirements{requirements: r.htmlRequirements.List()}
}
