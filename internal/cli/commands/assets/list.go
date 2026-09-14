package assets

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/media"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// NewList builds the `assets list NUMBER [SELECTOR...]` command.
// 处理顺序固定(计划 #2):获取详情 → --type 过滤 → 生成 1..N 编号 → selector
// → metadata probe(仅 TTY/JSON/NDJSON)→ 输出;
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
			"Pipe output is TYPE<TAB>URL per line and feeds `javdb assets download`. " +
			"JSON/NDJSON output includes optional width/height/duration metadata (best-effort probe).",
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
				// 只有真正消费元信息的输出模式执行 probe(计划 #2):
				// TTY/--json/--ndjson probe;普通 pipe 文本模式完全跳过,
				// 常规下载链路不为 20～30 张预览图增加 N 个无意义请求。
				if mode != pipeline.OutputText {
					assets = probeAssets(cmd.Context(), c, assets)
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

// probeAssets 对最终选中的资产做 best-effort metadata probe(计划 #2/#6)。
// probe 配置来自 config.toml [assets.probe];同一次调用内按 URL 去重;
// probe 失败只省略 metadata,不让整个 list 失败。
func probeAssets(ctx context.Context, c *javdb.Client, assets []javdb.MovieAsset) []javdb.MovieAsset {
	limits := loadProbeLimits()
	if !limits.Enabled || len(assets) == 0 {
		return assets
	}
	types := make([]string, len(assets))
	urls := make([]string, len(assets))
	for i, asset := range assets {
		types[i] = asset.Type
		urls[i] = asset.URL
	}
	results := media.ProbeAssetsConcurrent(ctx, c.ProbeMedia, types, urls, media.ProbeLimits{
		Enabled:              limits.Enabled,
		Concurrency:          limits.Concurrency,
		ImageMaxBytes:        limits.ImageMaxBytes,
		PlaylistMaxBytes:     limits.PlaylistMaxBytes,
		VideoSegmentMaxBytes: limits.VideoSegmentMaxBytes,
		TimeoutSeconds:       limits.TimeoutSeconds,
	})
	for i := range assets {
		if results[i].HasDimensions() {
			assets[i].Width = results[i].Width
			assets[i].Height = results[i].Height
		}
		if results[i].HasDuration() {
			assets[i].Duration = results[i].Duration
		}
	}
	return assets
}

// loadProbeLimits 从 config.toml 读取 [assets.probe] 配置;
// 配置缺失/损坏时使用默认值(probe 是 best-effort,不阻塞 list)。
func loadProbeLimits() javdb.ProbeLimits {
	path, err := paths.ConfigPath()
	if err != nil {
		return javdb.DefaultProbeLimits()
	}
	cfg, err := settings.LoadFile(path)
	if err != nil {
		return javdb.DefaultProbeLimits()
	}
	probe := cfg.Assets.Probe
	return javdb.ProbeLimits{
		Enabled:              probe.EnabledValue(),
		Concurrency:          probe.ConcurrencyValue(),
		ImageMaxBytes:        probe.ImageMaxBytesValue(),
		PlaylistMaxBytes:     probe.PlaylistMaxBytesValue(),
		VideoSegmentMaxBytes: probe.VideoSegmentMaxBytesValue(),
		TimeoutSeconds:       parseProbeTimeout(probe.TimeoutValue()),
	}
}

// parseProbeTimeout 把配置的 duration 字符串转成秒;解析失败返回 0
// (无额外超时),不阻塞 list。
func parseProbeTimeout(value string) float64 {
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0
	}
	return d.Seconds()
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
// assetCount 是过滤后资产总数:range 在展开前先校验上界(计划 #9)。
func selectAssets(assets []javdb.MovieAsset, descs []string, selector string) ([]javdb.MovieAsset, []string, error) {
	selected, err := parseAssetSelector(selector, len(assets))
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
