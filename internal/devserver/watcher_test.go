package devserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWatchDebouncesRecursiveChangesAndAddsDirectories(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "docs")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	source, err := Watch(root, nil, 25*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()

	markdown := filepath.Join(nested, "index.md")
	for _, content := range []string{"# one\n", "# two\n", "# three\n"} {
		if err := os.WriteFile(markdown, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	waitForChange(t, source.Changes())
	assertNoChange(t, source.Changes(), 80*time.Millisecond)

	created := filepath.Join(root, "new", "nested")
	if err := os.MkdirAll(created, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(created, "guide.md"), []byte("# guide\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitForChange(t, source.Changes())
	drainChanges(source.Changes())
	if err := os.WriteFile(filepath.Join(created, "guide.md"), []byte("# changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitForChange(t, source.Changes())
}

func TestWatchIgnoresGitWorktreesAndProjectExclusions(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "public")
	gitDirectory := filepath.Join(root, ".git", "objects")
	worktrees := filepath.Join(root, ".worktrees", "other")
	for _, directory := range []string{output, gitDirectory, worktrees} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ignored := func(name string) bool {
		relative, err := filepath.Rel(output, name)
		return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
	}
	source, err := Watch(root, ignored, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()

	for _, name := range []string{
		filepath.Join(output, "index.html"),
		filepath.Join(gitDirectory, "object"),
		filepath.Join(worktrees, "go.mod"),
	} {
		if err := os.WriteFile(name, []byte("ignored"), 0o644); err != nil {
			t.Fatal(err)
		}
		assertNoChange(t, source.Changes(), 60*time.Millisecond)
	}

	if err := os.WriteFile(filepath.Join(root, "index.md"), []byte("# watched\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitForChange(t, source.Changes())
}

func TestWatchCloseIsIdempotentAndClosesChannels(t *testing.T) {
	source, err := Watch(t.TempDir(), nil, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case _, ok := <-source.Changes():
		if ok {
			t.Fatal("changes channel remains open")
		}
	case <-time.After(time.Second):
		t.Fatal("changes channel did not close")
	}
}

func waitForChange(t *testing.T, changes <-chan struct{}) {
	t.Helper()
	select {
	case <-changes:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for filesystem change")
	}
}

func assertNoChange(t *testing.T, changes <-chan struct{}, duration time.Duration) {
	t.Helper()
	select {
	case _, ok := <-changes:
		if !ok {
			t.Fatal("filesystem changes channel closed unexpectedly")
		}
		t.Fatal("unexpected filesystem change")
	case <-time.After(duration):
	}
}

func drainChanges(changes <-chan struct{}) {
	for {
		select {
		case <-changes:
		default:
			return
		}
	}
}
