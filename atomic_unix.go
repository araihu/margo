//go:build !windows

package margo

import (
	"fmt"
	"os"
)

type unixAtomicOps struct{}

func defaultAtomicOps() atomicOps { return unixAtomicOps{} }

func (unixAtomicOps) createStage(dir string) (*os.File, error) {
	file, err := os.CreateTemp(dir, ".margo-artifact-*")
	if err != nil {
		return nil, err
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, err
	}
	return file, nil
}

func (unixAtomicOps) publishNoReplace(stage, target string) (bool, error) {
	// A same-directory hard link is an atomic no-replace publication for a
	// regular staged file: link(2) fails with EEXIST and never overwrites the
	// destination. The source is removed only after the target link is visible.
	if err := os.Link(stage, target); err != nil {
		return false, err
	}
	if err := os.Remove(stage); err != nil {
		return true, fmt.Errorf("stage cleanup after visibility: %w", err)
	}
	return true, nil
}

func (unixAtomicOps) remove(path string) error { return os.Remove(path) }

func (unixAtomicOps) syncParent(dir string) error {
	file, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	if err := file.Sync(); err != nil {
		return fmt.Errorf("directory sync: %w", err)
	}
	return nil
}
