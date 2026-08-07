package margo

import (
	"context"
	"io"
)

// ExtensionIdentity is the stable, serialized identity of one registered
// extension. ConfigurationHash is optional for the small root fixtures but is
// included in the compiler fingerprint whenever supplied.
type ExtensionIdentity struct {
	Name              string   `json:"name"`
	Version           string   `json:"version"`
	ConfigurationHash string   `json:"configurationHash,omitempty"`
	Capabilities      []string `json:"capabilities,omitempty"`
}

// ExtensionRegistration binds one immutable factory to its owned fences.
type ExtensionRegistration struct {
	Identity ExtensionIdentity
	Fences   []string
	Factory  ExtensionFactory
}

// RenderContext is the only root-to-extension policy delivery seam. It is a
// value so a session cannot mutate the compiler or another render operation.
type RenderContext struct {
	EffectivePolicy EffectivePolicy
}

// SourcePosition identifies a source location without exposing Goldmark
// segments as a public API.
type SourcePosition struct {
	Source string `json:"source"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// ExtensionNode is an immutable detached fence payload.
type ExtensionNode struct {
	Fence   string
	Payload []byte
	Source  SourcePosition
}

func (n ExtensionNode) clone() ExtensionNode {
	n.Payload = append([]byte(nil), n.Payload...)
	return n
}

// ExtensionSession is a per-render instance returned by a factory.
type ExtensionSession interface {
	Render(context.Context, ExtensionNode, io.Writer) error
}

// ExtensionFactory creates an independent render session for one operation.
type ExtensionFactory func(RenderContext) (ExtensionSession, error)
