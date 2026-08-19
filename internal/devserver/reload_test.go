package devserver

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBrokerSendsReadyThenReloadGeneration(t *testing.T) {
	broker := NewBroker()
	broker.Publish(1)
	server := httptest.NewServer(broker)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if got := response.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type = %q", got)
	}

	lines := make(chan string, 8)
	go func() {
		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()

	assertSSELine(t, lines, "event: ready")
	assertSSELine(t, lines, "data: 1")
	broker.Publish(2)
	assertSSELine(t, lines, "event: reload")
	assertSSELine(t, lines, "data: 2")
	cancel()
}

func TestReloadClientUsesEventSourceAndReloadEvents(t *testing.T) {
	for _, required := range []string{
		`new EventSource("/.margo/live-reload")`,
		`addEventListener("reload"`,
		`location.reload()`,
	} {
		if !strings.Contains(liveReloadClient, required) {
			t.Fatalf("client missing %q: %s", required, liveReloadClient)
		}
	}
}

func assertSSELine(t *testing.T, lines <-chan string, want string) {
	t.Helper()
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatalf("stream closed before %q", want)
			}
			if line == want {
				return
			}
		case <-timer.C:
			t.Fatal(fmt.Sprintf("timed out waiting for %q", want))
		}
	}
}
