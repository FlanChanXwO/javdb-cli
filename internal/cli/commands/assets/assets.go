// Package assets 提供 javdb assets 命令域:影片媒体资产(图片/预览视频)的发现与下载。
// 这里只处理影片附属媒体资产;完整影片/磁力下载不是本命令域的职责。
package assets

import (
	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

// New builds the assets command with its subcommands.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assets",
		Short: "Discover and download movie media assets",
		Long: "Discover (list) and download the media assets attached to a movie: " +
			"thumbnail, cover, preview images and the preview video. " +
			"This command does not download full movies or magnets.",
	}
	cmd.AddCommand(NewList(options, streams))
	cmd.AddCommand(NewDownload(options, streams))
	return cmd
}
