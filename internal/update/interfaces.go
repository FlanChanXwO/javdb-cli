package update

import (
	"context"

	"github.com/FlanChanXwO/javdb-cli/internal/buildinfo"
	"github.com/FlanChanXwO/javdb-cli/internal/update/model"
)

// SourceDetector determines which installer owns the currently running binary.
type SourceDetector interface {
	Detect(buildinfo.Info) (model.InstallSource, error)
}

// SourceDetectorFunc adapts a function for coordinator tests and composition.
type SourceDetectorFunc func(buildinfo.Info) (model.InstallSource, error)

// Detect calls the wrapped source detector.
func (f SourceDetectorFunc) Detect(info buildinfo.Info) (model.InstallSource, error) {
	return f(info)
}

// ReleaseChecker resolves the highest compatible published GitHub Release.
type ReleaseChecker interface {
	Check(context.Context, bool) (*model.Release, error)
}

// CommandRunner executes package-manager commands without shell interpolation.
type CommandRunner interface {
	Run(context.Context, string, ...string) error
}

// ReleaseInstaller installs an already verified release for the current platform.
type ReleaseInstaller interface {
	Install(context.Context, model.Release) error
}
