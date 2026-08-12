package margo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareDaggerGitSupportsNormalCloneAndLinkedWorktree(t *testing.T) {
	script, err := filepath.Abs("scripts/prepare-dagger-git.sh")
	if err != nil {
		t.Fatal(err)
	}

	origin := filepath.Join(t.TempDir(), "origin")
	runGit(t, "", "init", "--initial-branch=main", origin)
	runGit(t, origin, "config", "user.name", "Margo Test")
	runGit(t, origin, "config", "user.email", "margo@example.invalid")
	if err := os.WriteFile(filepath.Join(origin, "README.md"), []byte("main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, origin, "add", "README.md")
	runGit(t, origin, "commit", "-m", "main")
	runGit(t, origin, "tag", "v0.0.1")

	normalClone := filepath.Join(t.TempDir(), "clone")
	runGit(t, "", "clone", origin, normalClone)
	assertSelfContainedBundle(t, script, normalClone)

	linkedWorktree := filepath.Join(t.TempDir(), "worktree")
	runGit(t, normalClone, "config", "user.name", "Margo Test")
	runGit(t, normalClone, "config", "user.email", "margo@example.invalid")
	runGit(t, normalClone, "worktree", "add", "-b", "topic", linkedWorktree)
	if err := os.WriteFile(filepath.Join(linkedWorktree, "topic.txt"), []byte("topic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, linkedWorktree, "add", "topic.txt")
	runGit(t, linkedWorktree, "commit", "-m", "topic")
	gitFile, err := os.ReadFile(filepath.Join(linkedWorktree, ".git"))
	if err != nil || !strings.HasPrefix(string(gitFile), "gitdir: ") {
		t.Fatalf("fixture is not a linked worktree: %v %q", err, gitFile)
	}
	assertSelfContainedBundle(t, script, linkedWorktree)
}

func TestPrepareDaggerGitSupportsDetachedTagWithoutLocalMain(t *testing.T) {
	script, err := filepath.Abs("scripts/prepare-dagger-git.sh")
	if err != nil {
		t.Fatal(err)
	}

	origin := filepath.Join(t.TempDir(), "origin")
	runGit(t, "", "init", "--bare", "--initial-branch=main", origin)
	seed := filepath.Join(t.TempDir(), "seed")
	runGit(t, "", "clone", origin, seed)
	runGit(t, seed, "config", "user.name", "Margo Test")
	runGit(t, seed, "config", "user.email", "margo@example.invalid")
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("tagged\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, seed, "add", "README.md")
	runGit(t, seed, "commit", "-m", "tagged")
	runGit(t, seed, "tag", "v0.0.1")
	runGit(t, seed, "push", "origin", "main", "v0.0.1")

	detached := filepath.Join(t.TempDir(), "detached")
	runGit(t, "", "clone", origin, detached)
	runGit(t, detached, "checkout", "--detach", "v0.0.1")
	runGit(t, detached, "branch", "-D", "main")
	if output := strings.TrimSpace(runGit(t, detached, "branch", "--list", "main")); output != "" {
		t.Fatalf("fixture unexpectedly retains local main: %q", output)
	}
	assertSelfContainedBundle(t, script, detached)
}

func assertSelfContainedBundle(t *testing.T, script, repository string) {
	t.Helper()
	bundle := filepath.Join(repository, ".dagger-git.bundle")
	cmd := exec.Command(script, bundle)
	cmd.Dir = repository
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("prepare bundle in %s: %v\n%s", repository, err, output)
	}
	restored := filepath.Join(t.TempDir(), "restored")
	runGit(t, "", "clone", bundle, restored)
	if output := strings.TrimSpace(runGit(t, restored, "branch", "--list", "main")); output == "" {
		runGit(t, restored, "branch", "main", "refs/remotes/origin/main")
	}
	want := strings.TrimSpace(runGit(t, repository, "rev-parse", "HEAD"))
	got := strings.TrimSpace(runGit(t, restored, "rev-parse", "HEAD"))
	if got != want {
		t.Fatalf("restored HEAD = %s, want %s", got, want)
	}
	if output := runGit(t, restored, "tag", "--list", "v0.0.1"); strings.TrimSpace(output) != "v0.0.1" {
		t.Fatalf("restored tags = %q", output)
	}
	mainCommit := strings.TrimSpace(runGit(t, restored, "rev-parse", "refs/heads/main"))
	originMainCommit := strings.TrimSpace(runGit(t, repository, "rev-parse", "refs/remotes/origin/main"))
	if mainCommit != originMainCommit {
		t.Fatalf("portable main = %s, want origin/main %s", mainCommit, originMainCommit)
	}
}

func runGit(t *testing.T, directory string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if directory != "" {
		cmd.Dir = directory
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}
