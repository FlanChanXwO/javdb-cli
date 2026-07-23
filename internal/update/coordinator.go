package update

import (
	"context"
	"fmt"

	"github.com/FlanChanXwO/javdb-cli/internal/buildinfo"
)

// ReleaseInstaller installs an already selected release for the current platform.
type ReleaseInstaller interface {
	Install(context.Context, Release) error
}

// CoordinatorOptions provides external dependencies for a testable update flow.
type CoordinatorOptions struct {
	SourceDetector   SourceDetector
	ReleaseChecker   ReleaseChecker
	CommandRunner    CommandRunner
	ReleaseInstaller ReleaseInstaller
}

// Coordinator chooses a safe update strategy for the detected installation source.
type Coordinator struct {
	sourceDetector   SourceDetector
	releaseChecker   ReleaseChecker
	commandRunner    CommandRunner
	releaseInstaller ReleaseInstaller
}

// Request describes an explicit javdb update invocation.
type Request struct {
	BuildInfo         buildinfo.Info
	Check             bool
	IncludePrerelease bool
}

// Result is stable status output for human and JSON update checks.
type Result struct {
	Source           InstallSource `json:"source"`
	CurrentVersion   string        `json:"current_version"`
	LatestVersion    *string       `json:"latest_version"`
	LatestPrerelease bool          `json:"latest_prerelease"`
	UpdateAvailable  bool          `json:"update_available"`
}

// NewCoordinator validates the dependencies shared by command and tests.
func NewCoordinator(options CoordinatorOptions) (*Coordinator, error) {
	if options.SourceDetector == nil {
		return nil, fmt.Errorf("update source detector is required")
	}
	if options.ReleaseChecker == nil {
		return nil, fmt.Errorf("update release checker is required")
	}
	if options.ReleaseInstaller == nil {
		return nil, fmt.Errorf("release installer is required")
	}
	return &Coordinator{
		sourceDetector:   options.SourceDetector,
		releaseChecker:   options.ReleaseChecker,
		commandRunner:    options.CommandRunner,
		releaseInstaller: options.ReleaseInstaller,
	}, nil
}

// Execute checks for a newer release and, unless --check was requested, installs it.
func (c *Coordinator) Execute(ctx context.Context, request Request) (Result, error) {
	if c == nil {
		return Result{}, fmt.Errorf("update coordinator is nil")
	}
	source, err := c.sourceDetector.Detect(request.BuildInfo)
	if err != nil {
		return Result{}, fmt.Errorf("detect installation source: %w", err)
	}
	if source == InstallSourceDevelopment {
		return Result{}, fmt.Errorf("development builds cannot update themselves; install a published release first")
	}
	current, err := parseSemanticVersion(request.BuildInfo.Version)
	if err != nil {
		return Result{}, fmt.Errorf("parse current build version %q: %w", request.BuildInfo.Version, err)
	}
	latest, err := c.releaseChecker.Check(ctx, request.IncludePrerelease)
	if err != nil {
		return Result{}, fmt.Errorf("check available releases: %w", err)
	}
	result := Result{Source: source, CurrentVersion: request.BuildInfo.Version}
	if latest == nil {
		return result, nil
	}
	latestVersion, err := parseSemanticVersion(latest.TagName)
	if err != nil {
		return Result{}, fmt.Errorf("parse selected release tag %q: %w", latest.TagName, err)
	}
	result.LatestVersion = &latest.TagName
	result.LatestPrerelease = latest.Prerelease
	result.UpdateAvailable = latestVersion.compare(current) > 0
	if request.Check || !result.UpdateAvailable {
		return result, nil
	}
	switch source {
	case InstallSourceHomebrew:
		if request.IncludePrerelease {
			return Result{}, fmt.Errorf("Homebrew installation cannot install prerelease releases; use a Release archive or go install")
		}
		if c.commandRunner == nil {
			return Result{}, fmt.Errorf("Homebrew update command runner is unavailable")
		}
		if err := c.commandRunner.Run(ctx, "brew", "upgrade", homebrewFormula); err != nil {
			return Result{}, fmt.Errorf("run Homebrew update: %w", err)
		}
	case InstallSourceGoInstall:
		if c.commandRunner == nil {
			return Result{}, fmt.Errorf("go install update command runner is unavailable")
		}
		if err := c.commandRunner.Run(ctx, "go", "install", goInstallPackage+"@"+latest.TagName); err != nil {
			return Result{}, fmt.Errorf("run go install update: %w", err)
		}
	case InstallSourceRelease:
		if err := c.releaseInstaller.Install(ctx, *latest); err != nil {
			return Result{}, fmt.Errorf("install release update %q: %w", latest.TagName, err)
		}
	default:
		return Result{}, fmt.Errorf("unsupported installation source %q", source)
	}
	return result, nil
}
