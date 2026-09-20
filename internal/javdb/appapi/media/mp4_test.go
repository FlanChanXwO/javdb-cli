package media

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// MP4 契约：
// ISO BMFF,ftyp → moov → mdat(Fast Start),avc1/mp4a 轨,sample table 完整;
// Layer C 对最终文件重新解析,任何不一致拒绝发布。

func buildTestTracks(t *testing.T) (*h264Track, *aacTrack) {
	t.Helper()
	streams, err := parseTSStreams(validTSSegmentAt(0))
	if err != nil {
		t.Fatalf("parseTSStreams: %v", err)
	}
	video, err := parseH264Track(*streams[videoPID])
	if err != nil {
		t.Fatalf("parseH264Track: %v", err)
	}
	audio, err := parseAACTrack(*streams[audioPID])
	if err != nil {
		t.Fatalf("parseAACTrack: %v", err)
	}
	return video, audio
}

func TestBuildMP4ProducesFastStartStructure(t *testing.T) {
	video, audio := buildTestTracks(t)
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatalf("buildMP4: %v", err)
	}
	boxes := scanTopLevelBoxes(t, mp4)
	if len(boxes) < 3 {
		t.Fatalf("top-level boxes = %v", boxes)
	}
	if boxes[0].kind != "ftyp" || boxes[1].kind != "moov" || boxes[len(boxes)-1].kind != "mdat" {
		t.Fatalf("box order = %v, want ftyp → moov → mdat", boxes)
	}
	if !containsBox(t, mp4, boxes[1], "avc1") {
		t.Fatal("moov missing avc1 sample entry")
	}
	if !containsBox(t, mp4, boxes[1], "mp4a") {
		t.Fatal("moov missing mp4a sample entry")
	}
	// stsz 总和 + 8 字节头 == mdat 大小。
	mdat := boxes[len(boxes)-1]
	if got := sumStszSizes(t, mp4) + 8; got != mdat.size {
		t.Fatalf("mdat size %d != stsz sum + 8 (%d)", mdat.size, got)
	}
	// stco 必须落在文件范围内。
	checkChunkOffsetsInBounds(t, mp4)
}

func TestBuildMP4VideoOnly(t *testing.T) {
	video, _ := buildTestTracks(t)
	mp4, err := buildMP4(video, nil)
	if err != nil {
		t.Fatalf("buildMP4: %v", err)
	}
	boxes := scanTopLevelBoxes(t, mp4)
	if boxes[1].kind != "moov" {
		t.Fatalf("box order = %v", boxes)
	}
	if !containsBox(t, mp4, boxes[1], "avc1") {
		t.Fatal("moov missing avc1")
	}
	if containsBox(t, mp4, boxes[1], "mp4a") {
		t.Fatal("video-only mp4 must not contain mp4a")
	}
}

func TestLayerCRejectsTruncatedMP4(t *testing.T) {
	video, audio := buildTestTracks(t)
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatal(err)
	}
	tmp := filepath.Join(t.TempDir(), "video.tmp")
	if err := os.WriteFile(tmp, mp4[:len(mp4)-100], 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateMP4File(tmp); err == nil {
		t.Fatal("truncated MP4 must be rejected")
	}
}

func TestLayerCRejectsOutOfBoundsChunkOffset(t *testing.T) {
	video, audio := buildTestTracks(t)
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatal(err)
	}
	// 篡改第一个 stco 条目为超大值(在 moov 全量字节中定位)。
	stcoOff := bytes.Index(mp4, []byte("stco"))
	if stcoOff < 0 {
		t.Fatal("stco not found")
	}
	overwriteUint32(mp4, stcoOff+16, 0xFFFFFFF0)
	tmp := filepath.Join(t.TempDir(), "video.tmp")
	if err := os.WriteFile(tmp, mp4, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateMP4File(tmp); err == nil {
		t.Fatal("out-of-bounds chunk offset must be rejected")
	}
}

func TestLayerCAcceptsValidFile(t *testing.T) {
	video, audio := buildTestTracks(t)
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatal(err)
	}
	tmp := filepath.Join(t.TempDir(), "video.tmp")
	if err := os.WriteFile(tmp, mp4, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateMP4File(tmp); err != nil {
		t.Fatalf("valid MP4 rejected: %v", err)
	}
}

// ---- 端到端:HLS → .mp4(sdk 契约 #16/#47) ----

func TestDownloadHLSToMP4ProducesFastStartFile(t *testing.T) {
	server := hlsMP4Server(t)
	defer server.Close()
	target := filepath.Join(t.TempDir(), "preview.mp4")
	written, err := DownloadHLS(context.Background(), hlsFetchFrom(server.URL), server.URL+"/video.m3u8", target)
	if err != nil {
		t.Fatalf("download HLS to mp4: %v", err)
	}
	if written == 0 {
		t.Fatal("no bytes written")
	}
	if err := validateMP4File(target); err != nil {
		t.Fatalf("mp4 validation: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	boxes := scanTopLevelBoxes(t, data)
	if boxes[1].kind != "moov" {
		t.Fatalf("box order = %v, want moov before mdat", boxes)
	}
	assertAACEsdsFullBox(t, data)
}

func TestDownloadHLSToMP4NormalizesSegmentTimestampReset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/video.m3u8":
			_, _ = writer.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:2\n#EXTINF:1.0,\nseg1.ts\n#EXTINF:1.0,\nseg2.ts\n#EXT-X-ENDLIST\n"))
		case "/seg1.ts", "/seg2.ts":
			// JavDB 上游部分预览会在新的 TS segment 重新从较小时间戳开始。
			// remux 必须把这种 segment 级回跳接回连续时间轴，而不是拒绝下载
			// 或把非单调 DTS/PTS 原样写入 MP4。
			_, _ = writer.Write(validTSSegmentAt(0))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "preview.mp4")
	if _, err := DownloadHLS(context.Background(), hlsFetchFrom(server.URL), server.URL+"/video.m3u8", target); err != nil {
		t.Fatalf("download HLS with segment timestamp reset: %v", err)
	}
	if err := validateMP4File(target); err != nil {
		t.Fatalf("normalized MP4 validation: %v", err)
	}
}

func TestDownloadMP4IgnoresStaleFixedSpool(t *testing.T) {
	server := hlsMP4Server(t)
	defer server.Close()
	target := filepath.Join(t.TempDir(), "preview.mp4")
	staleSpool := target + ".spool"
	if err := os.WriteFile(staleSpool, []byte("stale spool"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := DownloadHLS(context.Background(), hlsFetchFrom(server.URL), server.URL+"/video.m3u8", target); err != nil {
		t.Fatalf("download HLS with stale spool: %v", err)
	}
	if _, err := os.Stat(staleSpool); err != nil {
		t.Fatalf("stale spool must remain unrelated to new download: %v", err)
	}
}

// ---- ctx 贯穿：取消立即停止工作且不落盘 ----

func TestDownloadMovieAssetContextCancelled(t *testing.T) {
	server := hlsMP4Server(t)
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	target := filepath.Join(t.TempDir(), "preview.ts")
	_, err := DownloadHLS(ctx, hlsFetchFrom(server.URL), server.URL+"/video.m3u8", target)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatal("cancelled download must not publish output")
	}
}

// ---- helpers ----

type boxRef struct {
	kind string
	off  int
	size int
}

// hlsMP4Server 起一个 HLS 服务器,segment 是 Layer A/B 都认可的完整媒体流。
func hlsMP4Server(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/video.m3u8":
			_, _ = writer.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:2\n#EXTINF:1.0,\nseg1.ts\n#EXT-X-ENDLIST\n"))
		case "/seg1.ts":
			_, _ = writer.Write(validTSSegmentAt(0))
		default:
			http.NotFound(writer, request)
		}
	}))
}

// assertAACEsdsFullBox 验证最终 MP4 的 AAC esds 符合 ISO BMFF FullBox 布局：
// box header 后必须先有 version/flags，再进入 ES_Descriptor(0x03)。
func assertAACEsdsFullBox(t *testing.T, data []byte) {
	t.Helper()
	kind := bytes.Index(data, []byte("esds"))
	if kind < 4 || kind+9 > len(data) {
		t.Fatal("AAC esds box not found")
	}
	boxStart := kind - 4
	size := int(beU32(data[boxStart : boxStart+4]))
	if size < 13 || boxStart+size > len(data) {
		t.Fatalf("invalid esds box size %d", size)
	}
	if !bytes.Equal(data[kind+4:kind+8], []byte{0, 0, 0, 0}) {
		t.Fatalf("esds FullBox version/flags = % x, want 00 00 00 00", data[kind+4:kind+8])
	}
	if data[kind+8] != 0x03 {
		t.Fatalf("esds first descriptor tag = 0x%02x, want ES_Descriptor 0x03", data[kind+8])
	}
}

func boxKind(data []byte, off int) string {
	if off+8 > len(data) {
		return ""
	}
	return string(data[off+4 : off+8])
}

func scanTopLevelBoxes(t *testing.T, data []byte) []boxRef {
	t.Helper()
	var boxes []boxRef
	off := 0
	for off+8 <= len(data) {
		size := int(uint32(data[off])<<24 | uint32(data[off+1])<<16 | uint32(data[off+2])<<8 | uint32(data[off+3]))
		if size < 8 {
			t.Fatalf("invalid box size %d at %d", size, off)
		}
		boxes = append(boxes, boxRef{kind: boxKind(data, off), off: off, size: size})
		off += size
	}
	if off != len(data) {
		t.Fatalf("trailing %d bytes after last box", len(data)-off)
	}
	return boxes
}

// containsBox 在 parent 范围内查找 box kind(含嵌套);fullbox 内的
// sample entry(如 stsd→avc1)没有独立 size 头,直接字节搜索即可满足断言。
func containsBox(t *testing.T, data []byte, parent boxRef, kind string) bool {
	t.Helper()
	return bytes.Contains(data[parent.off+8:parent.off+parent.size], []byte(kind))
}

// sumStszSizes 汇总 moov 内全部 stsz 的样本字节总和(视频+音频)。
func sumStszSizes(t *testing.T, data []byte) int {
	t.Helper()
	total := 0
	_ = walkBoxes(data, 0, int64(len(data)), func(ref boxWalker) error {
		if ref.kind != "stsz" {
			return nil
		}
		pos := int(ref.off)
		count := int(uint32(data[pos+16])<<24 | uint32(data[pos+17])<<16 | uint32(data[pos+18])<<8 | uint32(data[pos+19]))
		for i := 0; i < count; i++ {
			off := pos + 20 + i*4
			if off+4 > len(data) {
				t.Fatal("stsz entries truncated")
			}
			total += int(uint32(data[off])<<24 | uint32(data[off+1])<<16 | uint32(data[off+2])<<8 | uint32(data[off+3]))
		}
		return nil
	})
	return total
}

// checkChunkOffsetsInBounds 汇总全部 stco 条目,断言落在文件范围内。
func checkChunkOffsetsInBounds(t *testing.T, data []byte) {
	t.Helper()
	_ = walkBoxes(data, 0, int64(len(data)), func(ref boxWalker) error {
		if ref.kind != "stco" {
			return nil
		}
		pos := int(ref.off)
		count := int(uint32(data[pos+12])<<24 | uint32(data[pos+13])<<16 | uint32(data[pos+14])<<8 | uint32(data[pos+15]))
		for i := 0; i < count; i++ {
			off := pos + 16 + i*4
			if off+4 > len(data) {
				t.Fatal("stco entries truncated")
			}
			value := int(uint32(data[off])<<24 | uint32(data[off+1])<<16 | uint32(data[off+2])<<8 | uint32(data[off+3]))
			if value < 0 || value >= len(data) {
				t.Fatalf("chunk offset %d out of file bounds (%d bytes)", value, len(data))
			}
		}
		return nil
	})
}

func overwriteUint32(data []byte, off int, value uint32) {
	data[off] = byte(value >> 24)
	data[off+1] = byte(value >> 16)
	data[off+2] = byte(value >> 8)
	data[off+3] = byte(value)
}

// hlsFetchFrom 把服务器根地址包装成 FetchContext。
func hlsFetchFrom(base string) FetchContext {
	return byteFetch(func(ctx context.Context, uri string) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, errors.New("HTTP " + resp.Status)
		}
		buf := make([]byte, 0, 4096)
		tmp := make([]byte, 4096)
		for {
			n, readErr := resp.Body.Read(tmp)
			buf = append(buf, tmp[:n]...)
			if readErr != nil {
				break
			}
		}
		return buf, nil
	})
}

// ---- High Profile SPS scaling matrix 是 1-bit flag ----

// seq_scaling_matrix_present_flag 是 1 bit(readBit),不是 Exp-Golomb;
// 每一个 seq_scaling_list_present_flag 同样是 1 bit。
// 构造 High Profile(100)SPS:chroma=1、bit_depth 8/8、qpprime=0、
// scaling_matrix_present=1 后跟 8 个 list_present_flag=0,
// 之后正常字段 → 尺寸 640x480。
func TestParseSPSHighProfileScalingMatrixFlags(t *testing.T) {
	// bit 流(从 profile_idc 开始):profile=100, constraint=0, level=40,
	// sps_id=ue(0), chroma_format=ue(1), bit_depth=ue(0)x2, qpprime=1bit 0,
	// scaling_matrix_present=1bit 1, 8x list_present=1bit 0,
	// log2_max_frame_num=ue(0), pic_order=ue(0), max_ref=ue(1), gaps=1bit 0,
	// width_mbs=ue(39), height_map=ue(29), frame_only=1bit 1, direct8x8=0, crop=0
	bits := ""
	w := func(bitstr string) { bits += bitstr }
	ue := func(v uint64) string {
		// Exp-Golomb
		n := v + 1
		length := 0
		for tmp := n; tmp > 1; tmp >>= 1 {
			length++
		}
		return strings.Repeat("0", length) + strconv.FormatUint(n, 2)
	}
	w("01100100") // profile_idc = 100
	w("00000000") // constraint flags
	w("00101000") // level_idc = 40
	w(ue(0))      // sps_id
	w(ue(1))      // chroma_format_idc = 1
	w(ue(0))      // bit_depth_luma_minus8
	w(ue(0))      // bit_depth_chroma_minus8
	w("0")        // qpprime_y_zero_transform_bypass
	w("1")        // seq_scaling_matrix_present_flag(1 bit!)
	for i := 0; i < 8; i++ {
		w("0") // seq_scaling_list_present_flag × 8(each 1 bit)
	}
	w(ue(0))  // log2_max_frame_num_minus4
	w(ue(0))  // pic_order_cnt_type
	w(ue(0))  // log2_max_pic_order_cnt_lsb_minus4(pic_order=0 的必读字段)
	w(ue(1))  // max_num_ref_frames
	w("0")    // gaps_in_frame_num
	w(ue(39)) // pic_width_in_mbs_minus8 → (39+1)*16=640
	w(ue(29)) // pic_height_in_map_units_minus8 → (29+1)*16=480
	w("1")    // frame_mbs_only_flag
	w("0")    // direct_8x8_inference
	w("0")    // frame_cropping

	// 组字节
	nal := []byte{0x67}
	var current byte
	var count uint
	flush := func() {
		if count > 0 {
			nal = append(nal, current<<(8-count))
			current = 0
			count = 0
		}
	}
	for _, c := range bits {
		current = current<<1 | byte(c-'0')
		count++
		if count == 8 {
			flush()
		}
	}
	flush()

	width, height, err := parseSPSDimensions(nal)
	if err != nil {
		t.Fatalf("parseSPSDimensions: %v", err)
	}
	if width != 640 || height != 480 {
		t.Fatalf("dimensions = %dx%d, want 640x480", width, height)
	}
}
