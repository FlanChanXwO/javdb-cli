package media

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// Layer A segment 完整性契约(input.md 计划 #29-#32/#58):
// 每个 segment 下载→解密后必须通过结构校验(188 对齐、0x47、PAT/PMT 可解析、
// PES 前缀、明显截断)才能进入输出;损坏 segment 重试同一 segment 至多 3 次。

// ---- 最小确定性 TS fixture 构造(PAT → PMT → PES,单 packet PSI,0xFF 填充) ----

const (
	patPID   = 0x0000
	pmtPID   = 0x1000
	videoPID = 0x0101
	audioPID = 0x0102
)

// tsPacket 构造 payload-only 的 188 字节 TS 包(尾部 0xFF stuffing)。
func tsPacket(pid uint16, pusi bool, cc byte, payload []byte) []byte {
	if len(payload) > 184 {
		panic("payload too large for a single TS packet")
	}
	p := make([]byte, 188)
	p[0] = 0x47
	p[1] = byte(pid >> 8)
	if pusi {
		p[1] |= 0x40
	}
	p[2] = byte(pid)
	p[3] = 0x10 | (cc & 0x0F)
	copy(p[4:], payload)
	for i := 4 + len(payload); i < 188; i++ {
		p[i] = 0xFF
	}
	return p
}

// psiSection 构造 pointer_field + section(table_id/length/body/CRC 占位)。
func psiSection(tableID byte, body []byte) []byte {
	length := len(body) + 4 // CRC32 占位
	out := []byte{0x00, tableID, byte(length>>8)&0x0F | 0x30, byte(length)}
	out = append(out, body...)
	out = append(out, 0, 0, 0, 0)
	return out
}

// patSection:program 1 → PMT PID 0x1000。
func patSection() []byte {
	return psiSection(0x00, []byte{
		0x00, 0x01, // transport_stream_id
		0xC1, 0x00, 0x00, // version/section/last
		0x00, 0x01, // program_number 1
		0xE0 | byte(pmtPID>>8), byte(pmtPID & 0xFF),
	})
}

// pmtSection:H.264(0x1B)@videoPID + AAC ADTS(0x0F)@audioPID。
func pmtSection() []byte {
	return psiSection(0x02, []byte{
		0x00, 0x01, // program_number
		0xC1, 0x00, 0x00,
		0xE0 | byte(videoPID>>8), byte(videoPID & 0xFF), // PCR PID
		0xF0, 0x00, // program_info_length
		0x1B, 0xE0 | byte(videoPID>>8), byte(videoPID & 0xFF), 0xF0, 0x00,
		0x0F, 0xE0 | byte(audioPID>>8), byte(audioPID & 0xFF), 0xF0, 0x00,
	})
}

// videoPES:最小 ES(无 Annex-B),仅用于 Layer A 结构校验。
// validTSSegment 构造一个 Layer B 也认可的完整媒体 segment:
// PAT/PMT + 两个带 PTS 的 H.264 帧(SPS+PPS+IDR / 非 IDR)+ 一个 AAC ADTS 帧。
func validTSSegment() []byte { return validTSSegmentAt(0) }

func validTSSegmentAt(base uint64) []byte {
	sps := h264Frame(7, []byte{0xAA, 0xBB})
	pps := h264Frame(8, []byte{0xCC, 0xDD})
	frame0 := append(append(append([]byte{}, sps...), pps...), h264Frame(5, []byte{0x01, 0x02})...)
	frame1 := h264Frame(1, []byte{0x03, 0x04})
	var data []byte
	data = append(data, tsPacket(patPID, true, 0, patSection())...)
	data = append(data, tsPacket(pmtPID, true, 0, pmtSection())...)
	data = append(data, tsPacket(videoPID, true, 1, pesBytes(0xE0, base+90000, base+90000, true, frame0))...)
	data = append(data, tsPacket(videoPID, true, 2, pesBytes(0xE0, base+93600, base+93600, true, frame1))...)
	data = append(data, tsPacket(audioPID, true, 3, pesBytes(0xC0, base+90000, 0, false, adtsFrame()))...)
	return data
}

// ---- validateTSSegment 表驱动(input.md #58) ----

func TestValidateTSSegment(t *testing.T) {
	valid := validTSSegment()
	brokenPES := append([]byte{}, valid...)
	copy(brokenPES[2*188+4:], []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x00}) // 覆盖视频 PES 前缀
	noPMT := append([]byte{}, valid[:188]...)                             // 只剩 PAT
	noPAT := append([]byte{}, valid[188:]...)                             // 没有 PAT
	truncated := append([]byte{}, valid[:len(valid)-100]...)
	badSync := append([]byte{}, valid...)
	badSync[188*2] = 0x66 // 第三个 packet 同步字节损坏

	cases := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{"valid segment", valid, ""},
		{"empty", nil, "empty"},
		{"truncated packet", truncated, "not 188-byte aligned"},
		{"invalid sync byte", badSync, "invalid sync byte"},
		{"missing PAT", noPAT, "no parseable PAT"},
		{"missing PMT", noPMT, "declares no elementary streams"},
		{"broken PES prefix", brokenPES, "PES"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTSSegment(tc.data)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected valid, error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want contains %q", err, tc.wantErr)
			}
		})
	}
}

// ---- DownloadHLS 的 Layer A 集成:per-segment 校验 + bounded retry + 原子发布 ----

func hlsFetch(resources map[string][]byte) Fetch {
	return func(uri string) ([]byte, error) {
		body, ok := resources[uri]
		if !ok {
			return nil, errors.New("unexpected media URI " + uri)
		}
		return body, nil
	}
}

func TestDownloadHLSRetriesInvalidSegmentThenSucceeds(t *testing.T) {
	const playlistURL = "https://media.example.test/previews/index.m3u8"
	good := validTSSegment()
	attempts := 0
	fetch := func(uri string) ([]byte, error) {
		switch uri {
		case playlistURL:
			return []byte("#EXTM3U\n#EXT-X-TARGETDURATION:2\n#EXTINF:1.0,\ns1.ts\n#EXT-X-ENDLIST\n"), nil
		case "https://media.example.test/previews/s1.ts":
			attempts++
			if attempts == 1 {
				return []byte("HTML error page from CDN"), nil
			}
			return good, nil
		default:
			return nil, errors.New("unexpected URI")
		}
	}
	target := t.TempDir() + "/preview.ts"
	written, err := downloadHLS(fetch, playlistURL, target)
	if err != nil {
		t.Fatalf("download HLS: %v", err)
	}
	if written != int64(len(good)) {
		t.Fatalf("written = %d, want %d", written, len(good))
	}
}

func TestDownloadHLSFailsAfterExhaustedSegmentRetries(t *testing.T) {
	const playlistURL = "https://media.example.test/previews/index.m3u8"
	fetch := func(uri string) ([]byte, error) {
		switch uri {
		case playlistURL:
			return []byte("#EXTM3U\n#EXT-X-MEDIA-SEQUENCE:17\n#EXT-X-TARGETDURATION:2\n#EXTINF:1.0,\ns1.ts\n#EXT-X-ENDLIST\n"), nil
		case "https://media.example.test/previews/s1.ts":
			return []byte("garbage"), nil
		default:
			return nil, errors.New("unexpected URI")
		}
	}
	target := t.TempDir() + "/preview.ts"
	_, err := downloadHLS(fetch, playlistURL, target)
	if err == nil || !strings.Contains(err.Error(), "segment 17 remained invalid after 3 attempts") {
		t.Fatalf("error = %v, want exhausted retries", err)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("failed download left output: %v", statErr)
	}
	if _, statErr := os.Stat(target + ".part"); !os.IsNotExist(statErr) {
		t.Fatalf("failed download left temp file: %v", statErr)
	}
}

func TestDownloadHLSAcceptsDiscontinuityPlaylist(t *testing.T) {
	const playlistURL = "https://media.example.test/previews/index.m3u8"
	resources := map[string][]byte{
		playlistURL: []byte("#EXTM3U\n#EXT-X-TARGETDURATION:2\n#EXTINF:1.0,\na.ts\n#EXT-X-DISCONTINUITY\n#EXTINF:1.0,\nb.ts\n#EXT-X-ENDLIST\n"),
		"https://media.example.test/previews/a.ts": validTSSegmentAt(0),
		"https://media.example.test/previews/b.ts": validTSSegmentAt(180000),
	}
	target := t.TempDir() + "/preview.ts"
	written, err := downloadHLS(hlsFetch(resources), playlistURL, target)
	if err != nil {
		t.Fatalf("download HLS: %v", err)
	}
	if written != int64(2*len(validTSSegmentAt(0))) {
		t.Fatalf("written = %d, want %d", written, 2*len(validTSSegmentAt(0)))
	}
}

func TestDownloadHLSPublishesWithoutOverwriteAndLeavesNoTemp(t *testing.T) {
	const playlistURL = "https://media.example.test/previews/index.m3u8"
	resources := map[string][]byte{
		playlistURL: []byte("#EXTM3U\n#EXT-X-TARGETDURATION:2\n#EXTINF:1.0,\ns1.ts\n#EXT-X-ENDLIST\n"),
		"https://media.example.test/previews/s1.ts": validTSSegment(),
	}
	dir := t.TempDir()
	target := dir + "/preview.ts"
	if _, err := downloadHLS(hlsFetch(resources), playlistURL, target); err != nil {
		t.Fatalf("download HLS: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "preview.ts" {
		t.Fatalf("dir contents = %v, want only preview.ts", entries)
	}
	// 已存在的目标绝不被覆盖。
	if err := os.WriteFile(target, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := downloadHLS(hlsFetch(resources), playlistURL, target); err == nil {
		t.Fatal("expected failure for existing target")
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "keep" {
		t.Fatalf("existing target must stay untouched: %v %q", err, data)
	}
}

// 最终媒体校验(Layer B)在 .ts 发布前拦截结构合法但 codec 不支持的流。
func TestDownloadHLSRejectsUnsupportedCodecAtFinalValidation(t *testing.T) {
	hevcPMT := psiSection(0x02, []byte{
		0x00, 0x01, 0xC1, 0x00, 0x00, 0xE1, 0x01, 0xF0, 0x00,
		0x24, 0xE1, 0x01, 0xF0, 0x00,
	})
	hevcSegment := func() []byte {
		var data []byte
		data = append(data, tsPacket(patPID, true, 0, patSection())...)
		data = append(data, tsPacket(pmtPID, true, 0, hevcPMT)...)
		data = append(data, tsPacket(videoPID, true, 1, pesBytes(0xE0, 90000, 90000, false, h264Frame(5, []byte{0x01, 0x02})))...)
		return data
	}
	resources := map[string][]byte{
		"https://media.example.test/previews/hevc.m3u8": []byte("#EXTM3U\n#EXT-X-TARGETDURATION:2\n#EXTINF:1.0,\ns1.ts\n#EXT-X-ENDLIST\n"),
		"https://media.example.test/previews/s1.ts":     hevcSegment(),
	}
	target := t.TempDir() + "/preview.ts"
	_, err := downloadHLS(hlsFetch(resources), "https://media.example.test/previews/hevc.m3u8", target)
	if err == nil || !strings.Contains(err.Error(), "unsupported video codec") {
		t.Fatalf("error = %v, want unsupported video codec", err)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("rejected media left output: %v", statErr)
	}
}
