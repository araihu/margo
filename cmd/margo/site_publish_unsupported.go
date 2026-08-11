//go:build !darwin && !linux && !windows

package main

import "errors"

func renameSiteDirectoryNoReplace(string, string) error {
	return errors.New("atomic no-replace directory rename is unavailable on this platform")
}

func syncSiteOutputParent(string) error {
	return errors.New("directory synchronization is unavailable on this platform")
}
