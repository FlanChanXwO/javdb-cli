package update

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

// CommandRunner executes package-manager commands without shell interpolation.
type CommandRunner interface {
	Run(context.Context, string, ...string) error
}

type commandRunner struct {
	stdout io.Writer
	stderr io.Writer
}

// NewCommandRunner returns the production package-manager runner.
func NewCommandRunner(stdout, stderr io.Writer) CommandRunner {
	return commandRunner{stdout: stdout, stderr: stderr}
}

func (r commandRunner) Run(ctx context.Context, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Stdout = r.stdout
	command.Stderr = r.stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("run %s: %w", name, err)
	}
	return nil
}
