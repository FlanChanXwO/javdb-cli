// Package download 提供影片媒体下载命令。
package download

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the media download command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var thumbnailPath, previewImagePath, previewVideoPath string
	var isID bool
	var asJSON, asNDJSON bool

	pathFlags := func() []string {
		return []string{thumbnailPath, previewImagePath, previewVideoPath}
	}
	hasPlaceholder := func() bool {
		for _, path := range pathFlags() {
			if strings.Contains(path, "{number}") || strings.Contains(path, "{id}") {
				return true
			}
		}
		return false
	}
	// expand 对单个输入展开 {number}/{id} 占位符。
	expand := func(input pipeline.Envelope, c *javdb.Client, ctx context.Context) (expandedPaths, string, error) {
		number := input.Ref
		movieID := input.ID
		var err error
		if movieID == "" {
			if isID {
				// --id：ref 本身就是内部 movie id，绝不当作番号搜索
				// （ResolveMovieID 会在无精确匹配时回退首项，可能下载错影片）。
				movieID = number
			} else {
				movieID, err = c.ResolveMovieID(ctx, number)
				if err != nil {
					return expandedPaths{}, "", err
				}
			}
		}
		return expandedPaths{
			thumb: expandOne(thumbnailPath, number, movieID),
			image: expandOne(previewImagePath, number, movieID),
			video: expandOne(previewVideoPath, number, movieID),
		}, movieID, nil
	}
	// preflight 在开始写入前展开全部目标并检查唯一性、父目录与已存在文件。
	preflight := func(inputs []pipeline.Envelope, c *javdb.Client, ctx context.Context) error {
		seen := map[string]bool{}
		for _, input := range inputs {
			expanded, _, err := expand(input, c, ctx)
			if err != nil {
				return err
			}
			for _, path := range expanded.list() {
				if seen[path] {
					return fmt.Errorf("download targets must be unique, got duplicate %q", path)
				}
				seen[path] = true
				if _, err := os.Stat(path); err == nil {
					return fmt.Errorf("download target already exists: %s", path)
				} else if !os.IsNotExist(err) {
					return fmt.Errorf("check download target %q: %w", path, err)
				}
				parent := filepath.Dir(path)
				info, err := os.Stat(parent)
				if err != nil || !info.IsDir() {
					return fmt.Errorf("download target parent directory does not exist: %s", parent)
				}
			}
		}
		return nil
	}

	runner := &pipeline.BatchRunner{
		Name:  "download",
		Kinds: []pipeline.Kind{pipeline.KindMovie},
		ClientFactory: func() (*javdb.Client, error) {
			return client.NewWithDefaultToken(options)
		},
		Preflight: func(inputs []pipeline.Envelope) error {
			if len(inputs) <= 1 {
				return nil
			}
			if !hasPlaceholder() {
				return fmt.Errorf("download batch targets must contain {number} or {id} placeholders")
			}
			c, err := client.NewWithDefaultToken(options)
			if err != nil {
				return err
			}
			return preflight(inputs, c, context.Background())
		},
		RunOne: func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
			expanded, movieID, err := expand(input, c, ctx)
			if err != nil {
				return pipeline.Envelope{}, err
			}
			result, err := c.DownloadMovieMedia(ctx, movieID, javdb.MovieMediaDownloadOptions{
				ThumbnailPath:    expanded.thumb,
				PreviewImagePath: expanded.image,
				PreviewVideoPath: expanded.video,
			})
			if err != nil {
				return pipeline.Envelope{}, err
			}
			data := map[string]any{"movie_id": movieID}
			if result.ThumbnailPath != "" {
				data["thumbnail"] = result.ThumbnailPath
			}
			if result.PreviewImagePath != "" {
				data["preview_image"] = result.PreviewImagePath
			}
			if result.PreviewVideoPath != "" {
				data["preview_video"] = result.PreviewVideoPath
			}
			return pipeline.New(pipeline.KindDownload, input.Ref, movieID).WithData(data), nil
		},
		Legacy: func(args []string) error {
			if !hasPlaceholder() && len(pathFlags()) == 0 {
				return fmt.Errorf("set at least one of --thumbnail, --preview-image, or --preview-video")
			}
			if strings.TrimSpace(thumbnailPath) == "" && strings.TrimSpace(previewImagePath) == "" && strings.TrimSpace(previewVideoPath) == "" {
				return fmt.Errorf("set at least one of --thumbnail, --preview-image, or --preview-video")
			}
			return client.WithOptionalAuth(options, streams.Err, func(c *javdb.Client) error {
				ctx := context.Background()
				movieID := args[0]
				var err error
				if !isID {
					movieID, err = c.ResolveMovieID(ctx, args[0])
					if err != nil {
						return err
					}
				}
				// 单项且带占位符时也展开。
				thumb, image, video := thumbnailPath, previewImagePath, previewVideoPath
				if hasPlaceholder() {
					thumb = expandOne(thumb, args[0], movieID)
					image = expandOne(image, args[0], movieID)
					video = expandOne(video, args[0], movieID)
				}
				result, err := c.DownloadMovieMedia(ctx, movieID, javdb.MovieMediaDownloadOptions{
					ThumbnailPath:    thumb,
					PreviewImagePath: image,
					PreviewVideoPath: video,
				})
				if err != nil {
					return err
				}
				if result.ThumbnailPath != "" {
					fmt.Fprintf(streams.Out, "thumbnail\t%s\t%d bytes\n", result.ThumbnailPath, result.ThumbnailBytes)
				}
				if result.PreviewImagePath != "" {
					fmt.Fprintf(streams.Out, "preview-image\t%s\t%d bytes\n", result.PreviewImagePath, result.PreviewImageBytes)
				}
				if result.PreviewVideoPath != "" {
					fmt.Fprintf(streams.Out, "preview-video\t%s\t%d bytes\n", result.PreviewVideoPath, result.PreviewVideoBytes)
				}
				return nil
			})
		},
	}
	cmd := &cobra.Command{
		Use:   "download NUMBER",
		Short: "Download selected movie media to new files",
		Long:  "Download a thumbnail, only the first preview image, and/or the complete preview video. Output paths must not already exist.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(thumbnailPath) == "" && strings.TrimSpace(previewImagePath) == "" && strings.TrimSpace(previewVideoPath) == "" {
				return fmt.Errorf("set at least one of --thumbnail, --preview-image, or --preview-video")
			}
			return runner.Execute(streams, args, asNDJSON, asJSON)
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	cmd.Flags().StringVar(&thumbnailPath, "thumbnail", "", "Save thumbnail to PATH (supports {number}/{id})")
	cmd.Flags().StringVar(&previewImagePath, "preview-image", "", "Save only the first preview image to PATH (supports {number}/{id})")
	cmd.Flags().StringVar(&previewVideoPath, "preview-video", "", "Save preview HLS video to PATH (supports {number}/{id})")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, "Pipeline NDJSON envelopes")
	return cmd
}

func expandOne(path, number, id string) string {
	return strings.ReplaceAll(strings.ReplaceAll(path, "{number}", number), "{id}", id)
}

// expandedPaths 是单个输入展开后的三个媒体路径。
type expandedPaths struct {
	thumb string
	image string
	video string
}

func (p expandedPaths) list() []string {
	var paths []string
	for _, path := range []string{p.thumb, p.image, p.video} {
		if strings.TrimSpace(path) != "" {
			paths = append(paths, path)
		}
	}
	return paths
}
