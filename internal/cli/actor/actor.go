// Package actor 提供实体影片列表命令。
package actor

import (
	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/command"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// New builds the actor filmography command.
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	return command.New("actor", "actor REF", i18n.T("cli.list_movies_for_an_actor_id_or_name"), options, streams)
}
