package devserver

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/araihu/margo/site"
)

type buildCall struct {
	result chan buildAnswer
}

type buildAnswer struct {
	snapshot Snapshot
	err      error
}

type blockingBuilder struct {
	calls  chan buildCall
	active atomic.Int32
	max    atomic.Int32
}

func (builder *blockingBuilder) Build(ctx context.Context) (Snapshot, error) {
	active := builder.active.Add(1)
	defer builder.active.Add(-1)
	for {
		maximum := builder.max.Load()
		if active <= maximum || builder.max.CompareAndSwap(maximum, active) {
			break
		}
	}
	call := buildCall{result: make(chan buildAnswer, 1)}
	select {
	case builder.calls <- call:
	case <-ctx.Done():
		return Snapshot{}, ctx.Err()
	}
	select {
	case answer := <-call.result:
		return answer.snapshot, answer.err
	case <-ctx.Done():
		return Snapshot{}, ctx.Err()
	}
}

type channelChangeSource struct {
	changes chan struct{}
	errors  chan error
	closed  atomic.Int32
	once    sync.Once
}

func (source *channelChangeSource) Changes() <-chan struct{} { return source.changes }
func (source *channelChangeSource) Errors() <-chan error     { return source.errors }
func (source *channelChangeSource) Close() error {
	source.once.Do(func() { source.closed.Add(1) })
	return nil
}

func TestRunServesBeforeBuildAndQueuesOneFollowUp(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	url := "http://" + listener.Addr().String() + "/"
	builder := &blockingBuilder{calls: make(chan buildCall, 8)}
	changes := &channelChangeSource{changes: make(chan struct{}, 8), errors: make(chan error, 1)}
	store := NewSnapshotStore()
	broker := NewBroker()
	started := make(chan string, 1)
	reports := make(chan BuildEvent, 8)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, Options{
			Listener: listener,
			URL:      url,
			Builder:  builder,
			Changes:  changes,
			Store:    store,
			Broker:   broker,
			Started:  func(url string) { started <- url },
			BuildReported: func(event BuildEvent) {
				reports <- event
			},
		})
	}()

	if got := receive(t, started); got != url {
		t.Fatalf("started URL = %q, want %q", got, url)
	}
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("pre-build status = %d", response.StatusCode)
	}
	response.Body.Close()

	firstCall := receive(t, builder.calls)
	firstCall.result <- buildAnswer{snapshot: testSnapshot("first")}
	firstReport := receive(t, reports)
	if firstReport.Err != nil || !firstReport.Initial || firstReport.Generation != 1 {
		t.Fatalf("first report = %+v", firstReport)
	}
	assertHTTPBody(t, url, "first")

	changes.changes <- struct{}{}
	secondCall := receive(t, builder.calls)
	changes.changes <- struct{}{}
	changes.changes <- struct{}{}
	secondCall.result <- buildAnswer{err: errors.New("broken markdown")}
	secondReport := receive(t, reports)
	if secondReport.Err == nil || secondReport.Initial || secondReport.Generation != 1 {
		t.Fatalf("second report = %+v", secondReport)
	}
	assertHTTPBody(t, url, "first")

	thirdCall := receive(t, builder.calls)
	select {
	case unexpected := <-builder.calls:
		unexpected.result <- buildAnswer{}
		t.Fatal("more than one follow-up build queued")
	case <-time.After(50 * time.Millisecond):
	}
	thirdCall.result <- buildAnswer{snapshot: testSnapshot("second")}
	thirdReport := receive(t, reports)
	if thirdReport.Err != nil || thirdReport.Generation != 2 {
		t.Fatalf("third report = %+v", thirdReport)
	}
	assertHTTPBody(t, url, "second")
	if builder.max.Load() != 1 {
		t.Fatalf("maximum concurrent builds = %d", builder.max.Load())
	}

	cancel()
	if err := receive(t, done); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if changes.closed.Load() != 1 {
		t.Fatalf("change source closed %d times", changes.closed.Load())
	}
}

func TestRunStopsOnWatcherError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	builder := &blockingBuilder{calls: make(chan buildCall, 1)}
	changes := &channelChangeSource{changes: make(chan struct{}, 1), errors: make(chan error, 1)}
	done := make(chan error, 1)
	go func() {
		done <- Run(context.Background(), Options{Listener: listener, Builder: builder, Changes: changes})
	}()
	call := receive(t, builder.calls)
	call.result <- buildAnswer{snapshot: testSnapshot("ready")}
	wantErr := errors.New("watch failed")
	changes.errors <- wantErr
	if err := receive(t, done); !errors.Is(err, wantErr) {
		t.Fatalf("Run error = %v, want %v", err, wantErr)
	}
}

func testSnapshot(content string) Snapshot {
	return NewSnapshot(site.Result{Artifacts: []site.Artifact{{Path: "index.html", Content: []byte("<html><body>" + content + "</body></html>")}}})
}

func assertHTTPBody(t *testing.T, url, want string) {
	t.Helper()
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(data), want) {
		t.Fatalf("response = %d %q, want %q", response.StatusCode, data, want)
	}
}

func receive[T any](t *testing.T, channel <-chan T) T {
	t.Helper()
	select {
	case value := <-channel:
		return value
	case <-time.After(3 * time.Second):
		var zero T
		t.Fatal("timed out waiting for channel value")
		return zero
	}
}
