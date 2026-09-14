package media

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
)

// 资产元信息 probe(计划 #2/#4/#5/#6):
// JavDB 详情接口没有提供资产级 width/height/duration,需要在 assets list
// 内对最终选中的资产做 best-effort probe。
// 执行顺序:获取详情 → 构造资产列表 → --type 过滤 → selector 选择 →
// metadata probe → 输出。只有真正消费元信息的输出模式执行 probe。

// ProbeLimits 是单个 probe 请求的读取预算(计划 #3)。
type ProbeLimits struct {
	// Enabled 为 false 时完全关闭 metadata probe。
	Enabled bool
	// Concurrency 是 probe 最大并发。
	Concurrency int
	// ImageMaxBytes 是单张图片最多读取的 probe 数据量。
	ImageMaxBytes int64
	// PlaylistMaxBytes 是 HLS playlist probe 最大读取量。
	PlaylistMaxBytes int64
	// VideoSegmentMaxBytes 是为获取 SPS 读取首个 TS segment 的最大数据量。
	VideoSegmentMaxBytes int64
	// Timeout 是单个 probe 请求超时(秒);0 表示无额外超时。
	TimeoutSeconds float64
}

// DefaultProbeLimits 是 [assets.probe] 表不存在时的等效默认配置。
func DefaultProbeLimits() ProbeLimits {
	return ProbeLimits{
		Enabled:              true,
		Concurrency:          4,
		ImageMaxBytes:        65536,
		PlaylistMaxBytes:     262144,
		VideoSegmentMaxBytes: 262144,
		TimeoutSeconds:       10,
	}
}

// ProbeResult 是单资产的 best-effort 元信息;
// 无法取得的字段以零值表示,由上层省略(不使用 0 冒充有效元信息)。
type ProbeResult struct {
	Width    int
	Height   int
	Duration float64
	hasDims  bool
	hasDur   bool
}

// HasDimensions 报告 width/height 是否取得。
func (r ProbeResult) HasDimensions() bool { return r.hasDims }

// HasDuration 报告 duration 是否取得。
func (r ProbeResult) HasDuration() bool { return r.hasDur }

// probeFetcher 是 probe 的资源读取回调;与 FetchContext 同形。
type probeFetcher func(ctx context.Context, url string) ([]byte, error)

// ProbeAssetMetadata 对最终选中的资产做 best-effort metadata probe。
// 同一次调用内按 URL 去重(计划 #6);probe 失败只省略 metadata,
// 不让整个调用失败;context 取消正常传播。
func ProbeAssetMetadata(ctx context.Context, fetch probeFetcher, assetType, url string, limits ProbeLimits) ProbeResult {
	if !limits.Enabled {
		return ProbeResult{}
	}
	switch assetType {
	case "image":
		result, err := probeImage(ctx, fetch, url, limits)
		if err != nil {
			return ProbeResult{}
		}
		return result
	case "video":
		result, err := probeVideo(ctx, fetch, url, limits)
		if err != nil {
			return ProbeResult{}
		}
		return result
	default:
		return ProbeResult{}
	}
}

// ProbeAssetsConcurrent 并发 probe 多个资产,按 URL 去重(计划 #6);
// concurrency 来自配置,不加 CLI flag。
func ProbeAssetsConcurrent(ctx context.Context, fetch probeFetcher, types []string, urls []string, limits ProbeLimits) []ProbeResult {
	results := make([]ProbeResult, len(urls))
	if !limits.Enabled || len(urls) == 0 {
		return results
	}
	concurrency := limits.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	// 按 URL 去重:同一 URL 只 probe 一次。
	type key struct {
		assetType string
		url       string
	}
	dedup := map[key]int{}
	for i := range urls {
		k := key{types[i], urls[i]}
		if _, ok := dedup[k]; !ok {
			dedup[k] = i
		}
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for k, first := range dedup {
		wg.Add(1)
		go func(assetType, url string, idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := ctx.Err(); err != nil {
				return
			}
			results[idx] = ProbeAssetMetadata(ctx, fetch, assetType, url, limits)
		}(k.assetType, k.url, first)
	}
	wg.Wait()
	// 去重回填:同 URL 的后续位置复用首个结果。
	for i := range urls {
		k := key{types[i], urls[i]}
		if idx, ok := dedup[k]; ok && idx != i {
			results[i] = results[idx]
		}
	}
	return results
}

// probeImage 对单张图片执行一次 bounded Range 请求(计划 #4);
// 不做渐进式 Range 与扩大读取兜底。
func probeImage(ctx context.Context, fetch probeFetcher, url string, limits ProbeLimits) (ProbeResult, error) {
	raw, err := fetch(ctx, url)
	if err != nil {
		return ProbeResult{}, err
	}
	if int64(len(raw)) > limits.ImageMaxBytes {
		raw = raw[:limits.ImageMaxBytes]
	}
	// 图片 CDN 的 XOR 包装:probe 路径复用现有解包规则,
	// 但只处理当前已读取的有限前缀。
	payload := raw
	if _, ok := imagePayloadFormat(payload); !ok && len(payload) >= 2 {
		key := payload[0]
		decoded := make([]byte, len(payload)-1)
		for i := range decoded {
			decoded[i] = payload[i+1] ^ key
		}
		payload = decoded
	}
	w, h, ok := imageDimensionsFromPrefix(payload)
	if !ok {
		// 达到读取上限仍无法得到尺寸:省略 width/height,不扩大请求。
		return ProbeResult{}, nil
	}
	return ProbeResult{Width: w, Height: h, hasDims: true}, nil
}

// probeVideo 请求 HLS media playlist 并解析 EXTINF 求和 duration(计划 #5);
// 只 probe 首个 TS segment 获取 SPS 宽高,不从 rendition 文件名推断。
func probeVideo(ctx context.Context, fetch probeFetcher, url string, limits ProbeLimits) (ProbeResult, error) {
	var result ProbeResult
	playlistBody, err := fetch(ctx, url)
	if err != nil {
		return ProbeResult{}, err
	}
	if int64(len(playlistBody)) > limits.PlaylistMaxBytes {
		playlistBody = playlistBody[:limits.PlaylistMaxBytes]
	}
	// duration = Σ EXTINF(计划 #5):不用顶层 duration/文件大小/URL 名字。
	duration, ok := hlsPlaylistDuration(string(playlistBody))
	if ok {
		result.Duration = duration
		result.hasDur = true
	}
	// 只 probe 首个 TS segment:最多读取 video_segment_max_bytes。
	firstSegment := hlsFirstSegmentURI(string(playlistBody))
	if firstSegment == "" {
		return result, nil
	}
	resolved, err := resolveHLSURI(url, firstSegment)
	if err != nil {
		return result, nil
	}
	segmentData, err := fetch(ctx, resolved)
	if err != nil {
		return result, nil
	}
	if int64(len(segmentData)) > limits.VideoSegmentMaxBytes {
		segmentData = segmentData[:limits.VideoSegmentMaxBytes]
	}
	w, h, ok := videoDimensionsFromSegment(segmentData)
	if !ok {
		return result, nil
	}
	result.Width = w
	result.Height = h
	result.hasDims = true
	return result, nil
}

// hlsPlaylistDuration 解析全部 #EXTINF 并求和(秒)。
func hlsPlaylistDuration(playlist string) (float64, bool) {
	total := 0.0
	count := 0
	for _, line := range strings.Split(playlist, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#EXTINF:") {
			continue
		}
		value := strings.TrimSuffix(strings.TrimPrefix(line, "#EXTINF:"), ",")
		value = strings.TrimSpace(value)
		var d float64
		if _, err := fmt.Sscanf(value, "%f", &d); err != nil {
			continue
		}
		total += d
		count++
	}
	if count == 0 {
		return 0, false
	}
	return total, true
}

// hlsFirstSegmentURI 返回 playlist 中第一个非注释行(segment URI)。
func hlsFirstSegmentURI(playlist string) string {
	for _, line := range strings.Split(playlist, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}

// videoDimensionsFromSegment 从有限 TS segment 前缀提取 H.264 SPS 宽高。
// 当前实现只处理未加密 segment(真实 JavDB preview 无 AES-128)。
// 有限前缀内没有得到 SPS:省略 width/height。
func videoDimensionsFromSegment(segment []byte) (int, int, bool) {
	streams, err := parseTSStreams(segment)
	if err != nil {
		return 0, 0, false
	}
	for _, stream := range streams {
		if stream.streamType != streamTypeH264 {
			continue
		}
		track, err := parseH264Track(*stream)
		if err != nil || len(track.ParamSets) == 0 {
			continue
		}
		// 找 SPS(type 7)。
		for _, ps := range track.ParamSets {
			if ps[0]&0x1F == 7 {
				w, h, err := parseSPSDimensions(ps)
				if err != nil {
					continue
				}
				return int(w), int(h), true
			}
		}
	}
	return 0, 0, false
}

// imageDimensionsFromPrefix 从有限图片前缀获取尺寸(计划 #4):
// JPEG 扫描 SOF;PNG/GIF/WebP 使用头部结构;AVIF/HEIC 从 ISOBMFF ispe 获取。
func imageDimensionsFromPrefix(data []byte) (int, int, bool) {
	if len(data) < 8 {
		return 0, 0, false
	}
	switch {
	case data[0] == 0xFF && data[1] == 0xD8: // JPEG
		return jpegDimensions(data)
	case string(data[:8]) == "\x89PNG\r\n\x1a\n":
		if len(data) >= 24 {
			return int(binary.BigEndian.Uint32(data[16:20])), int(binary.BigEndian.Uint32(data[20:24])), true
		}
		return 0, 0, false
	case data[0] == 'G' && data[1] == 'I' && data[2] == 'F':
		if len(data) >= 10 {
			return int(binary.LittleEndian.Uint16(data[6:8])), int(binary.LittleEndian.Uint16(data[8:10])), true
		}
		return 0, 0, false
	case len(data) >= 12 && string(data[8:12]) == "WEBP":
		return webpDimensions(data)
	case len(data) >= 12 && (string(data[4:8]) == "ftyp"):
		return isobmffDimensions(data)
	default:
		return 0, 0, false
	}
}

// jpegDimensions 扫描 JPEG SOF marker 获取尺寸。
func jpegDimensions(data []byte) (int, int, bool) {
	pos := 2
	for pos+9 < len(data) {
		if data[pos] != 0xFF {
			pos++
			continue
		}
		marker := data[pos+1]
		// SOF0-SOF15(排除 DHT/DAC/RST/SOF 无关 marker)。
		if (marker >= 0xC0 && marker <= 0xC3) || (marker >= 0xC5 && marker <= 0xC7) ||
			(marker >= 0xC9 && marker <= 0xCB) || (marker >= 0xCD && marker <= 0xCF) {
			if pos+9 > len(data) {
				return 0, 0, false
			}
			height := int(binary.BigEndian.Uint16(data[pos+5 : pos+7]))
			width := int(binary.BigEndian.Uint16(data[pos+7 : pos+9]))
			return width, height, true
		}
		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			pos += 2
			continue
		}
		if pos+4 > len(data) {
			return 0, 0, false
		}
		segLen := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		if segLen < 2 {
			return 0, 0, false
		}
		pos += 2 + segLen
	}
	return 0, 0, false
}

// webpDimensions 从 VP8/VP8L/VP8X chunk 获取尺寸。
func webpDimensions(data []byte) (int, int, bool) {
	if len(data) < 30 {
		return 0, 0, false
	}
	chunk := string(data[12:16])
	switch chunk {
	case "VP8 ":
		// 帧头 3+3 字节后是 14 bit width/height(小端)。
		if len(data) < 30 {
			return 0, 0, false
		}
		w := int(binary.LittleEndian.Uint16(data[26:28]) & 0x3FFF)
		h := int(binary.LittleEndian.Uint16(data[28:30]) & 0x3FFF)
		return w, h, true
	case "VP8L":
		if len(data) < 25 {
			return 0, 0, false
		}
		bits := binary.LittleEndian.Uint32(data[21:25])
		w := int(bits&0x3FFF) + 1
		h := int((bits>>14)&0x3FFF) + 1
		return w, h, true
	case "VP8X":
		if len(data) < 30 {
			return 0, 0, false
		}
		w := int(binary.LittleEndian.Uint32(data[24:27])&0xFFFFFF) + 1
		h := int(binary.LittleEndian.Uint32(data[27:30])&0xFFFFFF) + 1
		return w, h, true
	default:
		return 0, 0, false
	}
}

// isobmffDimensions 从 ISOBMFF(AVIF/HEIC)ispe box 获取尺寸。
func isobmffDimensions(data []byte) (int, int, bool) {
	// 扫描 ispe box:box 头 8 字节 + version/flags 4 + width/height 各 4。
	for i := 0; i+20 <= len(data); i++ {
		if string(data[i+4:i+8]) == "ispe" {
			w := int(binary.BigEndian.Uint32(data[i+12 : i+16]))
			h := int(binary.BigEndian.Uint32(data[i+16 : i+20]))
			return w, h, true
		}
	}
	return 0, 0, false
}
