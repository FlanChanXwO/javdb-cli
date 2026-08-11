package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/buildinfo"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
)

// New builds the version command.
func New() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(cmd *cobra.Command, args []string) error {
			info := buildinfo.Current()
			if asJSON {
				// brew formula asserts version_info["version"] == "v#{version}"
				b, err := jsonx.MarshalLine(map[string]string{
					"version":    info.Version,
					"commit":     info.Commit,
					"build_date": info.BuildDate,
				})
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write(b)
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "javdb %s\ncommit: %s\nbuild date: %s\n",
				info.Version, info.Commit, info.BuildDate)
			return err
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	return cmd
}
