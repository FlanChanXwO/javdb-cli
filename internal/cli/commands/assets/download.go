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

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/common/atomicfile"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// stdinLineLimit 是 stdin 单行记录的明确上限(计划 #8):64 KiB。
const stdinLineLimit = 64 * 1024

// NewDownload builds the `assets download` pipe consumer.
// 输入是 `assets list` 的 TYPE<TAB>URL 记录流,流式逐条处理(计划 #8):
// scan → parse → download → next,不缓存全部记录。
// -d/-o 与默认命名遵循 input.md 计划 #13-#16/#42-#43:失败不落盘、绝不覆盖已有文件。
// stdout 只输出最终路径(计划 #7),不再输出 saved/bytes 装饰。
func NewDownload(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var dir, out string

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download assets from `assets list` pipe input",
		Long: "Download movie assets from TYPE<TAB>URL records piped by `javdb assets list`. " +
			"-d places auto-named files (image-001.jpg, video-001.mp4) into DIR; " +
			"-o writes a single asset to an exact path (.ts keeps the transport stream, " +
			".mp4 produces a fast-start MP4). Existing files are never overwritten. " +
			"Output is the final written path per line.",
		Example: "  javdb assets list SSIS-589 --type image 1-4 | javdb assets download -d ./media\n" +
			"  javdb assets list SSIS-589 --type video | javdb assets download -o preview.mp4",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return client.WithOptionalAuth(options, streams.Err, func(c *javdb.Client) error {
				return downloadAssetsStream(cmd.Context(), c, streams, dir, out)
			})
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Directory for auto-named output files")
	cmd.Flags().StringVarP(&out, "out", "o", "", "Exact output path for exactly one asset")
	return cmd
}

// downloadAssetsStream 流式处理 stdin 的 TYPE<TAB>URL 记录(计划 #8):
// 逐条 scan → parse → download → next,不缓存全部记录。
// -o 只需要读取第一条,再尝试读取第二条:存在第二条则报错,无需读完整个 stdin。
func downloadAssetsStream(ctx context.Context, c *javdb.Client, streams *invocation.Streams, dir, out string) error {
	dir = filepath.Clean(dir)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return fmt.Errorf("assets download: -d %q is not a directory", dir)
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
			// -o 模式只需要读取第一条,再尝试读取第二条(计划 #8):
			// 存在第二条则报错,无需把整个 stdin 读完。
			if firstRecord != nil {
				return fmt.Errorf("assets download: -o requires exactly one asset, got more")
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
		return fmt.Errorf("read stdin: %w", err)
	}
	if out != "" {
		if firstRecord == nil {
			return fmt.Errorf("assets download: no assets on stdin")
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
		return fmt.Errorf("assets download: no assets on stdin")
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
		return assetRecord{}, fmt.Errorf("invalid input at line %d: %q (want TYPE<TAB>URL)", lineNo, line)
	}
	assetType := strings.TrimSpace(parts[0])
	switch assetType {
	case "image", "video":
	default:
		return assetRecord{}, fmt.Errorf("invalid input at line %d: unsupported asset type %q", lineNo, assetType)
	}
	return assetRecord{Type: assetType, URL: strings.TrimSpace(parts[1])}, nil
}

// downloadAutoNamed 处理自动命名:图片下载后按 magic 定扩展名,视频默认 .mp4。
func downloadAutoNamed(ctx context.Context, c *javdb.Client, record assetRecord, dir string, pos int) (string, error) {
	switch record.Type {
	case "video":
		target := filepath.Join(dir, fmt.Sprintf("video-%03d.mp4", pos))
		if _, err := c.DownloadMovieAsset(ctx, javdb.MovieAsset{Type: record.Type, URL: record.URL}, target); err != nil {
			return "", err
		}
		return target, nil
	case "image":
		// 临时文件名唯一(计划 #27):os.MkdirTemp 生成一次性目录,
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
			return "", cleanupDir(fmt.Errorf("assets download: downloaded image is not a recognized format"))
		}
		target := filepath.Join(dir, fmt.Sprintf("image-%03d.%s", pos, format))
		if err := atomicfile.LinkNoReplace(tmp, target); err != nil {
			return "", cleanupDir(fmt.Errorf("assets download: publish %q: %w", target, err))
		}
		if err := os.Remove(tmp); err != nil {
			return "", cleanupDir(fmt.Errorf("assets download: remove image temp file after publish: %w", err))
		}
		if err := os.Remove(tmpDir); err != nil {
			return "", fmt.Errorf("assets download: remove image temp directory: %w", err)
		}
		return target, nil
	default:
		return "", fmt.Errorf("assets download: unsupported asset type %q", record.Type)
	}
}
