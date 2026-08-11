// Command releasenotes audits and validates the versioned changelog contract.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/audit"
	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/document"
	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/history"
	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/prepare"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("a subcommand is required: validate, audit, prepare, render, pr-validate, or sync-history")
	}
	switch arguments[0] {
	case "validate":
		return document.RunValidate(arguments[1:])
	case "audit":
		return audit.Run(arguments[1:])
	case "prepare":
		return prepare.Run(arguments[1:])
	case "render":
		return document.RunRender(arguments[1:])
	case "pr-validate":
		return audit.RunPullRequestValidate(arguments[1:])
	case "sync-history":
		return history.Run(arguments[1:])
	case "-h", "--help", "help":
		return errors.New("usage: releasenotes validate|audit|prepare|render|pr-validate|sync-history")
	default:
		return fmt.Errorf("unknown subcommand %q", arguments[0])
	}
}
