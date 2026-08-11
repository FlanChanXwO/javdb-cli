// Package download 提供影片媒体下载命令。
package download

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the media download command.
func New(flags *app.Flags, aio *app.IO) *cobra.Command {
	var thumbnailPath, previewImagePath, previewVideoPath string
	var isID bool
	cmd := &cobra.Command{
		Use:   "download NUMBER",
		Short: "Download selected movie media to new files",
		Long:  "Download a thumbnail, only the first preview image, and/or the complete preview video. Output paths must not already exist.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(thumbnailPath) == "" && strings.TrimSpace(previewImagePath) == "" && strings.TrimSpace(previewVideoPath) == "" {
				return fmt.Errorf("set at least one of --thumbnail, --preview-image, or --preview-video")
			}
			return app.WithOptionalAuthClient(flags, aio, func(c *javdb.Client) error {
				ctx := context.Background()
				movieID := args[0]
				var err error
				if !isID {
					movieID, err = c.ResolveMovieID(ctx, args[0])
					if err != nil {
						return err
					}
				}
				result, err := c.DownloadMovieMedia(ctx, movieID, javdb.MovieMediaDownloadOptions{
					ThumbnailPath:    thumbnailPath,
					PreviewImagePath: previewImagePath,
					PreviewVideoPath: previewVideoPath,
				})
				if err != nil {
					return err
				}
				if result.ThumbnailPath != "" {
					fmt.Fprintf(aio.Out, "thumbnail\t%s\t%d bytes\n", result.ThumbnailPath, result.ThumbnailBytes)
				}
				if result.PreviewImagePath != "" {
					fmt.Fprintf(aio.Out, "preview-image\t%s\t%d bytes\n", result.PreviewImagePath, result.PreviewImageBytes)
				}
				if result.PreviewVideoPath != "" {
					fmt.Fprintf(aio.Out, "preview-video\t%s\t%d bytes\n", result.PreviewVideoPath, result.PreviewVideoBytes)
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	cmd.Flags().StringVar(&thumbnailPath, "thumbnail", "", "Save thumbnail to PATH")
	cmd.Flags().StringVar(&previewImagePath, "preview-image", "", "Save only the first preview image to PATH")
	cmd.Flags().StringVar(&previewVideoPath, "preview-video", "", "Save preview HLS video to PATH")
	return cmd
}
