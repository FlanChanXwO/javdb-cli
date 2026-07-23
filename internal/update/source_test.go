package update

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/buildinfo"
)

func TestDetectInstallSourceTreatsDevelopmentAsNonUpdatable(t *testing.T) {
	source, err := detectInstallSource(buildinfo.Info{Version: "dev"}, sourceDetector{})
	if err != nil {
		t.Fatal(err)
	}
	if source != InstallSourceDevelopment {
		t.Fatalf("source = %q", source)
	}
}

func TestDetectInstallSourceRecognizesHomebrewReceipt(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "Cellar", "javdb-cli", "0.2.0", "bin", "javdb")
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	receipt := filepath.Join(root, "Cellar", "javdb-cli", "0.2.0", "INSTALL_RECEIPT.json")
	if err := os.WriteFile(receipt, []byte(`{"source":{"path":"/tmp/Formula/javdb-cli.rb"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	source, err := detectInstallSource(buildinfo.Info{Version: "v0.1.1"}, sourceDetector{
		executable:    func() (string, error) { return executable, nil },
		evalSymlinks:  func(path string) (string, error) { return path, nil },
		readFile:      os.ReadFile,
		readBuildInfo: func() (*debug.BuildInfo, bool) { return nil, false },
		getenv:        func(string) string { return "" },
		goos:          "linux",
	})
	if err != nil {
		t.Fatal(err)
	}
	if source != InstallSourceHomebrew {
		t.Fatalf("source = %q", source)
	}
}

func TestDetectInstallSourceRecognizesGoInstall(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "bin", "javdb")
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	source, err := detectInstallSource(buildinfo.Info{Version: "v0.1.1"}, sourceDetector{
		executable:   func() (string, error) { return executable, nil },
		evalSymlinks: func(path string) (string, error) { return path, nil },
		readFile:     os.ReadFile,
		readBuildInfo: func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Path: "github.com/FlanChanXwO/javdb-cli"}}, true
		},
		getenv: func(key string) string {
			if key == "GOBIN" {
				return filepath.Dir(executable)
			}
			return ""
		},
		goos: "linux",
	})
	if err != nil {
		t.Fatal(err)
	}
	if source != InstallSourceGoInstall {
		t.Fatalf("source = %q", source)
	}
}
