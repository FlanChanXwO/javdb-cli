package process

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveExecutablePath resolves an executable path and follows a symlink when
// the platform replacement must target the real binary.
func ResolveExecutablePath(executablePath string) (string, error) {
	target, err := filepath.Abs(executablePath)
	if err != nil {
		return "", fmt.Errorf("resolve current executable %q: %w", executablePath, err)
	}
	info, err := os.Lstat(target)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return target, nil
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", fmt.Errorf("resolve executable symlink %q: %w", target, err)
	}
	return filepath.Abs(resolved)
}
