package margo

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
)

type extensionSessionFunc func(context.Context, ExtensionNode, io.Writer) error

func (fn extensionSessionFunc) Render(ctx context.Context, node ExtensionNode, out io.Writer) error {
	return fn(ctx, node, out)
}

type testExtensionSession struct {
	mu    sync.Mutex
	nodes []ExtensionNode
}

func (s *testExtensionSession) Render(_ context.Context, node ExtensionNode, _ io.Writer) error {
	s.mu.Lock()
	s.nodes = append(s.nodes, node.clone())
	s.mu.Unlock()
	return nil
}

func TestRegistryRejectsDuplicateNames(t *testing.T) {
	registration := ExtensionRegistration{
		Identity: ExtensionIdentity{Name: "one", Version: "v1"},
		Factory:  func(RenderContext) (ExtensionSession, error) { return &testExtensionSession{}, nil },
	}
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate registration did not fail")
		}
	}()
	_ = New(WithExtension(registration), WithExtension(registration))
}

func TestExtensionRegistrationMutationIsFrozen(t *testing.T) {
	fences := []string{"demo"}
	registration := ExtensionRegistration{
		Identity: ExtensionIdentity{Name: "demo", Version: "v1", Capabilities: []string{"render"}},
		Fences:   fences,
		Factory:  func(RenderContext) (ExtensionSession, error) { return &testExtensionSession{}, nil },
	}
	compiler := New(WithExtension(registration))
	fences[0] = "changed"
	doc, err := compiler.Compile(context.Background(), Source{Name: "x.md", Content: []byte("```demo\npayload\n```")})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if _, err := compiler.Render(context.Background(), doc); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
}

func TestMissingChartsIntegrationFailsClosed(t *testing.T) {
	_, err := New().Compile(context.Background(), Source{Name: "chart.md", Content: []byte("```goshtosochart\n{}\n```")})
	if got := diagnosticCode(err); got != "extension.missing_integration" {
		t.Fatalf("diagnostic code = %q, err = %v", got, err)
	}
}

func TestExtensionRendersOnceAtMarkdownPosition(t *testing.T) {
	compiler := New(WithExtension(ExtensionRegistration{
		Identity: ExtensionIdentity{Name: "demo", Version: "v1"},
		Fences:   []string{"demo"},
		Factory: func(RenderContext) (ExtensionSession, error) {
			return extensionSessionFunc(func(_ context.Context, _ ExtensionNode, out io.Writer) error {
				_, err := io.WriteString(out, `<figure data-demo="true">rendered</figure>`)
				return err
			}), nil
		},
	}))
	document, err := compiler.Compile(context.Background(), Source{
		Name:    "ordered.md",
		Content: []byte("before\n\n```demo\npayload\n```\n\nafter\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, result.Content())
	before := strings.Index(markup, "<p>before</p>")
	figure := strings.Index(markup, `<figure data-demo="true">rendered</figure>`)
	after := strings.Index(markup, "<p>after</p>")
	if before < 0 || figure <= before || after <= figure {
		t.Fatalf("extension source order lost: %s", markup)
	}
	if strings.Count(markup, `data-demo="true"`) != 1 || strings.Contains(markup, "payload") {
		t.Fatalf("extension duplicated or fence leaked: %s", markup)
	}
}

func TestConcurrentExtensionSessions(t *testing.T) {
	var mu sync.Mutex
	created := 0
	compiler := New(WithExtension(ExtensionRegistration{
		Identity: ExtensionIdentity{Name: "demo", Version: "v1"},
		Fences:   []string{"demo"},
		Factory: func(RenderContext) (ExtensionSession, error) {
			mu.Lock()
			created++
			mu.Unlock()
			return &testExtensionSession{}, nil
		},
	}))
	doc, err := compiler.Compile(context.Background(), Source{Name: "x.md", Content: []byte("```demo\npayload\n```")})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	const renders = 8
	var wg sync.WaitGroup
	for i := 0; i < renders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := compiler.Render(context.Background(), doc); err != nil {
				t.Errorf("Render() error = %v", err)
			}
		}()
	}
	wg.Wait()
	mu.Lock()
	defer mu.Unlock()
	if created != renders {
		t.Fatalf("factory calls = %d, want %d", created, renders)
	}
}
