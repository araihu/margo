package margo

import (
	"context"
	"io"
	"sync/atomic"
	"testing"
)

type countingExtension struct{ calls atomic.Int64 }

func (e *countingExtension) factory(rc RenderContext) (ExtensionSession, error) {
	if rc.EffectivePolicy.OutputBytes <= 0 {
		panic("factory received invalid output policy")
	}
	e.calls.Add(1)
	return countingSession{}, nil
}

type countingSession struct{}

func (countingSession) Render(context.Context, ExtensionNode, io.Writer) error { return nil }

func TestDivergentCompilerRejectedBeforeHook(t *testing.T) {
	doc, err := New(WithTheme("goshtoso")).Compile(context.Background(), Source{Name: "x.md", Content: []byte("# x")})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	hook := &countingExtension{}
	_, err = New(WithTheme("minimal"), WithExtension(ExtensionRegistration{Identity: ExtensionIdentity{Name: "hook", Version: "v1"}, Factory: hook.factory})).Render(context.Background(), doc)
	if !isDiagnosticCode(err, "compiler.document_config_mismatch") {
		t.Fatalf("diagnostic code = %q, err = %v", diagnosticCode(err), err)
	}
	if got := hook.calls.Load(); got != 0 {
		t.Fatalf("hook calls = %d, want 0", got)
	}
}

func TestExtensionReceivesFrozenEffectiveOutputPolicy(t *testing.T) {
	var observed atomic.Int64
	extension := ExtensionRegistration{Identity: ExtensionIdentity{Name: "policy", Version: "v1"}, Factory: func(rc RenderContext) (ExtensionSession, error) {
		observed.Store(rc.EffectivePolicy.OutputBytes)
		return countingSession{}, nil
	}}
	compiler := New(WithHostPolicy(Policy{RawHTML: RawHTMLDeny, OutputBytes: 4096}), WithExtension(extension))
	doc, err := compiler.Compile(context.Background(), Source{Name: "policy.md", Content: []byte("# chart")})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if _, err = compiler.Render(context.Background(), doc); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if got := observed.Load(); got != 4096 {
		t.Fatalf("effective output bytes = %d, want 4096", got)
	}
}

func TestCompilerPolicyMutationAfterCompileFailsBeforeFactory(t *testing.T) {
	var calls atomic.Int64
	compiler := New(WithHostPolicy(Policy{RawHTML: RawHTMLDeny, OutputBytes: 4096}), WithExtension(ExtensionRegistration{
		Identity: ExtensionIdentity{Name: "policy", Version: "v1"},
		Factory:  func(RenderContext) (ExtensionSession, error) { calls.Add(1); return countingSession{}, nil },
	}))
	doc, err := compiler.Compile(context.Background(), Source{Name: "policy.md", Content: []byte("# chart")})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	mutateCompilerHostPolicyForTest(compiler, Policy{RawHTML: RawHTMLDeny, OutputBytes: 128})
	_, err = compiler.Render(context.Background(), doc)
	if !isDiagnosticCode(err, "compiler.document_config_mismatch") {
		t.Fatalf("diagnostic code = %q, err = %v", diagnosticCode(err), err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("factory calls = %d, want 0", got)
	}
}

func mutateCompilerHostPolicyForTest(compiler *Compiler, policy Policy) {
	compiler.mu.Lock()
	defer compiler.mu.Unlock()
	compiler.config.values["hostPolicy"] = policy
	compiler.fingerprint = compilerConfigFingerprint(compiler.config.values)
}
