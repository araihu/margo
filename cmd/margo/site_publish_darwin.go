//go:build darwin

package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func renameSiteDirectoryNoReplace(stage, target string) error {
	return unix.RenamexNp(stage, target, unix.RENAME_EXCL)
}

func syncSiteOutputParent(directory string) error {
	file, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Sync(); err != nil {
		return fmt.Errorf("directory sync: %w", err)
	}
	return nil
}
