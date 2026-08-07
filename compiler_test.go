package margo

import (
	"context"
	"sync"
	"testing"
)

func TestCompileSnapshotsSource(t *testing.T) {
	input := []byte("# one")
	doc, err := New().Compile(context.Background(), Source{Name: "x.md", Content: input})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	copy(input, []byte("# two"))
	if got := string(doc.sourceBytesForTest()); got != "# one" {
		t.Fatalf("source snapshot = %q, want %q", got, "# one")
	}
}

func TestCompilerConfigEquivalentFingerprint(t *testing.T) {
	if New().fingerprint != New().fingerprint {
		t.Fatal("equivalent compilers have different fingerprints")
	}
}

func TestCompileDefensiveSourceAccess(t *testing.T) {
	doc, err := New().Compile(context.Background(), Source{Name: "x.md", Content: []byte("# one")})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	first := doc.sourceBytesForTest()
	first[0] = 'X'
	if got := string(doc.sourceBytesForTest()); got != "# one" {
		t.Fatalf("document exposed mutable source = %q", got)
	}
}

func TestConcurrentCompile(t *testing.T) {
	c := New()
	const n = 16
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Compile(context.Background(), Source{Name: "x.md", Content: []byte("# one")})
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent Compile() error = %v", err)
		}
	}
}
