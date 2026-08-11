package update

import (
	"context"
	"fmt"

	"github.com/FlanChanXwO/javdb-cli/internal/update/model"
	"github.com/FlanChanXwO/javdb-cli/internal/update/release"
)

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
func (c *Coordinator) Execute(ctx context.Context, request model.Request) (model.Result, error) {
	if c == nil {
		return model.Result{}, fmt.Errorf("update coordinator is nil")
	}
	source, err := c.sourceDetector.Detect(request.BuildInfo)
	if err != nil {
		return model.Result{}, fmt.Errorf("detect installation source: %w", err)
	}
	if source == model.InstallSourceDevelopment {
		return model.Result{}, fmt.Errorf("development builds cannot update themselves; install a published release first")
	}
	current, err := release.ParseSemanticVersion(request.BuildInfo.Version)
	if err != nil {
		return model.Result{}, fmt.Errorf("parse current build version %q: %w", request.BuildInfo.Version, err)
	}
	latest, err := c.releaseChecker.Check(ctx, request.IncludePrerelease)
	if err != nil {
		return model.Result{}, fmt.Errorf("check available releases: %w", err)
	}
	result := model.Result{Source: source, CurrentVersion: request.BuildInfo.Version}
	if latest == nil {
		return result, nil
	}
	latestVersion, err := release.ParseSemanticVersion(latest.TagName)
	if err != nil {
		return model.Result{}, fmt.Errorf("parse selected release tag %q: %w", latest.TagName, err)
	}
	result.LatestVersion = &latest.TagName
	result.LatestPrerelease = latest.Prerelease
	result.UpdateAvailable = release.CompareSemanticVersions(latestVersion, current) > 0
	if request.Check || !result.UpdateAvailable {
		return result, nil
	}
	switch source {
	case model.InstallSourceHomebrew:
		if request.IncludePrerelease {
			return model.Result{}, fmt.Errorf("Homebrew installation cannot install prerelease releases; use a Release archive or go install")
		}
		if c.commandRunner == nil {
			return model.Result{}, fmt.Errorf("Homebrew update command runner is unavailable")
		}
		if err := c.commandRunner.Run(ctx, "brew", "upgrade", model.HomebrewFormula); err != nil {
			return model.Result{}, fmt.Errorf("run Homebrew update: %w", err)
		}
	case model.InstallSourceGoInstall:
		if c.commandRunner == nil {
			return model.Result{}, fmt.Errorf("go install update command runner is unavailable")
		}
		if err := c.commandRunner.Run(ctx, "go", "install", model.GoInstallPackage+"@"+latest.TagName); err != nil {
			return model.Result{}, fmt.Errorf("run go install update: %w", err)
		}
	case model.InstallSourceRelease:
		if err := c.releaseInstaller.Install(ctx, *latest); err != nil {
			return model.Result{}, fmt.Errorf("install release update %q: %w", latest.TagName, err)
		}
	default:
		return model.Result{}, fmt.Errorf("unsupported installation source %q", source)
	}
	return result, nil
}
