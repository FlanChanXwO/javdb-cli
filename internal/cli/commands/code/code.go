// Package code 提供实体影片列表命令。
package code

import (
	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/entity/entitycmd"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

// New builds the code filmography command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	return entitycmd.New("code", "code REF", "List movies for a code/prefix e.g. SSIS", options, streams)
}
