package cli

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
)

const usageTemplate = `{{T "usage.usage"}}{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

{{T "usage.aliases"}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{T "usage.examples"}}
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

{{T "usage.available_commands"}}{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

{{T "usage.additional_commands"}}{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{T "usage.flags"}}
{{localizedFlagUsages .LocalFlags | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{T "usage.global_flags"}}
{{localizedFlagUsages .InheritedFlags | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

{{T "usage.additional_help_topics"}}{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

{{printf (T "usage.more_info") .CommandPath}}{{end}}
`

func init() {
	cobra.AddTemplateFunc("T", i18n.T)
	defaultValue := regexp.MustCompile(`(?m)\(default .*\)$`)
	cobra.AddTemplateFunc("localizedFlagUsages", func(flags *pflag.FlagSet) string {
		return defaultValue.ReplaceAllStringFunc(flags.FlagUsages(), func(value string) string {
			return fmt.Sprintf(i18n.T("help.default_value"), strings.TrimSuffix(strings.TrimPrefix(value, "(default "), ")"))
		})
	})
}

func localizeBuiltinHelp(command *cobra.Command) {
	command.InitDefaultHelpFlag()
	path := command.CommandPath()
	if path == "javdb help" || path == "javdb completion" || strings.HasPrefix(path, "javdb completion ") {
		key := strings.ReplaceAll(strings.TrimPrefix(path, "javdb "), " ", ".")
		command.Short = i18n.T(key + ".short")
		if command.Long != "" {
			command.Long = i18n.T(key + ".long")
		}
		if flag := command.Flags().Lookup("no-descriptions"); flag != nil {
			flag.Usage = i18n.T("completion.no_descriptions")
		}
	}

	if flag := command.Flags().Lookup("help"); flag != nil {
		flag.Usage = fmt.Sprintf(i18n.T("help.for_command"), command.Name())
	}
	if flag := command.Flags().Lookup("version"); flag != nil {
		flag.Usage = fmt.Sprintf(i18n.T("help.version"), command.Name())
	}
	for _, child := range command.Commands() {
		localizeBuiltinHelp(child)
	}
}
