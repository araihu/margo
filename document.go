package margo

import "crypto/sha256"

// Document is an immutable compiled source. Its internal representation is
// deliberately opaque so parser and policy details remain versioned internals.
type Document struct {
	source              Source
	sourceHash          [32]byte
	compilerFingerprint CompilerConfigFingerprint
	documentFingerprint DocumentFingerprint
	metadata            Metadata
	assets              AssetSet
	diagnostics         []Diagnostic
	parsed              any
	effectivePolicy     EffectivePolicy
	plan                renderPlan
	htmlRequirements    HTMLRequirements
}

func (d *Document) sourceBytesForTest() []byte {
	if d == nil {
		return nil
	}
	return append([]byte(nil), d.source.Content...)
}

// Metadata returns a defensive value copy.
func (d *Document) Metadata() Metadata {
	if d == nil {
		return Metadata{}
	}
	return d.metadata.clone()
}

// Assets returns a defensive value copy.
func (d *Document) Assets() AssetSet {
	if d == nil {
		return AssetSet{}
	}
	return d.assets.clone()
}

// Diagnostics returns a defensive slice copy.
func (d *Document) Diagnostics() []Diagnostic {
	if d == nil {
		return nil
	}
	return cloneDiagnostics(d.diagnostics)
}

func (d *Document) sourceDigest() [32]byte {
	if d == nil {
		return [32]byte{}
	}
	return d.sourceHash
}

func (d *Document) editorialHTMLRequirements() HTMLRequirements {
	if d == nil {
		return HTMLRequirements{}
	}
	return HTMLRequirements{requirements: d.htmlRequirements.List()}
}

func sourceDigest(content []byte) [32]byte { return sha256.Sum256(content) }
