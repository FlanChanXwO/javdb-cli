package update

import (
	"context"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/buildinfo"
	"github.com/FlanChanXwO/javdb-cli/internal/update/model"
)

func TestCoordinatorInstallsReleaseOnlyWhenNewer(t *testing.T) {
	checker := &fakeReleaseChecker{release: &model.Release{TagName: "v0.2.0"}}
	installer := &fakeInstaller{}
	coordinator, err := NewCoordinator(CoordinatorOptions{
		SourceDetector:   SourceDetectorFunc(func(buildinfo.Info) (model.InstallSource, error) { return model.InstallSourceRelease, nil }),
		ReleaseChecker:   checker,
		ReleaseInstaller: installer,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := coordinator.Execute(context.Background(), model.Request{BuildInfo: buildinfo.Info{Version: "v0.1.1"}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.UpdateAvailable || len(installer.releases) != 1 || installer.releases[0].TagName != "v0.2.0" {
		t.Fatalf("result=%+v installer=%#v", result, installer.releases)
	}
	result, err = coordinator.Execute(context.Background(), model.Request{BuildInfo: buildinfo.Info{Version: "v0.2.0"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.UpdateAvailable || len(installer.releases) != 1 {
		t.Fatalf("result=%+v installer=%#v", result, installer.releases)
	}
}

func TestCoordinatorCheckDoesNotInstall(t *testing.T) {
	installer := &fakeInstaller{}
	coordinator, err := NewCoordinator(CoordinatorOptions{
		SourceDetector:   SourceDetectorFunc(func(buildinfo.Info) (model.InstallSource, error) { return model.InstallSourceRelease, nil }),
		ReleaseChecker:   &fakeReleaseChecker{release: &model.Release{TagName: "v0.2.0"}},
		ReleaseInstaller: installer,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := coordinator.Execute(context.Background(), model.Request{BuildInfo: buildinfo.Info{Version: "v0.1.1"}, Check: true})
	if err != nil {
		t.Fatal(err)
	}
	if !result.UpdateAvailable || len(installer.releases) != 0 {
		t.Fatalf("result=%+v installer=%#v", result, installer.releases)
	}
}

func TestCoordinatorUsesPackageManagerForManagedInstall(t *testing.T) {
	runner := &fakeRunner{}
	coordinator, err := NewCoordinator(CoordinatorOptions{
		SourceDetector:   SourceDetectorFunc(func(buildinfo.Info) (model.InstallSource, error) { return model.InstallSourceGoInstall, nil }),
		ReleaseChecker:   &fakeReleaseChecker{release: &model.Release{TagName: "v0.2.0"}},
		CommandRunner:    runner,
		ReleaseInstaller: &fakeInstaller{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.Execute(context.Background(), model.Request{BuildInfo: buildinfo.Info{Version: "v0.1.1"}}); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 1 || runner.calls[0] != "go install github.com/FlanChanXwO/javdb-cli/cmd/javdb@v0.2.0" {
		t.Fatalf("runner calls = %#v", runner.calls)
	}
}

func TestCoordinatorRejectsDevelopmentBuildWithoutNetwork(t *testing.T) {
	checker := &fakeReleaseChecker{release: &model.Release{TagName: "v0.2.0"}}
	coordinator, err := NewCoordinator(CoordinatorOptions{
		SourceDetector:   SourceDetectorFunc(func(buildinfo.Info) (model.InstallSource, error) { return model.InstallSourceDevelopment, nil }),
		ReleaseChecker:   checker,
		ReleaseInstaller: &fakeInstaller{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.Execute(context.Background(), model.Request{BuildInfo: buildinfo.Info{Version: "dev"}, Check: true}); err == nil {
		t.Fatal("development build check unexpectedly succeeded")
	}
	if checker.calls != 0 {
		t.Fatalf("release checker called %d times", checker.calls)
	}
}

type fakeReleaseChecker struct {
	release *model.Release
	calls   int
}

func (f *fakeReleaseChecker) Check(context.Context, bool) (*model.Release, error) {
	f.calls++
	return f.release, nil
}

type fakeInstaller struct {
	releases []model.Release
}

func (f *fakeInstaller) Install(_ context.Context, release model.Release) error {
	f.releases = append(f.releases, release)
	return nil
}

type fakeRunner struct {
	calls []string
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) error {
	call := name
	for _, arg := range args {
		call += " " + arg
	}
	f.calls = append(f.calls, call)
	return nil
}
