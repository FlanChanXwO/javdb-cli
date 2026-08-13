// Package list 提供实体影片列表命令。
package list

import (
	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/entity/entitycmd"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

// New builds the list filmography command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	return entitycmd.New("list", "list REF", "List movies inside a 合集 (user playlist)", options, streams)
}
