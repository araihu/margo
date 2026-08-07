package margo

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/a-h/templ"
)

var (
	ErrNilDocument              = errors.New("margo: nil document")
	ErrCompilerDocumentMismatch = errors.New("compiler.document_config_mismatch")
)

// Compiler owns one immutable configuration snapshot and is safe for
// concurrent Compile and Render calls.
type Compiler struct {
	mu          sync.RWMutex
	config      compilerConfig
	fingerprint CompilerConfigFingerprint
}

type sourceNormalization struct {
	metadata    Metadata
	diagnostics []Diagnostic
	parsed      any
}

var normalizeSource = func(source Source) (sourceNormalization, error) {
	return sourceNormalization{metadata: Metadata{Name: source.Name, BaseURL: source.BaseURL}}, nil
}

// New freezes options and returns a reusable compiler.
func New(options ...Option) *Compiler {
	config := newCompilerConfig()
	if err := applyOptions(&config, options); err != nil {
		panic(err)
	}
	config = config.clone()
	return &Compiler{config: config, fingerprint: compilerConfigFingerprint(config.values)}
}

// Compile snapshots source and returns an opaque immutable document.
func (c *Compiler) Compile(ctx context.Context, source Source) (*Document, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errors.New("margo: nil compiler")
	}
	c.mu.RLock()
	config := c.config.clone()
	fingerprint := c.fingerprint
	c.mu.RUnlock()
	snapshot := source.clone()
	normalized, err := normalizeSource(snapshot)
	if err != nil {
		return nil, err
	}
	sourceHash := sha256.Sum256(snapshot.Content)
	docFingerprint := documentFingerprint(snapshot, fingerprint, config.values)
	return &Document{
		source:              snapshot,
		sourceHash:          sourceHash,
		compilerFingerprint: fingerprint,
		documentFingerprint: docFingerprint,
		metadata:            normalized.metadata,
		diagnostics:         cloneDiagnostics(normalized.diagnostics),
		parsed:              normalized.parsed,
	}, nil
}

// Render creates an immutable result. Semantic rendering is added by the
// later render-plan task; this early contract still enforces compiler binding.
func (c *Compiler) Render(ctx context.Context, document *Document, options ...RenderOption) (*RenderResult, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if document == nil {
		return nil, ErrNilDocument
	}
	if c == nil {
		return nil, errors.New("margo: nil compiler")
	}
	c.mu.RLock()
	fingerprint := c.fingerprint
	c.mu.RUnlock()
	if fingerprint != document.compilerFingerprint {
		return nil, ErrCompilerDocumentMismatch
	}
	if _, err := applyRenderOptions(options); err != nil {
		return nil, err
	}
	return &RenderResult{
		content:     templ.ComponentFunc(func(context.Context, io.Writer) error { return nil }),
		metadata:    document.Metadata(),
		assets:      document.Assets(),
		diagnostics: document.Diagnostics(),
	}, nil
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("margo: compile canceled: %w", err)
	}
	return nil
}
