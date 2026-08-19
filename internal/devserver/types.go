package devserver

import (
	"errors"
	"path"
	"strings"
	"sync/atomic"

	"github.com/araihu/margo/site"
)

// Snapshot is one immutable, HTTP-ready site build.
type Snapshot struct {
	artifacts map[string][]byte
	basePath  string
	pages     int
}

// NewSnapshot copies a site result into an immutable serving snapshot.
func NewSnapshot(result site.Result) Snapshot {
	artifacts := make(map[string][]byte, len(result.Artifacts))
	for _, artifact := range result.Artifacts {
		artifacts[artifact.Path] = append([]byte(nil), artifact.Content...)
	}
	basePath := strings.TrimSpace(result.Site.BasePath)
	if basePath == "/" {
		basePath = ""
	} else if basePath != "" {
		basePath = path.Clean("/" + strings.Trim(basePath, "/"))
	}
	return Snapshot{artifacts: artifacts, basePath: basePath, pages: len(result.Pages)}
}

// BasePath returns the configured public path prefix.
func (snapshot Snapshot) BasePath() string { return snapshot.basePath }

// ArtifactCount returns the number of generated artifacts.
func (snapshot Snapshot) ArtifactCount() int { return len(snapshot.artifacts) }

// PageCount returns the number of generated pages.
func (snapshot Snapshot) PageCount() int { return snapshot.pages }

type snapshotState struct {
	snapshot   *Snapshot
	generation uint64
	failure    error
}

// SnapshotStore atomically exposes the last successful site build.
type SnapshotStore struct {
	state atomic.Pointer[snapshotState]
}

// NewSnapshotStore creates an empty serving store.
func NewSnapshotStore() *SnapshotStore {
	store := &SnapshotStore{}
	store.state.Store(&snapshotState{})
	return store
}

// Replace publishes a successful snapshot and returns its new generation.
func (store *SnapshotStore) Replace(snapshot Snapshot) uint64 {
	for {
		current := store.load()
		nextSnapshot := snapshot
		next := &snapshotState{snapshot: &nextSnapshot, generation: current.generation + 1}
		if store.state.CompareAndSwap(current, next) {
			return next.generation
		}
	}
}

// SetError records a failed build without discarding a last-good snapshot.
func (store *SnapshotStore) SetError(failure error) {
	if failure == nil {
		failure = errors.New("unknown development build failure")
	}
	for {
		current := store.load()
		next := &snapshotState{snapshot: current.snapshot, generation: current.generation, failure: failure}
		if store.state.CompareAndSwap(current, next) {
			return
		}
	}
}

func (store *SnapshotStore) load() *snapshotState {
	if store == nil {
		return &snapshotState{failure: errors.New("development snapshot store is unavailable")}
	}
	if current := store.state.Load(); current != nil {
		return current
	}
	initial := &snapshotState{}
	if store.state.CompareAndSwap(nil, initial) {
		return initial
	}
	return store.state.Load()
}
