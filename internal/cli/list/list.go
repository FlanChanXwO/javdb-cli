// Package list 提供实体影片列表命令。
package list

import (
	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/command"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// New builds the list filmography command.
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	return command.New("list", "list REF", i18n.T("command.list"), options, streams)
}
