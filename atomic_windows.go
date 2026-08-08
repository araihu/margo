//go:build windows

package margo

import (
	"os"

	"golang.org/x/sys/windows"
)

type windowsAtomicOps struct{}

func defaultAtomicOps() atomicOps { return windowsAtomicOps{} }

func (windowsAtomicOps) createStage(dir string) (*os.File, error) {
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

func (windowsAtomicOps) publishNoReplace(stage, target string) (bool, error) {
	return moveFile(stage, target, windows.MOVEFILE_WRITE_THROUGH)
}

func (windowsAtomicOps) publishReplace(stage, target string) (bool, error) {
	return moveFile(stage, target, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}

func moveFile(stage, target string, flags uint32) (bool, error) {
	from, err := windows.UTF16PtrFromString(stage)
	if err != nil {
		return false, err
	}
	to, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return false, err
	}
	if err := windows.MoveFileEx(from, to, flags); err != nil {
		return false, err
	}
	return true, nil
}

func (windowsAtomicOps) remove(path string) error { return os.Remove(path) }

// MOVEFILE_WRITE_THROUGH makes the move durable through the Windows file
// system. There is no portable directory fsync equivalent for this sink.
func (windowsAtomicOps) syncParent(string) error { return nil }
