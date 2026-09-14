package media

import (
	"bytes"
	"encoding/binary"
	"os"
	"sort"
	"strings"
	"testing"
)

// MP4 结构契约(计划 #18-#26/#30):
// mvhd/tkhd/mdhd 按 ISO BMFF 字段布局;track ID 唯一(video=1, audio=2);
// stts/ctts 带 sample_count 的 RLE entry;stss 语义正确;avcC 分 SPS[]/PPS[];
// mdat A/V chunk interleave;32-bit 溢出明确拒绝。

// parseTrackBoxes 在 moov 内定位全部 trak 的关键字段。
type trackBoxes struct {
	kind    string // vide / soun
	tkhdDur uint64 // movie timescale
	mdhdTS  uint32
	mdhdDur uint32
	sttsRaw []byte
	cttsRaw []byte // nil 表示省略
	stssRaw []byte // nil 表示省略
	avcC    []byte
}

func parseMP4Tracks(t *testing.T, mp4 []byte) []trackBoxes {
	t.Helper()
	var tracks []trackBoxes
	_ = walkBoxes(mp4, 0, int64(len(mp4)), func(ref boxWalker) error {
		if ref.kind != "trak" {
			return nil
		}
		trak := mp4[ref.off : ref.off+int64(boxSize(mp4, int(ref.off)))]
		tb := trackBoxes{}
		_ = walkBoxes(trak, 8, int64(len(trak)), func(tref boxWalker) error {
			switch tref.kind {
			case "tkhd":
				// box头8 + ver/flags4 + ctime4 + mtime4 + track_ID4 + reserved4 → duration 在 +28
				tb.tkhdDur = uint64(beU32(trak[tref.off+28 : tref.off+32]))
			case "mdhd":
				tb.mdhdTS = beU32(trak[tref.off+20 : tref.off+24])
				tb.mdhdDur = beU32(trak[tref.off+24 : tref.off+28])
			case "stts":
				tb.sttsRaw = append([]byte(nil), trak[tref.off:tref.off+int64(boxSize(trak, int(tref.off)))]...)
			case "ctts":
				tb.cttsRaw = append([]byte(nil), trak[tref.off:tref.off+int64(boxSize(trak, int(tref.off)))]...)
			case "stss":
				tb.stssRaw = append([]byte(nil), trak[tref.off:tref.off+int64(boxSize(trak, int(tref.off)))]...)
			case "avcC":
				tb.avcC = append([]byte(nil), trak[tref.off+8:tref.off+int64(boxSize(trak, int(tref.off)))]...)
			case "hdlr":
				tb.kind = string(trak[tref.off+16 : tref.off+20])
			}
			return nil
		})
		tracks = append(tracks, tb)
		return nil
	})
	return tracks
}

func boxSize(data []byte, off int) uint32 {
	if off+4 > len(data) {
		return 0
	}
	return binary.BigEndian.Uint32(data[off : off+4])
}

// ---- #21:unitMatrix 必须是完整单位矩阵(中间对角项 0x00010000) ----

func TestUnitMatrixIsCompleteIdentity(t *testing.T) {
	m := unitMatrix()
	if len(m) != 36 {
		t.Fatalf("matrix length = %d, want 36", len(m))
	}
	u32 := func(off int) uint32 {
		return binary.BigEndian.Uint32(m[off : off+4])
	}
	// a=0x00010000, b/u=0, c/v=0, d=0x00010000(中间对角项), e/w=0, f=0x40000000
	if got := u32(0); got != 0x00010000 {
		t.Fatalf("a = 0x%08X, want 0x00010000", got)
	}
	if got := u32(12); got != 0x00010000 {
		t.Fatalf("d(中间对角项) = 0x%08X, want 0x00010000", got)
	}
	if got := u32(32); got != 0x40000000 {
		t.Fatalf("f = 0x%08X, want 0x40000000", got)
	}
	// 其余项必须为 0。
	for _, off := range []int{4, 8, 16, 20, 24, 28} {
		if got := u32(off); got != 0 {
			t.Fatalf("matrix[%d] = 0x%08X, want 0", off, got)
		}
	}
}

// ---- #21/#22:mvhd/tkhd/mdhd 布局与时间基 ----

func TestMP4TrackIDsAndTimescales(t *testing.T) {
	video, audio := buildTestTracks(t)
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatalf("buildMP4: %v", err)
	}
	// track_ID 通过 tkhd 校验:video=1, audio=2。
	trackIDs := tTrackIDs(t, mp4)
	if len(trackIDs) != 2 {
		t.Fatalf("track count = %d, want 2", len(trackIDs))
	}
	if trackIDs[0] != 1 {
		t.Fatalf("video track_ID = %d, want 1", trackIDs[0])
	}
	if trackIDs[1] != 2 {
		t.Fatalf("audio track_ID = %d, want 2", trackIDs[1])
	}
	// mdhd:video=90000,audio=sample rate;movie timescale 独立。
	tracks := parseMP4Tracks(t, mp4)
	if tracks[0].mdhdTS != 90000 {
		t.Fatalf("video mdhd timescale = %d, want 90000", tracks[0].mdhdTS)
	}
	if tracks[1].mdhdTS != uint32(audio.SampleRate) {
		t.Fatalf("audio mdhd timescale = %d, want %d", tracks[1].mdhdTS, audio.SampleRate)
	}
	// tkhd duration 必须换算到 movie timescale,不能直接用 video 90k 时间。
	if tracks[0].tkhdDur == 0 {
		t.Fatal("tkhd duration is zero")
	}
}

func tTrackIDs(t *testing.T, mp4 []byte) []uint32 {
	t.Helper()
	var ids []uint32
	_ = walkBoxes(mp4, 0, int64(len(mp4)), func(ref boxWalker) error {
		if ref.kind != "tkhd" {
			return nil
		}
		// box 头 8 + ver/flags 4 + ctime 4 + mtime 4 → track_ID 在 +20。
		ids = append(ids, beU32(mp4[ref.off+20:ref.off+24]))
		return nil
	})
	return ids
}

// ---- #19:stts 必须是 {sample_count, sample_delta} 的 RLE entry ----

func TestMP4STTSHasSampleCountEntries(t *testing.T) {
	video, audio := buildTestTracks(t)
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatalf("buildMP4: %v", err)
	}
	tracks := parseMP4Tracks(t, mp4)
	// 测试流两帧 DTS delta 相同:1 个 entry {2, 3600},而不是 {entry_count=1, delta...}。
	vStts := tracks[0].sttsRaw
	if vStts == nil {
		t.Fatal("video track missing stts")
	}
	entryCount := beU32(vStts[12:16])
	if entryCount != 1 {
		t.Fatalf("stts entry_count = %d, want 1 (RLE of equal deltas)", entryCount)
	}
	sampleCount := beU32(vStts[16:20])
	sampleDelta := beU32(vStts[20:24])
	if sampleCount != 2 || sampleDelta != 3600 {
		t.Fatalf("stts entry = {%d, %d}, want {2, 3600}", sampleCount, sampleDelta)
	}
	// 音频 track 同理:N 个 1024 delta → 1 个 entry。
	aStts := tracks[1].sttsRaw
	entryCount = beU32(aStts[12:16])
	sampleCount = beU32(aStts[16:20])
	sampleDelta = beU32(aStts[20:24])
	if entryCount != 1 || sampleCount != 1 || sampleDelta != 1024 {
		t.Fatalf("audio stts entry = {%d, {%d, %d}}, want {1, {1, 1024}}", entryCount, sampleCount, sampleDelta)
	}
}

// ---- #20:ctts RLE + signed version 1;全零省略 ----

func TestMP4CTTSOmittedWhenAllZero(t *testing.T) {
	video, audio := buildTestTracks(t)
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatalf("buildMP4: %v", err)
	}
	tracks := parseMP4Tracks(t, mp4)
	// 测试流 PTS==DTS:ctts 必须省略。
	if tracks[0].cttsRaw != nil {
		t.Fatalf("ctts must be omitted when all offsets are zero, got % X", tracks[0].cttsRaw[:min(20, len(tracks[0].cttsRaw))])
	}
	_ = audio
}

// B 帧场景:PTS > DTS 的正 offset 用 ctts version 0 RLE。
func TestMP4CTTSRLEPositiveOffsets(t *testing.T) {
	// 手工构造 3 帧:sample1 PTS=DTS, sample2 PTS=DTS+3600(B 帧), sample3 PTS=DTS+1800。
	video := &h264Track{
		ParamSets: [][]byte{
			{0x67, 0x42, 0x00, 0x28, 0xF8, 0x14, 0x07, 0xB2},
			{0x68, 0xCC, 0xDD},
		},
		Samples: []h264Sample{
			{Data: avccFrame(0x65, 0x01), PTS: 0, DTS: 0, Sync: true},
			{Data: avccFrame(0x41, 0x02), PTS: 7200, DTS: 3600},
			{Data: avccFrame(0x41, 0x03), PTS: 3600, DTS: 7200},
		},
	}
	meta, err := planVideoTrack(video)
	if err != nil {
		t.Fatalf("planVideoTrack: %v", err)
	}
	stts, ctts, stss := buildVideoTimingBoxes(meta.Samples)
	// stts:delta 3600,3600,3600 → 1 个 entry {3, 3600}。
	if ec := beU32(stts[12:16]); ec != 1 {
		t.Fatalf("stts entry_count = %d, want 1", ec)
	}
	if ctts == nil {
		t.Fatal("B-frame samples must produce ctts")
	}
	version := ctts[8]
	if version != 1 {
		t.Fatalf("ctts version = %d, want 1 (sample3 PTS < DTS is a negative offset)", version)
	}
	entryCount := beU32(ctts[12:16])
	if entryCount != 3 {
		t.Fatalf("ctts entry_count = %d, want 3 (offsets 0,7200,-3600 are distinct)", entryCount)
	}
	_ = stss
}

// ---- #23:完全没有 sync sample 时必须拒绝生成 MP4 ----

func TestPlanVideoTrackRejectsNoSyncSamples(t *testing.T) {
	video := &h264Track{
		ParamSets: [][]byte{
			{0x67, 0x42, 0x00, 0x28, 0xF8, 0x14, 0x07, 0xB2},
			{0x68, 0xCC, 0xDD},
		},
		Samples: []h264Sample{
			{Data: avccFrame(0x41, 0x01), PTS: 0, DTS: 0, Sync: false},
			{Data: avccFrame(0x41, 0x02), PTS: 3600, DTS: 3600, Sync: false},
		},
	}
	_, err := planVideoTrack(video)
	if err == nil || !strings.Contains(err.Error(), "sync") {
		t.Fatalf("error = %v, want no sync sample rejection", err)
	}
}

// ---- #24:avcC 分 SPS[]/PPS[] 收集 ----

func TestBuildAVCCCollectsSPSAndPPSSeparately(t *testing.T) {
	sps1 := []byte{0x67, 0x42, 0x00, 0x28}
	sps2 := []byte{0x67, 0x42, 0x00, 0x29}
	pps1 := []byte{0x68, 0xCC, 0xDD}
	pps2 := []byte{0x68, 0xCC, 0xEE}
	avcC, err := buildAVCC([][]byte{sps1, pps1, sps2, pps2})
	if err != nil {
		t.Fatalf("buildAVCC: %v", err)
	}
	if len(avcC) < 8 {
		t.Fatalf("avcC too short: % X", avcC)
	}
	payload := avcC[8:]
	if payload[0] != 0x01 {
		t.Fatalf("configurationVersion = %d", payload[0])
	}
	numSPS := payload[5] & 0x1F
	if numSPS != 2 {
		t.Fatalf("numOfSequenceParameterSets = %d, want 2", numSPS)
	}
	// 第一个 SPS 长度 = 4。
	spsLen := int(binary.BigEndian.Uint16(payload[6:8]))
	if spsLen != len(sps1) {
		t.Fatalf("first SPS length = %d, want %d", spsLen, len(sps1))
	}
	// 扫描到 PPS 区:numOfPictureParameterSets 必须为 2。
	// payload 布局:[0]=configVersion,[1..3] profile/compat/level,[4]=0xFF,
	// [5]=0xE0|numSPS,之后逐 SPS(len 2B + 数据),最后是 numPPS 计数。
	pos := 6 + (2 + len(sps1)) + (2 + len(sps2))
	if payload[pos] != byte(len(pps1)+len(pps2)) && payload[pos] != 2 {
		t.Fatalf("expected numOfPictureParameterSets at %d, got 0x%02X", pos, payload[pos])
	}
	numPPS := payload[pos]
	if numPPS != 2 {
		t.Fatalf("numOfPictureParameterSets = %d, want 2", numPPS)
	}
}

// ---- #18:video-only MP4 正常(audio 为 nil) ----

func TestBuildMP4VideoOnlyNoPanic(t *testing.T) {
	video, _ := buildTestTracks(t)
	mp4, err := buildMP4(video, nil)
	if err != nil {
		t.Fatalf("buildMP4 video-only: %v", err)
	}
	if len(mp4) == 0 {
		t.Fatal("video-only mp4 is empty")
	}
	tracks := parseMP4Tracks(t, mp4)
	if len(tracks) != 1 || tracks[0].kind != "vide" {
		t.Fatalf("video-only tracks = %v", tracks)
	}
}

// ---- #26:32-bit 溢出明确拒绝 ----

func TestBuildMoovRejects32BitOverflow(t *testing.T) {
	// mdatStart 接近 4 GiB:stco 偏移会溢出 32-bit。
	videoRaw, audioRaw := buildTestTracks(t)
	video, err := planVideoTrack(videoRaw)
	if err != nil {
		t.Fatal(err)
	}
	audio, err := planAudioTrack(audioRaw)
	if err != nil {
		t.Fatal(err)
	}
	_, err = buildMoov(video, audio, int64(0xFFFFFFF0))
	if err == nil || !strings.Contains(err.Error(), "32") {
		t.Fatalf("error = %v, want 32-bit overflow rejection", err)
	}
}

// ---- #25:mdat A/V chunk interleave ----

// interleaveSamples 单元级:长视频(多 GOP)+ 长音频(多秒)必须产生
// 多 chunk 交错,且 chunk 首样本 Offset 单调递增、覆盖全部样本。
func TestInterleaveSamplesProducesAVChunks(t *testing.T) {
	video := &mp4TrackMeta{
		Timescale: 90000,
		Width:     64, Height: 64,
		ParamSets: [][]byte{{0x67, 0x42}, {0x68, 0xCC}},
	}
	// 4 个 GOP,每 GOP 2 帧,每帧 3000 字节;PTS 每 3600 推进。
	sampleSize := uint32(3000)
	for gop := 0; gop < 4; gop++ {
		for i := 0; i < 2; i++ {
			video.Samples = append(video.Samples, mp4SampleMeta{
				Size: sampleSize,
				PTS:  uint64(gop*2+i) * 3600,
				DTS:  uint64(gop*2+i) * 3600,
				Sync: i == 0,
			})
		}
	}
	audio := &mp4TrackMeta{
		Timescale:  48000,
		SampleRate: 48000,
	}
	// 4 秒音频,每秒 46 个样本,每个 400 字节。
	for i := 0; i < 46*4; i++ {
		audio.Samples = append(audio.Samples, mp4SampleMeta{
			Size: 400,
			PTS:  uint64(i) * 48000 / 46,
		})
	}
	const mdatStart = 1000000
	videoChunks, audioChunks := interleaveSamples(video, audio, mdatStart)
	if len(videoChunks) < 2 {
		t.Fatalf("video chunks = %d, want GOP-level chunks (>= 2)", len(videoChunks))
	}
	if len(audioChunks) < 2 {
		t.Fatalf("audio chunks = %d, want time-window chunks (>= 2)", len(audioChunks))
	}
	// 全部样本被覆盖且 Offset 单调(与 writer 相同:按 Offset 排序后校验)。
	allChunks := append(videoChunks, audioChunks...)
	sort.Slice(allChunks, func(i, j int) bool { return allChunks[i].offset < allChunks[j].offset })
	sampleTotal := 0
	prev := int64(mdatStart - 1)
	for _, chunk := range allChunks {
		if chunk.offset < prev {
			t.Fatalf("chunk offset %d < previous %d (interleave must be monotonic)", chunk.offset, prev)
		}
		prev = chunk.offset
		for range chunk.samples {
			sampleTotal++
		}
	}
	if sampleTotal != len(video.Samples)+len(audio.Samples) {
		t.Fatalf("chunk samples = %d, want %d", sampleTotal, len(video.Samples)+len(audio.Samples))
	}
	// 每个样本的 Offset 都被重分配为交错后的绝对位置。
	for i, s := range video.Samples {
		if s.Offset < mdatStart {
			t.Fatalf("video sample %d Offset = %d, want >= mdatStart", i, s.Offset)
		}
	}
	for i, s := range audio.Samples {
		if s.Offset < mdatStart {
			t.Fatalf("audio sample %d Offset = %d, want >= mdatStart", i, s.Offset)
		}
	}
}

// ---- #30:validator 独立检查 stts/ctts/stco/track ID/interleave ----

func TestValidateMP4RejectsBadSTTSEntryCount(t *testing.T) {
	video, audio := buildTestTracks(t)
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatal(err)
	}
	// 篡改 video stts 的 entry_count 为超大值,使 stts entries 越出 box。
	stcoOff := bytes.Index(mp4, []byte("stts"))
	if stcoOff < 0 {
		t.Fatal("stts not found")
	}
	overwriteUint32(mp4, stcoOff+8, 0xFFFFFFF0)
	tmp := t.TempDir() + "/v.mp4"
	if err := writeFile(tmp, mp4); err != nil {
		t.Fatal(err)
	}
	if err := validateMP4File(tmp); err == nil {
		t.Fatal("oversized stts entry_count must be rejected")
	}
}

func TestValidateMP4RejectsSingleTrackIDCollision(t *testing.T) {
	// 两个 track 都用 track ID 1:validator 必须拒绝。
	video, audio := buildTestTracks(t)
	mp4, err := buildMP4(video, audio)
	if err != nil {
		t.Fatal(err)
	}
	// 把第二个 tkhd 的 track_ID 改成 1(与第一个冲突)。
	seen := 0
	_ = walkBoxes(mp4, 0, int64(len(mp4)), func(ref boxWalker) error {
		if ref.kind != "tkhd" {
			return nil
		}
		seen++
		if seen == 2 {
			overwriteUint32(mp4, int(ref.off)+20, 1)
		}
		return nil
	})
	tmp := t.TempDir() + "/v.mp4"
	if err := writeFile(tmp, mp4); err != nil {
		t.Fatal(err)
	}
	if err := validateMP4File(tmp); err == nil {
		t.Fatal("duplicate track ID must be rejected")
	}
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func avccFrame(nalType, payload byte) []byte {
	nal := []byte{(nalType & 0x1F) | 0x60, payload, payload}
	out := make([]byte, 4+len(nal))
	binary.BigEndian.PutUint32(out, uint32(len(nal)))
	copy(out[4:], nal)
	return out
}
