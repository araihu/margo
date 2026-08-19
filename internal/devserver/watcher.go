package devserver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type fsChangeSource struct {
	root      string
	inputRoot string
	watcher   *fsnotify.Watcher
	ignored   func(string) bool
	debounce  time.Duration
	changes   chan struct{}
	errors    chan error
	done      chan struct{}
	close     sync.Once
	wait      sync.WaitGroup
}

// Watch recursively observes one project tree and emits debounced changes.
func Watch(root string, ignored func(string) bool, debounce time.Duration) (ChangeSource, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("serve.watch_invalid: %w", err)
	}
	realRoot, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("serve.watch_unreadable: %w", err)
	}
	info, err := os.Stat(realRoot)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("serve.watch_unreadable: watch root is not a directory")
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("serve.watch_unavailable: %w", err)
	}
	source := &fsChangeSource{
		root: realRoot, inputRoot: absolute, watcher: watcher, ignored: ignored, debounce: debounce,
		changes: make(chan struct{}, 1), errors: make(chan error, 1), done: make(chan struct{}),
	}
	if source.debounce <= 0 {
		source.debounce = 100 * time.Millisecond
	}
	if err := source.addTree(realRoot); err != nil {
		_ = watcher.Close()
		return nil, err
	}
	source.wait.Add(1)
	go source.run()
	return source, nil
}

func (source *fsChangeSource) Changes() <-chan struct{} { return source.changes }
func (source *fsChangeSource) Errors() <-chan error     { return source.errors }

func (source *fsChangeSource) Close() error {
	var failure error
	source.close.Do(func() {
		close(source.done)
		failure = source.watcher.Close()
		source.wait.Wait()
	})
	return failure
}

func (source *fsChangeSource) run() {
	defer source.wait.Done()
	defer close(source.changes)
	defer close(source.errors)
	var timer *time.Timer
	var timerChannel <-chan time.Time
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	trigger := func() {
		if timer == nil {
			timer = time.NewTimer(source.debounce)
		} else {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(source.debounce)
		}
		timerChannel = timer.C
	}
	for {
		select {
		case <-source.done:
			return
		case event, ok := <-source.watcher.Events:
			if !ok {
				return
			}
			if source.isIgnored(event.Name) {
				continue
			}
			info, statErr := os.Stat(event.Name)
			isDirectory := statErr == nil && info.IsDir()
			if event.Op&fsnotify.Create != 0 && isDirectory {
				if err := source.addTree(event.Name); err != nil {
					source.reportError(err)
					return
				}
			}
			if isDirectory && event.Op&(fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) != 0 {
				trigger()
			}
		case <-timerChannel:
			timerChannel = nil
			select {
			case source.changes <- struct{}{}:
			default:
			}
		case err, ok := <-source.watcher.Errors:
			if !ok {
				return
			}
			source.reportError(err)
			return
		}
	}
}

func (source *fsChangeSource) addTree(root string) error {
	err := filepath.WalkDir(root, func(name string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if source.isIgnored(name) {
			return filepath.SkipDir
		}
		return source.watcher.Add(name)
	})
	if err != nil {
		return fmt.Errorf("serve.watch_failed: %w", err)
	}
	return nil
}

func (source *fsChangeSource) isIgnored(name string) bool {
	relative, err := filepath.Rel(source.root, name)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return true
	}
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == ".git" || component == ".worktrees" {
			return true
		}
	}
	if source.ignored == nil {
		return false
	}
	projectPath := filepath.Join(source.inputRoot, relative)
	return source.ignored(projectPath)
}

func (source *fsChangeSource) reportError(err error) {
	select {
	case source.errors <- err:
	default:
	}
}
