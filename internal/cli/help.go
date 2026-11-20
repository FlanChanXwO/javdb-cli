package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
)

func init() {
	cobra.AddTemplateFunc("localizedFlagUsages", func(flags *pflag.FlagSet) string {
		return strings.ReplaceAll(flags.FlagUsages(), "(default ", "("+i18n.T("help.default")+" ")
	})
}

func localizeBuiltinHelp(command *cobra.Command) {
	command.InitDefaultHelpFlag()
	path := command.CommandPath()
	if path == "javdb help" || path == "javdb completion" || strings.HasPrefix(path, "javdb completion ") {
		key := strings.ReplaceAll(path, " ", ".")
		command.Short = i18n.T(key + ".short")
		if command.Long != "" {
			command.Long = i18n.T(key + ".long")
		}
		command.Flags().VisitAll(func(flag *pflag.Flag) {
			if flag.Name != "help" && flag.Name != "version" {
				flag.Usage = i18n.T(key + ".flag." + flag.Name)
			}
		})
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
