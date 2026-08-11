//go:build !windows

package process

import "os"

// ReplaceExecutable performs an atomic same-filesystem replacement on Unix-like hosts.
func ReplaceExecutable(sourcePath, targetPath string) error {
	return os.Rename(sourcePath, targetPath)
}

// CleanupPendingWindowsUpdate has no work outside Windows.
func CleanupPendingWindowsUpdate() error {
	return nil
}
