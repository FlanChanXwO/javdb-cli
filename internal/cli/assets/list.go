package assets

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

type assetListItem struct {
	Type     string `json:"type"`
	URL      string `json:"url"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Duration int    `json:"duration,omitempty"`
}

// NewList builds the `assets list NUMBER [SELECTOR...]` command.
// 处理顺序固定:获取详情 → --type 过滤 → 生成 1..N 编号 → selector → metadata probe → 输出;
// 编号只是当前过滤结果的顺序位置,不是长期资产 ID。
func NewList(options *session.Options, streams *environment.Streams) *cobra.Command {
	var typeFilter string
	var asJSON, asNDJSON bool

	cmd := &cobra.Command{
		Use:   "list NUMBER [SELECTOR...]",
		Short: i18n.T("cli.list_media_assets_attached_to_a_movie"),
		Long:  i18n.T("cli.list_the_media_assets_of_a_movie_thumbnail_cover_preview_images_and_pr"),

		Example: i18n.T("cli.javdb_assets_list_abc_javdb_assets_list_abc_type_image_javdb_assets_do"),

		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := record.ResolveOutputMode(asNDJSON, asJSON, streams.OutIsTerminal)
			if err != nil {
				return err
			}
			probeSettings, err := resolveAssetProbeSettings()
			if err != nil {
				return err
			}
			return session.WithOptionalAuth(options, streams.Err, func(c *session.Client) error {
				ctx := cmd.Context()
				movieID, err := c.ResolveMovieID(ctx, args[0])
				if err != nil {
					return err
				}
				movie, err := c.MovieDetail(ctx, movieID)
				if err != nil {
					return err
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
				infos := assetInfosWithoutMetadata(assets)
				if probeSettings.Enabled {
					infos, err = c.ProbeMovieAssets(ctx, assets, javdb.MovieAssetProbeOptions{
						Concurrency: probeSettings.Concurrency,
					})
					if err != nil {
						return err
					}
				}
				return renderAssetList(streams.Out, mode, infos, descs)
			})
		},
	}
	cmd.Flags().StringVar(&typeFilter, "type", "", i18n.T("cli.filter_by_asset_type_image_or_video"))
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("cli.machine_readable_json_array"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("cli.one_json_object_per_line"))
	return cmd
}

func resolveAssetProbeSettings() (settings.ResolvedAssetProbe, error) {
	path, err := paths.ConfigPath()
	if err != nil {
		return settings.ResolvedAssetProbe{}, err
	}
	configured, err := settings.LoadFile(path)
	if err != nil {
		return settings.ResolvedAssetProbe{}, err
	}
	return settings.ResolveAssetProbe(configured)
}

func assetInfosWithoutMetadata(assets []javdb.MovieAsset) []javdb.MovieAssetInfo {
	infos := make([]javdb.MovieAssetInfo, len(assets))
	for index, asset := range assets {
		infos[index].Asset = asset
	}
	return infos
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
		return nil, nil, fmt.Errorf(i18n.T("cli.invalid_type_q_image_or_video"), typeFilter)
	}
}

// selectAssets 应用 1-based selector;无 selector 时原样返回全部。
// assetCount 是过滤后资产总数，range 在展开前先校验上界。
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
			return nil, nil, fmt.Errorf(i18n.T("cli.asset_number_d_out_of_range_d"), n, len(assets))
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
func renderAssetList(out io.Writer, mode record.OutputMode, infos []javdb.MovieAssetInfo, descs []string) error {
	switch mode {
	case record.OutputHuman:
		return renderAssetTable(out, infos, descs)
	case record.OutputText:
		for _, info := range infos {
			asset := info.Asset
			if _, err := fmt.Fprintf(out, "%s\t%s\n", asset.Type, asset.URL); err != nil {
				return err
			}
		}
	case record.OutputJSON:
		line, err := jsonx.MarshalLine(flattenAssetInfos(infos))
		if err != nil {
			return err
		}
		n, err := out.Write(line)
		if err != nil {
			return err
		}
		if n != len(line) {
			return io.ErrShortWrite
		}
		return nil
	case record.OutputNDJSON:
		for _, item := range flattenAssetInfos(infos) {
			line, err := jsonx.MarshalLine(item)
			if err != nil {
				return err
			}
			n, err := out.Write(line)
			if err != nil {
				return err
			}
			if n != len(line) {
				return io.ErrShortWrite
			}
		}
	}
	return nil
}

func flattenAssetInfos(infos []javdb.MovieAssetInfo) []assetListItem {
	items := make([]assetListItem, len(infos))
	for index, info := range infos {
		items[index] = assetListItem{
			Type:     info.Asset.Type,
			URL:      info.Asset.URL,
			Width:    info.Metadata.Width,
			Height:   info.Metadata.Height,
			Duration: info.Metadata.Duration,
		}
	}
	return items
}

// renderAssetTable 输出 TTY 编号表格;描述列仅存在于 TTY 渲染层。
func renderAssetTable(out io.Writer, infos []javdb.MovieAssetInfo, descs []string) error {
	width := len(strconv.Itoa(len(infos)))
	if width < 1 {
		width = 1
	}
	if _, err := fmt.Fprintf(out, i18n.T("cli.s_s_s_s_description"), width, "#", i18n.T("asset.type"), i18n.T("asset.size"), i18n.T("asset.duration")); err != nil {
		return err
	}
	for i, info := range infos {
		asset := info.Asset
		if _, err := fmt.Fprintf(out, "%*d  %-5s  %-9s  %-8s  %s\n", width, i+1, asset.Type, formatAssetSize(info.Metadata), formatAssetDuration(info.Metadata), descs[i]); err != nil {
			return err
		}
	}
	return nil
}

func formatAssetSize(metadata javdb.MovieAssetMetadata) string {
	if metadata.Width <= 0 || metadata.Height <= 0 {
		return "-"
	}
	return fmt.Sprintf("%dx%d", metadata.Width, metadata.Height)
}

func formatAssetDuration(metadata javdb.MovieAssetMetadata) string {
	if metadata.Duration <= 0 {
		return "-"
	}
	return strconv.Itoa(metadata.Duration) + "s"
}
