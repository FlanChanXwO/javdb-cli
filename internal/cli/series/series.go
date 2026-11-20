// Package series 提供实体影片列表命令。
package series

import (
	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/command"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// New builds the series filmography command.
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	return command.New("series", "series REF", i18n.T("command.series"), options, streams)
}
