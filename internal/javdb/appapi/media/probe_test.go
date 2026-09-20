package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
)

// probe 契约：只读取识别媒体 header 所需的字节。
// 图片探测在得到尺寸后停止；HLS 探测扫描完整 playlist 取 duration，
// 并只读取首段直到找到 H.264 SPS。解析上限与 best-effort 语义在这里覆盖。

type trackedProbeBody struct {
	reader io.Reader
	read   int
	closed bool
}

func (b *trackedProbeBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	b.read += n
	return n, err
}

func (b *trackedProbeBody) Close() error {
	b.closed = true
	return nil
}

func encodedProbeImage(t *testing.T, format string, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.RGBA{R: 0x80, G: 0x40, B: 0x20, A: 0xff})
	var out bytes.Buffer
	var err error
	switch format {
	case "jpeg":
		err = jpeg.Encode(&out, img, nil)
	case "png":
		err = png.Encode(&out, img)
	case "gif":
		err = gif.Encode(&out, img, nil)
	default:
		t.Fatalf("unsupported test image format %q", format)
	}
	if err != nil {
		t.Fatalf("encode %s fixture: %v", format, err)
	}
	return out.Bytes()
}

func webPProbeFixture(kind string, width, height int) []byte {
	var payload []byte
	switch kind {
	case "VP8 ":
		payload = []byte{0, 0, 0, 0x9d, 0x01, 0x2a, byte(width), byte(width >> 8), byte(height), byte(height >> 8)}
	case "VP8L":
		packed := uint32(width-1) | uint32(height-1)<<14
		payload = []byte{0x2f, byte(packed), byte(packed >> 8), byte(packed >> 16), byte(packed >> 24)}
	case "VP8X":
		payload = []byte{0, 0, 0, 0, byte(width - 1), byte((width - 1) >> 8), byte((width - 1) >> 16), byte(height - 1), byte((height - 1) >> 8), byte((height - 1) >> 16)}
	}
	chunkSize := len(payload)
	fixture := make([]byte, 20+chunkSize)
	copy(fixture[:4], "RIFF")
	binary.LittleEndian.PutUint32(fixture[4:8], uint32(len(fixture)-8))
	copy(fixture[8:12], "WEBP")
	copy(fixture[12:16], kind)
	binary.LittleEndian.PutUint32(fixture[16:20], uint32(chunkSize))
	copy(fixture[20:], payload)
	return fixture
}

// TestProbeImageMetadataStreamsSupportedFormats 覆盖图片格式识别与 header 级流式读取。
func TestProbeImageMetadataStreamsSupportedFormats(t *testing.T) {
	const width, height = 37, 23
	tests := []struct {
		name    string
		payload []byte
	}{
		{name: "jpeg", payload: encodedProbeImage(t, "jpeg", width, height)},
		{name: "png", payload: encodedProbeImage(t, "png", width, height)},
		{name: "gif", payload: encodedProbeImage(t, "gif", width, height)},
		{name: "webp vp8", payload: webPProbeFixture("VP8 ", width, height)},
		{name: "webp vp8l", payload: webPProbeFixture("VP8L", width, height)},
		{name: "webp vp8x", payload: webPProbeFixture("VP8X", width, height)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := &trackedProbeBody{reader: bytes.NewReader(tc.payload)}
			endpoint := NewMedia(func(context.Context, string) (io.ReadCloser, error) {
				return body, nil
			})
			metadata, err := endpoint.ProbeImageMetadata(context.Background(), "https://media.example.test/image")
			if err != nil {
				t.Fatalf("ProbeImageMetadata() error = %v", err)
			}
			if metadata.Width != width || metadata.Height != height {
				t.Fatalf("metadata = %+v, want %dx%d", metadata, width, height)
			}
			if !body.closed {
				t.Fatal("ProbeImageMetadata() did not close response body")
			}
		})
	}
}

// TestProbeImageMetadataUnwrapsXORAndStopsAfterHeader 是 #48 的核心资源保证：
// 识别出尺寸后不得继续读取图片尾部。
func TestProbeImageMetadataUnwrapsXORAndStopsAfterHeader(t *testing.T) {
	const width, height = 41, 29
	payload := encodedProbeImage(t, "png", width, height)
	const key = byte(0x97)
	wrapped := make([]byte, len(payload)+1)
	wrapped[0] = key
	for i, value := range payload {
		wrapped[i+1] = value ^ key
	}
	wrapped = append(wrapped, bytes.Repeat([]byte{0xaa}, 1<<20)...)
	body := &trackedProbeBody{reader: bytes.NewReader(wrapped)}
	endpoint := NewMedia(func(context.Context, string) (io.ReadCloser, error) {
		return body, nil
	})
	metadata, err := endpoint.ProbeImageMetadata(context.Background(), "https://media.example.test/image")
	if err != nil {
		t.Fatalf("ProbeImageMetadata() error = %v", err)
	}
	if metadata.Width != width || metadata.Height != height {
		t.Fatalf("metadata = %+v, want %dx%d", metadata, width, height)
	}
	if body.read >= len(wrapped) {
		t.Fatalf("ProbeImageMetadata() read full response: %d bytes", body.read)
	}
	if !body.closed {
		t.Fatal("ProbeImageMetadata() did not close response body")
	}
}

// TestProbeHLSMetadataExtractsDurationAndSPS 覆盖完整 HLS 链路：
// playlist → EXTINF duration → AES-128 → TS → PAT/PMT → H.264 SPS。
// 首段附带巨大尾部，验证拿到 SPS 后停止读取并关闭 response body。
func TestProbeHLSMetadataExtractsDurationAndSPS(t *testing.T) {
	const playlistURL = "https://media.example.test/previews/index.m3u8"
	const segmentURL = "https://media.example.test/previews/a.ts"
	const keyURL = "https://media.example.test/previews/key.bin"
	key := []byte("0123456789abcdef")
	segment := append(validTSSegmentAt(0), bytes.Repeat([]byte{0xaa}, 1<<20)...)
	for _, tc := range []struct {
		name     string
		playlist []byte
		segment  []byte
		key      []byte
	}{
		{
			name:     "plain",
			playlist: []byte("#EXTM3U\n#EXTINF:10.125,\na.ts\n#EXTINF:9.875,\nb.ts\n#EXT-X-ENDLIST\n"),
			segment:  segment,
		},
		{
			name:     "aes-128",
			playlist: []byte("#EXTM3U\n#EXT-X-MEDIA-SEQUENCE:7\n#EXT-X-KEY:METHOD=AES-128,URI=\"key.bin\"\n#EXTINF:10.125,\na.ts\n#EXTINF:9.875,\nb.ts\n#EXT-X-ENDLIST\n"),
			segment:  encryptHLSTestPayload(t, segment, key, hlsSequenceIV(7)),
			key:      key,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			segmentBody := &trackedProbeBody{reader: bytes.NewReader(tc.segment)}
			endpoint := NewMedia(func(_ context.Context, uri string) (io.ReadCloser, error) {
				switch uri {
				case playlistURL:
					return io.NopCloser(bytes.NewReader(tc.playlist)), nil
				case segmentURL:
					return segmentBody, nil
				case keyURL:
					return io.NopCloser(bytes.NewReader(tc.key)), nil
				default:
					return nil, fmt.Errorf("unexpected media URI %q", uri)
				}
			})
			metadata, err := endpoint.ProbeHLSMetadata(context.Background(), playlistURL)
			if err != nil {
				t.Fatalf("ProbeHLSMetadata() error = %v", err)
			}
			if metadata.Width != 640 || metadata.Height != 480 || metadata.DurationSeconds != 20 {
				t.Fatalf("metadata = %+v, want 640x480 and 20 seconds", metadata)
			}
			if segmentBody.read >= len(tc.segment) {
				t.Fatalf("ProbeHLSMetadata() read full segment: %d/%d bytes", segmentBody.read, len(tc.segment))
			}
			if !segmentBody.closed {
				t.Fatal("ProbeHLSMetadata() did not close segment body")
			}
		})
	}
}

// TestProbeHLSMetadataPartialAndMalformedSafety 覆盖 best-effort 与解析资源边界：
// duration 与 SPS 相互独立；超长 playlist 行明确失败且提前停止读取。
func TestProbeHLSMetadataPartialAndMalformedSafety(t *testing.T) {
	const playlistURL = "https://media.example.test/previews/index.m3u8"

	t.Run("keeps duration when segment probe fails", func(t *testing.T) {
		endpoint := NewMedia(byteFetch(func(_ context.Context, uri string) ([]byte, error) {
			if uri == playlistURL {
				return []byte("#EXTM3U\n#EXTINF:20.0,\nbroken.ts\n#EXT-X-ENDLIST\n"), nil
			}
			return nil, errors.New("segment unavailable")
		}))
		metadata, err := endpoint.ProbeHLSMetadata(context.Background(), playlistURL)
		if err != nil {
			t.Fatalf("ProbeHLSMetadata() error = %v", err)
		}
		if metadata.DurationSeconds != 20 || metadata.Width != 0 || metadata.Height != 0 {
			t.Fatalf("metadata = %+v, want duration-only result", metadata)
		}
	})

	t.Run("does not fabricate duration from a non-representable EXTINF total", func(t *testing.T) {
		// 每行 EXTINF 都是有限正数，但累加结果溢出为 +Inf：对调用方只有一个
		// 可见契约——duration unknown，不得退化为虚假时长。
		playlist := "#EXTM3U\n#EXTINF:1.7e308,\na.ts\n#EXTINF:1.7e308,\nb.ts\n#EXT-X-ENDLIST\n"
		endpoint := NewMedia(func(_ context.Context, uri string) (io.ReadCloser, error) {
			if uri == playlistURL {
				return io.NopCloser(strings.NewReader(playlist)), nil
			}
			return io.NopCloser(bytes.NewReader(validTSSegmentAt(0))), nil
		})
		metadata, err := endpoint.ProbeHLSMetadata(context.Background(), playlistURL)
		if err != nil {
			t.Fatalf("ProbeHLSMetadata() error = %v", err)
		}
		if metadata.DurationSeconds != 0 {
			t.Fatalf("duration = %v, want unknown for overflowing EXTINF total", metadata.DurationSeconds)
		}
	})

	t.Run("rejects oversized playlist line without buffering it", func(t *testing.T) {
		// 缺失换行的响应；读取必须在行上限处失败，而不是把整段缓存进内存。
		payload := append([]byte("#EXTM3U\n"), bytes.Repeat([]byte{'a'}, 1<<20)...)
		body := &trackedProbeBody{reader: bytes.NewReader(payload)}
		endpoint := NewMedia(func(_ context.Context, uri string) (io.ReadCloser, error) {
			return body, nil
		})
		_, err := endpoint.ProbeHLSMetadata(context.Background(), playlistURL)
		if err == nil || !strings.Contains(err.Error(), "line too long") {
			t.Fatalf("ProbeHLSMetadata() error = %v, want line-too-long failure", err)
		}
		if body.read >= len(payload) {
			t.Fatalf("oversized playlist line buffered %d/%d bytes", body.read, len(payload))
		}
	})
}

// TestProbeMediaResourceStress 是显式的资源伸缩回归：大段畸形/合法 SPS 下，
// 解析状态不得随输入规模成比例增长。默认 skip，只在 JAVDB_ASSET_PROBE_STRESS=1 时运行。
func TestProbeMediaResourceStress(t *testing.T) {
	if os.Getenv("JAVDB_ASSET_PROBE_STRESS") == "" {
		t.Skip("set JAVDB_ASSET_PROBE_STRESS=1 to run media resource stress")
	}
	const playlistURL = "https://media.example.test/previews/index.m3u8"
	// 判定依据是“分配量不得随输入规模成比例增长”：输入约 34 MiB，若解析器
	// 缓存 payload，分配量至少会是输入量级；阈值取输入长度的 1/16，而实际
	// 观察值只有几 KiB。
	const inputPayloadBytes = 32 << 20
	for _, tc := range []struct {
		name    string
		firstES []byte
		fill    byte
		wantErr bool
	}{
		{
			name:    "implausible SPS fails deterministically",
			firstES: append([]byte{0x00, 0x00, 0x00, 0x01, 0x67}, bytes.Repeat([]byte{0x08}, 16)...),
			fill:    0x08,
			wantErr: true,
		},
		{
			name:    "valid SPS stops at the first NAL",
			firstES: []byte{0x00, 0x00, 0x00, 0x01, 0x67, 0x42, 0x00, 0x28, 0xF8, 0x14, 0x07, 0xB2},
			fill:    0x00,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			segment := spsProbeSegment(tc.firstES, inputPayloadBytes, tc.fill)
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			endpoint := NewMedia(func(_ context.Context, uri string) (io.ReadCloser, error) {
				if uri == playlistURL {
					return io.NopCloser(strings.NewReader("#EXTM3U\nmalformed.ts\n#EXT-X-ENDLIST\n")), nil
				}
				return io.NopCloser(bytes.NewReader(segment)), nil
			})
			metadata, err := endpoint.ProbeHLSMetadata(context.Background(), playlistURL)
			runtime.ReadMemStats(&after)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("ProbeHLSMetadata() = %+v, want an SPS parse error", metadata)
				}
			} else if err != nil {
				t.Fatalf("ProbeHLSMetadata() error = %v", err)
			} else if metadata.Width != 640 || metadata.Height != 480 {
				t.Fatalf("metadata = %+v, want 640x480", metadata)
			}

			if alloc := after.TotalAlloc - before.TotalAlloc; alloc > uint64(len(segment))/16 {
				t.Fatalf("parser allocated %d bytes for a %d-byte segment, want bounded state", alloc, len(segment))
			}
		})
	}
}

// spsProbeSegment 构造结构合法的 TS segment：PAT/PMT 后是单个 H.264 PES 及其
// 续包。firstES 是首个 PES 的 ES 前缀（不足处用 fill 补到 payload 上限），后续
// 每个包都填满 184 字节 ES，避免 0xFF stuffing 混入 elementary stream。
func spsProbeSegment(firstES []byte, payloadBytes int, fill byte) []byte {
	const (
		packetPayload = 184
		// 首个 PES 含 19 字节 PES 头（起始码/长度/flags/PTS+DTS），
		// 因此 ES 前缀上限是 184-19。续包没有 PES 头，可填满 184 字节 ES。
		firstESLength = packetPayload - 19
	)
	if len(firstES) > firstESLength {
		panic("first ES prefix exceeds a single TS packet")
	}
	segment := append([]byte{}, tsPacket(patPID, true, 0, patSection())...)
	segment = append(segment, tsPacket(pmtPID, true, 0, pmtSection())...)
	head := append([]byte{}, firstES...)
	head = append(head, bytes.Repeat([]byte{fill}, firstESLength-len(head))...)
	segment = append(segment, tsPacket(videoPID, true, 1, pesBytes(0xE0, 90000, 90000, true, head))...)
	remaining := payloadBytes
	cc := byte(2)
	for remaining > 0 {
		size := packetPayload
		if size > remaining {
			size = remaining
		}
		segment = append(segment, tsPacket(videoPID, false, cc, bytes.Repeat([]byte{fill}, size))...)
		cc = (cc + 1) & 0x0F
		remaining -= size
	}
	return segment
}
