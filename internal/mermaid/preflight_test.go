package mermaid

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestMermaidFrontmatterRejectsEveryKeyBeforeTaskCreation(t *testing.T) {
	for _, name := range []string{"frontmatter-title.mmd", "frontmatter-config.mmd", "frontmatter-unknown.mmd"} {
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join("..", "..", "testdata", "mermaid", "configuration", name))
			if err != nil {
				t.Fatal(err)
			}
			descriptor, err := Compile(source, 0)
			requireConfigurationForbidden(t, err)
			var diagnostic *DiagnosticError
			errors.As(err, &diagnostic)
			if diagnostic.Offset < 0 || diagnostic.Offset >= len(source) {
				t.Fatalf("diagnostic offset = %d", diagnostic.Offset)
			}
			if descriptor != (TaskDescriptor{}) {
				t.Fatalf("descriptor created for forbidden frontmatter: %#v", descriptor)
			}
		})
	}
}

func TestMermaidRejectsInitAndInitializeTokenVariants(t *testing.T) {
	for _, name := range []string{"directive-init.mmd", "directive-initialize-whitespace.mmd"} {
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join("..", "..", "testdata", "mermaid", "configuration", name))
			if err != nil {
				t.Fatal(err)
			}
			descriptor, err := Compile(source, 0)
			requireConfigurationForbidden(t, err)
			if descriptor != (TaskDescriptor{}) {
				t.Fatalf("descriptor created for forbidden directive: %#v", descriptor)
			}
		})
	}
}

func TestMermaidStrictAcceptsOrdinarySourceAndKeepsConfigurationHash(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "testdata", "mermaid", "configuration", "strict.mmd"))
	if err != nil {
		t.Fatal(err)
	}
	before := StrictConfigurationHash()
	if _, err := Compile(source, 0); err != nil {
		t.Fatal(err)
	}
	after := StrictConfigurationHash()
	if after != before {
		t.Fatal("strict configuration hash changed during compile")
	}
}

func requireConfigurationForbidden(t *testing.T, err error) {
	t.Helper()
	var diagnostic *DiagnosticError
	if !errors.As(err, &diagnostic) {
		t.Fatalf("error = %v, want DiagnosticError", err)
	}
	if diagnostic.Code != ConfigurationForbiddenCode {
		t.Fatalf("diagnostic code = %q, want %q", diagnostic.Code, ConfigurationForbiddenCode)
	}
}
