package appapi

import (
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

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/protocol/httpx"
)

// DownloadImage 下载并还原图片 CDN 返回的图片数据，再以新文件方式写入 target。
func (c *Client) DownloadImage(sourceURL, target string) (int64, error) {
	raw, err := c.fetchMedia(sourceURL)
	if err != nil {
		return 0, err
	}
	imageData, err := decodeImagePayload(raw)
	if err != nil {
		return 0, err
	}
	return writeNewMediaFile(target, func(w io.Writer) (int64, error) {
		n, err := w.Write(imageData)
		if err == nil && n != len(imageData) {
			err = io.ErrShortWrite
		}
		return int64(n), err
	})
}

// DownloadHLS 下载一个已结束的 HLS 媒体播放列表，并将解密后的媒体分片串接到 target。
func (c *Client) DownloadHLS(playlistURL, target string) (int64, error) {
	return downloadHLS(c.fetchMedia, playlistURL, target)
}

func (c *Client) fetchMedia(rawURL string) ([]byte, error) {
	if err := validateMediaURL(rawURL); err != nil {
		return nil, err
	}
	resp, err := c.http.Get(rawURL, map[string]string{"user-agent": UserAgent})
	if err != nil {
		return nil, fmt.Errorf("request media: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if closeErr := resp.Body.Close(); closeErr != nil {
			return nil, fmt.Errorf("close media response after HTTP %d: %w", resp.StatusCode, closeErr)
		}
		return nil, fmt.Errorf("media request returned HTTP %d", resp.StatusCode)
	}
	body, err := httpx.ReadAll(resp)
	if err != nil {
		return nil, fmt.Errorf("read media: %w", err)
	}
	return body, nil
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

// decodeImagePayload 只接受已知图片格式，避免把 CDN 的包装字节误写成“下载成功”的图片。
func decodeImagePayload(raw []byte) ([]byte, error) {
	if knownImagePayload(raw) {
		return raw, nil
	}
	if len(raw) < 2 {
		return nil, fmt.Errorf("media response is not a recognized image")
	}

	// 图片 CDN 的实际响应以首字节为 XOR key；去掉该字节并异或后才是原始图片。
	key := raw[0]
	decoded := make([]byte, len(raw)-1)
	for i := range decoded {
		decoded[i] = raw[i+1] ^ key
	}
	if !knownImagePayload(decoded) {
		return nil, fmt.Errorf("media response is not a recognized image")
	}
	return decoded, nil
}

func knownImagePayload(data []byte) bool {
	switch {
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		return true // JPEG
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		return true // PNG
	case len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a"):
		return true
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return true
	case len(data) >= 12 && string(data[4:8]) == "ftyp":
		switch string(data[8:12]) {
		case "avif", "avis", "heic", "heix", "mif1":
			return true
		}
	}
	return false
}

type mediaFetch func(string) ([]byte, error)

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

func downloadHLS(fetch mediaFetch, playlistURL, target string) (int64, error) {
	playlistBody, err := fetch(playlistURL)
	if err != nil {
		return 0, fmt.Errorf("download HLS playlist: %w", err)
	}
	playlist, err := parseHLSMediaPlaylist(playlistURL, playlistBody)
	if err != nil {
		return 0, err
	}

	return writeNewMediaFile(target, func(w io.Writer) (int64, error) {
		var total int64
		keys := map[string][]byte{}
		for _, segment := range playlist.segments {
			payload, err := fetch(segment.uri)
			if err != nil {
				return total, fmt.Errorf("download HLS segment: %w", err)
			}
			if segment.key != nil {
				key, ok := keys[segment.key.uri]
				if !ok {
					key, err = fetch(segment.key.uri)
					if err != nil {
						return total, fmt.Errorf("download HLS key: %w", err)
					}
					if len(key) != aes.BlockSize {
						return total, fmt.Errorf("HLS AES-128 key has %d bytes, want %d", len(key), aes.BlockSize)
					}
					keys[segment.key.uri] = key
				}
				iv := segment.key.iv
				if len(iv) == 0 {
					iv = hlsSequenceIV(segment.sequence)
				}
				payload, err = decryptHLSSegment(payload, key, iv)
				if err != nil {
					return total, err
				}
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
	})
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

func writeNewMediaFile(path string, write func(io.Writer) (int64, error)) (written int64, err error) {
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

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return 0, fmt.Errorf("create media file: %w", err)
	}
	closed := false
	completed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
		if !completed {
			_ = os.Remove(path)
		}
	}()

	written, err = write(file)
	if err != nil {
		return 0, err
	}
	if err = file.Close(); err != nil {
		return 0, fmt.Errorf("close media file: %w", err)
	}
	closed = true
	completed = true
	return written, nil
}
