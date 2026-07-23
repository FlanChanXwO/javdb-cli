//go:build windows

package update

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var replaceFileW = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReplaceFileW")

// replaceExecutable uses ReplaceFileW when a target exists. The old executable
// becomes .old and is removed at the next invocation, after Windows releases it.
func replaceExecutable(sourcePath, targetPath string) error {
	from, err := windows.UTF16PtrFromString(sourcePath)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(targetPath)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(targetPath); os.IsNotExist(err) {
		return windows.MoveFileEx(from, to, 0)
	} else if err != nil {
		return err
	}
	backupPath := targetPath + ".old"
	if _, err := os.Lstat(backupPath); err == nil {
		return fmt.Errorf("pending previous update backup exists at %q", backupPath)
	} else if !os.IsNotExist(err) {
		return err
	}
	backup, err := windows.UTF16PtrFromString(backupPath)
	if err != nil {
		return err
	}
	if err := replaceFileW.Find(); err != nil {
		return err
	}
	success, _, callErr := replaceFileW.Call(
		uintptr(unsafe.Pointer(to)),
		uintptr(unsafe.Pointer(from)),
		uintptr(unsafe.Pointer(backup)),
		0,
		0,
		0,
	)
	if success == 0 {
		return callErr
	}
	return nil
}

// CleanupPendingWindowsUpdate removes the backup retained by ReplaceFileW.
func CleanupPendingWindowsUpdate() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable for update cleanup: %w", err)
	}
	executable, err = resolveExecutablePath(executable)
	if err != nil {
		return err
	}
	if err := os.Remove(executable + ".old"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove previous update backup: %w", err)
	}
	return nil
}
