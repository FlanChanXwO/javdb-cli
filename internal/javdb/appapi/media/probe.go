package media

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"strconv"
	"strings"
)

// ProbeMetadata 是媒体流本身可确定的元数据。DurationSeconds 保留浮点精度，
// 对外整数化由 SDK 边界统一处理。
type ProbeMetadata struct {
	Width           int
	Height          int
	DurationSeconds float64
}

type hlsProbePlaylist struct {
	firstSegment    hlsSegment
	hasFirstSegment bool
	durationSeconds float64
	durationValid   bool
	endList         bool
}

// ProbeImageMetadata 流式读取图片 header；得到尺寸后立即停止，不下载像素或尾部。
func (e *MediaEndpoint) ProbeImageMetadata(ctx context.Context, sourceURL string) (ProbeMetadata, error) {
	if err := ctx.Err(); err != nil {
		return ProbeMetadata{}, err
	}
	body, err := e.fetch(ctx, sourceURL)
	if err != nil {
		return ProbeMetadata{}, err
	}
	if body == nil {
		return ProbeMetadata{}, fmt.Errorf("image probe response body is nil")
	}
	metadata, probeErr := probeImageMetadata(&contextReader{ctx: ctx, reader: body})
	closeErr := body.Close()
	if err := ctx.Err(); err != nil {
		return ProbeMetadata{}, errors.Join(err, closeErr)
	}
	return metadata, errors.Join(probeErr, closeErr)
}

// ProbeHLSMetadata 扫描完整 VOD playlist 获取 duration，并只读取首段直到找到 H.264 SPS。
// 首段探测失败时，已经由完整 EXTINF 列表确定的 duration 仍然有效。
func (e *MediaEndpoint) ProbeHLSMetadata(ctx context.Context, playlistURL string) (ProbeMetadata, error) {
	if err := ctx.Err(); err != nil {
		return ProbeMetadata{}, err
	}
	body, err := e.fetch(ctx, playlistURL)
	if err != nil {
		return ProbeMetadata{}, fmt.Errorf("download HLS playlist: %w", err)
	}
	if body == nil {
		return ProbeMetadata{}, fmt.Errorf("download HLS playlist: media response body is nil")
	}
	playlist, parseErr := parseHLSProbePlaylistReader(playlistURL, &contextReader{ctx: ctx, reader: body})
	closeErr := body.Close()
	if parseErr != nil || closeErr != nil {
		return ProbeMetadata{}, errors.Join(parseErr, closeErr)
	}

	metadata := ProbeMetadata{}
	if playlist.durationValid {
		metadata.DurationSeconds = playlist.durationSeconds
	}
	width, height, probeErr := e.probeHLSSegmentDimensions(ctx, playlist.firstSegment)
	if err := ctx.Err(); err != nil {
		return ProbeMetadata{}, err
	}
	if probeErr != nil {
		// 只有父 ctx 真正结束才放弃；这里是上面的 ctx.Err() 检查之后的代码，
		// 因此单项 transport 超时（即使错误本身包装了 context.DeadlineExceeded）
		// 一律按 best-effort 处理，不影响已经确定的 duration。
		if playlist.durationValid {
			return metadata, nil
		}
		return ProbeMetadata{}, probeErr
	}
	metadata.Width = width
	metadata.Height = height
	return metadata, nil
}

func (e *MediaEndpoint) probeHLSSegmentDimensions(ctx context.Context, segment hlsSegment) (int, int, error) {
	var (
		key []byte
		err error
	)
	if segment.key != nil {
		key, err = fetchHLSKey(ctx, e.fetch, segment.key.uri)
		if err != nil {
			return 0, 0, fmt.Errorf("download HLS key: %w", err)
		}
	}
	body, err := e.fetch(ctx, segment.uri)
	if err != nil {
		return 0, 0, fmt.Errorf("download HLS segment: %w", err)
	}
	if body == nil {
		return 0, 0, fmt.Errorf("download HLS segment: media response body is nil")
	}
	var payload io.Reader = body
	if segment.key != nil {
		iv := segment.key.iv
		if len(iv) == 0 {
			iv = hlsSequenceIV(segment.sequence)
		}
		payload, err = newProbeCBCReader(body, key, iv)
		if err != nil {
			return 0, 0, errors.Join(err, body.Close())
		}
	}
	width, height, probeErr := probeTSDimensions(&contextReader{ctx: ctx, reader: payload})
	closeErr := body.Close()
	if err := ctx.Err(); err != nil {
		return 0, 0, errors.Join(err, closeErr)
	}
	return width, height, errors.Join(probeErr, closeErr)
}

// contextReader 在底层 reader 即使继续产出数据时，也把取消及时传播给解析器。
type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	if ctxErr := r.ctx.Err(); ctxErr != nil {
		return 0, ctxErr
	}
	return n, err
}

func probeImageMetadata(raw io.Reader) (ProbeMetadata, error) {
	reader := bufio.NewReader(raw)
	prefix, peekErr := reader.Peek(13)
	if peekErr != nil && !errors.Is(peekErr, io.EOF) {
		return ProbeMetadata{}, peekErr
	}
	format, ok := imagePayloadFormat(prefix)
	var decoded io.Reader = reader
	if !ok {
		if len(prefix) < 2 {
			return ProbeMetadata{}, fmt.Errorf("media response is not a recognized image")
		}
		key := prefix[0]
		decodedPrefix := make([]byte, len(prefix)-1)
		for i := range decodedPrefix {
			decodedPrefix[i] = prefix[i+1] ^ key
		}
		format, ok = imagePayloadFormat(decodedPrefix)
		if !ok {
			return ProbeMetadata{}, fmt.Errorf("media response is not a recognized image")
		}
		if _, err := reader.ReadByte(); err != nil {
			return ProbeMetadata{}, err
		}
		decoded = &xorProbeReader{reader: reader, key: key}
	}

	if format == "webp" {
		width, height, err := probeWebPDimensions(decoded)
		return ProbeMetadata{Width: width, Height: height}, err
	}
	if format == "avif" || format == "heic" {
		return ProbeMetadata{}, fmt.Errorf("%s image metadata probing is not supported", format)
	}
	config, _, err := image.DecodeConfig(decoded)
	if err != nil {
		return ProbeMetadata{}, err
	}
	return ProbeMetadata{Width: config.Width, Height: config.Height}, nil
}

type xorProbeReader struct {
	reader io.Reader
	key    byte
}

func (r *xorProbeReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	for i := 0; i < n; i++ {
		p[i] ^= r.key
	}
	return n, err
}

func probeWebPDimensions(reader io.Reader) (int, int, error) {
	var header [12]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return 0, 0, err
	}
	if string(header[:4]) != "RIFF" || string(header[8:]) != "WEBP" {
		return 0, 0, fmt.Errorf("invalid WebP header")
	}
	remaining := int64(binary.LittleEndian.Uint32(header[4:8])) - 4
	for remaining >= 8 {
		var chunkHeader [8]byte
		if _, err := io.ReadFull(reader, chunkHeader[:]); err != nil {
			return 0, 0, err
		}
		remaining -= 8
		chunkSize := int64(binary.LittleEndian.Uint32(chunkHeader[4:8]))
		if chunkSize < 0 || chunkSize > remaining {
			return 0, 0, fmt.Errorf("WebP chunk exceeds RIFF payload")
		}
		width, height, found, err := probeWebPChunkDimensions(string(chunkHeader[:4]), io.LimitReader(reader, chunkSize), chunkSize)
		if err != nil {
			return 0, 0, err
		}
		if found {
			return width, height, nil
		}
		if _, err := io.CopyN(io.Discard, reader, chunkSize); err != nil {
			return 0, 0, err
		}
		remaining -= chunkSize
		if chunkSize%2 != 0 {
			if _, err := io.CopyN(io.Discard, reader, 1); err != nil {
				return 0, 0, err
			}
			remaining--
		}
	}
	return 0, 0, fmt.Errorf("WebP has no supported image chunk")
}

func probeWebPChunkDimensions(kind string, reader io.Reader, size int64) (width, height int, found bool, err error) {
	var required int
	switch kind {
	case "VP8 ":
		required = 10
	case "VP8L":
		required = 5
	case "VP8X":
		required = 10
	default:
		return 0, 0, false, nil
	}
	if size < int64(required) {
		return 0, 0, false, fmt.Errorf("WebP %s chunk is truncated", kind)
	}
	payload := make([]byte, required)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return 0, 0, false, err
	}
	switch kind {
	case "VP8 ":
		if payload[3] != 0x9d || payload[4] != 0x01 || payload[5] != 0x2a {
			return 0, 0, false, fmt.Errorf("invalid WebP VP8 frame header")
		}
		width = int(binary.LittleEndian.Uint16(payload[6:8]) & 0x3fff)
		height = int(binary.LittleEndian.Uint16(payload[8:10]) & 0x3fff)
	case "VP8L":
		if payload[0] != 0x2f {
			return 0, 0, false, fmt.Errorf("invalid WebP VP8L frame header")
		}
		packed := binary.LittleEndian.Uint32(payload[1:5])
		width = int(packed&0x3fff) + 1
		height = int((packed>>14)&0x3fff) + 1
	case "VP8X":
		width = int(payload[4]) | int(payload[5])<<8 | int(payload[6])<<16
		height = int(payload[7]) | int(payload[8])<<8 | int(payload[9])<<16
		width++
		height++
	}
	if width <= 0 || height <= 0 {
		return 0, 0, false, fmt.Errorf("invalid WebP dimensions %dx%d", width, height)
	}
	return width, height, true, nil
}

// maxHLSProbeLineBytes 是单行 HLS playlist 的解析上限。合法行（标签、EXTINF、
// 密钥 URI、分片 URI）远小于该值；超长行只可能来自缺失换行的异常响应，
// 若不加限制 bufio 会为单行持续扩容直到读完全部响应。该上限属于 parser 的
// 内存安全实现细节，不是用户可配置项，也不截断合法内容。
const maxHLSProbeLineBytes = 64 << 10

// errHLSProbeLineTooLong 表示单行 HLS playlist 超过解析上限。
var errHLSProbeLineTooLong = errors.New("HLS playlist line too long")

// readHLSProbeLine 读取一行，同时把单行长度限制在 maxHLSProbeLineBytes 内。
// 用 ReadSlice 而不是 ReadString：缓冲区被填满时不会为超长行扩容，
// 拼接只在累计长度未越界的范围内进行，随后立即以明确的错误失败。
// 返回的错误可能是 io.EOF，由调用方保持原有的行处理语义。
func readHLSProbeLine(reader *bufio.Reader) (string, error) {
	var line []byte
	for {
		chunk, err := reader.ReadSlice('\n')
		line = append(line, chunk...)
		if len(line) > maxHLSProbeLineBytes {
			return "", fmt.Errorf("%w: exceeds %d bytes", errHLSProbeLineTooLong, maxHLSProbeLineBytes)
		}
		switch {
		case err == nil:
			return string(line), nil
		case errors.Is(err, bufio.ErrBufferFull):
			// 行尚未结束，继续读取；累计长度已由上面的检查约束。
			continue
		case errors.Is(err, io.EOF):
			return string(line), io.EOF
		default:
			return "", err
		}
	}
}

func parseHLSProbePlaylistReader(playlistURL string, raw io.Reader) (hlsProbePlaylist, error) {
	reader := bufio.NewReader(raw)
	firstLine, err := readHLSProbeLine(reader)
	if err != nil && !errors.Is(err, io.EOF) {
		return hlsProbePlaylist{}, err
	}
	if strings.TrimSpace(firstLine) != "#EXTM3U" {
		return hlsProbePlaylist{}, fmt.Errorf("media response is not an HLS playlist")
	}

	var (
		playlist        hlsProbePlaylist
		key             *hlsKey
		sequence        uint64
		durationTotal   float64
		durationValid   = true
		pendingEXTINF   bool
		pendingValid    bool
		pendingDuration float64
	)
	processLine := func(rawLine string) error {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			return nil
		}
		switch {
		case line == "#EXT-X-ENDLIST":
			playlist.endList = true
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
		case strings.HasPrefix(line, "#EXTINF:"):
			if pendingEXTINF {
				durationValid = false
			}
			pendingEXTINF = true
			pendingValid = false
			value := strings.TrimSpace(strings.SplitN(strings.TrimPrefix(line, "#EXTINF:"), ",", 2)[0])
			parsed, err := strconv.ParseFloat(value, 64)
			if err == nil && parsed > 0 && !math.IsNaN(parsed) && !math.IsInf(parsed, 0) {
				pendingDuration = parsed
				pendingValid = true
			}
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
			if !playlist.hasFirstSegment {
				segment := hlsSegment{uri: uri, sequence: sequence}
				if key != nil {
					copied := *key
					copied.iv = append([]byte(nil), key.iv...)
					segment.key = &copied
				}
				playlist.firstSegment = segment
				playlist.hasFirstSegment = true
			}
			if pendingEXTINF && pendingValid {
				durationTotal += pendingDuration
			} else {
				durationValid = false
			}
			pendingEXTINF = false
			pendingValid = false
			pendingDuration = 0
			sequence++
		}
		return nil
	}
	for {
		rawLine, readErr := readHLSProbeLine(reader)
		if len(rawLine) > 0 {
			if err := processLine(rawLine); err != nil {
				return hlsProbePlaylist{}, err
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return hlsProbePlaylist{}, readErr
		}
	}
	if !playlist.endList {
		return hlsProbePlaylist{}, fmt.Errorf("HLS playlist has no end marker")
	}
	if !playlist.hasFirstSegment {
		return hlsProbePlaylist{}, fmt.Errorf("HLS playlist has no media segments")
	}
	if pendingEXTINF {
		durationValid = false
	}
	// 每段 EXTINF 都已校验为有限正数，唯一可能的越界来源是累加溢出；
	// 一旦累加值不再是有限正数就把 duration 视为 unknown，交给调用方
	// 按 best-effort 处理，不做任何 float→int 的错误降级。
	if durationValid && (math.IsNaN(durationTotal) || math.IsInf(durationTotal, 0) || durationTotal <= 0) {
		durationValid = false
	}
	if durationValid {
		playlist.durationSeconds = durationTotal
		playlist.durationValid = true
	}
	return playlist, nil
}
