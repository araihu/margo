package mermaid_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/araihu/margo"
	internalmermaid "github.com/araihu/margo/internal/mermaid"
)

func TestMermaidTaskDescriptorIsDeterministic(t *testing.T) {
	const wantProfileFingerprint = "fdcd7a02605775b63074d40b4786e3f8e29fa6f1e6ec2b060ae6ba44f365fe16"
	source := []byte("flowchart TD\n  A --> B\n")
	wantSource := sha256.Sum256(source)

	descriptor, err := internalmermaid.Compile(source, 7)
	if err != nil {
		t.Fatal(err)
	}
	if descriptor.Kind != "mermaid" {
		t.Fatalf("kind = %q, want mermaid", descriptor.Kind)
	}
	if descriptor.BlockID != "mermaid:00000007:"+hex.EncodeToString(wantSource[:]) {
		t.Fatalf("block ID = %q", descriptor.BlockID)
	}
	if descriptor.Input.SourceSHA256 != wantSource {
		t.Fatal("source SHA-256 mismatch")
	}
	if descriptor.Input.BlockOrdinal != 7 {
		t.Fatalf("block ordinal = %d, want 7", descriptor.Input.BlockOrdinal)
	}
	if descriptor.Input.RuntimeDigest != internalmermaid.RuntimeDigest {
		t.Fatalf("runtime digest = %q", descriptor.Input.RuntimeDigest)
	}
	if got := hex.EncodeToString(descriptor.Input.ProfileFingerprint[:]); got != wantProfileFingerprint {
		t.Fatalf("profile fingerprint = %s, want %s", got, wantProfileFingerprint)
	}
	if descriptor.ConfigurationHash != internalmermaid.StrictConfigurationHash() {
		t.Fatal("strict configuration hash mismatch")
	}

	source[0] = 'X'
	repeated, err := internalmermaid.Compile([]byte("flowchart TD\n  A --> B\n"), 7)
	if err != nil {
		t.Fatal(err)
	}
	if repeated != descriptor {
		t.Fatal("descriptor changed after caller source mutation")
	}
}

func TestMermaidStrictConfigurationHashIsPinned(t *testing.T) {
	const want = "38c4b58dcdb0fa27325e3b89b9732ea6665ee66396b0f4f3c52438c3cd0cbd2b"
	digest := internalmermaid.StrictConfigurationHash()
	if got := hex.EncodeToString(digest[:]); got != want {
		t.Fatalf("strict configuration hash = %s, want %s", got, want)
	}
}

func TestMermaidStrictIsDefaultRegisteredAtCompile(t *testing.T) {
	for _, source := range []string{
		"```mermaid\nflowchart TD\n  A --> B\n```\n",
		"---\ngoshtoso:\n  security:\n    mermaid: strict\n---\n```mermaid\nflowchart TD\n  A --> B\n```\n",
	} {
		compiler := margo.New()
		_, err := compiler.Compile(context.Background(), margo.Source{Name: "strict.md", Content: []byte(source)})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestMermaidStrictRejectsDocumentConfigurationOtherThanLiteralStrict(t *testing.T) {
	for _, value := range []string{"deny", "loose", "{}"} {
		t.Run(value, func(t *testing.T) {
			compiler := margo.New()
			source := "---\ngoshtoso:\n  security:\n    mermaid: " + value + "\n---\n```mermaid\nflowchart TD\n  A --> B\n```\n"
			_, err := compiler.Compile(context.Background(), margo.Source{Name: "mode.md", Content: []byte(source)})
			var diagnostic *margo.DiagnosticError
			if !errors.As(err, &diagnostic) {
				t.Fatalf("error = %v, want DiagnosticError", err)
			}
			if len(diagnostic.Diagnostics) != 1 || diagnostic.Diagnostics[0].Code != internalmermaid.ConfigurationForbiddenCode {
				t.Fatalf("diagnostics = %#v", diagnostic.Diagnostics)
			}
		})
	}
}

func TestMermaidRejectsLegacyInitializeBeforeRender(t *testing.T) {
	compiler := margo.New()
	_, err := compiler.Compile(context.Background(), margo.Source{
		Name:    "directive.md",
		Content: []byte("```mermaid\n  %%{ InItIaLiZe : { 'htmlLabels': true } }%%\nflowchart TD\n  A --> B\n```\n"),
	})
	var diagnostic *margo.DiagnosticError
	if !errors.As(err, &diagnostic) {
		t.Fatalf("error = %v, want DiagnosticError", err)
	}
	if len(diagnostic.Diagnostics) != 1 || diagnostic.Diagnostics[0].Code != internalmermaid.ConfigurationForbiddenCode {
		t.Fatalf("diagnostics = %#v", diagnostic.Diagnostics)
	}
	if diagnostic.Diagnostics[0].Line != 2 || diagnostic.Diagnostics[0].Column != 3 {
		t.Fatalf("diagnostic position = %d:%d, want 2:3", diagnostic.Diagnostics[0].Line, diagnostic.Diagnostics[0].Column)
	}
}
