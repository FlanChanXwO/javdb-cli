package cli

import (
	"errors"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/logging"
	"github.com/FlanChanXwO/javdb-cli/internal/update/process"
)

func configureLogging(cmd *cobra.Command, streams *environment.Streams) error {
	streams.Command = cmd.CommandPath()
	streams.Logger = streams.Logger.With("command", streams.Command)
	path, err := paths.ConfigPath()
	if err != nil {
		return err
	}
	cfg, err := settings.LoadFile(path)
	if err != nil {
		return err
	}
	flags := cmd.Root().PersistentFlags()
	level, format := cfg.LogLevel, cfg.LogFormat
	if flags.Changed("log-level") {
		level, _ = flags.GetString("log-level")
	}
	if flags.Changed("log-format") {
		format, _ = flags.GetString("log-format")
	}
	logger, err := logging.New(streams.Err, level, format)
	if err != nil {
		// A bad level still reports its error in the explicitly selected JSON format.
		if format == "json" {
			fallback, _ := logging.New(streams.Err, "error", "json")
			streams.Logger = fallback.With("command", streams.Command)
		}
		return err
	}
	streams.Command = cmd.CommandPath()
	streams.Logger = logger.With("command", streams.Command)
	return nil
}

// Run executes the CLI, owns final diagnostics, and maps failures to exit status 1.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	command, streams := newRoot(stdin, stdout, stderr)
	command.SetArgs(args)
	command.SetIn(stdin)
	command.SetOut(stdout)
	command.SetErr(stderr)
	err := process.CleanupPendingWindowsUpdate()
	if err == nil {
		err = command.Execute()
	}
	if err == nil {
		return 0
	}
	if streams.Command == "" {
		flags := pflag.NewFlagSet("logging", pflag.ContinueOnError)
		flags.SetOutput(io.Discard)
		flags.ParseErrorsWhitelist.UnknownFlags = true
		flags.String("log-format", "", "")
		flags.String("log-level", "", "")
		if flagErr := flags.Parse(args); flagErr != nil && flagErr.Error() != err.Error() {
			err = errors.Join(err, flagErr)
		}
		flags.Visit(func(flag *pflag.Flag) {
			if setErr := command.PersistentFlags().Set(flag.Name, flag.Value.String()); setErr != nil {
				err = errors.Join(err, setErr)
			}
		})

		target := command
		if found, _, findErr := command.Find(args); findErr == nil {
			target = found
		}
		if configErr := configureLogging(target, streams); configErr != nil {
			err = errors.Join(err, configErr)
		}
	}
	if logErr := logging.Failure(streams.Logger, err); logErr != nil {
		// The diagnostic sink itself failed; preserve that failure on the execution's stderr.
		io.WriteString(stderr, logErr.Error()+"\n")
	}
	return 1
}
