package devserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Builder produces one immutable site snapshot.
type Builder interface {
	Build(context.Context) (Snapshot, error)
}

// ChangeSource emits debounced filesystem changes and watcher failures.
type ChangeSource interface {
	Changes() <-chan struct{}
	Errors() <-chan error
	Close() error
}

// BuildEvent reports one completed development build.
type BuildEvent struct {
	Generation uint64
	Err        error
	Initial    bool
	Snapshot   Snapshot
}

// Options configures one development server run.
type Options struct {
	Listener      net.Listener
	URL           string
	Builder       Builder
	Changes       ChangeSource
	Store         *SnapshotStore
	Broker        *Broker
	Started       func(string)
	BuildReported func(BuildEvent)
}

type buildResult struct {
	snapshot Snapshot
	err      error
	initial  bool
}

// Run serves snapshots and serializes rebuilds until cancellation or failure.
func Run(ctx context.Context, options Options) error {
	if ctx == nil {
		return fmt.Errorf("serve.context_required: context is required")
	}
	if options.Listener == nil {
		return fmt.Errorf("serve.listener_required: listener is required")
	}
	if options.Builder == nil {
		_ = options.Listener.Close()
		return fmt.Errorf("serve.builder_required: builder is required")
	}
	if options.Changes == nil {
		_ = options.Listener.Close()
		return fmt.Errorf("serve.watcher_required: change source is required")
	}
	defer options.Changes.Close()
	if options.Store == nil {
		options.Store = NewSnapshotStore()
	}
	if options.Broker == nil {
		options.Broker = NewBroker()
	}
	if options.URL == "" {
		options.URL = "http://" + options.Listener.Addr().String() + "/"
	}

	runContext, cancel := context.WithCancel(ctx)
	defer cancel()
	server := &http.Server{
		Handler:           NewHandler(options.Store, options.Broker),
		ReadHeaderTimeout: 5 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		failure := server.Serve(options.Listener)
		if errors.Is(failure, http.ErrServerClosed) || errors.Is(failure, net.ErrClosed) {
			failure = nil
		}
		serverErrors <- failure
	}()
	defer func() {
		cancel()
		shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			_ = server.Close()
		}
	}()
	if options.Started != nil {
		options.Started(options.URL)
	}

	results := make(chan buildResult, 1)
	building := false
	pending := false
	startBuild := func(initial bool) {
		building = true
		go func() {
			snapshot, err := options.Builder.Build(runContext)
			results <- buildResult{snapshot: snapshot, err: err, initial: initial}
		}()
	}
	startBuild(true)

	changes := options.Changes.Changes()
	watchErrors := options.Changes.Errors()
	for {
		select {
		case <-ctx.Done():
			return nil
		case failure := <-serverErrors:
			if failure != nil {
				return fmt.Errorf("serve.http_failed: %w", failure)
			}
			return nil
		case _, ok := <-changes:
			if !ok {
				changes = nil
				continue
			}
			if building {
				pending = true
			} else {
				startBuild(false)
			}
		case failure, ok := <-watchErrors:
			if !ok {
				watchErrors = nil
				continue
			}
			if failure != nil {
				return fmt.Errorf("serve.watch_failed: %w", failure)
			}
		case result := <-results:
			building = false
			event := BuildEvent{Err: result.err, Initial: result.initial, Snapshot: result.snapshot}
			if result.err != nil {
				options.Store.SetError(result.err)
				event.Generation = options.Store.load().generation
			} else {
				event.Generation = options.Store.Replace(result.snapshot)
				options.Broker.Publish(event.Generation)
			}
			if options.BuildReported != nil {
				options.BuildReported(event)
			}
			if pending {
				pending = false
				startBuild(false)
			}
		}
	}
}
