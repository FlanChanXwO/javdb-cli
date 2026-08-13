// Package actor 提供实体影片列表命令。
package actor

import (
	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/entity/entitycmd"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

// New builds the actor filmography command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	return entitycmd.New("actor", "actor REF", "List movies for an actor (id or name)", options, streams)
}
