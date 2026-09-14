package media

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// FetchContext 是 client 提供给媒体解码器的原始资源读取回调。
// context 贯穿全部媒体请求(计划 #44):取消时立即停止网络与工作。
type FetchContext func(ctx context.Context, url string) ([]byte, error)

// MediaEndpoint 提供图片与 HLS 预览媒体的下载 capability。
type MediaEndpoint struct {
	fetch FetchContext
}

// NewMedia 用 transport 提供的原始资源读取回调构造 media capability。
func NewMedia(fetch FetchContext) *MediaEndpoint {
	return &MediaEndpoint{fetch: fetch}
}

// DownloadImage 下载并还原图片媒体,委托包级纯实现。
func (e *MediaEndpoint) DownloadImage(ctx context.Context, sourceURL, target string) (int64, error) {
	return DownloadImage(ctx, e.fetch, sourceURL, target)
}

// DownloadHLS 下载已结束的 HLS 预览媒体,委托包级纯实现;输出格式由
// target 后缀决定(.ts 保留 Transport Stream,.mp4 输出 Fast Start MP4)。
func (e *MediaEndpoint) DownloadHLS(ctx context.Context, playlistURL, target string) (int64, error) {
	return DownloadHLS(ctx, e.fetch, playlistURL, target)
}

// DownloadImage 下载并还原图片 CDN 返回的图片数据,再原子发布到 target。
func DownloadImage(ctx context.Context, fetch FetchContext, sourceURL, target string) (int64, error) {
	raw, err := fetchBounded(ctx, fetch, sourceURL, maxImageBytes, "image")
	if err != nil {
		return 0, err
	}
	imageData, err := decodeImagePayload(raw)
	if err != nil {
		return 0, err
	}
	// payload 在写入前已通过魔数校验,发布后无需再次验证。
	return publishMediaFile(target, func(w io.Writer) (int64, error) {
		n, err := w.Write(imageData)
		if err == nil && n != len(imageData) {
			err = io.ErrShortWrite
		}
		return int64(n), err
	}, nil)
}

// DownloadHLS 下载一个已结束的 HLS 播放列表到 target,输出格式由后缀决定:
// .ts 保留 Transport Stream(解密+校验后的拼接流);.mp4 输出 Fast Start MP4;
// 其余后缀明确拒绝,不做转码(计划 #17/#18/#25)。
func DownloadHLS(ctx context.Context, fetch FetchContext, playlistURL, target string) (int64, error) {
	switch strings.ToLower(filepath.Ext(target)) {
	case ".ts":
		return downloadTS(ctx, fetch, playlistURL, target)
	case ".mp4":
		return downloadMP4(ctx, fetch, playlistURL, target)
	default:
		return 0, fmt.Errorf("unsupported video output format %q", filepath.Ext(target))
	}
}

func validateMediaURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid media URL")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("unsupported media URL scheme %q", u.Scheme)
	}
	return nil
}

// decodeImagePayload 只接受已知图片格式,避免把 CDN 的包装字节误写成“下载成功”的图片。
func decodeImagePayload(raw []byte) ([]byte, error) {
	if _, ok := imagePayloadFormat(raw); ok {
		return raw, nil
	}
	if len(raw) < 2 {
		return nil, fmt.Errorf("media response is not a recognized image")
	}

	// 图片 CDN 的实际响应以首字节为 XOR key;去掉该字节并异或后才是原始图片。
	key := raw[0]
	decoded := make([]byte, len(raw)-1)
	for i := range decoded {
		decoded[i] = raw[i+1] ^ key
	}
	if _, ok := imagePayloadFormat(decoded); !ok {
		return nil, fmt.Errorf("media response is not a recognized image")
	}
	return decoded, nil
}

// ImagePayloadFormat 通过魔数识别图片 payload,返回稳定格式名
// (jpg/png/gif/webp/avif/heic),供下载层确定落盘扩展名。
func ImagePayloadFormat(data []byte) (string, bool) {
	return imagePayloadFormat(data)
}

func imagePayloadFormat(data []byte) (string, bool) {
	switch {
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		return "jpg", true // JPEG
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		return "png", true
	case len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a"):
		return "gif", true
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "webp", true
	case len(data) >= 12 && string(data[4:8]) == "ftyp":
		switch string(data[8:12]) {
		case "avif", "avis":
			return "avif", true
		case "heic", "heix", "mif1":
			return "heic", true
		}
	}
	return "", false
}

// Fetch 是 client 提供给媒体解码器的原始资源读取回调。
type Fetch func(string) ([]byte, error)

type hlsKey struct {
	uri string
	iv  []byte
}

type hlsSegment struct {
	uri      string
	sequence uint64
	key      *hlsKey
}

type hlsMediaPlaylist struct {
	segments []hlsSegment
}

// maxSegmentAttempts 是单个 segment 的最大尝试次数。
// 依据 input.md 计划 #32 的明文要求(损坏 segment 重试同一 segment,至多 3 次)。
const maxSegmentAttempts = 3

// 下载路径的内部安全上限(计划 #11)。这些是 downloader 的内部边界,
// 不扩展成 CLI tuning flags。读取模式:Content-Length 预检查 +
// io.LimitReader(limit+1) + 超过 limit 明确报错。
const (
	maxPlaylistBytes   = 2 << 20  // HLS playlist 2 MiB
	maxSegmentBytes    = 32 << 20 // TS segment 32 MiB
	maxImageBytes      = 64 << 20 // image 64 MiB
	maxKeyBytes        = 17       // AES-128 key exactly 16 B,最多读 17 B
	maxTotalVideoBytes = 512 << 20
	maxSegmentCount    = 4096
	maxSamplesPerTrack = 1_000_000
)

// downloadTS 产出保留 Transport Stream 的 .ts:每个 segment 通过 Layer A 与
// codec 检查后写入 .part,最终做分块结构校验后原子发布(计划 #33)。
func downloadTS(ctx context.Context, fetch FetchContext, playlistURL, target string) (int64, error) {
	playlistBody, err := fetchBounded(ctx, fetch, playlistURL, maxPlaylistBytes, "HLS playlist")
	if err != nil {
		return 0, fmt.Errorf("download HLS playlist: %w", err)
	}
	playlist, err := parseHLSMediaPlaylist(playlistURL, playlistBody)
	if err != nil {
		return 0, err
	}
	if err := checkSegmentCount(len(playlist.segments)); err != nil {
		return 0, err
	}

	return publishMediaFile(target, func(w io.Writer) (int64, error) {
		var total int64
		keys := map[string][]byte{}
		for _, segment := range playlist.segments {
			payload, err := fetchValidatedSegment(ctx, fetch, segment, keys)
			if err != nil {
				return total, err
			}
			if err := validateSegmentCodecs(payload); err != nil {
				return total, err
			}
			n, err := w.Write(payload)
			if err != nil {
				return total, err
			}
			if n != len(payload) {
				return total, io.ErrShortWrite
			}
			total += int64(n)
		}
		return total, nil
	}, validateTSFileStream)
}

// downloadMP4 走 spool 管线产出 Fast Start MP4(计划 #35/#36)。
func downloadMP4(ctx context.Context, fetch FetchContext, playlistURL, target string) (int64, error) {
	playlistBody, err := fetchBounded(ctx, fetch, playlistURL, maxPlaylistBytes, "HLS playlist")
	if err != nil {
		return 0, fmt.Errorf("download HLS playlist: %w", err)
	}
	playlist, err := parseHLSMediaPlaylist(playlistURL, playlistBody)
	if err != nil {
		return 0, err
	}
	if err := checkSegmentCount(len(playlist.segments)); err != nil {
		return 0, err
	}

	spoolPath := target + ".spool"
	spool, err := newMP4Spooler(spoolPath)
	if err != nil {
		return 0, err
	}
	defer os.Remove(spoolPath)
	defer spool.file.Close()

	return publishMediaFile(target, func(w io.Writer) (int64, error) {
		keys := map[string][]byte{}
		for _, segment := range playlist.segments {
			payload, err := fetchValidatedSegment(ctx, fetch, segment, keys)
			if err != nil {
				return 0, err
			}
			if err := spool.addSegment(payload); err != nil {
				return 0, err
			}
		}
		spool.finalize()
		return writeMP4Body(spool, w)
	}, validateMP4File)
}

// fetchValidatedSegment 获取、解密并校验单个 segment(Layer A per-segment gate)。
// 校验失败或传输失败时重试同一 segment,至多 maxSegmentAttempts 次。
func fetchValidatedSegment(ctx context.Context, fetch FetchContext, segment hlsSegment, keys map[string][]byte) ([]byte, error) {
	var lastErr error
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for attempt := 1; attempt <= maxSegmentAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		raw, err := fetchBounded(ctx, fetch, segment.uri, maxSegmentBytes, "HLS segment")
		if err != nil {
			lastErr = fmt.Errorf("download HLS segment: %w", err)
			continue
		}
		if len(raw) == 0 {
			lastErr = fmt.Errorf("download HLS segment: empty response")
			continue
		}
		payload := raw
		if segment.key != nil {
			key, ok := keys[segment.key.uri]
			if !ok {
				key, err = fetchBounded(ctx, fetch, segment.key.uri, maxKeyBytes, "HLS key")
				if err != nil {
					lastErr = fmt.Errorf("download HLS key: %w", err)
					continue
				}
				if len(key) != aes.BlockSize {
					lastErr = fmt.Errorf("HLS AES-128 key has %d bytes, want %d", len(key), aes.BlockSize)
					continue
				}
				keys[segment.key.uri] = key
			}
			iv := segment.key.iv
			if len(iv) == 0 {
				iv = hlsSequenceIV(segment.sequence)
			}
			payload, err = decryptHLSSegment(payload, key, iv)
			if err != nil {
				lastErr = err
				continue
			}
		}
		if err := validateTSSegment(payload); err != nil {
			lastErr = err
			continue
		}
		return payload, nil
	}
	return nil, fmt.Errorf("segment %d remained invalid after %d attempts: %w", segment.sequence, maxSegmentAttempts, lastErr)
}

// fetchBounded 用有界读取获取媒体资源(计划 #11):
// Content-Length 预检查(transport 层 LimitReader(limit+1))+ size 检查,
// 超过 limit 明确报错,不无界进内存。
func fetchBounded(ctx context.Context, fetch FetchContext, url string, limit int64, what string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	raw, err := fetch(ctx, url)
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, fmt.Errorf("%s size %d exceeds limit %d", what, len(raw), limit)
	}
	return raw, nil
}

// boundedFetchContext 是 client 提供给媒体解码器的有界读取回调;
// limit 透传给 transport 的 LimitReader(计划 #11)。
func boundedFetchContext(c mediaBoundedClient, limit int64) FetchContext {
	return func(ctx context.Context, url string) ([]byte, error) {
		return c.FetchMediaBounded(ctx, url, limit)
	}
}

// mediaBoundedClient 是 FetchMediaBounded 所需的最小接口。
type mediaBoundedClient interface {
	FetchMediaBounded(ctx context.Context, rawURL string, limit int64) ([]byte, error)
}

// checkSegmentCount 校验 segment 数不超过内部安全上限(计划 #11)。
func checkSegmentCount(count int) error {
	if count > maxSegmentCount {
		return fmt.Errorf("HLS playlist has %d segments, exceeds limit %d", count, maxSegmentCount)
	}
	return nil
}

func parseHLSMediaPlaylist(playlistURL string, raw []byte) (hlsMediaPlaylist, error) {
	lines := strings.Split(string(raw), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "#EXTM3U" {
		return hlsMediaPlaylist{}, fmt.Errorf("media response is not an HLS playlist")
	}

	var (
		playlist hlsMediaPlaylist
		key      *hlsKey
		sequence uint64
		endList  bool
	)
	for _, rawLine := range lines[1:] {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		switch {
		case line == "#EXT-X-ENDLIST":
			endList = true
		case strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:"):
			parsed, err := strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(line, "#EXT-X-MEDIA-SEQUENCE:")), 10, 64)
			if err != nil {
				return hlsMediaPlaylist{}, fmt.Errorf("invalid HLS media sequence: %w", err)
			}
			sequence = parsed
		case strings.HasPrefix(line, "#EXT-X-KEY:"):
			parsed, err := parseHLSKey(playlistURL, strings.TrimPrefix(line, "#EXT-X-KEY:"))
			if err != nil {
				return hlsMediaPlaylist{}, err
			}
			key = parsed
		case strings.HasPrefix(line, "#EXT-X-BYTERANGE:"):
			return hlsMediaPlaylist{}, fmt.Errorf("HLS byte-range segments are not supported")
		case strings.HasPrefix(line, "#EXT-X-MAP:"):
			return hlsMediaPlaylist{}, fmt.Errorf("fragmented MP4 HLS playlists are not supported")
		case strings.HasPrefix(line, "#EXT-X-STREAM-INF:") || strings.HasPrefix(line, "#EXT-X-I-FRAME-STREAM-INF:"):
			return hlsMediaPlaylist{}, fmt.Errorf("HLS master playlists are not supported")
		case strings.HasPrefix(line, "#"):
			continue
		default:
			uri, err := resolveHLSURI(playlistURL, line)
			if err != nil {
				return hlsMediaPlaylist{}, err
			}
			segment := hlsSegment{uri: uri, sequence: sequence}
			if key != nil {
				copied := *key
				copied.iv = append([]byte(nil), key.iv...)
				segment.key = &copied
			}
			playlist.segments = append(playlist.segments, segment)
			sequence++
		}
	}
	if !endList {
		// 直播列表会持续增长；没有结束标记时无法声称已完整下载预览视频。
		return hlsMediaPlaylist{}, fmt.Errorf("HLS playlist has no end marker")
	}
	if len(playlist.segments) == 0 {
		return hlsMediaPlaylist{}, fmt.Errorf("HLS playlist has no media segments")
	}
	return playlist, nil
}

func parseHLSKey(playlistURL, value string) (*hlsKey, error) {
	attrs, err := parseHLSAttributes(value)
	if err != nil {
		return nil, fmt.Errorf("invalid HLS key attributes: %w", err)
	}
	switch attrs["METHOD"] {
	case "NONE":
		return nil, nil
	case "AES-128":
		uriValue := attrs["URI"]
		if uriValue == "" {
			return nil, fmt.Errorf("HLS AES-128 key has no URI")
		}
		uri, err := resolveHLSURI(playlistURL, uriValue)
		if err != nil {
			return nil, err
		}
		var iv []byte
		if rawIV := attrs["IV"]; rawIV != "" {
			iv, err = parseHLSIV(rawIV)
			if err != nil {
				return nil, err
			}
		}
		return &hlsKey{uri: uri, iv: iv}, nil
	default:
		return nil, fmt.Errorf("unsupported HLS encryption method %q", attrs["METHOD"])
	}
}

func parseHLSAttributes(raw string) (map[string]string, error) {
	attrs := map[string]string{}
	for pos := 0; pos < len(raw); {
		for pos < len(raw) && (raw[pos] == ',' || raw[pos] == ' ' || raw[pos] == '\t') {
			pos++
		}
		if pos == len(raw) {
			break
		}
		keyStart := pos
		for pos < len(raw) && raw[pos] != '=' && raw[pos] != ',' {
			pos++
		}
		if pos == len(raw) || raw[pos] != '=' {
			return nil, fmt.Errorf("missing '='")
		}
		key := strings.ToUpper(strings.TrimSpace(raw[keyStart:pos]))
		if key == "" {
			return nil, fmt.Errorf("empty attribute name")
		}
		pos++
		var value string
		if pos < len(raw) && raw[pos] == '"' {
			pos++
			valueStart := pos
			for pos < len(raw) && raw[pos] != '"' {
				pos++
			}
			if pos == len(raw) {
				return nil, fmt.Errorf("unterminated quoted value")
			}
			value = raw[valueStart:pos]
			pos++
		} else {
			valueStart := pos
			for pos < len(raw) && raw[pos] != ',' {
				pos++
			}
			value = strings.TrimSpace(raw[valueStart:pos])
		}
		attrs[key] = value
		for pos < len(raw) && (raw[pos] == ' ' || raw[pos] == '\t') {
			pos++
		}
		if pos < len(raw) && raw[pos] != ',' {
			return nil, fmt.Errorf("expected ','")
		}
	}
	return attrs, nil
}

func parseHLSIV(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(strings.TrimPrefix(raw, "0x"), "0X")
	if raw == "" || len(raw) > aes.BlockSize*2 {
		return nil, fmt.Errorf("invalid HLS IV")
	}
	if len(raw)%2 != 0 {
		raw = "0" + raw
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid HLS IV: %w", err)
	}
	iv := make([]byte, aes.BlockSize)
	copy(iv[aes.BlockSize-len(decoded):], decoded)
	return iv, nil
}

func resolveHLSURI(playlistURL, reference string) (string, error) {
	base, err := url.Parse(playlistURL)
	if err != nil {
		return "", fmt.Errorf("invalid HLS playlist URL")
	}
	ref, err := url.Parse(reference)
	if err != nil {
		return "", fmt.Errorf("invalid HLS resource URL")
	}
	resolved := base.ResolveReference(ref).String()
	if err := validateMediaURL(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

func hlsSequenceIV(sequence uint64) []byte {
	iv := make([]byte, aes.BlockSize)
	binary.BigEndian.PutUint64(iv[aes.BlockSize-8:], sequence)
	return iv
}

func decryptHLSSegment(payload, key, iv []byte) ([]byte, error) {
	if len(key) != aes.BlockSize {
		return nil, fmt.Errorf("HLS AES-128 key has %d bytes, want %d", len(key), aes.BlockSize)
	}
	if len(iv) != aes.BlockSize {
		return nil, fmt.Errorf("HLS AES-128 IV has %d bytes, want %d", len(iv), aes.BlockSize)
	}
	if len(payload) == 0 || len(payload)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("encrypted HLS segment has invalid length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	plain := append([]byte(nil), payload...)
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, plain)
	return removePKCS7Padding(plain)
}

func removePKCS7Padding(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("invalid PKCS#7 padding")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > aes.BlockSize || padding > len(data) {
		return nil, fmt.Errorf("invalid PKCS#7 padding")
	}
	for _, b := range data[len(data)-padding:] {
		if int(b) != padding {
			return nil, fmt.Errorf("invalid PKCS#7 padding")
		}
	}
	return data[:len(data)-padding], nil
}

// publishMediaFile 把媒体原子发布到 path:先写入 path+".part" 临时文件,
// 全部写入并通过 validate 后 rename 到最终路径。
// 最终路径已存在时绝不覆盖(计划 #41/#42);失败时清理临时文件,不留半成品。
func publishMediaFile(path string, write func(io.Writer) (int64, error), validate func(path string) error) (written int64, err error) {
	if strings.TrimSpace(path) == "" {
		return 0, fmt.Errorf("output path is required")
	}
	dir := filepath.Dir(path)
	info, statErr := os.Stat(dir)
	if statErr != nil {
		return 0, fmt.Errorf("output directory: %w", statErr)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("output directory is not a directory")
	}
	if _, err := os.Lstat(path); err == nil {
		return 0, fmt.Errorf("output file already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return 0, fmt.Errorf("check output file %q: %w", path, err)
	}

	// 临时文件用 os.CreateTemp 保证唯一(计划 #27):并发或上次异常残留时
	// 不会与固定 target.part 名冲突。CreateTemp 位于同一目录,保证同一文件
	// 系统,支持原子发布。
	tmpFile, err := os.CreateTemp(dir, ".mediadl-*.part")
	if err != nil {
		return 0, fmt.Errorf("create media temp file: %w", err)
	}
	tmp := tmpFile.Name()
	closed := false
	completed := false
	defer func() {
		if !closed {
			_ = tmpFile.Close()
		}
		if !completed {
			_ = os.Remove(tmp)
		}
	}()

	written, err = write(tmpFile)
	if err != nil {
		return 0, err
	}
	if err = tmpFile.Close(); err != nil {
		return 0, fmt.Errorf("close media temp file: %w", err)
	}
	closed = true
	if validate != nil {
		if err = validate(tmp); err != nil {
			return 0, err
		}
	}
	// 真 no-replace 发布(计划 #28):os.Link 在目标已存在时返回 EEXIST,
	// 两个进程同时写相同 target 时一方成功另一方 ErrExist,
	// 不依赖"先检查再 rename"(TOCTOU)。Windows 由 replace_windows.go 处理。
	if err = linkNoReplace(tmp, path); err != nil {
		return 0, fmt.Errorf("publish media file: %w", err)
	}
	completed = true
	// 链接成功后删除临时文件;删除失败不能静默吞掉(计划 #29)。
	if err = os.Remove(tmp); err != nil {
		return 0, fmt.Errorf("remove media temp file after publish: %w", err)
	}
	return written, nil
}

// validateMediaFile 对已落盘的 TS 做最终媒体校验:
// 结构层(Layer A)之上再做媒体模型校验(Layer B:codec/轨道/时间戳)。
func validateMediaFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return validateMediaStream(data)
}
