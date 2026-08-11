package paths

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", filepath.VolumeName(home))
	t.Setenv("HOMEPATH", strings.TrimPrefix(home, filepath.VolumeName(home)))
	return home
}

func TestEnsureDefaultConfigFileCreatesPrivateBaseline(t *testing.T) {
	home := isolateHome(t)
	if err := EnsureDefaultConfigFile(); err != nil {
		t.Fatalf("EnsureDefaultConfigFile() error = %v", err)
	}

	path := filepath.Join(home, AppDirName, "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(data)
	for _, expected := range []string{
		"# javdb-cli configuration",
		`host = "auto"`,
		`https_proxy = ""`,
		"auto_relogin = false",
		`lang = "en"`,
	} {
		if !strings.Contains(content, expected) {
			t.Errorf("config.toml missing %q:\n%s", expected, content)
		}
	}
	if strings.Contains(content, "device_uuid") {
		t.Fatalf("config.toml contains advanced device_uuid:\n%s", content)
	}

	if runtime.GOOS != "windows" {
		assertPermissions(t, path, 0o600)
		assertPermissions(t, filepath.Dir(path), 0o700)
	}
}

func TestEnsureDefaultConfigFileDoesNotOverwriteExistingFile(t *testing.T) {
	home := isolateHome(t)
	path := filepath.Join(home, AppDirName, "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	const original = "# custom\nhost = \"main\"\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := EnsureDefaultConfigFile(); err != nil {
		t.Fatalf("EnsureDefaultConfigFile() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got := string(data); got != original {
		t.Fatalf("config.toml = %q, want original %q", got, original)
	}
}

func TestEnsureDefaultConfigFileRemovesPartialFileOnWriteFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.toml")
	wantErr := errors.New("injected write failure")
	err := ensureDefaultConfigFileAt(path, func(file *os.File) error {
		if _, writeErr := file.WriteString("partial"); writeErr != nil {
			return writeErr
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("ensureDefaultConfigFileAt() error = %v, want %v", err, wantErr)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("partial config remains: Stat() error = %v", statErr)
	}
}

func TestEnsureDefaultConfigFileSupportsConcurrentCreation(t *testing.T) {
	home := isolateHome(t)
	const callers = 8
	start := make(chan struct{})
	errors := make(chan error, callers)
	var group sync.WaitGroup
	group.Add(callers)
	for range callers {
		go func() {
			defer group.Done()
			<-start
			errors <- EnsureDefaultConfigFile()
		}()
	}
	close(start)
	group.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("EnsureDefaultConfigFile() concurrent error = %v", err)
		}
	}

	path := filepath.Join(home, AppDirName, "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got := string(data); got != defaultConfigTemplate {
		t.Fatalf("concurrent config.toml = %q, want baseline %q", got, defaultConfigTemplate)
	}
}

func assertPermissions(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s permissions = %04o, want %04o", path, got, want)
	}
}
