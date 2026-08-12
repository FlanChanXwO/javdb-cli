// Package paths 管理 javdb-cli 本机状态文件的路径。
package paths

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// AppDirName is the directory under the user home used for credentials/config.
const AppDirName = ".javdb-cli"

const defaultConfigTemplate = `# javdb-cli configuration
# Use "javdb config set KEY VALUE" to change a setting.

host = "auto"
https_proxy = ""
auto_relogin = false
lang = "en"
`

// Dir returns ~/.javdb-cli (or $HOME/.javdb-cli), creating nothing.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, AppDirName), nil
}

// EnsureDir creates the config directory with 0700 permissions.
func EnsureDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func AuthPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "auth.json"), nil
}

func ConfigPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// RouteCachePath 返回自动选线路由缓存文件的路径（~/.javdb-cli/route.json）。
func RouteCachePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "route.json"), nil
}

// EnsureDefaultConfigFile 在配置首次缺失时创建只含常用选项的私有基线文件。
func EnsureDefaultConfigFile() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return ensureDefaultConfigFileAt(path, func(file *os.File) error {
		written, err := io.WriteString(file, defaultConfigTemplate)
		if err != nil {
			return err
		}
		if written != len(defaultConfigTemplate) {
			return io.ErrShortWrite
		}
		return nil
	})
}

func ensureDefaultConfigFileAt(path string, populate func(*os.File) error) (err error) {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	file, err := os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	temporaryPath := file.Name()
	closed := false
	defer func() {
		if !closed {
			err = errors.Join(err, file.Close())
		}
		if removeErr := os.Remove(temporaryPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		return err
	}
	if err := populate(file); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		closed = true
		return err
	}
	closed = true
	// 同目录 hard link 以 no-replace 语义原子发布：目标出现时，内容已经写完、同步并关闭；
	// 并发创建者若已先发布，则保留现有赢家，绝不覆盖用户或另一调用方的完整配置。
	if err := os.Link(temporaryPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return err
	}
	return nil
}

func DeviceUUIDPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "device_uuid"), nil
}

func TagTaxonomyPath(zone string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tags-"+zone+".json"), nil
}

// ReverseSearchCacheDir 返回反搜响应缓存的目录（~/.javdb-cli/reverse-search-cache）。
func ReverseSearchCacheDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "reverse-search-cache"), nil
}
