// Package assets 提供 javdb assets 命令域:影片媒体资产(图片/预览视频)的发现与下载。
// 这里只处理影片附属媒体资产;完整影片/磁力下载不是本命令域的职责。
package assets

import (
	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// New builds the assets command with its subcommands.
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assets",
		Short: i18n.T("cli.discover_and_download_movie_media_assets"),
		Long:  i18n.T("cli.discover_list_and_download_the_media_assets_attached_to_a_movie_thumbn"),
	}
	cmd.AddCommand(NewList(options, streams))
	cmd.AddCommand(NewDownload(options, streams))
	return cmd
}
