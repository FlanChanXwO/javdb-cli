package assets

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// NewDownload builds the `assets download` pipe consumer.
// 输入是 `assets list` 的 TYPE<TAB>URL 记录流;-d/-o 与默认命名遵循
// input.md 计划 #13-#16/#42-#43:失败不落盘、绝不覆盖已有文件。
func NewDownload(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var dir, out string

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download assets from `assets list` pipe input",
		Long: "Download movie assets from TYPE<TAB>URL records piped by `javdb assets list`. " +
			"-d places auto-named files (image-001.jpg, video-001.mp4) into DIR; " +
			"-o writes a single asset to an exact path (.ts keeps the transport stream, " +
			".mp4 produces a fast-start MP4). Existing files are never overwritten.",
		Example: "  javdb assets list SSIS-589 --type image 1-4 | javdb assets download -d ./media\n" +
			"  javdb assets list SSIS-589 --type video | javdb assets download -o preview.mp4",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			assets, err := readAssetRecords(streams.In)
			if err != nil {
				return err
			}
			if len(assets) == 0 {
				return fmt.Errorf("assets download: no assets on stdin")
			}
			if out != "" && len(assets) != 1 {
				return fmt.Errorf("assets download: -o requires exactly one asset, got %d", len(assets))
			}
			return client.WithOptionalAuth(options, streams.Err, func(c *javdb.Client) error {
				return downloadAssets(cmd.Context(), c, streams, assets, dir, out)
			})
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Directory for auto-named output files")
	cmd.Flags().StringVarP(&out, "out", "o", "", "Exact output path for exactly one asset")
	return cmd
}

// assetRecord 是 pipe 输入中的一条 TYPE<TAB>URL 记录。
type assetRecord struct {
	Type string
	URL  string
}

// readAssetRecords 解析 stdin 的 TYPE<TAB>URL 记录流;空行跳过,坏行带行号报错。
func readAssetRecords(in io.Reader) ([]assetRecord, error) {
	scanner := bufio.NewScanner(in)
	var records []assetRecord
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimRight(scanner.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("invalid input at line %d: %q (want TYPE<TAB>URL)", lineNo, line)
		}
		assetType := strings.TrimSpace(parts[0])
		switch assetType {
		case "image", "video":
		default:
			return nil, fmt.Errorf("invalid input at line %d: unsupported asset type %q", lineNo, assetType)
		}
		records = append(records, assetRecord{Type: assetType, URL: strings.TrimSpace(parts[1])})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}
	return records, nil
}

// downloadAssets 逐个下载并落盘,输出 saved 行。
// 自动命名的图片先写到目录内临时文件,按 magic 检测结果确定最终扩展名后再发布,
// 发布前做冲突检查(#43);任何失败都保证最终目标不出现半成品。
func downloadAssets(ctx context.Context, c *javdb.Client, streams *invocation.Streams, records []assetRecord, dir, out string) error {
	dir = filepath.Clean(dir)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return fmt.Errorf("assets download: -d %q is not a directory", dir)
	}
	for i, record := range records {
		pos := i + 1
		if out != "" {
			written, err := c.DownloadMovieAsset(ctx, javdb.MovieAsset{Type: record.Type, URL: record.URL}, out)
			if err != nil {
				return err
			}
			fmt.Fprintf(streams.Out, "saved %s (%d bytes)\n", out, written)
			continue
		}
		target, written, err := downloadAutoNamed(ctx, c, record, dir, pos)
		if err != nil {
			return err
		}
		fmt.Fprintf(streams.Out, "saved %s (%d bytes)\n", target, written)
	}
	return nil
}

// downloadAutoNamed 处理自动命名:图片下载后按 magic 定扩展名,视频默认 .mp4。
func downloadAutoNamed(ctx context.Context, c *javdb.Client, record assetRecord, dir string, pos int) (string, int64, error) {
	switch record.Type {
	case "video":
		target := filepath.Join(dir, fmt.Sprintf("video-%03d.mp4", pos))
		written, err := c.DownloadMovieAsset(ctx, javdb.MovieAsset{Type: record.Type, URL: record.URL}, target)
		if err != nil {
			return "", 0, err
		}
		return target, written, nil
	case "image":
		tmp := filepath.Join(dir, fmt.Sprintf(".assets-download-%03d.tmp", pos))
		defer os.Remove(tmp)
		if _, err := c.DownloadMovieAsset(ctx, javdb.MovieAsset{Type: record.Type, URL: record.URL}, tmp); err != nil {
			return "", 0, err
		}
		format, ok := javdb.ImageAssetFormat(tmp)
		if !ok {
			return "", 0, fmt.Errorf("assets download: downloaded image is not a recognized format")
		}
		target := filepath.Join(dir, fmt.Sprintf("image-%03d.%s", pos, format))
		if _, err := os.Lstat(target); err == nil {
			return "", 0, fmt.Errorf("assets download: target already exists: %s", target)
		} else if !os.IsNotExist(err) {
			return "", 0, fmt.Errorf("assets download: check target %q: %w", target, err)
		}
		if err := os.Rename(tmp, target); err != nil {
			return "", 0, fmt.Errorf("assets download: publish %q: %w", target, err)
		}
		info, err := os.Stat(target)
		if err != nil {
			return "", 0, err
		}
		return target, info.Size(), nil
	default:
		return "", 0, fmt.Errorf("assets download: unsupported asset type %q", record.Type)
	}
}
