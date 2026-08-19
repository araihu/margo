package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestServeEndToEndReloadsRawMarkdownTree(t *testing.T) {
	root := t.TempDir()
	markdown := filepath.Join(root, "index.md")
	writeSiteFixture(t, markdown, "# First version\n")

	var stdout lockedBuffer
	var stderr bytes.Buffer
	command := NewRootCommand(Dependencies{
		Stdout: &stdout, Stderr: &stderr, WorkingDirectory: root, Build: testBuildInfo(),
	})
	command.SetArgs([]string{"serve", "."})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- command.ExecuteContext(ctx) }()
	stopped := false
	defer func() {
		cancel()
		if stopped {
			return
		}
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("serve command did not stop")
		}
	}()

	url := strings.TrimSuffix(waitForServingURL(t, &stdout), "/")
	waitForServedContent(t, url+"/", "First version")
	reload := openReloadStream(t, ctx, url+"/.margo/live-reload")
	if err := os.WriteFile(markdown, []byte("# Second version\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitForReloadEvent(t, reload)
	waitForServedContent(t, url+"/", "Second version")
	if _, err := os.Stat(filepath.Join(root, "dist")); !os.IsNotExist(err) {
		t.Fatalf("development server wrote output: %v", err)
	}

	cancel()
	select {
	case err := <-done:
		stopped = true
		if err != nil {
			t.Fatalf("serve command: %v\nstderr: %s", err, stderr.String())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serve command did not stop after cancellation")
	}
}

type lockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (buffer *lockedBuffer) Write(data []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.Write(data)
}

func (buffer *lockedBuffer) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.String()
}

func waitForServingURL(t *testing.T, output *lockedBuffer) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, line := range strings.Split(output.String(), "\n") {
			if strings.HasPrefix(line, "Serving ") {
				return strings.TrimPrefix(line, "Serving ")
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for serving URL: %s", output.String())
	return ""
}

func waitForServedContent(t *testing.T, url, required string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get(url)
		if err == nil {
			data, readErr := io.ReadAll(response.Body)
			response.Body.Close()
			if readErr == nil && response.StatusCode == http.StatusOK && strings.Contains(string(data), required) {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q from %s", required, url)
}

func openReloadStream(t *testing.T, ctx context.Context, url string) <-chan string {
	t.Helper()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	lines := make(chan string, 16)
	go func() {
		defer response.Body.Close()
		defer close(lines)
		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()
	return lines
}

func waitForReloadEvent(t *testing.T, lines <-chan string) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatal("reload stream closed")
			}
			if line == "event: reload" {
				return
			}
		case <-deadline.C:
			t.Fatal(fmt.Sprintf("timed out waiting for reload event"))
		}
	}
}
