package assets

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/common/atomicfile"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

type assetDownloader interface {
	DownloadMovieAsset(context.Context, javdb.MovieAsset, string) (int64, error)
}

// stdinLineLimit 是 stdin 单行记录的明确上限：64 KiB。
const stdinLineLimit = 64 * 1024

// NewDownload builds the `assets download` pipe consumer.
// 输入是 `assets list` 的 TYPE<TAB>URL 记录流，流式逐条处理：
// scan → parse → download → next,不缓存全部记录。
// -d/-o 与默认命名在失败时不落盘，并且绝不覆盖已有文件。
// stdout 只输出最终路径，不输出 saved/bytes 装饰。
func NewDownload(options *session.Options, streams *environment.Streams) *cobra.Command {
	var dir, out string

	cmd := &cobra.Command{
		Use:   "download",
		Short: i18n.T("command.assets_download"),
		Long:  i18n.T("assets.download_description"),

		Example: "  javdb assets list ABC-123 --type image 1-4 | javdb assets download -d ./media\n  javdb assets list ABC-123 --type video | javdb assets download -o preview.mp4",

		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return session.WithOptionalAuth(options, streams.Err, func(c *session.Client) error {
				return downloadAssetsStream(cmd.Context(), c, streams, dir, out)
			})
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", i18n.T("flag.output_directory"))
	cmd.Flags().StringVarP(&out, "out", "o", "", i18n.T("flag.output_file"))
	return cmd
}

// downloadAssetsStream 流式处理 stdin 的 TYPE<TAB>URL 记录：
// 逐条 scan → parse → download → next,不缓存全部记录。
// -o 只需要读取第一条,再尝试读取第二条:存在第二条则报错,无需读完整个 stdin。
func downloadAssetsStream(ctx context.Context, c assetDownloader, streams *environment.Streams, dir, out string) error {
	dir = filepath.Clean(dir)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return fmt.Errorf(i18n.T("assets.output_not_directory"), dir)
	}
	scanner := bufio.NewScanner(streams.In)
	scanner.Buffer(make([]byte, 0, stdinLineLimit), stdinLineLimit)
	pos := 0
	lineNo := 0
	var firstRecord *assetRecord
	for scanner.Scan() {
		lineNo++
		line := strings.TrimRight(scanner.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		// pos 是当前记录在非空记录流中的位置,空行不占编号。
		pos++
		record, err := parseAssetRecord(line, lineNo)
		if err != nil {
			return err
		}
		if out != "" {
			// -o 模式只需要读取第一条，再尝试读取第二条：
			// 存在第二条则报错,无需把整个 stdin 读完。
			if firstRecord != nil {
				return i18n.Errorf("assets.single_output_required")
			}
			firstRecord = &record
			continue
		}
		target, err := downloadAutoNamed(ctx, c, record, dir, pos)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(streams.Out, target); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf(i18n.T("input.read_failed"), err)
	}
	if out != "" {
		if firstRecord == nil {
			return i18n.Errorf("assets.input_required")
		}
		written, err := c.DownloadMovieAsset(ctx, javdb.MovieAsset{Type: firstRecord.Type, URL: firstRecord.URL}, out)
		if err != nil {
			return err
		}
		_ = written
		if _, err := fmt.Fprintln(streams.Out, out); err != nil {
			return err
		}
		return nil
	}
	if pos == 0 {
		return i18n.Errorf("assets.input_required")
	}
	return nil
}

// assetRecord 是 pipe 输入中的一条 TYPE<TAB>URL 记录。
type assetRecord struct {
	Type string
	URL  string
}

// parseAssetRecord 解析单条 TYPE<TAB>URL 记录;空行跳过,坏行带行号报错。
func parseAssetRecord(line string, lineNo int) (assetRecord, error) {
	parts := strings.Split(line, "\t")
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return assetRecord{}, fmt.Errorf(i18n.T("assets.invalid_input"), lineNo, line)
	}
	assetType := strings.TrimSpace(parts[0])
	switch assetType {
	case "image", "video":
	default:
		return assetRecord{}, fmt.Errorf(i18n.T("assets.invalid_input_type"), lineNo, assetType)
	}
	return assetRecord{Type: assetType, URL: strings.TrimSpace(parts[1])}, nil
}

// downloadAutoNamed 处理自动命名:图片下载后按 magic 定扩展名,视频默认 .mp4。
func downloadAutoNamed(ctx context.Context, c assetDownloader, record assetRecord, dir string, pos int) (string, error) {
	switch record.Type {
	case "video":
		target := filepath.Join(dir, fmt.Sprintf("video-%03d.mp4", pos))
		if _, err := c.DownloadMovieAsset(ctx, javdb.MovieAsset{Type: record.Type, URL: record.URL}, target); err != nil {
			return "", err
		}
		return target, nil
	case "image":
		// os.MkdirTemp 生成一次性目录以保证临时文件名唯一，
		// 下载产物发布到目录内唯一路径,无并发/残留冲突。
		tmpDir, err := os.MkdirTemp(dir, ".assets-download-")
		if err != nil {
			return "", err
		}
		cleanupDir := func(primary error) error {
			cleanupErr := os.RemoveAll(tmpDir)
			if primary != nil {
				return errors.Join(primary, cleanupErr)
			}
			return cleanupErr
		}
		tmp := filepath.Join(tmpDir, "image.raw")
		if _, err := c.DownloadMovieAsset(ctx, javdb.MovieAsset{Type: record.Type, URL: record.URL}, tmp); err != nil {
			return "", cleanupDir(err)
		}
		format, ok := javdb.ImageAssetFormat(tmp)
		if !ok {
			return "", cleanupDir(i18n.Errorf("assets.unrecognized_image"))
		}
		target := filepath.Join(dir, fmt.Sprintf("image-%03d.%s", pos, format))
		if err := atomicfile.LinkNoReplace(tmp, target); err != nil {
			return "", cleanupDir(fmt.Errorf(i18n.T("assets.publish_failed"), target, err))
		}
		if err := os.Remove(tmp); err != nil {
			return "", cleanupDir(fmt.Errorf(i18n.T("assets.remove_temp_file"), err))
		}
		if err := os.Remove(tmpDir); err != nil {
			return "", fmt.Errorf(i18n.T("assets.remove_temp_directory"), err)
		}
		return target, nil
	default:
		return "", fmt.Errorf(i18n.T("assets.unsupported_type"), record.Type)
	}
}
