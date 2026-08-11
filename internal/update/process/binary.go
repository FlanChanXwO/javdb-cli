package process

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// BinaryChecker validates a downloaded binary before it can replace javdb.
type BinaryChecker interface {
	Check(context.Context, string, string) error
}

// BinaryCheckerFunc adapts a function for installer tests and调用方注入。
type BinaryCheckerFunc func(context.Context, string, string) error

// Check invokes the wrapped binary checker.
func (f BinaryCheckerFunc) Check(ctx context.Context, path, version string) error {
	return f(ctx, path, version)
}

// NewBinaryChecker returns the production checker for a staged javdb binary.
func NewBinaryChecker() BinaryChecker {
	return binaryChecker{}
}

type binaryChecker struct{}

func (binaryChecker) Check(ctx context.Context, executablePath, expectedVersion string) error {
	command := exec.CommandContext(ctx, executablePath, "version", "--json")
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("run candidate version command: %w", err)
	}
	var version struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(output, &version); err != nil {
		return fmt.Errorf("decode candidate version JSON: %w", err)
	}
	if version.Version != expectedVersion {
		return fmt.Errorf("candidate reports version %q, expected %q", version.Version, expectedVersion)
	}
	return nil
}
