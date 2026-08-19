package main

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

const serveSignalHelper = "MARGO_TEST_SERVE_SIGNAL_HELPER"

func TestServeProcessStopsCleanlyOnInterrupt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.Interrupt is not implemented on Windows")
	}
	root := t.TempDir()
	writeSiteFixture(t, root+"/index.md", "# Signal test\n")
	command := exec.Command(os.Args[0], "-test.run=^TestServeSignalHelper$")
	command.Env = append(os.Environ(), serveSignalHelper+"=1", "MARGO_TEST_SERVE_ROOT="+root)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	waited := false
	t.Cleanup(func() {
		if waited {
			return
		}
		_ = command.Process.Kill()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("serve helper cleanup timed out")
		}
	})
	url := readServingURL(t, bufio.NewScanner(stdout))
	waitForServedContent(t, url, "Signal test")
	if err := command.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		waited = true
		if err != nil {
			t.Fatalf("interrupted process: %v\nstderr: %s", err, stderr.String())
		}
	case <-time.After(5 * time.Second):
		_ = command.Process.Kill()
		t.Fatal("interrupted process did not stop")
	}
}

func TestServeSignalHelper(t *testing.T) {
	if os.Getenv(serveSignalHelper) != "1" {
		return
	}
	os.Args = []string{"margo", "serve", os.Getenv("MARGO_TEST_SERVE_ROOT")}
	main()
}

func readServingURL(t *testing.T, scanner *bufio.Scanner) string {
	t.Helper()
	lines := make(chan string, 8)
	go func() {
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatal("serve helper exited before reporting URL")
			}
			if strings.HasPrefix(line, "Serving ") {
				return strings.TrimPrefix(line, "Serving ")
			}
		case <-timer.C:
			t.Fatal("timed out waiting for serve helper URL")
		}
	}
}
