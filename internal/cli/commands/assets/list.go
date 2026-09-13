package assets

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// NewList builds the `assets list NUMBER [SELECTOR...]` command.
// 处理顺序固定:获取全部资产 → --type 过滤 → 生成 1..N 编号 → selector;
// 编号只是当前过滤结果的顺序位置,不是长期资产 ID。
func NewList(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var typeFilter string
	var asJSON, asNDJSON bool

	cmd := &cobra.Command{
		Use:   "list NUMBER [SELECTOR...]",
		Short: "List media assets attached to a movie",
		Long: "List the media assets of a movie: thumbnail, cover, preview images and preview video. " +
			"Optional selectors are 1-based positions in the current (filtered) list, e.g. 1, 1-4, 1,3-5, or several: 1 3 5. " +
			"Without a selector every asset of the requested type is listed. " +
			"Pipe output is TYPE<TAB>URL per line and feeds `javdb assets download`.",
		Example: "  javdb assets list SSIS-589\n" +
			"  javdb assets list SSIS-589 --type image 1-4 | javdb assets download -d ./images\n" +
			"  javdb assets list SSIS-589 --type video | javdb assets download -o preview.mp4",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := pipeline.ResolveOutputMode(asNDJSON, asJSON, streams.OutIsTerminal)
			if err != nil {
				return err
			}
			return client.WithOptionalAuth(options, streams.Err, func(c *javdb.Client) error {
				ctx := cmd.Context()
				movieID, err := c.ResolveMovieID(ctx, args[0])
				if err != nil {
					return err
				}
				movie, err := c.MovieDetail(ctx, movieID)
				if err != nil {
					return fmt.Errorf("assets list failed: %w", err)
				}
				assets := javdb.MovieAssetsFromDetail(movie)
				descs := javdb.MovieAssetDescriptions(movie)
				assets, descs, err = filterAssets(assets, descs, typeFilter)
				if err != nil {
					return err
				}
				assets, descs, err = selectAssets(assets, descs, strings.Join(args[1:], " "))
				if err != nil {
					return err
				}
				return renderAssetList(streams.Out, mode, assets, descs)
			})
		},
	}
	cmd.Flags().StringVar(&typeFilter, "type", "", "Filter by asset type: image or video")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON array")
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, "One JSON object per line")
	return cmd
}

// filterAssets 按 --type 过滤;仅接受 image/video,空串表示全部。
func filterAssets(assets []javdb.MovieAsset, descs []string, typeFilter string) ([]javdb.MovieAsset, []string, error) {
	switch typeFilter {
	case "":
		return assets, descs, nil
	case "image", "video":
		filtered := make([]javdb.MovieAsset, 0, len(assets))
		filteredDescs := make([]string, 0, len(descs))
		for i, asset := range assets {
			if asset.Type == typeFilter {
				filtered = append(filtered, asset)
				filteredDescs = append(filteredDescs, descs[i])
			}
		}
		return filtered, filteredDescs, nil
	default:
		return nil, nil, fmt.Errorf("invalid --type %q (image or video)", typeFilter)
	}
}

// selectAssets 应用 1-based selector;无 selector 时原样返回全部。
func selectAssets(assets []javdb.MovieAsset, descs []string, selector string) ([]javdb.MovieAsset, []string, error) {
	selected, err := parseAssetSelector(selector)
	if err != nil {
		return nil, nil, err
	}
	if selected == nil {
		return assets, descs, nil
	}
	for _, n := range selected {
		if n > len(assets) {
			return nil, nil, fmt.Errorf("asset number %d out of range (1-%d)", n, len(assets))
		}
	}
	picked := make([]javdb.MovieAsset, 0, len(selected))
	pickedDescs := make([]string, 0, len(selected))
	for _, n := range selected {
		picked = append(picked, assets[n-1])
		pickedDescs = append(pickedDescs, descs[n-1])
	}
	return picked, pickedDescs, nil
}

// renderAssetList 按输出模式写出资产列表。
// 描述文本只在 TTY 模式渲染;pipe/JSON/NDJSON 严格只含 type 与 url。
func renderAssetList(out io.Writer, mode pipeline.OutputMode, assets []javdb.MovieAsset, descs []string) error {
	switch mode {
	case pipeline.OutputHuman:
		renderAssetTable(out, assets, descs)
	case pipeline.OutputText:
		for _, asset := range assets {
			if _, err := fmt.Fprintf(out, "%s\t%s\n", asset.Type, asset.URL); err != nil {
				return err
			}
		}
	case pipeline.OutputJSON:
		line, err := jsonx.MarshalLine(assets)
		if err != nil {
			return err
		}
		_, err = out.Write(line)
		return err
	case pipeline.OutputNDJSON:
		for _, asset := range assets {
			line, err := jsonx.MarshalLine(asset)
			if err != nil {
				return err
			}
			if _, err := out.Write(line); err != nil {
				return err
			}
		}
	}
	return nil
}

// renderAssetTable 输出 TTY 编号表格;描述列仅存在于 TTY 渲染层。
func renderAssetTable(out io.Writer, assets []javdb.MovieAsset, descs []string) {
	width := len(strconv.Itoa(len(assets)))
	if width < 1 {
		width = 1
	}
	fmt.Fprintf(out, "%*s  %-5s  DESCRIPTION\n", width, "#", "TYPE")
	for i, asset := range assets {
		fmt.Fprintf(out, "%*d  %-5s  %s\n", width, i+1, asset.Type, descs[i])
	}
}
