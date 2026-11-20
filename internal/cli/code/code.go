// Package code 提供实体影片列表命令。
package code

import (
	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/command"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// New builds the code filmography command.
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	return command.New("code", "code REF", i18n.T("cli.list_movies_for_a_code_prefix_e_g_ssis"), options, streams)
}
