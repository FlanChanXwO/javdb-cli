// Package director 提供实体影片列表命令。
package director

import (
	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/entity/entitycmd"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

// New builds the director filmography command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	return entitycmd.New("director", "director REF", "List movies for a director (id or name)", options, streams)
}
