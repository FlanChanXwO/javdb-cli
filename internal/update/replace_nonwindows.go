//go:build !windows

package update

import "os"

// replaceExecutable is an atomic same-filesystem replacement on Unix-like hosts.
func replaceExecutable(sourcePath, targetPath string) error {
	return os.Rename(sourcePath, targetPath)
}

// CleanupPendingWindowsUpdate has no work outside Windows.
func CleanupPendingWindowsUpdate() error {
	return nil
}
