package media

import (
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/internal/common/atomicfile"
)

// FetchContext 是 client 提供给媒体解码器的原始资源读取回调。
// context 贯穿全部媒体请求，取消时立即停止网络与媒体处理。
type FetchContext func(ctx context.Context, url string) (io.ReadCloser, error)

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
	body, err := fetch(ctx, sourceURL)
	if err != nil {
		return 0, err
	}
	rawPath, _, err := copyMediaResponseToTemp(body, ".javdb-image-*")
	if err != nil {
		return 0, err
	}
	imagePath, cleanup, err := prepareImageFile(rawPath)
	if err != nil {
		return 0, errors.Join(err, removeMediaTemp(rawPath, "image response"))
	}
	// payload 在写入前已通过魔数校验,发布后无需再次验证。
	written, publishErr := publishMediaFile(target, func(w io.Writer) (int64, error) {
		return copyFileToWriter(imagePath, w)
	}, nil)
	removeErr := removeMediaTemp(rawPath, "image response")
	if cleanup {
		removeErr = errors.Join(removeErr, removeMediaTemp(imagePath, "decoded image"))
	}
	return written, errors.Join(publishErr, removeErr)
}

// DownloadHLS 下载一个已结束的 HLS 播放列表到 target,输出格式由后缀决定:
// .ts 保留 Transport Stream(解密+校验后的拼接流);.mp4 输出 Fast Start MP4;
// 其余后缀明确拒绝，不做转码。
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
// 损坏的 segment 重试同一 segment，至多 3 次。
const maxSegmentAttempts = 3

// downloadTS 产出保留 Transport Stream 的 .ts:每个 segment 通过 Layer A 与
// codec 检查后写入 .part,最终通过结构与 Layer B 媒体校验后原子发布。
func downloadTS(ctx context.Context, fetch FetchContext, playlistURL, target string) (int64, error) {
	playlist, err := fetchHLSMediaPlaylist(ctx, fetch, playlistURL)
	if err != nil {
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
			if err := validateSegmentCodecsFile(payload.path); err != nil {
				return total, errors.Join(err, removeMediaTemp(payload.path, "HLS segment"))
			}
			n, copyErr := copyFileToWriter(payload.path, w)
			total += n
			removeErr := removeMediaTemp(payload.path, "HLS segment")
			if copyErr != nil || removeErr != nil {
				return total, errors.Join(copyErr, removeErr)
			}
		}
		return total, nil
	}, validateTSOutput)
}

// downloadMP4 走 spool 管线产出 Fast Start MP4。
func downloadMP4(ctx context.Context, fetch FetchContext, playlistURL, target string) (int64, error) {
	playlist, err := fetchHLSMediaPlaylist(ctx, fetch, playlistURL)
	if err != nil {
		return 0, err
	}
	spool, err := newMP4Spooler(filepath.Dir(target))
	if err != nil {
		return 0, err
	}
	written, publishErr := publishMediaFile(target, func(w io.Writer) (int64, error) {
		keys := map[string][]byte{}
		for _, segment := range playlist.segments {
			payload, err := fetchValidatedSegment(ctx, fetch, segment, keys)
			if err != nil {
				return 0, err
			}
			addErr := spool.addSegmentFile(payload.path)
			removeErr := removeMediaTemp(payload.path, "HLS segment")
			if addErr != nil || removeErr != nil {
				return 0, errors.Join(addErr, removeErr)
			}
		}
		if err := spool.finalize(); err != nil {
			return 0, err
		}
		return writeMP4Body(spool, w)
	}, validateMP4File)
	closeErr := spool.file.Close()
	removeErr := os.Remove(spool.file.Name())
	var closeCleanupErr, removeCleanupErr error
	if closeErr != nil {
		closeCleanupErr = fmt.Errorf("close MP4 spool: %w", closeErr)
	}
	if removeErr != nil {
		removeCleanupErr = fmt.Errorf("remove MP4 spool: %w", removeErr)
	}
	if publishErr != nil {
		return written, errors.Join(publishErr, closeCleanupErr, removeCleanupErr)
	}
	if closeCleanupErr != nil || removeCleanupErr != nil {
		return written, errors.Join(closeCleanupErr, removeCleanupErr)
	}
	return written, nil
}

// fetchValidatedSegment 获取、解密并校验单个 segment(Layer A per-segment gate)。
// 校验失败或传输失败时重试同一 segment,至多 maxSegmentAttempts 次。
type fetchedSegment struct {
	path string
	size int64
}

func fetchValidatedSegment(ctx context.Context, fetch FetchContext, segment hlsSegment, keys map[string][]byte) (*fetchedSegment, error) {
	var lastErr error
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for attempt := 1; attempt <= maxSegmentAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rawPath, rawSize, err := fetchMediaToTemp(ctx, fetch, segment.uri, ".javdb-segment-*")
		if err != nil {
			lastErr = fmt.Errorf("download HLS segment: %w", err)
			continue
		}
		if rawSize == 0 {
			lastErr = errors.Join(fmt.Errorf("download HLS segment: empty response"), removeMediaTemp(rawPath, "HLS segment"))
			continue
		}
		payloadPath := rawPath
		if segment.key != nil {
			key, ok := keys[segment.key.uri]
			if !ok {
				key, err = fetchHLSKey(ctx, fetch, segment.key.uri)
				if err != nil {
					lastErr = errors.Join(fmt.Errorf("download HLS key: %w", err), removeMediaTemp(rawPath, "HLS segment"))
					continue
				}
				keys[segment.key.uri] = key
			}
			iv := segment.key.iv
			if len(iv) == 0 {
				iv = hlsSequenceIV(segment.sequence)
			}
			payloadPath, err = decryptHLSSegmentFile(payloadPath, key, iv)
			rawRemoveErr := removeMediaTemp(rawPath, "encrypted HLS segment")
			if err != nil {
				lastErr = errors.Join(err, rawRemoveErr)
				continue
			}
			if rawRemoveErr != nil {
				lastErr = errors.Join(rawRemoveErr, removeMediaTemp(payloadPath, "decrypted HLS segment"))
				continue
			}
		}
		if err := validateTSSegmentFile(payloadPath); err != nil {
			lastErr = errors.Join(err, removeMediaTemp(payloadPath, "HLS segment"))
			continue
		}
		info, err := os.Stat(payloadPath)
		if err != nil {
			lastErr = errors.Join(err, removeMediaTemp(payloadPath, "HLS segment"))
			continue
		}
		return &fetchedSegment{path: payloadPath, size: info.Size()}, nil
	}
	return nil, fmt.Errorf("segment %d remained invalid after %d attempts: %w", segment.sequence, maxSegmentAttempts, lastErr)
}

func fetchHLSMediaPlaylist(ctx context.Context, fetch FetchContext, playlistURL string) (hlsMediaPlaylist, error) {
	body, err := fetch(ctx, playlistURL)
	if err != nil {
		return hlsMediaPlaylist{}, fmt.Errorf("download HLS playlist: %w", err)
	}
	if body == nil {
		return hlsMediaPlaylist{}, fmt.Errorf("download HLS playlist: media response body is nil")
	}
	playlist, parseErr := parseHLSMediaPlaylistReader(playlistURL, body)
	closeErr := body.Close()
	if parseErr != nil || closeErr != nil {
		return hlsMediaPlaylist{}, errors.Join(parseErr, closeErr)
	}
	return playlist, nil
}

// mediaCopyBufferSize 只是网络/文件搬运的固定工作缓冲，不限制任何媒体资源的总大小。
const mediaCopyBufferSize = 32 * 1024

func removeMediaTemp(path, kind string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove %s temp file: %w", kind, err)
	}
	return nil
}

func copyMediaResponseToTemp(body io.ReadCloser, pattern string) (path string, size int64, err error) {
	if body == nil {
		return "", 0, fmt.Errorf("media response body is nil")
	}
	tmp, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", 0, errors.Join(fmt.Errorf("create media response temp file: %w", err), body.Close())
	}
	path = tmp.Name()
	_, copyErr := io.CopyBuffer(tmp, body, make([]byte, mediaCopyBufferSize))
	bodyCloseErr := body.Close()
	tmpCloseErr := tmp.Close()
	if copyErr != nil || bodyCloseErr != nil || tmpCloseErr != nil {
		return "", 0, errors.Join(copyErr, bodyCloseErr, tmpCloseErr, removeMediaTemp(path, "media response"))
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", 0, errors.Join(err, removeMediaTemp(path, "media response"))
	}
	return path, info.Size(), nil
}

func fetchMediaToTemp(ctx context.Context, fetch FetchContext, uri, pattern string) (string, int64, error) {
	body, err := fetch(ctx, uri)
	if err != nil {
		return "", 0, err
	}
	return copyMediaResponseToTemp(body, pattern)
}

func copyFileToWriter(path string, dst io.Writer) (int64, error) {
	src, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	n, copyErr := io.CopyBuffer(dst, src, make([]byte, mediaCopyBufferSize))
	closeErr := src.Close()
	if copyErr != nil || closeErr != nil {
		return n, errors.Join(copyErr, closeErr)
	}
	return n, nil
}

func fetchHLSKey(ctx context.Context, fetch FetchContext, uri string) ([]byte, error) {
	body, err := fetch(ctx, uri)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, fmt.Errorf("HLS AES-128 key response body is nil")
	}
	key, readErr := readMediaBodyBounded(body, aes.BlockSize)
	closeErr := body.Close()
	if readErr != nil || closeErr != nil {
		return nil, errors.Join(readErr, closeErr)
	}
	if len(key) != aes.BlockSize {
		return nil, fmt.Errorf("HLS AES-128 key has %d bytes, want %d", len(key), aes.BlockSize)
	}
	return key, nil
}

// readMediaBodyBounded 只用于协议上明确固定大小的响应,目前是 HLS AES-128 key(16 字节)。
// LimitReader 读取 maxBytes+1 以区分“刚好完整”与“响应过长”,不会为普通媒体响应设置总量上限。
func readMediaBodyBounded(body io.Reader, maxBytes int64) ([]byte, error) {
	if body == nil {
		return nil, fmt.Errorf("media response body is nil")
	}
	if maxBytes < 0 {
		return nil, fmt.Errorf("media body bound must be non-negative")
	}
	probe := maxBytes
	if maxBytes < int64(^uint64(0)>>1) {
		probe++
	}
	data, err := io.ReadAll(io.LimitReader(body, probe))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("media response exceeds fixed bound of %d bytes", maxBytes)
	}
	return data, nil
}

func prepareImageFile(rawPath string) (path string, cleanup bool, err error) {
	_, ok, err := imageFileFormat(rawPath)
	if err != nil {
		return "", false, err
	}
	if ok {
		return rawPath, false, nil
	}
	raw, err := os.Open(rawPath)
	if err != nil {
		return "", false, err
	}
	var key [1]byte
	if _, err := io.ReadFull(raw, key[:]); err != nil {
		return "", false, errors.Join(fmt.Errorf("read image XOR key: %w", err), raw.Close())
	}
	decoded, err := os.CreateTemp("", ".javdb-image-decoded-*")
	if err != nil {
		return "", false, errors.Join(err, raw.Close())
	}
	decodedPath := decoded.Name()
	cleanupDecoded := func(primary error) error {
		return errors.Join(primary, raw.Close(), decoded.Close(), removeMediaTemp(decodedPath, "decoded image"))
	}
	buf := make([]byte, mediaCopyBufferSize)
	for {
		n, readErr := raw.Read(buf)
		for i := 0; i < n; i++ {
			buf[i] ^= key[0]
		}
		if n > 0 {
			written, writeErr := decoded.Write(buf[:n])
			if writeErr == nil && written != n {
				writeErr = io.ErrShortWrite
			}
			if writeErr != nil {
				return "", false, cleanupDecoded(writeErr)
			}
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				return "", false, cleanupDecoded(readErr)
			}
			break
		}
	}
	rawCloseErr := raw.Close()
	decodedCloseErr := decoded.Close()
	if rawCloseErr != nil || decodedCloseErr != nil {
		return "", false, errors.Join(rawCloseErr, decodedCloseErr, removeMediaTemp(decodedPath, "decoded image"))
	}
	if _, ok, err := imageFileFormat(decodedPath); err != nil {
		return "", false, errors.Join(err, removeMediaTemp(decodedPath, "decoded image"))
	} else if !ok {
		return "", false, errors.Join(fmt.Errorf("media response is not a recognized image"), removeMediaTemp(decodedPath, "decoded image"))
	}
	return decodedPath, true, nil
}

func imageFileFormat(path string) (string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	var prefix [12]byte
	n, readErr := file.Read(prefix[:])
	closeErr := file.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return "", false, errors.Join(readErr, closeErr)
	}
	if closeErr != nil {
		return "", false, closeErr
	}
	format, ok := imagePayloadFormat(prefix[:n])
	return format, ok, nil
}

func parseHLSMediaPlaylist(playlistURL string, raw []byte) (hlsMediaPlaylist, error) {
	return parseHLSMediaPlaylistReader(playlistURL, bytes.NewReader(raw))
}

func parseHLSMediaPlaylistReader(playlistURL string, raw io.Reader) (hlsMediaPlaylist, error) {
	reader := bufio.NewReader(raw)
	firstLine, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return hlsMediaPlaylist{}, err
	}
	if strings.TrimSpace(firstLine) != "#EXTM3U" {
		return hlsMediaPlaylist{}, fmt.Errorf("media response is not an HLS playlist")
	}

	var (
		playlist hlsMediaPlaylist
		key      *hlsKey
		sequence uint64
		endList  bool
	)
	processLine := func(rawLine string) error {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			return nil
		}
		switch {
		case line == "#EXT-X-ENDLIST":
			endList = true
		case line == "#EXT-X-DISCONTINUITY":
			return fmt.Errorf("HLS discontinuity is not supported")
		case strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:"):
			parsed, err := strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(line, "#EXT-X-MEDIA-SEQUENCE:")), 10, 64)
			if err != nil {
				return fmt.Errorf("invalid HLS media sequence: %w", err)
			}
			sequence = parsed
		case strings.HasPrefix(line, "#EXT-X-KEY:"):
			parsed, err := parseHLSKey(playlistURL, strings.TrimPrefix(line, "#EXT-X-KEY:"))
			if err != nil {
				return err
			}
			key = parsed
		case strings.HasPrefix(line, "#EXT-X-BYTERANGE:"):
			return fmt.Errorf("HLS byte-range segments are not supported")
		case strings.HasPrefix(line, "#EXT-X-MAP:"):
			return fmt.Errorf("fragmented MP4 HLS playlists are not supported")
		case strings.HasPrefix(line, "#EXT-X-STREAM-INF:") || strings.HasPrefix(line, "#EXT-X-I-FRAME-STREAM-INF:"):
			return fmt.Errorf("HLS master playlists are not supported")
		case strings.HasPrefix(line, "#"):
			return nil
		default:
			uri, err := resolveHLSURI(playlistURL, line)
			if err != nil {
				return err
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
		return nil
	}
	for {
		rawLine, readErr := reader.ReadString('\n')
		if len(rawLine) > 0 {
			if err := processLine(rawLine); err != nil {
				return hlsMediaPlaylist{}, err
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return hlsMediaPlaylist{}, readErr
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

// decryptHLSSegmentFile 对加密 segment 做 CBC 流式解密。只保留一个待定的明文 block,
// 以便在 EOF 时验证并去除 PKCS#7 padding;完整 segment 保留在临时文件而非内存。
func decryptHLSSegmentFile(inputPath string, key, iv []byte) (string, error) {
	if len(key) != aes.BlockSize {
		return "", fmt.Errorf("HLS AES-128 key has %d bytes, want %d", len(key), aes.BlockSize)
	}
	if len(iv) != aes.BlockSize {
		return "", fmt.Errorf("HLS AES-128 IV has %d bytes, want %d", len(iv), aes.BlockSize)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	in, err := os.Open(inputPath)
	if err != nil {
		return "", err
	}
	out, err := os.CreateTemp(filepath.Dir(inputPath), ".javdb-segment-decrypted-*")
	if err != nil {
		return "", errors.Join(err, in.Close())
	}
	outPath := out.Name()
	cleanup := func(primary error) error {
		return errors.Join(primary, in.Close(), out.Close(), removeMediaTemp(outPath, "decrypted segment"))
	}

	decrypter := cipher.NewCBCDecrypter(block, iv)
	var encrypted [aes.BlockSize]byte
	var plain [aes.BlockSize]byte
	pending := make([]byte, 0, aes.BlockSize)
	blocks := 0
	for {
		n, readErr := io.ReadFull(in, encrypted[:])
		if errors.Is(readErr, io.EOF) && n == 0 {
			break
		}
		if readErr != nil {
			return "", cleanup(fmt.Errorf("encrypted HLS segment has invalid length: %w", readErr))
		}
		decrypter.CryptBlocks(plain[:], encrypted[:])
		if len(pending) > 0 {
			if _, err := out.Write(pending); err != nil {
				return "", cleanup(err)
			}
		}
		pending = append(pending[:0], plain[:]...)
		blocks++
	}
	if blocks == 0 {
		return "", cleanup(fmt.Errorf("encrypted HLS segment has invalid length"))
	}
	unpadded, err := removePKCS7Padding(pending)
	if err != nil {
		return "", cleanup(err)
	}
	if _, err := out.Write(unpadded); err != nil {
		return "", cleanup(err)
	}
	inCloseErr := in.Close()
	outCloseErr := out.Close()
	if inCloseErr != nil || outCloseErr != nil {
		return "", errors.Join(inCloseErr, outCloseErr, removeMediaTemp(outPath, "decrypted segment"))
	}
	return outPath, nil
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

// publishMediaFile 把媒体原子发布到 path:先写入同目录唯一临时文件,
// 全部写入并通过 validate 后以 no-replace 硬链接发布。
// 最终路径已存在时绝不覆盖；失败时清理临时文件，不留半成品。
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
	// 临时文件用 os.CreateTemp 保证唯一：并发或上次异常残留时
	// 不会与固定 target.part 名冲突。CreateTemp 位于同一目录,保证同一文件
	// 系统,支持原子发布。
	tmpFile, err := os.CreateTemp(dir, ".mediadl-*.part")
	if err != nil {
		return 0, fmt.Errorf("create media temp file: %w", err)
	}
	tmp := tmpFile.Name()
	closed := false
	closeTemp := func() error {
		if closed {
			return nil
		}
		closed = true
		return tmpFile.Close()
	}
	removeTemp := func() error { return os.Remove(tmp) }
	cleanupTemp := func() error {
		return errors.Join(closeTemp(), removeTemp())
	}

	written, err = write(tmpFile)
	if err != nil {
		return written, errors.Join(err, cleanupTemp())
	}
	if closeErr := closeTemp(); closeErr != nil {
		return written, errors.Join(fmt.Errorf("close media temp file: %w", closeErr), removeTemp())
	}
	if validate != nil {
		if err = validate(tmp); err != nil {
			return written, errors.Join(err, removeTemp())
		}
	}
	// 真 no-replace 发布：LinkNoReplace 在目标已存在时由操作系统
	// 原子返回 ErrExist，不依赖“先检查再 rename”(TOCTOU)。
	if err = atomicfile.LinkNoReplace(tmp, path); err != nil {
		return written, errors.Join(fmt.Errorf("publish media file: %w", err), removeTemp())
	}
	// 链接成功后删除临时文件；删除失败不能静默吞掉。
	if err = removeTemp(); err != nil {
		return 0, fmt.Errorf("remove media temp file after publish: %w", err)
	}
	return written, nil
}

// validateTSOutput 在发布前复核结构和媒体模型，验证器都按 reader 读取文件。
func validateTSOutput(path string) error {
	if err := validateTSFileStream(path); err != nil {
		return err
	}
	return validateMediaFile(path)
}

// validateMediaFile 对已落盘的 TS 做最终媒体校验:
// 结构层(Layer A)之上再做媒体模型校验(Layer B:codec/轨道/时间戳)。
func validateMediaFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	validateErr := validateMediaStreamReader(file)
	closeErr := file.Close()
	return errors.Join(validateErr, closeErr)
}
