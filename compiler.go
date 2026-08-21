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
	registry    extensionRegistry
}

type sourceNormalization struct {
	metadata    Metadata
	diagnostics []Diagnostic
	parsed      any
	sourceBytes int64
	// Check owns its richer asset diagnostics and skips compile-only remote
	// image rejection when reusing policy evaluation.
	skipRemoteImages bool
}

var normalizeSource = func(source Source) (sourceNormalization, error) {
	return sourceNormalization{metadata: Metadata{Name: source.Name, BaseURL: source.BaseURL}}, nil
}

var evaluatePolicy = defaultEvaluatePolicy

var renderDocumentBytes = renderExtensionPlanBytes

// New freezes options and returns a reusable compiler.
func New(options ...Option) *Compiler {
	config := newCompilerConfig()
	if err := installDefaultExtensions(&config); err != nil {
		panic(err)
	}
	if err := applyOptions(&config, options); err != nil {
		panic(err)
	}
	config = config.clone()
	return &Compiler{
		config:      config,
		fingerprint: compilerConfigFingerprint(config.values),
		registry:    registryFromConfig(config),
	}
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
	registry := c.registry.clone()
	c.mu.RUnlock()
	snapshot := source.clone()
	inputLimit, err := configuredInputLimit(config)
	if err != nil {
		return nil, err
	}
	if int64(len(snapshot.Content)) > inputLimit {
		return nil, policyDiagnostic("policy.resource.document_too_large", "document exceeds the maximum byte limit")
	}
	normalized, err := normalizeSource(snapshot)
	if err != nil {
		return nil, err
	}
	normalized.sourceBytes = int64(len(snapshot.Content))
	effectivePolicy, err := evaluatePolicy(config, normalized)
	if err != nil {
		return nil, err
	}
	sourceHash := sha256.Sum256(snapshot.Content)
	docFingerprint := documentFingerprint(snapshot, fingerprint, config.values)
	plan, err := buildRenderPlan(snapshot, normalized, registry, fingerprint, docFingerprint, effectivePolicy)
	if err != nil {
		return nil, err
	}
	return &Document{
		source:              snapshot,
		sourceHash:          sourceHash,
		compilerFingerprint: fingerprint,
		documentFingerprint: docFingerprint,
		metadata:            normalized.metadata,
		diagnostics:         cloneDiagnostics(normalized.diagnostics),
		parsed:              normalized.parsed,
		effectivePolicy:     effectivePolicy,
		plan:                plan,
		htmlRequirements:    HTMLRequirements{requirements: plan.htmlRequirements.List()},
	}, nil
}

// SupportsRenderIDAllocator reports whether every registered extension has
// opted into the deck render-wide identity capability.
func (c *Compiler) SupportsRenderIDAllocator() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, registration := range c.registry.registrations {
		found := false
		for _, capability := range registration.Identity.Capabilities {
			if capability == "NamespacedIDsV1" {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
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
	if fingerprint != document.compilerFingerprint || fingerprint != document.plan.compilerFingerprint {
		return nil, ErrCompilerDocumentMismatch
	}
	renderConfig, err := applyRenderOptions(options)
	if err != nil {
		return nil, err
	}
	bytes, err := renderDocumentBytes(ctx, document, options)
	if err != nil {
		return nil, err
	}
	runtimeTasks, err := runtimeTaskTemplates(document.plan.clone())
	if err != nil {
		return nil, err
	}
	return &RenderResult{
		content: templ.ComponentFunc(func(_ context.Context, out io.Writer) error {
			_, err := out.Write(bytes)
			return err
		}),
		metadata:            document.Metadata(),
		assets:              document.Assets(),
		diagnostics:         document.Diagnostics(),
		htmlRequirements:    document.projectedHTMLRequirements(),
		documentFingerprint: document.documentFingerprint,
		runtimeTasks:        cloneRuntimeTaskTemplates(runtimeTasks),
		target:              renderTarget(renderConfig),
	}, nil
}

func renderExtensionPlanBytes(ctx context.Context, document *Document, options []RenderOption) ([]byte, error) {
	if _, err := applyRenderOptions(options); err != nil {
		return nil, err
	}
	return executeRenderPlan(ctx, document.plan.clone())
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
