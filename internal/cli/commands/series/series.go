// Package series 提供实体影片列表命令。
package series

import (
	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/entity/entitycmd"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

// New builds the series filmography command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	return entitycmd.New("series", "series REF", "List movies for a series (id or name)", options, streams)
}
