package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/build"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/internal/buildinfo"
)

// SourceDetector determines which installer owns the currently running binary.
type SourceDetector interface {
	Detect(buildinfo.Info) (InstallSource, error)
}

// SourceDetectorFunc adapts a function for coordinator tests and composition.
type SourceDetectorFunc func(buildinfo.Info) (InstallSource, error)

// Detect calls the wrapped source detector.
func (f SourceDetectorFunc) Detect(info buildinfo.Info) (InstallSource, error) {
	return f(info)
}

// DetectInstallSource identifies managed binaries before a write is attempted.
func DetectInstallSource(info buildinfo.Info) (InstallSource, error) {
	if info.IsDevelopment() {
		return InstallSourceDevelopment, nil
	}
	return detectInstallSource(info, sourceDetector{
		executable:    os.Executable,
		evalSymlinks:  filepath.EvalSymlinks,
		readFile:      os.ReadFile,
		readBuildInfo: debug.ReadBuildInfo,
		getenv:        os.Getenv,
		goos:          runtime.GOOS,
	})
}

type sourceDetector struct {
	executable    func() (string, error)
	evalSymlinks  func(string) (string, error)
	readFile      func(string) ([]byte, error)
	readBuildInfo func() (*debug.BuildInfo, bool)
	getenv        func(string) string
	goos          string
}

func detectInstallSource(info buildinfo.Info, deps sourceDetector) (InstallSource, error) {
	if info.IsDevelopment() {
		return InstallSourceDevelopment, nil
	}
	rawExecutable, err := deps.executable()
	if err != nil {
		return "", fmt.Errorf("determine executable path: %w", err)
	}
	executable, err := deps.evalSymlinks(rawExecutable)
	if err != nil {
		return "", fmt.Errorf("resolve executable symlink %q: %w", rawExecutable, err)
	}
	if source, isKeg, err := detectHomebrewSource(executable, deps); isKeg {
		return source, err
	}
	isGoInstalled, err := isGoInstall(executable, deps)
	if err != nil {
		return "", err
	}
	if isGoInstalled {
		return InstallSourceGoInstall, nil
	}
	return InstallSourceRelease, nil
}

func detectHomebrewSource(executable string, deps sourceDetector) (InstallSource, bool, error) {
	formula, kegRoot, ok := homebrewKeg(executable, executableName(deps.goos))
	if !ok {
		return "", false, nil
	}
	if formula != "javdb-cli" {
		return "", true, fmt.Errorf("unsupported Homebrew formula %q for executable %q", formula, executable)
	}
	receiptPath := filepath.Join(kegRoot, "INSTALL_RECEIPT.json")
	body, err := deps.readFile(receiptPath)
	if err != nil {
		return "", true, fmt.Errorf("read Homebrew receipt %q: %w", receiptPath, err)
	}
	var receipt struct {
		Source struct {
			Path string `json:"path"`
		} `json:"source"`
	}
	if err := json.Unmarshal(body, &receipt); err != nil {
		return "", true, fmt.Errorf("parse Homebrew receipt %q: %w", receiptPath, err)
	}
	if path.Base(receipt.Source.Path) != "javdb-cli.rb" {
		return "", true, fmt.Errorf("Homebrew receipt %q does not identify javdb-cli.rb", receiptPath)
	}
	return InstallSourceHomebrew, true, nil
}

func homebrewKeg(executablePath, binaryName string) (formula, kegRoot string, ok bool) {
	executablePath = filepath.Clean(executablePath)
	if filepath.Base(executablePath) != binaryName || filepath.Base(filepath.Dir(executablePath)) != "bin" {
		return "", "", false
	}
	kegRoot = filepath.Dir(filepath.Dir(executablePath))
	formulaDirectory := filepath.Dir(kegRoot)
	if filepath.Base(filepath.Dir(formulaDirectory)) != "Cellar" {
		return "", "", false
	}
	return filepath.Base(formulaDirectory), kegRoot, true
}

func isGoInstall(executable string, deps sourceDetector) (bool, error) {
	buildInfo, ok := deps.readBuildInfo()
	if !ok || buildInfo == nil || buildInfo.Main.Path != "github.com/FlanChanXwO/javdb-cli" {
		return false, nil
	}
	binDirectory := deps.getenv("GOBIN")
	if binDirectory == "" {
		goPath := deps.getenv("GOPATH")
		if goPath == "" {
			goPath = build.Default.GOPATH
		}
		paths := filepath.SplitList(goPath)
		if len(paths) == 0 || paths[0] == "" {
			return false, nil
		}
		binDirectory = filepath.Join(paths[0], "bin")
	}
	expected, err := deps.evalSymlinks(filepath.Join(binDirectory, executableName(deps.goos)))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("resolve go-install executable: %w", err)
	}
	return sameExecutablePath(executable, expected, deps.goos), nil
}

func executableName(goos string) string {
	if goos == "windows" {
		return "javdb.exe"
	}
	return "javdb"
}

func sameExecutablePath(first, second, goos string) bool {
	first, second = filepath.Clean(first), filepath.Clean(second)
	if goos == "windows" {
		return strings.EqualFold(first, second)
	}
	return first == second
}
